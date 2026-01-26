package runtime

import (
	"fmt"
)

var _ CallableRuntimeValue = &Closure{}

type Closure struct {
	Fn   *CompiledFunction
	Free []RuntimeValue
}

func MakeClosure(fun *CompiledFunction, free []RuntimeValue) *Closure {
	return &Closure{
		Fn:   fun,
		Free: free,
	}
}

// Arity implements CallableRuntimeValue.
func (c *Closure) Arity() int {
	return c.Fn.Params
}

// Inspect implements CallableRuntimeValue.
func (c *Closure) Inspect() string {
	return fmt.Sprintf("func %s(#%d)", c.Fn.Symbol.Decl.DeclName(), c.Arity())
}

// Lookup implements CallableRuntimeValue.
func (c *Closure) Lookup(name string) RuntimeValue {
	if name == "arity" {
		return Int(c.Arity())
	}
	return nil
}

// TypeConstantId implements CallableRuntimeValue.
func (c *Closure) TypeConstantId() TypeId {
	return TypeId(*c.Fn.Symbol.TypeSymbol.ConstantId)
}
