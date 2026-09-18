package orchestra

import (
	"context"
	"fmt"
	"sync"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/pkgmanager"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/resolver"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

type ModuleResolver struct {
	mu          sync.Mutex
	pm          *pkgmanager.PackageManager
	cave        cavefile.Cavefile
	installed   []registry.ResolvedPackage
	modules     map[registry.LogicalURI]*ast.ContextModule
	prelude     *ast.ContextModule
	ready       bool
	readOnly    bool
	onInstalled pkgmanager.InstallProgress
}

// ResolverOption configures a ModuleResolver.
type ResolverOption func(*ModuleResolver)

// ReadOnly returns a ResolverOption that puts the resolver in read-only mode.
// In this mode, dependencies are only resolved from locally-installed packages.
// Missing dependencies produce errors rather than triggering remote installation.
func ReadOnly() ResolverOption {
	return func(r *ModuleResolver) { r.readOnly = true }
}

// WithInstallProgress returns a ResolverOption that reports each dependency as it finishes installing.
func WithInstallProgress(fn pkgmanager.InstallProgress) ResolverOption {
	return func(r *ModuleResolver) { r.onInstalled = fn }
}

func NewModuleResolver(pm *pkgmanager.PackageManager, cave cavefile.Cavefile, opts ...ResolverOption) (*ModuleResolver, error) {
	if cave.Name == "" && cave.Source == "" {
		return nil, fmt.Errorf("cavefile must have a name or source to resolve modules")
	}
	if cave.Name == "" {
		cave.Name = string(registry.CanonicalizeModuleSource(cave.Source))
	}

	r := &ModuleResolver{
		pm:      pm,
		cave:    cave,
		modules: map[registry.LogicalURI]*ast.ContextModule{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r, nil
}

var _ resolver.ModuleResolver = (*ModuleResolver)(nil)

func (r *ModuleResolver) RegisterModule(uri registry.LogicalURI, module *ast.ContextModule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules[uri] = module
}

func (r *ModuleResolver) MainModule() *ast.ContextModule {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.modules[registry.LogicalURI(r.cave.Name)]
}

// InvalidateModules clears cached project modules so they will be re-parsed
// on next access. Prelude and installed packages are preserved.
func (r *ModuleResolver) InvalidateModules() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules = make(map[registry.LogicalURI]*ast.ContextModule)
	// Keep: r.installed, r.prelude, r.ready, r.pm, r.cave
}

func (r *ModuleResolver) ensureDependencies(ctx context.Context) ([]registry.ResolvedPackage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	depErr := r.installMissingDependencies(ctx)

	// Install the project itself even if a dependency failed, so its modules stay discoverable in r.installed.
	caveOnly := cavefile.Cavefile{Package: r.cave.Package}
	projectTask := r.pm.Install(caveOnly)
	projectTask.ReadOnly = r.readOnly
	projectTask.OnInstalled = r.onInstalled
	projectPkgs, err := projectTask.Run(ctx)
	if err != nil {
		return nil, err
	}
	r.installed = append(r.installed, projectPkgs...)

	if depErr != nil {
		return r.installed, depErr
	}
	return r.installed, nil
}

func (r *ModuleResolver) installMissingDependencies(ctx context.Context) error {
	missing := r.filterMissingDependencies(r.cave.Dependencies)
	if len(missing) == 0 {
		return nil
	}
	depTask := r.pm.Install(cavefile.Cavefile{Dependencies: missing})
	depTask.ReadOnly = r.readOnly
	depTask.OnInstalled = r.onInstalled
	depPkgs, err := depTask.Run(ctx)
	if err != nil {
		// In read-only mode, still use whatever was found
		if r.readOnly && len(depPkgs) > 0 {
			r.installed = append(r.installed, depPkgs...)
		}
		return err
	}
	r.installed = append(r.installed, depPkgs...)
	return nil
}

func (r *ModuleResolver) ResolveModule(ctx context.Context, uri registry.LogicalURI) (*ast.ContextModule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.ensureInstalledLocked(ctx); err != nil {
		return nil, err
	}
	if mod, ok := r.modules[uri]; ok {
		return mod, nil
	}
	if uri == preludeModuleURI {
		return r.preludeLocked(ctx)
	}
	prelude, err := r.preludeLocked(ctx)
	if err != nil {
		return nil, err
	}
	mod, findErr := r.findResolvedModule(uri)
	if findErr != nil {
		return nil, fmt.Errorf("module %q not found", uri)
	}
	ctxMod, err := parseResolvedModule(mod, prelude)
	// Cache even partial results
	if ctxMod != nil {
		r.modules[uri] = ctxMod
	}
	if err != nil {
		return ctxMod, err
	}
	return ctxMod, nil
}

func (r *ModuleResolver) Prelude(ctx context.Context) (*ast.ContextModule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.preludeLocked(ctx)
}

func (r *ModuleResolver) preludeLocked(ctx context.Context) (*ast.ContextModule, error) {
	if r.prelude != nil {
		return r.prelude, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.ensureInstalledLocked(ctx); err != nil {
		return nil, err
	}
	mod, err := r.findResolvedModule(preludeModuleURI)
	if err != nil {
		return nil, err
	}
	ctxMod, err := parseResolvedModule(mod, nil)
	// Cache even partial results
	if ctxMod != nil {
		r.prelude = ctxMod
		r.modules[preludeModuleURI] = ctxMod
	}
	if err != nil {
		return ctxMod, err
	}
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

// EnsureInstalled installs the project and its declared dependencies, returning the resolved packages.
func (r *ModuleResolver) EnsureInstalled(ctx context.Context) ([]registry.ResolvedPackage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureInstalledLocked(ctx); err != nil {
		return nil, err
	}
	return r.installed, nil
}

func (r *ModuleResolver) ensureInstalledLocked(ctx context.Context) error {
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
		return module, err
	}
	if parent != nil && mod.URI() != preludeModuleURI {
		injectPreludeImport(module, parent)
	}
	if errs := mp.Errors(); len(errs) > 0 {
		return module, parser.ParseErrors(errs)
	}
	return module, nil
}
