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
	if c.Fn.Symbol != nil && c.Fn.Symbol.Decl != nil {
		return fmt.Sprintf("fn %s(#%d)", c.Fn.Symbol.Decl.DeclName(), c.Arity())
	}
	return fmt.Sprintf("fn(#%d)", c.Arity())
}

// Lookup implements CallableRuntimeValue.
func (c *Closure) Lookup(name string) RuntimeValue {
	switch name {
	case "arity":
		return Int(c.Arity())
	case "name":
		if c.Fn.Symbol != nil && c.Fn.Symbol.Decl != nil {
			return String(c.Fn.Symbol.Decl.DeclName().String())
		}
		return String(c.Inspect())
	}
	return nil
}

// TypeConstantId implements CallableRuntimeValue.
func (c *Closure) TypeConstantId() TypeId {
	return c.Fn.TypeConstantId()
}
