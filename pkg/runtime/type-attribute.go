package runtime

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ CallableRuntimeValue = &AttributeType{}

type AttributeType struct {
	Symbol       *ast.Symbol
	FieldSymbols []*ast.Symbol
	Attributes   map[TypeId]int
}

func MakeAttributeType(symbol *ast.Symbol) (*AttributeType, error) {
	decl, ok := symbol.Decl.(*ast.DeclAttr)
	if !ok {
		return nil, fmt.Errorf("declaration is not a DeclAttr, got %T", symbol.Decl)
	}
	fieldSymbols := make([]*ast.Symbol, len(decl.Fields))
	for i, f := range decl.Fields {
		for _, fsym := range symbol.ChildTable.Symbols {
			if fsym.Decl.DeclName().String() == f.DeclName().String() {
				fieldSymbols[i] = fsym
			}
		}
		if fieldSymbols[i] == nil {
			return nil, fmt.Errorf("no symbol for field: %q", f.DeclName().String())
		}
	}

	return &AttributeType{
		Symbol:       symbol,
		FieldSymbols: fieldSymbols,
		Attributes:   nil,
	}, nil
}

// Arity implements CallableRuntimeValue.
func (*AttributeType) Arity() int {
	return 1
}

// Inspect implements RuntimeValue.
func (at *AttributeType) Inspect() string {
	return fmt.Sprintf("attr %s", at.Symbol.Decl.DeclName())
}

// Lookup implements RuntimeValue.
func (*AttributeType) Lookup(name string) RuntimeValue {
	return nil
}

// TypeConstantId implements RuntimeValue.
func (at *AttributeType) TypeConstantId() TypeId {
	return TypeId(*at.Symbol.ConstantId)
}

func (at *AttributeType) MakeValue(values []RuntimeValue) *AttributeValue {
	return MakeAttributeValue(at, values)
}

func (at *AttributeType) TypeAttributes() map[TypeId]int {
	return at.Attributes
}
