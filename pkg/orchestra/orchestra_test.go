package orchestra_test

import (
	"context"
	_ "embed"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/pkgmanager"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
)

//go:embed testdata/Cavefile
var cavefileContent string

func TestParseModulePreloadsPrelude(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "app/main.zirr", "mod app\nconst greeting = \"hi\"\n")

	orch := newTestOrchestra(t, projectFS, "project")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}

	module, err := orch.ParseModulePath(context.Background(), "app", resolver)
	if err != nil {
		t.Fatalf("parse module: %v", err)
	}

	if module.Decls.Parent == nil {
		t.Fatal("expected module decls to have a prelude parent")
	}
	sym := module.Decls.Parent.Symbols["String"]
	if sym == nil || sym.Decl == nil {
		t.Fatal("expected prelude symbol String to be present")
	}
	if _, ok := sym.Decl.(*ast.DeclExternType); !ok {
		t.Fatalf("expected String to be an extern type, got %T", sym.Decl)
	}
}

func TestRunFile(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\n1\n")

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func TestIsTypeCheckAgainstCrossModuleType(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

fn assertTrue(actual, label) {
	if !actual {
		panic(label)
	}
}

assertTrue(3 is Int, "FAIL: is Int")
assertTrue([1, 2] is Array, "FAIL: is Array")
assertTrue([1, 2] is [Int], "FAIL: is [Int]")

const described = switch 3 {
case is Int: "int"
case _: "other"
}
assertTrue(described == "int", "FAIL: switch case is Int")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func TestIterableAgainstRealPrelude(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

fn sumRange() {
	var sum = 0
	for x <- Range(0, 5) {
		sum = sum + x
	}
	return sum
}
assertEqual(sumRange(), 0 + 1 + 2 + 3 + 4, "FAIL: Range sum")

fn sumClosedRange() {
	var sum = 0
	for x <- ClosedRange(0, 5) {
		sum = sum + x
	}
	return sum
}
assertEqual(sumClosedRange(), 0 + 1 + 2 + 3 + 4 + 5, "FAIL: ClosedRange sum")

fn countVisits(collection) {
	var visits = 0
	for x <- collection {
		visits = visits + 1
	}
	return visits
}
assertEqual(countVisits(Range(0, 0)), 0, "FAIL: empty Range visits nothing")
assertEqual(countVisits(Range(3, 3)), 0, "FAIL: empty Range (equal bounds) visits nothing")
assertEqual(countVisits(ClosedRange(3, 3)), 1, "FAIL: single-element ClosedRange visits once")
assertEqual(countVisits(ClosedRange(5, 3)), 0, "FAIL: inverted ClosedRange visits nothing")

fn breaksEarly() {
	var sum = 0
	for x <- Range(0, 100) {
		if x == 3 {
			break
		}
		sum = sum + x
	}
	return sum
}
assertEqual(breaksEarly(), 0 + 1 + 2, "FAIL: Range break")

fn arrayViaGenericPathStillWorks() {
	var sum = 0
	for x <- [10, 20, 30] {
		sum = sum + x
	}
	return sum
}
assertEqual(arrayViaGenericPathStillWorks(), 60, "FAIL: Array fast path")

const doubled = for x <- Range(0, 3) { x * 2 }
assertEqual(doubled[0], 0, "FAIL: expr-form Range [0]")
assertEqual(doubled[1], 2, "FAIL: expr-form Range [1]")
assertEqual(doubled[2], 4, "FAIL: expr-form Range [2]")

assertEqual(Countable(Range(0, 5)).length(Range(0, 5)), 5, "FAIL: Countable.length on Range")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func TestDictIterableYieldsPairs(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

const pair = Pair("k", "v")
assertEqual(pair.key, "k", "FAIL: Pair.key")
assertEqual(pair.value, "v", "FAIL: Pair.value")

const d = [1: "one", 2: "two", 3: "three"]

assertEqual(d.length, 3, "FAIL: Dict.length")
assertEqual(d.keys.length, 3, "FAIL: Dict.keys length")
assertEqual([10, 20, 30].length, 3, "FAIL: Array.length")
assertEqual(Countable(d).length(d), 3, "FAIL: Countable.length on Dict")

fn checkAllPairsAndCountKeySum() {
	var count = 0
	var keySum = 0
	for p <- d {
		assertEqual(d[p.key], p.value, "FAIL: Dict iterate pair mismatch")
		count = count + 1
		keySum = keySum + p.key
	}
	return count * 1000 + keySum
}
assertEqual(checkAllPairsAndCountKeySum(), 3*1000 + (1+2+3), "FAIL: Dict iterate visit count/key sum")

const collected = for p <- d { p.key }
assertEqual(collected.length, 3, "FAIL: expr-form Dict iterate collected wrong count")

fn breaksEarly() {
	var count = 0
	for p <- d {
		count = count + 1
		break
	}
	return count
}
assertEqual(breaksEarly(), 1, "FAIL: break stops Dict iteration early")

fn emptyDict() {
	var count = 0
	for p <- [:] {
		count = count + 1
	}
	return count
}
assertEqual(emptyDict(), 0, "FAIL: empty Dict visits nothing")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func TestStringIterableYieldsChars(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

const s = "café"

assertEqual(s.length, 4, "FAIL: String.length is character count, not byte count")
assertEqual(s[0], 'c', "FAIL: String index 0")
assertEqual(s[3], 'é', "FAIL: String index 3 (multi-byte character)")
assertEqual(Countable(s).length(s), 4, "FAIL: Countable.length on String")

assertEqual(s.chars.length, 4, "FAIL: String.chars length")
assertEqual(s.chars[0], 'c', "FAIL: String.chars[0]")
assertEqual(s.chars[3], 'é', "FAIL: String.chars[3] (multi-byte character)")
assertEqual("".chars.length, 0, "FAIL: empty String.chars")

fn countChars(str) {
	var count = 0
	for c <- str {
		count = count + 1
	}
	return count
}
assertEqual(countChars(s), 4, "FAIL: String iterate visits one Char per character, not per byte")
assertEqual(countChars(""), 0, "FAIL: empty String visits nothing")

fn breaksEarly() {
	var count = 0
	for c <- s {
		count = count + 1
		if count == 2 {
			break
		}
	}
	return count
}
assertEqual(breaksEarly(), 2, "FAIL: break stops String iteration early")

const collected = for c <- "ab" { c }
assertEqual(collected.length, 2, "FAIL: expr-form String iterate collected wrong count")
assertEqual(collected[0], 'a', "FAIL: expr-form String iterate [0]")
assertEqual(collected[1], 'b', "FAIL: expr-form String iterate [1]")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func TestParseFileUsesPreludeAttribute(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\n@Deprecated(\"use NewExample\")\ndata Example { name }\n")

	orch := newTestOrchestra(t, projectFS, "project")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	module, err := orch.ParseFile(context.Background(), "main.zirr", resolver)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}
	if module.Decls.Parent == nil {
		t.Fatal("expected prelude decls to be parented")
	}
	if module.Decls.Parent.Symbols["Deprecated"] == nil {
		t.Fatal("expected prelude attribute Deprecated to be present")
	}
	if module.Decls.Parent.Symbols["String"] == nil {
		t.Fatal("expected prelude type String to be present")
	}

	sym := module.Decls.Symbols["Example"]
	if sym == nil || sym.Decl == nil {
		t.Fatal("expected data Example to be declared")
	}
	decl, ok := sym.Decl.(*ast.DeclData)
	if !ok {
		t.Fatalf("expected Example to be data, got %T", sym.Decl)
	}
	if len(decl.Attributes) != 1 {
		t.Fatalf("expected 1 attribute, got %d", len(decl.Attributes))
	}
	anno := decl.Attributes[0]
	if len(anno.Reference) != 1 || anno.Reference[0].Value != "Deprecated" {
		t.Fatalf("expected @Deprecated attribute, got %v", anno.Reference)
	}
	if len(anno.Arguments) != 1 {
		t.Fatalf("expected 1 attribute argument, got %d", len(anno.Arguments))
	}
}

// TestRunFileNoDeclaredDependencies verifies that a project with no explicit
// dependencies (only stdlib injected automatically) can still resolve and run
// its own modules. This guards against a coupling bug where the project package
// itself was only installed when declared dependencies were missing.
func TestRunFileNoDeclaredDependencies(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst answer = \"42\"\n")

	// Deliberately no PackageName-derived dependencies beyond stdlib (injected automatically).
	orch := newTestOrchestra(t, projectFS, "main")

	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestRunFileWithCrossModuleImport verifies that a file importing another module
// within the same project is resolved correctly. This exercises the cavereg
// filesystem path: the resolver must discover project sub-modules from
// ProjectFS (not RegistryFS) via findResolvedModule.
func TestRunFileWithCrossModuleImport(t *testing.T) {
	projectFS := memfs.New()
	// utils/ subdirectory → URI "project.utils" (directory name is the URI segment)
	writeFile(t, projectFS, "utils/greet.zirr", "mod utils\nconst greeting = \"hello\"\n")
	writeFile(t, projectFS, "main.zirr", "mod main\nimport utils = project.utils\n")

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestResolveModuleBareProjectSubmodule is a regression test: only the fully-qualified form used to resolve at the compiler/analyzer level, so a bare import (e.g. "flow") produced "unknown module" even though go-to-definition worked fine via a plain filesystem lookup.
func TestResolveModuleBareProjectSubmodule(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "utils/greet.zirr", "mod utils\nconst greeting = \"hello\"\n")

	orch := newTestOrchestra(t, projectFS, "project")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}

	mod, err := resolver.ResolveModule(context.Background(), "utils")
	if err != nil {
		t.Fatalf("resolve bare project submodule %q: %v", "utils", err)
	}
	if mod == nil {
		t.Fatal("expected non-nil module")
	}
	if mod.Decls.Symbols["greeting"] == nil {
		t.Fatal("expected 'greeting' to be declared in the resolved module")
	}

	qualified, err := resolver.ResolveModule(context.Background(), "project.utils")
	if err != nil {
		t.Fatalf("resolve qualified project submodule: %v", err)
	}
	if qualified != mod {
		t.Fatal("expected bare and fully-qualified imports to resolve to the same module instance")
	}
}

func newTestOrchestra(t *testing.T, projectFS billy.Filesystem, name string) *orchestra.Orchestra {
	t.Helper()
	registryFS := memfs.New()
	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:   projectFS,
		RegistryFS:  registryFS,
		PackageName: name,
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}
	return orch
}

// TestMainModuleIsRegisteredAfterParse verifies that after ParseFile the
// resolver's MainModule returns the same pointer, which is required for the
// compiler's pointer-equality checks to work correctly.
func TestMainModuleIsRegisteredAfterParse(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst x = 42\n")

	orch := newTestOrchestra(t, projectFS, "main")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}

	module, err := orch.ParseFile(context.Background(), "main.zirr", resolver)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}

	if resolver.MainModule() == nil {
		t.Fatal("expected MainModule to be non-nil after ParseFile")
	}
	if resolver.MainModule() != module {
		t.Fatal("expected MainModule to be the same pointer as the parsed module")
	}
}

