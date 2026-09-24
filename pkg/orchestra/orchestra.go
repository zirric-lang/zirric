package orchestra

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/compiler"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/pkgmanager"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/cavereg"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/fsmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/localreg"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
	"code.knabel.dev/zirric-lang/zirric/pkg/toolchain"
	"code.knabel.dev/zirric-lang/zirric/pkg/vm"
	"github.com/go-git/go-billy/v5"
)

const preludeModuleURI registry.LogicalURI = "prelude"

const DefaultCavefileName = "Cavefile"

type Config struct {
	ProjectFS   billy.Filesystem
	RegistryFS  billy.Filesystem
	PackageName string // required; used as the project's logical URI base and package identity

	Cavefile     *cavefile.Cavefile // optional; overrides auto-detecting the Cavefile in ProjectFS
	CavefilePath string             // optional; overrides DefaultCavefileName as the Cavefile's path within ProjectFS

	// IgnoreLanguageVersion opens a project whose @cave.LanguageVersion this Zirric does not satisfy.
	// Only `zirric cave describe` sets it, since reporting what the manifest says is how the refusal is explained.
	IgnoreLanguageVersion bool
}

type Orchestra struct {
	// ReportWarnings is called with the diagnostics that do not stop a build, so that the caller decides how to show them. Nil means they are not reported at all.
	ReportWarnings func(analyzer.AnalysisErrors)

	cave           cavefile.Cavefile
	cavefilePath   string
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

	cave, cavefilePath, err := loadCave(cfg)
	if err != nil {
		return nil, err
	}
	// Refused here rather than at each command, so that nothing acts on a package this Zirric cannot build.
	if !cfg.IgnoreLanguageVersion {
		if err := cavefile.CheckLanguageVersion(cave.Package, toolchain.Version()); err != nil {
			return nil, err
		}
	}
	cave = ensureStandardLibraryDependency(cave)

	caveReg, err := cavereg.New(cave, cfg.ProjectFS)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cave registry: %w", err)
	}

	pm, err := pkgmanager.New(
		cfg.RegistryFS,
		pkgmanager.WithRegistry(caveReg),
		pkgmanager.WithRegistry(localreg.New()),
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
		cavefilePath:   cavefilePath,
	}, nil
}

func suggestedModuleDeclaration(packageName string) string {
	candidate := registry.CanonicalizeModulePath(packageName)
	if !registry.IsModulePath(candidate) {
		return "add a 'mod' declaration"
	}
	return fmt.Sprintf("add 'mod %s'", candidate)
}

// loadCave resolves the project's Cavefile and its path within ProjectFS (empty when a synthetic Cavefile was used instead).
func loadCave(cfg Config) (cavefile.Cavefile, string, error) {
	if cfg.Cavefile != nil {
		return *cfg.Cavefile, "", nil
	}
	path := cfg.CavefilePath
	if path == "" {
		path = DefaultCavefileName
	}
	if _, err := cfg.ProjectFS.Stat(path); err == nil {
		cave, err := ParseCavefile(context.Background(), cfg.ProjectFS, cfg.RegistryFS, path)
		if err != nil {
			return cavefile.Cavefile{}, "", fmt.Errorf("parse %s: %w", path, err)
		}
		// Every module of the package is named under this path, so a Cavefile that declares none leaves the package unnamed.
		if cave.ModulePath == "" {
			return cavefile.Cavefile{}, "", fmt.Errorf("%s declares no module: %s to name the package", path, suggestedModuleDeclaration(cfg.PackageName))
		}
		return cave, path, nil
	}
	if cfg.CavefilePath != "" {
		return cavefile.Cavefile{}, "", fmt.Errorf("cavefile not found: %s", path)
	}
	return cavefile.Cavefile{
		Package: cavefile.Package{
			Name:   cfg.PackageName,
			Source: "file://" + cfg.ProjectFS.Root(),
		},
	}, "", nil
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
	// A file belongs to the module its directory names, however it was reached.
	mod := staticmodule.NewModule(registry.JoinModuleURI(o.projectBaseURI, filepath.ToSlash(filepath.Dir(filePath))), []registry.Source{source})
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
		// Sources() failed — return whatever partial module we have
		return module, err
	}
	// Auto-import prelude so that `prelude.X` references work in all modules except the prelude itself.
	if mod.URI() != preludeModuleURI {
		injectPreludeImport(module, prelude)
	}
	resolver.RegisterModule(mod.URI(), module)

	if errs := mp.Errors(); len(errs) > 0 {
		return module, parser.ParseErrors(errs)
	}
	return module, nil
}

