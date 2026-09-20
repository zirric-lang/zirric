package runtime

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ CallableRuntimeValue = &ExternFunc{}

type ExternFuncImpl func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error)

type ExternFunc struct {
	name            string
	symbol          *ast.Symbol
	arity           int
	Impl            ExternFuncImpl
	Attributes      map[TypeId]int
	ParamAttributes []map[TypeId]int
}

func MakeExternFunc(symbol *ast.Symbol, impl ExternFuncImpl) *ExternFunc {
	decl, ok := symbol.Decl.(*ast.DeclExternFunc)
	if !ok {
		panic(fmt.Errorf("declaration is not a DeclExternFunc, got %T", symbol.Decl))
	}
	return &ExternFunc{
		name:   symbol.Name,
		symbol: symbol,
		arity:  len(decl.Parameters),
		Impl:   impl,
	}
}

// MakeNativeFunc creates a callable ExternFunc without an ast.Symbol.
// Use this for anonymous native closures (e.g. Writer.write implementations).
func MakeNativeFunc(name string, arity int, impl ExternFuncImpl) *ExternFunc {
	return &ExternFunc{
		name:  name,
		arity: arity,
		Impl:  impl,
	}
}

// Arity implements CallableRuntimeValue.
func (ef ExternFunc) Arity() int {
	return ef.arity
}

// Inspect implements CallableRuntimeValue.
func (ef ExternFunc) Inspect() string {
	return fmt.Sprintf("extern %s(#%d)", ef.name, ef.arity)
}

// Lookup implements CallableRuntimeValue.
func (ef ExternFunc) Lookup(name string) RuntimeValue {
	switch name {
	case "arity":
		return Int(ef.Arity())
	case "name":
		return String(ef.name)
	}
	return nil
}

// TypeConstantId implements CallableRuntimeValue.
// All extern functions are of type Func.
func (ef ExternFunc) TypeConstantId() TypeId {
	return BuiltinTypeIds["Func"]
}

func (ef ExternFunc) TypeAttributes() map[TypeId]int {
	return ef.Attributes
}
