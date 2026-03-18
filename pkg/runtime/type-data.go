package runtime

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ CallableRuntimeValue = &DataType{}

type DataType struct {
	Symbol       *ast.Symbol
	FieldSymbols []*ast.Symbol
	Attributes   map[TypeId]int
}

func MakeDataType(symbol *ast.Symbol) (*DataType, error) {
	decl, ok := symbol.Decl.(*ast.DeclData)
	if !ok {
		return nil, fmt.Errorf("declaration is not a DeclData, got %T", symbol.Decl)
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

	return &DataType{
		Symbol:       symbol,
		FieldSymbols: fieldSymbols,
		Attributes:   nil,
	}, nil
}

// Arity implements Callable.
func (dt *DataType) Arity() int {
	return len(dt.FieldSymbols)
}

// Inspect implements Callable.
func (dt *DataType) Inspect() string {
	return fmt.Sprintf("data %s", dt.Symbol.Decl.DeclName())
}

// Lookup implements Callable.
func (dt *DataType) Lookup(name string) RuntimeValue {
	panic("unimplemented")
}

// TypeConstantId implements Callable.
func (dt *DataType) TypeConstantId() TypeId {
	return TypeId(*dt.Symbol.TypeSymbol.ConstantId)
}

func (dt *DataType) TypeAttributes() map[TypeId]int {
	return dt.Attributes
}