// TestRunFileWithConstBinding verifies that a file with a const binding compiles
// and runs correctly — exercising full main-module symbol compilation.
func TestRunFileWithLetBinding(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst greeting = \"hello\"\n")

	orch := newTestOrchestra(t, projectFS, "main")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestREPLLoop verifies the REPL's parse→compile pattern: a single resolver
// is reused across multiple ParseFile+Compile calls, as the REPL does.
func TestREPLLoop(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "repl.zirr", "mod repl\n")

	orch := newTestOrchestra(t, projectFS, "repl")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}

	iterations := []string{
		"mod repl\n1\n",
		"mod repl\n2\n",
	}
	for i, src := range iterations {
		if err := projectFS.Remove("repl.zirr"); err != nil {
			t.Fatalf("iter %d: remove: %v", i, err)
		}
		writeFile(t, projectFS, "repl.zirr", src)

		module, err := orch.ParseFile(context.Background(), "repl.zirr", resolver)
		if err != nil {
			t.Fatalf("iter %d: parse file: %v", i, err)
		}
		if _, err := orch.Compile(module, resolver); err != nil {
			t.Fatalf("iter %d: compile: %v", i, err)
		}
	}
}

func writeFile(t *testing.T, fs billy.Filesystem, path string, contents string) {
	t.Helper()

	dir := filepath.Dir(path)
	if dir != "." {
		if err := fs.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	f, err := fs.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}

	if _, err := f.Write([]byte(contents)); err != nil {
		_ = f.Close()
		t.Fatalf("write %s: %v", path, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close %s: %v", path, err)
	}
}

// TestRunFileImportPrelude verifies that explicitly importing the prelude
// (which contains extern const declarations like void) compiles and runs
// without "unknown declaration *ast.DeclExternValue" errors.
func TestRunFileImportPrelude(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nimport prelude = prelude\n")

	orch := newTestOrchestra(t, projectFS, "main")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestAutoImportPrelude verifies that the prelude module is automatically
// imported so that `prelude.X` references work without an explicit import.
func TestAutoImportPrelude(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\n")

	orch := newTestOrchestra(t, projectFS, "main")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	module, err := orch.ParseFile(context.Background(), "main.zirr", resolver)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}

	sym, ok := module.Decls.Symbols["prelude"]
	if !ok || sym == nil {
		t.Fatal("expected synthetic prelude import in module decls")
	}
	importDecl, ok := sym.Decl.(*ast.DeclImport)
	if !ok {
		t.Fatalf("expected DeclImport, got %T", sym.Decl)
	}
	if importDecl.Alias.Value != "prelude" {
		t.Fatalf("expected alias 'prelude', got %q", importDecl.Alias.Value)
	}
}

// TestAutoImportPreludeSkipsPrelude verifies that the prelude module itself
// does not get a synthetic prelude import (avoiding circularity).
func TestAutoImportPreludeSkipsPrelude(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\n")

	orch := newTestOrchestra(t, projectFS, "main")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	prelude, err := resolver.Prelude(context.Background())
	if err != nil {
		t.Fatalf("get prelude: %v", err)
	}
	if _, ok := prelude.Decls.Symbols["prelude"]; ok {
		t.Fatal("prelude module should not have a synthetic prelude import")
	}
}

// TestShortModulePath verifies that short module names like "prelude"
// resolve to the full stdlib URI automatically.
func TestShortModulePath(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nimport p = prelude\n")

	orch := newTestOrchestra(t, projectFS, "main")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestShortModulePathFuture verifies that "future" resolves
// to the full stdlib path.
func TestShortModulePathFuture(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\n")

	orch := newTestOrchestra(t, projectFS, "main")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	mod, err := resolver.ResolveModule(context.Background(), "future")
	if err != nil {
		t.Fatalf("resolve short path future: %v", err)
	}
	if mod == nil {
		t.Fatal("expected non-nil module for future")
	}
}

// TestShortModulePathNotFoundUsesOriginalName verifies that when a short
// module name can't be found even with the stdlib prefix, the error
// message uses the original short name.
func TestShortModulePathNotFoundUsesOriginalName(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\n")

	orch := newTestOrchestra(t, projectFS, "main")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	_, err = resolver.ResolveModule(context.Background(), "nonexistent.module")
	if err == nil {
		t.Fatal("expected error for nonexistent module")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "nonexistent.module") {
		t.Fatalf("error should reference original short name, got: %s", errStr)
	}
	if strings.Contains(errStr, "code.knabel.dev.zirric_lang.zirric.nonexistent") {
		t.Fatalf("error should not use expanded name, got: %s", errStr)
	}
}

// TestBarePreludeSymbol verifies that prelude symbols like `String` are
// directly accessible without the `prelude.` prefix.
func TestBarePreludeSymbol(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst x = String\n")

	orch := newTestOrchestra(t, projectFS, "main")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestBarePreludeSymbolInAttribute verifies that prelude types used inside
// attribute arguments (e.g. @Deprecated("reason")) compile correctly.
func TestBarePreludeSymbolInAttribute(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\n@Deprecated(\"use NewExample\")\ndata Example { name }\n")

	orch := newTestOrchestra(t, projectFS, "main")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestBarePreludeSymbolShadowed verifies that a user declaration can shadow
// a prelude symbol.
func TestBarePreludeSymbolShadowed(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst String = 42\n")

	orch := newTestOrchestra(t, projectFS, "main")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestBarePreludeAndQualified verifies that both `String` and `prelude.String`
// work in the same file.
func TestBarePreludeAndQualified(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst x = String\nconst y = prelude.String\n")

	orch := newTestOrchestra(t, projectFS, "main")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestCavefileExample verifies that the example Cavefile from
// examples/project/Cavefile compiles and runs without errors.
// The Cavefile uses cross-module attribute references like @cave.Dependencies
// and @tasks.Name which require file-local imports to be visible during
// attribute resolution of promoted declarations.
func TestCavefileExample(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "Cavefile", cavefileContent)

	orch := newTestOrchestra(t, projectFS, "examples.project")
	if err := orch.RunFile(context.Background(), "Cavefile"); err != nil {
		t.Fatalf("run Cavefile: %v", err)
	}
}

// TestParseModuleWithErrors verifies that ParseModule returns a non-nil module
// alongside a parser.ParseErrors error when the source has syntax errors.
func TestParseModuleWithErrors(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst = \n")

	orch := newTestOrchestra(t, projectFS, "main")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}

	module, err := orch.ParseFile(context.Background(), "main.zirr", resolver)
	if err == nil {
		t.Fatal("expected parse error")
	}
	var parseErrs parser.ParseErrors
	if !errors.As(err, &parseErrs) {
		t.Fatalf("expected parser.ParseErrors, got %T: %v", err, err)
	}
	if len(parseErrs) == 0 {
		t.Fatal("expected at least one parse error")
	}
	if module == nil {
		t.Fatal("expected non-nil partial module alongside errors")
	} else if module.Decls.Parent == nil {
		t.Fatal("expected partial module to have prelude parent")
	}
}

// TestInvalidateModules verifies that InvalidateModules clears cached modules
// but preserves the prelude.
func TestInvalidateModules(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst x = 1\n")

	orch := newTestOrchestra(t, projectFS, "main")
	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}

	_, err = orch.ParseFile(context.Background(), "main.zirr", resolver)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}
	if resolver.MainModule() == nil {
		t.Fatal("expected MainModule before invalidation")
	}

	resolver.InvalidateModules()
	if resolver.MainModule() != nil {
		t.Fatal("expected MainModule to be nil after invalidation")
	}

	// Re-parse should work
	_, err = orch.ParseFile(context.Background(), "main.zirr", resolver)
	if err != nil {
		t.Fatalf("re-parse after invalidation: %v", err)
	}
	if resolver.MainModule() == nil {
		t.Fatal("expected MainModule after re-parse")
	}
}

// TestReadOnlyResolverMissingDependencyIsScoped is a regression test: an unresolvable Cavefile dependency used to fail Prelude()/ParseModule for the whole project, not just files that import it; resolving the missing dependency itself must surface a typed pkgmanager.DependencyNotInstalledError.
func TestReadOnlyResolverMissingDependencyIsScoped(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "Cavefile", `import cave

@cave.Dependencies()
data Dependencies {
  @cave.Local("../does-not-exist")
  missing
}
`)
	writeFile(t, projectFS, "main.zirr", "mod main\nconst x = 1\n")

	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:   projectFS,
		RegistryFS:  memfs.New(),
		PackageName: "project",
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}
	resolver, err := orch.NewResolver(orchestra.ReadOnly())
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}

	module, err := orch.ParseFile(context.Background(), "main.zirr", resolver)
	if err != nil {
		t.Fatalf("expected main.zirr to parse despite an unrelated missing dependency, got: %v", err)
	}
	if module == nil {
		t.Fatal("expected non-nil module")
	}

	_, err = resolver.ResolveModule(context.Background(), "missing")
	var notInstalled *pkgmanager.DependencyNotInstalledError
	if !errors.As(err, &notInstalled) {
		t.Fatalf("expected *pkgmanager.DependencyNotInstalledError, got %T: %v", err, err)
	}
}

// TestReadOnlyResolver verifies that a read-only resolver can parse modules
// using only locally-available packages.
func TestReadOnlyResolver(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst x = 1\n")

	orch := newTestOrchestra(t, projectFS, "main")
	resolver, err := orch.NewResolver(orchestra.ReadOnly())
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}

	module, err := orch.ParseFile(context.Background(), "main.zirr", resolver)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}
	if module == nil {
		t.Fatal("expected non-nil module")
	} else if module.Decls.Parent == nil {
		t.Fatal("expected prelude parent in read-only mode")
	}
}
