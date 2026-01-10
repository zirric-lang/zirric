package runtime

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/ast"
)

var _ RuntimeValue = &EnumType{}

type EnumType struct {
	symbol *ast.Symbol
}

// Inspect implements RuntimeValue.
func (et *EnumType) Inspect() string {
	return fmt.Sprintf("data %s", et.symbol.Decl.DeclName())
}

// Lookup implements RuntimeValue.
func (*EnumType) Lookup(name string) RuntimeValue {
	return nil
}

// TypeConstantId implements RuntimeValue.
func (et *EnumType) TypeConstantId() TypeId {
	return TypeId(*et.symbol.TypeSymbol.ConstantId)
}
