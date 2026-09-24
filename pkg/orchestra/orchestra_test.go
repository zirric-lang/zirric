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

assertEqual(len(d), 3, "FAIL: Dict length via len()")
assertEqual(len(d.keys()), 3, "FAIL: Dict.keys() length")
assertEqual(len([10, 20, 30]), 3, "FAIL: Array length via len()")
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
assertEqual(len(collected), 3, "FAIL: expr-form Dict iterate collected wrong count")

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

assertEqual(len(s), 5, "FAIL: String length via len() is byte count, matching Go's len(), not character count")
assertEqual(s[0] is Byte, true, "FAIL: String index yields Byte")
assertEqual(Countable(s).length(s), 5, "FAIL: Countable.length on String is byte count")

assertEqual(len(s.chars()), 4, "FAIL: String.chars() length")
assertEqual(s.chars()[0], 'c', "FAIL: String.chars()[0]")
assertEqual(s.chars()[3], 'é', "FAIL: String.chars()[3] (multi-byte character)")
assertEqual(len("".chars()), 0, "FAIL: empty String.chars()")

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
assertEqual(len(collected), 2, "FAIL: expr-form String iterate collected wrong count")
assertEqual(collected[0], 'a', "FAIL: expr-form String iterate [0]")
assertEqual(collected[1], 'b', "FAIL: expr-form String iterate [1]")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func TestBinaryAndStringIndexingYieldByte(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

import bytes

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

const b = bytes.fromString("hi")

assertEqual(len(b), 2, "FAIL: Binary length via len()")
assertEqual(Countable(b).length(b), 2, "FAIL: Countable.length on Binary")

assertEqual(b[0] is Byte, true, "FAIL: Binary index yields Byte")
assertEqual(b[0] is Binary, false, "FAIL: a Byte is not itself a Binary")
assertEqual(b[0] is Int, false, "FAIL: a Byte is not an Int")

const s = "hi"
assertEqual(s[0] is Byte, true, "FAIL: String index yields Byte")

fn countBytes(binary) {
	var count = 0
	for x <- binary {
		assertEqual(x is Byte, true, "FAIL: Binary iterate visits Byte values")
		count = count + 1
	}
	return count
}
assertEqual(countBytes(b), 2, "FAIL: Binary iterate visits one Byte per element")
assertEqual(countBytes(bytes.fromString("")), 0, "FAIL: empty Binary visits nothing")

fn breaksEarly() {
	var count = 0
	for x <- bytes.fromString("abcdef") {
		count = count + 1
		if count == 3 {
			break
		}
	}
	return count
}
assertEqual(breaksEarly(), 3, "FAIL: break stops Binary iteration early")

const collected = for x <- bytes.fromString("ab") { x is Byte }
assertEqual(len(collected), 2, "FAIL: expr-form Binary iterate collected wrong count")
assertEqual(collected[0], true, "FAIL: expr-form Binary iterate [0]")
assertEqual(collected[1], true, "FAIL: expr-form Binary iterate [1]")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestSwitchWithBreakAndContinueInsideForLoop is a regression test: a switch statement nested directly in a (statement-form) for loop's body had no case in compileLoopBlock, so break/continue inside one of its cases fell through to the generic compileStmtBreak/compileStmtContinue stubs, which always error "break/continue used outside of loop" — even though they plainly are inside one. Mirrors the same, separately-fixed gap in expr-for's own loop-body compiler.
func TestSwitchWithBreakAndContinueInsideForLoop(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

fn run() -> Int {
	var sum = 0
	for x <- [1, 2, 3, 4, 5] {
		switch x {
		case is Int:
			if x == 2 {
				continue
			}
			if x == 4 {
				break
			}
			sum = sum + x
		case _:
			panic("unreachable")
		}
	}
	return sum
}
assertEqual(run(), 4, "FAIL: expected 1 + 3 (2 skipped via continue, loop stopped before 4 via break)")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestExprForReferencesOuterScopeVariable is a regression test: any identifier referenced inside an expr-for body but declared in its enclosing function used to crash with "free variable ... not found in free mapping". The analyzer's generic SymbolTable.resolve() promotes any cross-table lookup to a FreeScope symbol (defineFree) whenever a name isn't found in the current table — including an expr-for body's own same-frame SymbolTable, which exists purely for lexical scoping, not because expr-for is an actual closure boundary (compileExprFor never calls enterScope). The compiler's free-symbol handling only ever jumped straight to symbol.Original(), which is only safe when no real closure boundary sits in between.
func TestExprForReferencesOuterScopeVariable(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

fn run() -> [Int] {
	const multiplier = 10
	var flag = false
	for x <- [1] {
		flag = true
	}
	return for x <- [1, 2, 3] {
		if flag { x * multiplier } else { 0 }
	}
}
const r = run()
assertEqual(r[0], 10, "FAIL: r[0] should reflect the outer const and the mutated outer var")
assertEqual(r[1], 20, "FAIL: r[1] should reflect the outer const and the mutated outer var")
assertEqual(r[2], 30, "FAIL: r[2] should reflect the outer const and the mutated outer var")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestExprForInRealClosureStillCapturesGrandparentScope is a regression test guarding TestExprForReferencesOuterScopeVariable's fix against the one case it must NOT take a same-frame shortcut for: an expr-for nested inside a real closure (ExprFunc, which does call enterScope) referencing a variable from a scope further out still needs an actual runtime capture, not a direct same-frame local read (which would silently read the wrong frame's slot).
func TestExprForInRealClosureStillCapturesGrandparentScope(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

fn outer() {
	const x = 100
	return fn() {
		return for y <- [1, 2, 3] {
			x + y
		}
	}
}
const inner = outer()
const r = inner()
assertEqual(r[0], 101, "FAIL: r[0] should be the captured outer const (100) + 1")
assertEqual(r[1], 102, "FAIL: r[1] should be the captured outer const (100) + 2")
assertEqual(r[2], 103, "FAIL: r[2] should be the captured outer const (100) + 3")
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

// TestRunFileNoDeclaredDependencies verifies that a project with no explicit dependencies (only stdlib injected automatically) can still resolve and run its own modules. This guards against a coupling bug where the project package itself was only installed when declared dependencies were missing.
func TestRunFileNoDeclaredDependencies(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst answer = \"42\"\n")

	// Deliberately no PackageName-derived dependencies beyond stdlib (injected automatically).
	orch := newTestOrchestra(t, projectFS, "main")

	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestRunFileWithCrossModuleImport verifies that a file importing another module within the same project is resolved correctly. This exercises the cavereg filesystem path: the resolver must discover project sub-modules from ProjectFS (not RegistryFS) via findResolvedModule.
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

// TestMainModuleIsRegisteredAfterParse verifies that after ParseFile the resolver's MainModule returns the same pointer, which is required for the compiler's pointer-equality checks to work correctly.
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

// TestRunFileWithConstBinding verifies that a file with a const binding compiles and runs correctly — exercising full main-module symbol compilation.
func TestRunFileWithLetBinding(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst greeting = \"hello\"\n")

	orch := newTestOrchestra(t, projectFS, "main")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestREPLLoop verifies the REPL's parse→compile pattern: a single resolver is reused across multiple ParseFile+Compile calls, as the REPL does.
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

// TestRunFileImportPrelude verifies that explicitly importing the prelude (which contains extern const declarations like void) compiles and runs without "unknown declaration *ast.DeclExternValue" errors.
func TestRunFileImportPrelude(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nimport prelude = prelude\n")

	orch := newTestOrchestra(t, projectFS, "main")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestAutoImportPrelude verifies that the prelude module is automatically imported so that `prelude.X` references work without an explicit import.
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

// TestAutoImportPreludeSkipsPrelude verifies that the prelude module itself does not get a synthetic prelude import (avoiding circularity).
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

// TestShortModulePath verifies that short module names like "prelude" resolve to the full stdlib URI automatically.
func TestShortModulePath(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nimport p = prelude\n")

	orch := newTestOrchestra(t, projectFS, "main")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestShortModulePathFuture verifies that "future" resolves to the full stdlib path.
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

// TestShortModulePathNotFoundUsesOriginalName verifies that when a short module name can't be found even with the stdlib prefix, the error message uses the original short name.
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

// TestBarePreludeSymbol verifies that prelude symbols like `String` are directly accessible without the `prelude.` prefix.
func TestBarePreludeSymbol(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst x = String\n")

	orch := newTestOrchestra(t, projectFS, "main")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestBarePreludeSymbolInAttribute verifies that prelude types used inside attribute arguments (e.g. @Deprecated("reason")) compile correctly.
func TestBarePreludeSymbolInAttribute(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\n@Deprecated(\"use NewExample\")\ndata Example { name }\n")

	orch := newTestOrchestra(t, projectFS, "main")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestBarePreludeSymbolShadowed verifies that a user declaration can shadow a prelude symbol.
func TestBarePreludeSymbolShadowed(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst String = 42\n")

	orch := newTestOrchestra(t, projectFS, "main")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestBarePreludeAndQualified verifies that both `String` and `prelude.String` work in the same file.
func TestBarePreludeAndQualified(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", "mod main\nconst x = String\nconst y = prelude.String\n")

	orch := newTestOrchestra(t, projectFS, "main")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestCavefileExample verifies that the example Cavefile from examples/project/Cavefile compiles and runs without errors.
// The Cavefile uses cross-module attribute references like @cave.Package and @tasks.Name which require file-local imports to be visible during attribute resolution of promoted declarations.
func TestCavefileExample(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "Cavefile", cavefileContent)

	orch := newTestOrchestra(t, projectFS, "examples.project")
	if err := orch.RunFile(context.Background(), "Cavefile"); err != nil {
		t.Fatalf("run Cavefile: %v", err)
	}
}

// TestParseModuleWithErrors verifies that ParseModule returns a non-nil module alongside a parser.ParseErrors error when the source has syntax errors.
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

// TestInvalidateModules verifies that InvalidateModules clears cached modules but preserves the prelude.
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
	writeFile(t, projectFS, "Cavefile", `mod proj

import cave

@cave.Package()
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

// TestReadOnlyResolver verifies that a read-only resolver can parse modules using only locally-available packages.
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

// TestTailIfAndSwitchImplicitlyReturnValue is a regression test: the parser only converts a function body into an implicit return when the body is a single bare expression statement (e.g. `fn f() { Ok(true) }`), not when it's an `if`/`switch` statement (e.g. `fn f() { if c { Ok(true) } else { Err() } }`) — even as the last statement of a multi-statement body, or inside a closure.
// Those used to silently return Void.
// Fixed at the compiler level (compileFuncBody/compileTailStmt in pkg/compiler), routing a trailing if/switch through the same value-producing path a trailing bare expression already used, recursively for nested branches.
func TestTailIfAndSwitchImplicitlyReturnValue(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

fn ifTail(x) -> Int {
	if x > 0 { 1 } else { -1 }
}

fn ifTailMultiStatement(x) -> Int {
	const doubled = x * 2
	if doubled > 0 { 1 } else { -1 }
}

fn switchTail(x) -> Int {
	switch x {
	case is Int:
		1
	case _:
		-1
	}
}

const closureIfTail = fn(x) -> Int {
	if x > 0 { 1 } else { -1 }
}

assertEqual(ifTail(5), 1, "FAIL: bare if-statement as function tail should return its branch value")
assertEqual(ifTail(-5), -1, "FAIL: bare if-statement as function tail should return its else value")
assertEqual(ifTailMultiStatement(5), 1, "FAIL: trailing if in a multi-statement body should still return its value")
assertEqual(switchTail(5), 1, "FAIL: bare switch-statement as function tail should return its matched case's value")
assertEqual(closureIfTail(5), 1, "FAIL: bare if-statement as a closure's tail should return its branch value")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestTailSwitchNoMatchingCaseDoesNotHaltProgram is a regression test for a bug introduced while fixing TestTailIfAndSwitchImplicitlyReturnValue: a trailing switch with no case matching the runtime value (no exhaustive `case _:` present) must still return (Void), not fall off the end of the function's instructions — which silently halted the VM's whole dispatch loop, not just that function call, leaving every statement after the call unexecuted with no error.
func TestTailSwitchNoMatchingCaseDoesNotHaltProgram(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

union Ev {
	data A {}
	data B { x: Int }
}

fn handle(e: Ev) {
	switch e {
	case is A:
		1
	}
}

handle(A())
handle(B(1))
panic("reached end")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	err := orch.RunFile(context.Background(), "main.zirr")
	if err == nil {
		t.Fatal("expected the script to run to completion and panic, got nil error")
	}
	if !strings.Contains(err.Error(), "reached end") {
		t.Fatalf("expected execution to continue past the unmatched tail switch case and reach the final panic, got: %v", err)
	}
}

// TestStmtIfWithoutElseDoesNotInfiniteLoop is a regression test for a compiler bug: compileStmtIf's no-else branch tracked the trailing unconditional Jump it emits after the if-block (to skip a would-be else) in jumpEnds, then overwrote that same slot with jumpNext (the condition-false JumpFalse) instead of adding to it — losing track of the trailing Jump entirely.
// Its operand was left at the placeholder address (math.MinInt), so whenever the condition was true, the VM jumped to a garbage/truncated address — here, one that looped backward, executing the whole test setup forever.
// Only manifests when the if-block's last statement leaves something to Pop (e.g. a bare call), since that's what the vestigial "remove trailing pop" branch this replaced was (incompletely, buggily) reacting to.
func TestStmtIfWithoutElseDoesNotInfiniteLoop(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

var calls = 0

fn sideEffect() -> Int {
	calls = calls + 1
	return calls
}

fn maybe(cond) {
	if cond {
		sideEffect()
	}
}

maybe(true)
maybe(false)
maybe(true)
assertEqual(calls, 2, "FAIL: expected exactly 2 calls, no infinite loop or skipped calls")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestMainPackageReflectionExcludesTheEntryModule is a regression test: the running script is itself a member of the project package, so materializing every module used to load the program a second time.
// Depending on how the entry file was reached that either re-entered an initialization already in progress, which surfaced as "recursive initialization of global variable", or silently executed the whole script again.
func TestMainPackageReflectionExcludesTheEntryModule(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "helper/helper.zirr", `mod helper

fn value() { 7 }
`)
	writeFile(t, projectFS, "other/other.zirr", `mod other

fn value() { 9 }
`)
	writeFile(t, projectFS, "main.zirr", `mod main

import pkgs = reflect.packages

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

const project = pkgs.mainPackage()

var sawEntry = false
for name <- project.moduleNames {
	if name == "project" {
		sawEntry = true
	}
}
assertEqual(sawEntry, false, "FAIL: the entry module must not be listed")

// Previously this re-entered the entry module's own initialization instead of completing.
const loaded = pkgs.modulesExcept(project, [])
assertEqual(len(loaded), 2, "FAIL: expected both project modules to load")

assertEqual(len(pkgs.modulesExcept(project, ["project.other"])), 1, "FAIL: modulesExcept must skip the named module")
assertEqual(len(pkgs.modulesExcept(project, ["project.helper", "project.other"])), 0, "FAIL: modulesExcept must skip every named module")

const helpers = pkgs.modulesWhere(project, fn(name) { name == "project.helper" })
assertEqual(len(helpers), 1, "FAIL: modulesWhere must keep only matching modules")
assertEqual(len(pkgs.modulesWhere(project, fn(name) { false })), 0, "FAIL: modulesWhere must load nothing when the predicate rejects every name")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestMainPackageReflectionSkipsShadowedModules is a regression test: reflect.packages compiles every module of the project package so that each has a global slot, and a project carrying its own directory named after a loaded standard library module used to get that copy compiled too.
// For a module like prelude that declares the core types, the program then held two incompatible definitions of them, and something as ordinary as len() on a value produced by the first would fail to find @Countable via the second.
// Such a module is unreachable anyway, since the bare name resolves to the loaded one, so it is left out entirely.
func TestMainPackageReflectionSkipsShadowedModules(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "prelude/shadow.zirr", `mod prelude

const shadowMarker = 1
`)
	writeFile(t, projectFS, "greeting/greeting.zirr", `mod greeting

fn hello() { "hello" }
`)
	writeFile(t, projectFS, "main.zirr", `mod main

import pkgs = reflect.packages

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

const project = pkgs.mainPackage()

// len() still resolves @Countable, which is what a second compiled prelude used to break.
assertEqual(len(project.moduleNames) > 0, true, "FAIL: expected the project package to declare modules")

var sawShadowedPrelude = false
var sawGreeting = false
for name <- project.moduleNames {
	if name == "project.prelude" {
		sawShadowedPrelude = true
	}
	if name == "project.greeting" {
		sawGreeting = true
	}
}
assertEqual(sawShadowedPrelude, false, "FAIL: the shadowed prelude copy must not be reported")
assertEqual(sawGreeting, true, "FAIL: an ordinary project module must still be reported")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestSequentialForLoopsMayReuseABindingName is a regression test: a statement-form `for x <- v` body shares the enclosing scope, and the binding was inserted there unconditionally, so a second loop binding the same name in the same function failed to compile with "symbol already defined".
// A loop nested inside another that binds the same name is still rejected, since there the two bindings really are live at once.
func TestSequentialForLoopsMayReuseABindingName(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

fn sums(v) {
	var first = 0
	for c <- v {
		first = first + c
	}
	var second = 0
	for c <- v {
		second = second + c * 10
	}
	var third = 0
	for c <- v {
		third = third + c * 100
	}
	return [first, second, third]
}

fn nestedDistinct(v) {
	var n = 0
	for outer <- v {
		for inner <- v {
			n = n + 1
		}
	}
	for inner <- v {
		n = n + 100
	}
	return n
}

const totals = sums([1, 2, 3])
assertEqual(totals[0], 6, "FAIL: first loop")
assertEqual(totals[1], 60, "FAIL: second loop reusing the binding name")
assertEqual(totals[2], 600, "FAIL: third loop reusing the binding name")
assertEqual(nestedDistinct([1, 2]), 204, "FAIL: nested distinct names plus sequential reuse")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestNestedForLoopsMayNotReuseABindingName guards the other half of TestSequentialForLoopsMayReuseABindingName: reuse is only safe once the earlier loop has finished, so an inner loop rebinding an enclosing loop's name must still be reported rather than silently sharing its slot.
func TestNestedForLoopsMayNotReuseABindingName(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

fn f(v) {
	var n = 0
	for c <- v {
		for c <- v {
			n = n + 1
		}
	}
	return n
}

f([1, 2])
`)

	orch := newTestOrchestra(t, projectFS, "project")
	err := orch.RunFile(context.Background(), "main.zirr")
	if err == nil {
		t.Fatal("expected a redeclaration error for a nested loop rebinding the same name")
	}
	if !strings.Contains(err.Error(), "symbol already defined") {
		t.Fatalf("expected a redeclaration error, got: %v", err)
	}
}

// TestStmtIfWithoutElseInsideForLoopDoesNotInfiniteLoop is a regression test for the exact same bug as TestStmtIfWithoutElseDoesNotInfiniteLoop, but in a second, independent copy of the buggy code: compileStmtIf's no-else jump-patching bug was duplicated (not shared) across compileStmtIfInLoop (statement-form `for x <- v {...}` bodies) and compileExprForIfInLoop (expr-form `for x <- v {...}` bodies, i.e. `for`-comprehensions) — fixing compileStmtIf alone left both of those still broken.
// Found via the `arrays` module's `filter`, whose `for x <- v { if predicate(x) { append(...) } }` (if without else, last statement is a bare call) hit exactly this shape and hung forever.
func TestStmtIfWithoutElseInsideForLoopDoesNotInfiniteLoop(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

fn filter(v, predicate) {
	var result = []
	for x <- v {
		if predicate(x) {
			result = append(result, x)
		}
	}
	return result
}

const evens = filter([1, 2, 3, 4, 5, 6], fn(x) { x % 2 == 0 })
assertEqual(len(evens), 3, "FAIL: expected exactly 3 matches, no infinite loop or skipped elements")
assertEqual(evens[0], 2, "FAIL: first even")
assertEqual(evens[2], 6, "FAIL: last even")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

// TestTopLevelConstCanReferenceImport is a regression test: a top-level (module-scope) `const`/`var` whose value expression references an imported module used to fail with "undefined identifier" for the import, even though the exact same reference worked fine as a bare statement or inside a function body.
// Root cause: ast.DeclTable.Insert promotes an exported top-level const/var into the module's flattened symbol table (module.Decls.Symbols) rather than leaving it in its declaring file's own table — and analyzer.resolveIdentifiers resolved every promoted decl's body using that flattened module.Symbols table, which (unlike a file's own symbol table) never includes that file's imports, since imports are never promoted.
// DeclFunc bodies were unaffected because they resolve using their own dedicated, correctly file-parented symbol table (n.Impl.Symbols) instead of whatever table the caller passed in — const and var had no equivalent, so this fix (symbolsForPromotedDecl in pkg/analyzer/resolve_identifiers.go) gives them one by matching the declaring file via source path, the same pattern compiler.sourceFileSymbols already uses elsewhere.
func TestTopLevelConstCanReferenceImport(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", `mod main

import fmt

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

const greeting = fmt.sprint("hi")
var farewell = fmt.sprint("bye")

assertEqual(greeting, "hi", "FAIL: top-level const referencing an import")
assertEqual(farewell, "bye", "FAIL: top-level var referencing an import")
`)

	orch := newTestOrchestra(t, projectFS, "project")
	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}
