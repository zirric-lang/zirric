package runtime

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ CallableRuntimeValue = ExternFunc{}

type ExternFuncImpl func(args []RuntimeValue) RuntimeValue

type ExternFunc struct {
	symbol          *ast.Symbol
	arity           int
	Impl            ExternFuncImpl
	Attributes      map[TypeId]int
	ParamAttributes []map[TypeId]int
}

func MakeExternFunc(symbol *ast.Symbol, impl ExternFuncImpl) (ExternFunc, error) {
	decl, ok := symbol.Decl.(*ast.DeclExternFunc)
	if !ok {
		return ExternFunc{}, fmt.Errorf("declaration is not a DeclExternFunc, got %T", symbol.Decl)
	}
	return ExternFunc{
		symbol: symbol,
		arity:  len(decl.Parameters),
		Impl:   impl,
	}, nil
}

// Arity implements CallableRuntimeValue.
func (ef ExternFunc) Arity() int {
	return ef.arity
}

// Inspect implements CallableRuntimeValue.
func (ef ExternFunc) Inspect() string {
	return fmt.Sprintf("extern %s(#%d)", ef.symbol.Decl.DeclName(), ef.arity)
}

// Lookup implements CallableRuntimeValue.
func (ef ExternFunc) Lookup(name string) RuntimeValue {
	if name == "arity" {
		// return ef.arity
		panic("unimplemented: how to create ints?")
	}
	return nil
}

// TypeConstantId implements CallableRuntimeValue.
func (ef ExternFunc) TypeConstantId() TypeId {
	return TypeId(*ef.symbol.TypeSymbol.ConstantId)
}
