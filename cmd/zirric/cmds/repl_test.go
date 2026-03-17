package cmds

import (
	"context"
	"fmt"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/compiler"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/vm"
)

// replTestResolver is a minimal module resolver for REPL tests.
type replTestResolver struct {
	main    *ast.ContextModule
	modules map[registry.LogicalURI]*ast.ContextModule
}

func newReplTestResolver(main *ast.ContextModule) replTestResolver {
	return replTestResolver{
		main:    main,
		modules: map[registry.LogicalURI]*ast.ContextModule{main.Name: main},
	}
}

func (r replTestResolver) MainModule() *ast.ContextModule { return r.main }

func (r replTestResolver) ResolveModule(ctx context.Context, name registry.LogicalURI) (*ast.ContextModule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m, ok := r.modules[name]
	if !ok {
		return nil, fmt.Errorf("module %q not found", name)
	}
	return m, nil
}

// newTestReplState builds a minimal replState backed by an in-memory "module repl" module.
// The resolver field is intentionally left nil because replEvalLine never accesses it.
func newTestReplState(t testing.TB) *replState {
	t.Helper()

	moduleURI := registry.LogicalURI("repl")
	src := staticmodule.NewSourceString(moduleURI.Join("main.zirr"), "mod repl\n")
	mod := staticmodule.NewModule(moduleURI, []registry.Source{src})
	mp := parser.NewModuleParse(mod)
	ctxMod, err := mp.Parse(mod)
	if err != nil {
		t.Fatal(err)
	}
	if len(mp.Errors()) > 0 {
		t.Fatalf("parse errors: %v", mp.Errors())
	}

	res := newReplTestResolver(ctxMod)

	analysis := analyzer.New(res)
	if errs, _ := analysis.Analyze(ctxMod, true); len(errs) > 0 {
		t.Fatalf("initial analyze: %v", errs[0])
	}

	comp := compiler.NewWithAnalyzer(res, analysis)
	if err := comp.Compile(ctxMod); err != nil {
		t.Fatal(err)
	}

	machine := vm.New(comp.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatal(err)
	}

	return &replState{
		module:   ctxMod,
		analysis: analysis,
		comp:     comp,
		machine:  machine,
	}
}

// --- Helper tests ---

func TestSnapshotMapKeys(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	snap := snapshotMapKeys(m)

	if _, ok := snap["a"]; !ok {
		t.Error("expected 'a' in snapshot")
	}
	if _, ok := snap["b"]; !ok {
		t.Error("expected 'b' in snapshot")
	}

	// Mutating the original map must not affect the snapshot.
	m["c"] = 3
	if _, ok := snap["c"]; ok {
		t.Error("snapshot must not reflect keys added after snapshot")
	}
}

func TestSnapshotMapKeys_Empty(t *testing.T) {
	snap := snapshotMapKeys(map[string]int{})
	if len(snap) != 0 {
		t.Errorf("expected empty snapshot, got %d keys", len(snap))
	}
}

func TestRemoveAddedKeys(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	before := snapshotMapKeys(m)

	m["c"] = 3
	m["d"] = 4

	removeAddedKeys(m, before)

	if _, ok := m["c"]; ok {
		t.Error("'c' should have been removed by rollback")
	}
	if _, ok := m["d"]; ok {
		t.Error("'d' should have been removed by rollback")
	}
	if m["a"] != 1 {
		t.Error("'a' should still be present after rollback")
	}
	if m["b"] != 2 {
		t.Error("'b' should still be present after rollback")
	}
}

func TestRemoveAddedKeys_NoOp(t *testing.T) {
	m := map[string]int{"a": 1}
	before := snapshotMapKeys(m)
	removeAddedKeys(m, before)
	if len(m) != 1 || m["a"] != 1 {
		t.Error("map should be unchanged when no keys were added")
	}
}

// --- replEvalLine tests ---

