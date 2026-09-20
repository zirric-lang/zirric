package runtime

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
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

func TestOSPluginModule(t *testing.T) {
	plugin := &OSPlugin{}
	if plugin.Module() != "os" {
		t.Errorf("Module: got %q, want %q", plugin.Module(), "os")
	}
}

func TestOSPluginBindStdout(t *testing.T) {
	plugin := &OSPlugin{}
	sym := makeExternFuncSymbol("stdout", 0)

	val := plugin.Bind(&mockBindContext{}, &ast.SymbolTable{}, sym)
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
	// The stream type is resolved through the running VM, so there is nothing to build without one.
	if _, err := ef.Impl(nil, []RuntimeValue{}); err == nil {
		t.Error("expected an error when called without a VM")
	}
}

func TestOSPluginBindStdin(t *testing.T) {
	plugin := &OSPlugin{}
	sym := makeExternFuncSymbol("stdin", 0)

	val := plugin.Bind(&mockBindContext{}, &ast.SymbolTable{}, sym)
	if val == nil {
		t.Fatal("Bind returned nil for stdin")
	}
	ef, ok := val.(*ExternFunc)
	if !ok {
		t.Fatalf("expected *ExternFunc, got %T", val)
	}
	if _, err := ef.Impl(nil, []RuntimeValue{}); err == nil {
		t.Error("expected an error when called without a VM")
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

// noModuleCaller stands in for a VM that cannot reach the io module, which is the only way stream construction fails now that the type is resolved at runtime rather than passed in.
type noModuleCaller struct{}

func (noModuleCaller) CallFunction(RuntimeValue, ...RuntimeValue) (RuntimeValue, error) {
	return nil, fmt.Errorf("not supported")
}
func (noModuleCaller) AttributesOf(RuntimeValue) map[TypeId]int { return nil }
func (noModuleCaller) ResolveGlobal(int) (RuntimeValue, error) {
	return nil, fmt.Errorf("not supported")
}
func (noModuleCaller) ResolveModuleMember(string, string) (RuntimeValue, error) {
	return nil, fmt.Errorf("module not part of this program")
}

func TestMakeWriteStreamRequiresTheIOModule(t *testing.T) {
	if _, err := MakeWriteStream(noModuleCaller{}, io.Discard); err == nil {
		t.Error("expected an error when io.WriteStream cannot be resolved")
	}
}

func TestMakeReadStreamRequiresTheIOModule(t *testing.T) {
	if _, err := MakeReadStream(noModuleCaller{}, strings.NewReader("")); err == nil {
		t.Error("expected an error when io.ReadStream cannot be resolved")
	}
}
