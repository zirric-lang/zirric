package vm_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/compiler"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
	"code.knabel.dev/zirric-lang/zirric/pkg/vm"
)

func compileFunction(t *testing.T, input string, name string) (*compiler.Bytecode, runtime.RuntimeValue) {
	t.Helper()
	module, program := prepareSourceFileParsing(t, input)
	resolver := newTestModuleResolver(module)

	comp := compiler.New(resolver)
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compile: %v", err)
	}
	bytecode := comp.Bytecode()

	sym, ok := module.Symbols.Symbols[name]
	if !ok || sym.ConstantId == nil {
		t.Fatalf("symbol %q not found or missing a ConstantId", name)
	}
	return bytecode, bytecode.Constants[*sym.ConstantId]
}

func TestCallFunctionWithArguments(t *testing.T) {
	bytecode, fn := compileFunction(t, "fn add(a, b) { return a + b }", "add")
	machine := vm.New(bytecode)
	if err := machine.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}

	result, err := machine.CallFunction(fn, runtime.Int(2), runtime.Int(3))
	if err != nil {
		t.Fatalf("call function: %v", err)
	}
	testExpectedValue(t, 5, result)
}

func TestCallFunctionZeroArguments(t *testing.T) {
	bytecode, fn := compileFunction(t, "fn greet() { return \"hi\" }", "greet")
	machine := vm.New(bytecode)
	if err := machine.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}

	result, err := machine.CallFunction(fn)
	if err != nil {
		t.Fatalf("call function: %v", err)
	}
	testExpectedValue(t, "hi", result)
}

func TestCallFunctionWrongArgumentCount(t *testing.T) {
	bytecode, fn := compileFunction(t, "fn add(a, b) { return a + b }", "add")
	machine := vm.New(bytecode)
	if err := machine.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}

	if _, err := machine.CallFunction(fn, runtime.Int(1)); err == nil {
		t.Fatal("expected an error for a wrong argument count, got nil")
	}
}
