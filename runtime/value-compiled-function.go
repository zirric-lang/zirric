package runtime

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/ast"
	"code.knabel.dev/zirric-lang/zirric/op"
)

var _ CallableRuntimeValue = CompiledFunction{}

type CompiledFunction struct {
	Instructions op.Instructions
	Params       int
	Symbol       *ast.Symbol
}

func MakeCompiledFunction(
	instructions op.Instructions,
	params int,
	symbol *ast.Symbol,
) *CompiledFunction {
	return &CompiledFunction{
		Instructions: instructions,
		Params:       params,
		Symbol:       symbol,
	}
}

// Arity implements CallableRuntimeValue.
func (c CompiledFunction) Arity() int {
	return c.Params
}

// Inspect implements CallableRuntimeValue.
func (c CompiledFunction) Inspect() string {
	return fmt.Sprintf("func %s(#%d)", c.Symbol.Decl.DeclName(), c.Arity())
}

// Lookup implements CallableRuntimeValue.
func (c CompiledFunction) Lookup(name string) RuntimeValue {
	if name == "arity" {
		return Int(c.Arity())
	}
	return nil
}

// TypeConstantId implements CallableRuntimeValue.
func (c CompiledFunction) TypeConstantId() TypeId {
	return TypeId(*c.Symbol.TypeSymbol.ConstantId)
}