func (o *Orchestra) Compile(module *ast.ContextModule, resolver *ModuleResolver) (*compiler.Bytecode, error) {
	// Orchestra owns analysis and passes the analyzer into the compiler.
	analysis := analyzer.New(resolver)
	found, _ := analysis.Analyze(module, true)
	diagnostics := analyzer.AnalysisErrors(found)
	// Warnings describe something worth saying rather than something that stops a build, so they are reported and then stepped over.
	if warnings := diagnostics.Warnings(); len(warnings) > 0 && o.ReportWarnings != nil {
		o.ReportWarnings(warnings)
	}
	// Everything else, and as it is: analysis finds all of it in one pass, and each carries the position a renderer needs to show the line it refers to.
	if failing := diagnostics.Failing(); len(failing) > 0 {
		return nil, failing
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

// generatedModuleSegment names the module a generated program is given, one level beneath the project's root.
// The leading underscore keeps it out of reach of a real directory, which cannot start with one.
const generatedModuleSegment = "_generated"

// RunSource compiles and runs body as a program of a module beneath the project's root.
//
// The module declaration is written here, since it has to name a module under the base the project's Cavefile fixed. A project with no Cavefile fixed no base, and none is written.
//
// It is deliberately not the root module itself. Registering a synthesized single-source module under the root URI would shadow the real root module, so a package whose own declarations live at its root — the usual shape of a small one-module package — would see them vanish: anything importing that package by name, its own tests included, would resolve to this program instead and find it empty.
func (o *Orchestra) RunSource(ctx context.Context, name string, body string) error {
	// A project with no Cavefile fixed no base, so there is no root module to step out of and nothing to name.
	uri := o.projectBaseURI
	source := body
	if registry.IsModulePath(string(uri)) {
		uri = registry.JoinModuleURI(uri, generatedModuleSegment)
		source = "mod " + string(uri) + "\n\n" + body
	}

	resolver, err := o.NewResolver()
	if err != nil {
		return err
	}
	mod := staticmodule.NewModule(uri, []registry.Source{
		staticmodule.NewSource(uri.Join(name), []byte(source)),
	})
	module, err := o.ParseModule(ctx, mod, resolver)
	if err != nil {
		return err
	}
	bytecode, err := o.Compile(module, resolver)
	if err != nil {
		return err
	}
	return o.runBytecode(bytecode)
}

// RunFileWithArgs runs filePath like RunFile, but rewrites os.Args to [filePath, args...] first, so the script's own os.args() call sees this argv instead of the CLI's.
func (o *Orchestra) RunFileWithArgs(ctx context.Context, filePath string, args []string) error {
	prevArgs := os.Args
	os.Args = buildArgv(filePath, args)
	defer func() { os.Args = prevArgs }()

	return o.RunFile(ctx, filePath)
}

func buildArgv(filePath string, args []string) []string {
	argv := make([]string, 0, 1+len(args))
	argv = append(argv, filePath)
	argv = append(argv, args...)
	return argv
}

func (o *Orchestra) NewResolver(opts ...ResolverOption) (*ModuleResolver, error) {
	return NewModuleResolver(o.pkgmanager, o.cave, opts...)
}

// compileCavefileForTasks compiles the Cavefile through the real project resolver, unlike ParseCavefile's stdlib-only bootstrap parse.
func (o *Orchestra) compileCavefileForTasks(ctx context.Context) (*compiler.Bytecode, *ast.ContextModule, *ModuleResolver, error) {
	resolver, err := o.NewResolver()
	if err != nil {
		return nil, nil, nil, err
	}

	module, err := o.ParseFile(ctx, o.cavefilePath, resolver)
	if err != nil {
		return nil, nil, nil, err
	}

	bytecode, err := o.Compile(module, resolver)
	if err != nil {
		return nil, nil, nil, err
	}

	return bytecode, module, resolver, nil
}

func (o *Orchestra) Cavefile() cavefile.Cavefile {
	return o.cave
}

// CavefilePath returns the Cavefile's path within ProjectFS, or "" if the project has no physical Cavefile (a synthetic one was used instead).
func (o *Orchestra) CavefilePath() string {
	return o.cavefilePath
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

// injectPreludeImport adds a synthetic `import prelude = <preludeURI>` into the module's DeclTable so that `prelude.X` references work automatically.
// It also injects DeclImportMember entries for each public prelude symbol so that bare identifiers like `String` resolve without a prefix.
func injectPreludeImport(module *ast.ContextModule, prelude *ast.ContextModule) {
	if module == nil || module.Decls == nil {
		return
	}
	// Don't overwrite an explicit prelude import.
	if _, exists := module.Decls.Symbols["prelude"]; exists {
		return
	}

	syntheticTok := token.Token{Type: token.IMPORT, Literal: "import"}
	parts := strings.Split(string(preludeModuleURI), ".")
	refs := make(ast.StaticReference, len(parts))
	for i, part := range parts {
		refs[i] = ast.Identifier{
			Token: token.Token{Type: token.IDENT, Literal: part},
			Value: part,
		}
	}
	alias := ast.Identifier{
		Token: token.Token{Type: token.IDENT, Literal: "prelude"},
		Value: "prelude",
	}
	importDecl := ast.MakeDeclAliasImport(syntheticTok, alias, refs)

	if prelude != nil && prelude.Decls != nil {
		for name, sym := range prelude.Decls.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}
			if sym.Decl.ExportScope() != ast.ExportScopePublic {
				continue
			}
			memberIdent := ast.Identifier{
				Token: token.Token{Type: token.IDENT, Literal: name},
				Value: name,
			}
			member := ast.MakeDeclImportMember(syntheticTok, importDecl.ModuleName, memberIdent)
			importDecl.AddMember(member)
		}
	}

	module.Decls.Symbols["prelude"] = &ast.DeclSymbol{
		Name: "prelude",
		Decl: importDecl,
	}

	// Insert each import member into the module's DeclTable so bare identifiers like `String` can be resolved without the `prelude.` prefix.
	for _, member := range importDecl.Members {
		name := member.Name.Value
		if _, exists := module.Decls.Symbols[name]; exists {
			continue
		}
		module.Decls.Symbols[name] = &ast.DeclSymbol{
			Name: name,
			Decl: member,
		}
	}
}
