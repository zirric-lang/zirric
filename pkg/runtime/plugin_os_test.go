package runtime

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// mockBindContext provides symbol resolution for tests.
type mockBindContext struct {
	symbols map[string]map[string]*ast.Symbol
}

func (m *mockBindContext) ResolveModuleSymbol(moduleName string, symbolName string) *ast.Symbol {
	if mod, ok := m.symbols[moduleName]; ok {
		return mod[symbolName]
	}
	return nil
}

func (m *mockBindContext) MainPackageModules() (string, map[string]int) {
	return "", nil
}

func makeDataSymbol(name string, constantId int) *ast.Symbol {
	ident := ast.MakeIdentifier(token.Token{Literal: name})
	decl := ast.MakeDeclData(token.Token{}, ident)
	return &ast.Symbol{
		Name:       name,
		ConstantId: &constantId,
		Decl:       decl,
	}
}

func TestOSPluginModule(t *testing.T) {
	plugin := &OSPlugin{}
	if plugin.Module() != "os" {
		t.Errorf("Module: got %q, want %q", plugin.Module(), "os")
	}
}

func TestOSPluginBindStdout(t *testing.T) {
	plugin := &OSPlugin{}
	writerConstId := 42
	ctx := &mockBindContext{
		symbols: map[string]map[string]*ast.Symbol{
			"io": {"Writer": makeDataSymbol("Writer", writerConstId)},
		},
	}
	sym := makeExternFuncSymbol("stdout", 0)
	module := &ast.SymbolTable{}

	val := plugin.Bind(ctx, module, sym)
	if val == nil {
		t.Fatal("Bind returned nil for stdout")
	}
	ef, ok := val.(*ExternFunc)
	if !ok {
		t.Fatalf("expected *ExternFunc, got %T", val)
	}
	if ef.Arity() != 0 {
		t.Errorf("arity: got %d, want 0", ef.Arity())
	}

	// Call the extern fn to get a Writer DataValue.
	result, err := ef.Impl(nil, []RuntimeValue{})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	dv, ok := result.(*DataValue)
	if !ok {
		t.Fatalf("expected *DataValue, got %T", result)
	}
	if dv.TypeId != TypeId(writerConstId) {
		t.Errorf("TypeId: got %d, want %d", dv.TypeId, writerConstId)
	}
	if _, ok := dv.Fields["write"]; !ok {
		t.Error("Writer DataValue missing 'write' field")
	}
	// Verify the write function is callable.
	writeFn, ok := dv.Values[dv.Fields["write"]].(*ExternFunc)
	if !ok {
		t.Fatalf("write field is not *ExternFunc, got %T", dv.Values[dv.Fields["write"]])
	}
	if writeFn.Arity() != 1 {
		t.Errorf("write fn arity: got %d, want 1", writeFn.Arity())
	}
}

func TestOSPluginBindStdin(t *testing.T) {
	plugin := &OSPlugin{}
	readerConstId := 43
	ctx := &mockBindContext{
		symbols: map[string]map[string]*ast.Symbol{
			"io": {"Reader": makeDataSymbol("Reader", readerConstId)},
		},
	}
	sym := makeExternFuncSymbol("stdin", 0)
	module := &ast.SymbolTable{}

	val := plugin.Bind(ctx, module, sym)
	if val == nil {
		t.Fatal("Bind returned nil for stdin")
	}
	ef := val.(*ExternFunc)
	result, err := ef.Impl(nil, []RuntimeValue{})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	dv, ok := result.(*DataValue)
	if !ok {
		t.Fatalf("expected *DataValue, got %T", result)
	}
	if dv.TypeId != TypeId(readerConstId) {
		t.Errorf("TypeId: got %d, want %d", dv.TypeId, readerConstId)
	}
	if _, ok := dv.Fields["read"]; !ok {
		t.Error("Reader DataValue missing 'read' field")
	}
}

func TestOSPluginBindExit(t *testing.T) {
	plugin := &OSPlugin{}
	ctx := &mockBindContext{}
	sym := makeExternFuncSymbol("exit", 1)
	module := &ast.SymbolTable{}

	val := plugin.Bind(ctx, module, sym)
	if val == nil {
		t.Fatal("Bind returned nil for exit")
	}
	ef, ok := val.(*ExternFunc)
	if !ok {
		t.Fatalf("expected *ExternFunc, got %T", val)
	}
	if ef.Arity() != 1 {
		t.Errorf("arity: got %d, want 1", ef.Arity())
	}
}

func TestOSPluginBindEnv(t *testing.T) {
	plugin := &OSPlugin{}
	ctx := &mockBindContext{}
	sym := makeExternFuncSymbol("env", 1)
	module := &ast.SymbolTable{}

	val := plugin.Bind(ctx, module, sym)
	if val == nil {
		t.Fatal("Bind returned nil for env")
	}
	ef := val.(*ExternFunc)
	// Call with a known env variable.
	result, err := ef.Impl(nil, []RuntimeValue{String("PATH")})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if _, ok := result.(String); !ok {
		t.Errorf("expected String result, got %T", result)
	}
}

func TestOSPluginBindArgs(t *testing.T) {
	plugin := &OSPlugin{}
	ctx := &mockBindContext{}
	sym := makeExternFuncSymbol("args", 0)
	module := &ast.SymbolTable{}

	val := plugin.Bind(ctx, module, sym)
	if val == nil {
		t.Fatal("Bind returned nil for args")
	}
	ef := val.(*ExternFunc)
	result, err := ef.Impl(nil, []RuntimeValue{})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	arr, ok := result.(Array)
	if !ok {
		t.Fatalf("expected Array result, got %T", result)
	}
	// os.Args always has at least one element (the binary name).
	if len(arr) == 0 {
		t.Error("expected at least one arg")
	}
}

func TestOSPluginBindUnknown(t *testing.T) {
	plugin := &OSPlugin{}
	ctx := &mockBindContext{}
	sym := makeExternFuncSymbol("nonexistent", 0)
	module := &ast.SymbolTable{}

	val := plugin.Bind(ctx, module, sym)
	if val != nil {
		t.Errorf("expected nil for unknown declaration, got %T", val)
	}
}

func TestOSPluginWriterWithNilSymbol(t *testing.T) {
	// Verify that makeWriterValue returns an error when the symbol is nil.
	_, err := makeWriterValue(nil, nil)
	if err == nil {
		t.Error("expected error for nil writer symbol")
	}
}

func TestOSPluginReaderWithNilSymbol(t *testing.T) {
	// Verify that makeReaderValue returns an error when the symbol is nil.
	_, err := makeReaderValue(nil, nil)
	if err == nil {
		t.Error("expected error for nil reader symbol")
	}
}