func TestReplEvalLine_IntExpression(t *testing.T) {
	state := newTestReplState(t)
	result, err := replEvalLine(state, "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected a result value, got nil")
	}
	if got := result.Inspect(); got != "42" {
		t.Errorf("expected \"42\", got %q", got)
	}
}

func TestReplEvalLine_StringExpression(t *testing.T) {
	state := newTestReplState(t)
	result, err := replEvalLine(state, `"hello"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected a result value, got nil")
	}
	if got := result.Inspect(); got != `"hello"` {
		t.Errorf("expected %q, got %q", `"hello"`, got)
	}
}

func TestReplEvalLine_BoolExpression(t *testing.T) {
	state := newTestReplState(t)

	trueResult, err := replEvalLine(state, "true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if trueResult == nil || trueResult.Inspect() != "true" {
		t.Errorf("expected \"true\", got %v", trueResult)
	}

	falseResult, err := replEvalLine(state, "false")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if falseResult == nil || falseResult.Inspect() != "false" {
		t.Errorf("expected \"false\", got %v", falseResult)
	}
}

func TestReplEvalLine_ArithmeticExpression(t *testing.T) {
	state := newTestReplState(t)
	result, err := replEvalLine(state, "1 + 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.Inspect() != "3" {
		t.Errorf("expected \"3\", got %v", result)
	}
}

func TestReplEvalLine_Declaration_ReturnsVoid(t *testing.T) {
	state := newTestReplState(t)
	result, err := replEvalLine(state, "const x = 42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// A declaration has no executable body; replEvalLine returns nil.
	if result != nil {
		t.Errorf("expected nil result for a declaration, got %v", result.Inspect())
	}
}

func TestReplEvalLine_ReadDeclaredVariable(t *testing.T) {
	state := newTestReplState(t)

	if _, err := replEvalLine(state, "const x = 13"); err != nil {
		t.Fatalf("declaration failed: %v", err)
	}

	result, err := replEvalLine(state, "x")
	if err != nil {
		t.Fatalf("unexpected error reading x: %v", err)
	}
	if result == nil || result.Inspect() != "13" {
		t.Errorf("expected \"13\", got %v", result)
	}
}

func TestReplEvalLine_AccumulatesState(t *testing.T) {
	state := newTestReplState(t)

	if _, err := replEvalLine(state, "const a = 10"); err != nil {
		t.Fatalf("const a failed: %v", err)
	}
	if _, err := replEvalLine(state, "const b = 20"); err != nil {
		t.Fatalf("const b failed: %v", err)
	}

	result, err := replEvalLine(state, "a + b")
	if err != nil {
		t.Fatalf("expression failed: %v", err)
	}
	if result == nil || result.Inspect() != "30" {
		t.Errorf("expected \"30\", got %v", result)
	}
}

func TestReplEvalLine_FunctionDeclarationAndCall(t *testing.T) {
	state := newTestReplState(t)

	// Declaring a function produces no executable __init__, so result is nil.
	result, err := replEvalLine(state, "fn double(n) { return n + n }")
	if err != nil {
		t.Fatalf("func declaration failed: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for func declaration, got %v", result.Inspect())
	}

	// Calling the function should return the computed value.
	result, err = replEvalLine(state, "double(21)")
	if err != nil {
		t.Fatalf("func call failed: %v", err)
	}
	if result == nil || result.Inspect() != "42" {
		t.Errorf("expected \"42\", got %v", result)
	}
}

func TestReplEvalLine_ParseError_RollsBack(t *testing.T) {
	state := newTestReplState(t)

	// An invalid declaration triggers a parse error.
	_, err := replEvalLine(state, "const =")
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}

	// The state must be intact: a valid subsequent line must still work.
	result, err := replEvalLine(state, "42")
	if err != nil {
		t.Fatalf("expected success after parse error rollback: %v", err)
	}
	if result == nil || result.Inspect() != "42" {
		t.Errorf("expected \"42\", got %v", result)
	}
}

func TestReplEvalLine_CompileError_RollsBack(t *testing.T) {
	state := newTestReplState(t)

	// Referencing an undeclared identifier is caught at compile time.
	_, err := replEvalLine(state, "const x = undeclaredVar")
	if err == nil {
		t.Fatal("expected compile error for undeclared identifier, got nil")
	}

	// After rollback, declaring x under the same name must succeed.
	if _, err := replEvalLine(state, "const x = 42"); err != nil {
		t.Fatalf("expected success re-declaring x after rollback: %v", err)
	}

	// And x must hold the value from the successful declaration.
	result, err := replEvalLine(state, "x")
	if err != nil {
		t.Fatalf("unexpected error reading x: %v", err)
	}
	if result == nil || result.Inspect() != "42" {
		t.Errorf("expected \"42\", got %v", result)
	}
}

func TestReplEvalLine_CompileError_DoesNotCorruptSubsequentLines(t *testing.T) {
	state := newTestReplState(t)

	// Multiple failed lines must not accumulate zombie state that
	// corrupts the index mapping of later successful declarations.
	for i := 0; i < 3; i++ {
		if _, err := replEvalLine(state, "const y = undeclaredVar"); err == nil {
			t.Fatalf("iteration %d: expected compile error, got nil", i)
		}
	}

	if _, err := replEvalLine(state, "const y = 99"); err != nil {
		t.Fatalf("let y after repeated failures: %v", err)
	}

	result, err := replEvalLine(state, "y")
	if err != nil {
		t.Fatalf("reading y: %v", err)
	}
	if result == nil || result.Inspect() != "99" {
		t.Errorf("expected \"99\", got %v", result)
	}
}

func TestReplEvalLine_UndeclaredIdentifier_ReturnsError(t *testing.T) {
	state := newTestReplState(t)
	_, err := replEvalLine(state, "thisVarDoesNotExist")
	if err == nil {
		t.Fatal("expected error for undeclared identifier, got nil")
	}
}

func TestReplEvalLine_LineIndexIncrements(t *testing.T) {
	state := newTestReplState(t)
	if state.lineIdx != 0 {
		t.Fatalf("expected initial lineIdx 0, got %d", state.lineIdx)
	}
	replEvalLine(state, "1") //nolint:errcheck
	if state.lineIdx != 1 {
		t.Errorf("expected lineIdx 1 after first eval, got %d", state.lineIdx)
	}
	replEvalLine(state, "2") //nolint:errcheck
	if state.lineIdx != 2 {
		t.Errorf("expected lineIdx 2 after second eval, got %d", state.lineIdx)
	}
}

func TestReplEvalLine_RollbackReclaims_AnalyzerIDs(t *testing.T) {
	// After a failed line, the analyzer's ID counters must be restored so that
	// successful declarations following failures receive contiguous IDs with no gaps.
	state := newTestReplState(t)

	snapBefore := state.analysis.Snapshot()

	// Three failed compile attempts.
	for i := 0; i < 3; i++ {
		if _, err := replEvalLine(state, "const z = undeclaredVar"); err == nil {
			t.Fatalf("iteration %d: expected error, got nil", i)
		}
	}

	snapAfterFailures := state.analysis.Snapshot()

	// Counters must be identical to pre-failure state.
	if snapAfterFailures.NextGlobal() != snapBefore.NextGlobal() {
		t.Errorf("nextGlobal leaked: want %d, got %d", snapBefore.NextGlobal(), snapAfterFailures.NextGlobal())
	}
	if snapAfterFailures.NextConstant() != snapBefore.NextConstant() {
		t.Errorf("nextConstant leaked: want %d, got %d", snapBefore.NextConstant(), snapAfterFailures.NextConstant())
	}

	// A successful declaration after failures must still work correctly.
	if _, err := replEvalLine(state, "const z = 7"); err != nil {
		t.Fatalf("declaration after rollbacks: %v", err)
	}
	result, err := replEvalLine(state, "z")
	if err != nil {
		t.Fatalf("reading z: %v", err)
	}
	if result == nil || result.Inspect() != "7" {
		t.Errorf("expected \"7\", got %v", result)
	}
}
