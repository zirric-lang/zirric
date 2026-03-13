package orchestra

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/compiler"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/pkgmanager"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/cavereg"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/fsmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/vm"
	"github.com/go-git/go-billy/v5"
)

const preludeModuleURI registry.LogicalURI = defaultStandardLibraryName + ".prelude"

type Config struct {
	ProjectFS   billy.Filesystem
	RegistryFS  billy.Filesystem
	PackageName string // required; used as the project's logical URI base and package identity
}

type Orchestra struct {
	cave           cavefile.Cavefile
	projectFS      billy.Filesystem
	projectBaseURI registry.LogicalURI
	pkgmanager     *pkgmanager.PackageManager
}

func New(cfg Config) (*Orchestra, error) {
	if cfg.ProjectFS == nil {
		return nil, errors.New("project filesystem is required")
	}
	if cfg.RegistryFS == nil {
		return nil, errors.New("registry filesystem is required")
	}
	if cfg.PackageName == "" {
		return nil, errors.New("package name is required")
	}
	if err := cfg.RegistryFS.MkdirAll("git", 0o755); err != nil {
		return nil, err
	}

	cave := ensureStandardLibraryDependency(cavefile.Cavefile{
		Package: cavefile.Package{
			Name:   cfg.PackageName,
			Source: "file://" + cfg.ProjectFS.Root(),
		},
	})

	caveReg, err := cavereg.New(cave, cfg.ProjectFS)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cave registry: %w", err)
	}

	pm, err := pkgmanager.New(
		cfg.RegistryFS,
		pkgmanager.WithRegistry(caveReg),
		pkgmanager.WithGitRegistry(),
		withDefaultStdlibRegistry(),
	)
	if err != nil {
		return nil, err
	}

	return &Orchestra{
		projectFS:      cfg.ProjectFS,
		projectBaseURI: registry.LogicalURI(cave.Name),
		pkgmanager:     pm,
		cave:           cave,
	}, nil
}

func (o *Orchestra) ParseModulePath(ctx context.Context, modulePath string, resolver *ModuleResolver) (*ast.ContextModule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	mod, err := o.resolveModule(modulePath)
	if err != nil {
		return nil, err
	}
	return o.ParseModule(ctx, mod, resolver)
}

func (o *Orchestra) ParseFile(ctx context.Context, filePath string, resolver *ModuleResolver) (*ast.ContextModule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	source, err := o.readSourceFile(filePath)
	if err != nil {
		return nil, err
	}
	moduleURI := o.projectBaseURI
	if dir := filepath.Dir(filePath); dir != "." {
		moduleURI = registry.JoinModuleURI(moduleURI, filepath.ToSlash(dir))
	}
	mod := staticmodule.NewModule(moduleURI, []registry.Source{source})
	return o.ParseModule(ctx, mod, resolver)
}

func (o *Orchestra) ParseModule(ctx context.Context, mod registry.ResolvedModule, resolver *ModuleResolver) (*ast.ContextModule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if resolver == nil {
		return nil, errors.New("module resolver is required")
	}
	prelude, err := resolver.Prelude(ctx)
	if err != nil {
		return nil, err
	}
	mp := parser.NewModuleParse(mod)
	mp.Decls().Parent = prelude.Decls
	module, err := mp.Parse(mod)
	if err != nil {
		return nil, err
	}
	if err := joinParseErrors(mp.Errors()); err != nil {
		return nil, err
	}
	resolver.RegisterModule(mod.URI(), module)
	return module, nil
}

func (o *Orchestra) Compile(module *ast.ContextModule, resolver *ModuleResolver) (*compiler.Bytecode, error) {
	// Orchestra owns analysis and passes the analyzer into the compiler.
	analysis := analyzer.New(resolver)
	if errs, _ := analysis.Analyze(module, true); len(errs) > 0 {
		return nil, fmt.Errorf("%s", errs[0].Error())
	}
	comp := compiler.NewWithAnalyzer(resolver, analysis)
	if err := comp.Compile(module); err != nil {
		return nil, err
	}
	return comp.Bytecode(), nil
}

func (o *Orchestra) RunModulePath(ctx context.Context, modulePath string) error {
	resolver, err := o.NewResolver()
	if err != nil {
		return err
	}

	module, err := o.ParseModulePath(ctx, modulePath, resolver)
	if err != nil {
		return err
	}

	bytecode, err := o.Compile(module, resolver)
	if err != nil {
		return err
	}
	return o.runBytecode(bytecode)
}

func (o *Orchestra) RunFile(ctx context.Context, filePath string) error {
	resolver, err := o.NewResolver()
	if err != nil {
		return err
	}

	module, err := o.ParseFile(ctx, filePath, resolver)
	if err != nil {
		return err
	}

	bytecode, err := o.Compile(module, resolver)
	if err != nil {
		return err
	}
	return o.runBytecode(bytecode)
}

func (o *Orchestra) NewResolver() (*ModuleResolver, error) {
	return NewModuleResolver(o.pkgmanager, o.cave)
}

func (o *Orchestra) runBytecode(bytecode *compiler.Bytecode) error {
	machine := vm.New(bytecode)
	return machine.Run()
}

func (o *Orchestra) resolveModule(modulePath string) (registry.ResolvedModule, error) {
	modules, err := fsmodule.DiscoverModules(o.projectBaseURI, o.projectFS)
	if err != nil {
		return nil, err
	}
	if modulePath == "." || modulePath == "" {
		for _, module := range modules {
			if module.URI() == o.projectBaseURI {
				return module, nil
			}
		}
	}
	targetURI := registry.JoinModuleURI(o.projectBaseURI, filepath.ToSlash(modulePath))
	for _, module := range modules {
		if module.URI() == targetURI {
			return module, nil
		}
	}
	return nil, fmt.Errorf("module %q not found", modulePath)
}

func (o *Orchestra) readSourceFile(filePath string) (registry.Source, error) {
	file, err := o.projectFS.Open(filePath)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(file)
	closeErr := file.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	logicalURI := o.projectBaseURI.Join(filepath.ToSlash(filePath))
	return staticmodule.NewSource(logicalURI, data), nil
}

func joinParseErrors(errs []parser.ParseError) error {
	if len(errs) == 0 {
		return nil
	}
	joined := make([]error, 0, len(errs))
	for _, err := range errs {
		joined = append(joined, err)
	}
	return errors.Join(joined...)
}
