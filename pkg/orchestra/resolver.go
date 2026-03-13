package orchestra

import (
	"context"
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/pkgmanager"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/resolver"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

type ModuleResolver struct {
	pm        *pkgmanager.PackageManager
	cave      cavefile.Cavefile
	installed []registry.ResolvedPackage
	modules   map[registry.LogicalURI]*ast.ContextModule
	prelude   *ast.ContextModule
	ready     bool
}

func NewModuleResolver(pm *pkgmanager.PackageManager, cave cavefile.Cavefile) (*ModuleResolver, error) {
	if cave.Name == "" && cave.Source == "" {
		return nil, fmt.Errorf("cavefile must have a name or source to resolve modules")
	}
	if cave.Name == "" {
		cave.Name = string(registry.CanonicalizeModuleSource(cave.Source))
	}

	return &ModuleResolver{
		pm:      pm,
		cave:    cave,
		modules: map[registry.LogicalURI]*ast.ContextModule{},
	}, nil
}

var _ resolver.ModuleResolver = (*ModuleResolver)(nil)

func (r *ModuleResolver) RegisterModule(uri registry.LogicalURI, module *ast.ContextModule) {
	r.modules[uri] = module
}

func (r *ModuleResolver) MainModule() *ast.ContextModule {
	return r.modules[registry.LogicalURI(r.cave.Name)]
}

func (r *ModuleResolver) ensureDependencies(ctx context.Context) ([]registry.ResolvedPackage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Always install the project package itself so its modules are discoverable
	// in r.installed, regardless of whether any declared dependencies are missing.
	caveOnly := cavefile.Cavefile{Package: r.cave.Package}
	projectTask := r.pm.Install(caveOnly)
	projectPkgs, err := projectTask.Run(ctx)
	if err != nil {
		return nil, err
	}
	r.installed = append(r.installed, projectPkgs...)

	// Install any declared dependencies that are not yet satisfied.
	missing := r.filterMissingDependencies(r.cave.Dependencies)
	if len(missing) == 0 {
		return r.installed, nil
	}
	depTask := r.pm.Install(cavefile.Cavefile{Dependencies: missing})
	depPkgs, err := depTask.Run(ctx)
	if err != nil {
		return nil, err
	}
	r.installed = append(r.installed, depPkgs...)
	return r.installed, nil
}

func (r *ModuleResolver) ResolveModule(ctx context.Context, uri registry.LogicalURI) (*ast.ContextModule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.ensureInstalled(ctx); err != nil {
		return nil, err
	}
	if mod, ok := r.modules[uri]; ok {
		return mod, nil
	}
	if uri == preludeModuleURI {
		return r.Prelude(ctx)
	}
	prelude, err := r.Prelude(ctx)
	if err != nil {
		return nil, err
	}
	mod, err := r.findResolvedModule(uri)
	if err != nil {
		return nil, err
	}
	ctxMod, err := parseResolvedModule(mod, prelude)
	if err != nil {
		return nil, err
	}
	r.modules[uri] = ctxMod
	return ctxMod, nil
}

func (r *ModuleResolver) Prelude(ctx context.Context) (*ast.ContextModule, error) {
	if r.prelude != nil {
		return r.prelude, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.ensureInstalled(ctx); err != nil {
		return nil, err
	}
	mod, err := r.findResolvedModule(preludeModuleURI)
	if err != nil {
		return nil, err
	}
	ctxMod, err := parseResolvedModule(mod, nil)
	if err != nil {
		return nil, err
	}
	r.prelude = ctxMod
	r.modules[preludeModuleURI] = ctxMod
	return ctxMod, nil
}

func (r *ModuleResolver) findResolvedModule(uri registry.LogicalURI) (registry.ResolvedModule, error) {
	for _, pkg := range r.installed {
		mods, err := pkg.ResolveModules()
		if err != nil {
			return nil, err
		}
		for _, mod := range mods {
			if mod.URI() != uri {
				continue
			}
			return mod, nil
		}
	}
	return nil, fmt.Errorf("module %q not found", uri)
}

func (r *ModuleResolver) ensureInstalled(ctx context.Context) error {
	if r.ready {
		return nil
	}
	_, err := r.ensureDependencies(ctx)
	if err != nil {
		return err
	}
	r.ready = true
	return nil
}

func (r *ModuleResolver) filterMissingDependencies(deps []cavefile.Dependency) []cavefile.Dependency {
	if len(deps) == 0 {
		return nil
	}
	var missing []cavefile.Dependency
	for _, dep := range deps {
		if r.hasDependency(dep) {
			continue
		}
		missing = append(missing, dep)
	}
	return missing
}

func (r *ModuleResolver) hasDependency(dep cavefile.Dependency) bool {
	for _, pkg := range r.installed {
		if dep.Source == "" {
			continue
		}
		if pkg.Source() != dep.Source {
			continue
		}
		if version.MatchesAll(pkg.Version(), dep.Predicates...) {
			return true
		}
	}
	return false
}

func parseResolvedModule(mod registry.ResolvedModule, parent *ast.ContextModule) (*ast.ContextModule, error) {
	mp := parser.NewModuleParse(mod)
	if parent != nil {
		mp.Decls().Parent = parent.Decls
	}
	module, err := mp.Parse(mod)
	if err != nil {
		return nil, err
	}
	if err := joinParseErrors(mp.Errors()); err != nil {
		return nil, err
	}
	return module, nil
}
