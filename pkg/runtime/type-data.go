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
	// FieldAttributes holds the attributes written on each field, by the same index as FieldSymbols.
	// It is nil when no field carries any, so reach for FieldAttributesAt rather than indexing it directly.
	FieldAttributes []map[TypeId]int
	// FieldTypes holds the type hint written on each field, by the same index as FieldSymbols.
	// It is nil when no field carries one, so reach for FieldTypeAt rather than indexing it directly.
	FieldTypes []TypeRef
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

// FieldAttributesAt returns the attributes written on the field at index i, or nil when that field carries none.
func (dt *DataType) FieldAttributesAt(i int) map[TypeId]int {
	if i < 0 || i >= len(dt.FieldAttributes) {
		return nil
	}
	return dt.FieldAttributes[i]
}

// FieldTypeAt returns the type hint written on the field at index i, or an unknown TypeRef when that field carries none.
func (dt *DataType) FieldTypeAt(i int) TypeRef {
	if i < 0 || i >= len(dt.FieldTypes) {
		return TypeRef{}
	}
	return dt.FieldTypes[i]
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
	return nil
}

// TypeConstantId implements Callable.
func (dt *DataType) TypeConstantId() TypeId {
	return TypeId(*dt.Symbol.TypeSymbol.ConstantId)
}

func (dt *DataType) TypeAttributes() map[TypeId]int {
	return dt.Attributes
}
