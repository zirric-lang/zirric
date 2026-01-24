package runtime

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/ast"
)

var _ RuntimeValue = &UnionType{}

type UnionType struct {
	symbol *ast.Symbol
}

// Inspect implements RuntimeValue.
func (et *UnionType) Inspect() string {
	return fmt.Sprintf("union %s", et.symbol.Decl.DeclName())
}

// Lookup implements RuntimeValue.
func (*UnionType) Lookup(name string) RuntimeValue {
	return nil
}

// TypeConstantId implements RuntimeValue.
func (et *UnionType) TypeConstantId() TypeId {
	return TypeId(*et.symbol.TypeSymbol.ConstantId)
}
