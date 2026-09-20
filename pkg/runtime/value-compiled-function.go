package runtime

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/op"
)

var _ CallableRuntimeValue = CompiledFunction{}

type CompiledFunction struct {
	Instructions    op.Instructions
	Params          int
	Locals          int
	Symbol          *ast.Symbol
	Attributes      map[TypeId]int
	ParamAttributes []map[TypeId]int
}

func MakeCompiledFunction(
	instructions op.Instructions,
	params int,
	locals int,
	symbol *ast.Symbol,
) *CompiledFunction {
	return &CompiledFunction{
		Instructions: instructions,
		Params:       params,
		Locals:       locals,
		Symbol:       symbol,
	}
}

// Arity implements CallableRuntimeValue.
func (c CompiledFunction) Arity() int {
	return c.Params
}

// Inspect implements CallableRuntimeValue.
func (c CompiledFunction) Inspect() string {
	if c.Symbol != nil && c.Symbol.Decl != nil {
		return fmt.Sprintf("fn %s(#%d)", c.Symbol.Decl.DeclName(), c.Arity())
	}
	return fmt.Sprintf("fn(#%d)", c.Arity())
}

// Lookup implements CallableRuntimeValue.
func (c CompiledFunction) Lookup(name string) RuntimeValue {
	switch name {
	case "arity":
		return Int(c.Arity())
	case "name":
		return String(c.Symbol.Name)
	}
	return nil
}

// TypeConstantId implements CallableRuntimeValue.
// All compiled functions are of type Func.
func (c CompiledFunction) TypeConstantId() TypeId {
	return BuiltinTypeIds["Func"]
}

func (c CompiledFunction) TypeAttributes() map[TypeId]int {
	return c.Attributes
}
