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
	// FieldAttributes holds the attributes written on each field, by the same index as FieldSymbols.
	// It is nil when no field carries any, so reach for FieldAttributesAt rather than indexing it directly.
	FieldAttributes []map[TypeId]int
	// FieldTypes holds the type hint written on each field, by the same index as FieldSymbols.
	// It is nil when no field carries one, so reach for FieldTypeAt rather than indexing it directly.
	FieldTypes []TypeRef
	// FieldDocs holds the comment written above each field, by the same index as FieldSymbols.
	// It is nil when no field carries one, so reach for FieldDocsAt rather than indexing it directly.
	FieldDocs []string
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

// Fields implements FieldedType.
func (at *AttributeType) Fields() []*ast.Symbol {
	return at.FieldSymbols
}

// FieldAttributesAt implements FieldedType.
func (at *AttributeType) FieldAttributesAt(i int) map[TypeId]int {
	if i < 0 || i >= len(at.FieldAttributes) {
		return nil
	}
	return at.FieldAttributes[i]
}

// FieldTypeAt implements FieldedType.
func (at *AttributeType) FieldTypeAt(i int) TypeRef {
	if i < 0 || i >= len(at.FieldTypes) {
		return TypeRef{}
	}
	return at.FieldTypes[i]
}

// FieldDocsAt implements FieldedType.
func (at *AttributeType) FieldDocsAt(i int) string {
	if i < 0 || i >= len(at.FieldDocs) {
		return ""
	}
	return at.FieldDocs[i]
}
