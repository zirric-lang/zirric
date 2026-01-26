package runtime

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ CallableRuntimeValue = &AnnotationType{}

type AnnotationType struct {
	Symbol       *ast.Symbol
	FieldSymbols []*ast.Symbol
	Annotations  map[TypeId]int
}

func MakeAnnotationType(symbol *ast.Symbol) (*AnnotationType, error) {
	decl, ok := symbol.Decl.(*ast.DeclAnnotation)
	if !ok {
		return nil, fmt.Errorf("declaration is not a DeclAnnotation, got %T", symbol.Decl)
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

	return &AnnotationType{
		Symbol:       symbol,
		FieldSymbols: fieldSymbols,
		Annotations:  nil,
	}, nil
}

// Arity implements CallableRuntimeValue.
func (*AnnotationType) Arity() int {
	return 1
}

// Inspect implements RuntimeValue.
func (at *AnnotationType) Inspect() string {
	return fmt.Sprintf("annotation %s", at.Symbol.Decl.DeclName())
}

// Lookup implements RuntimeValue.
func (*AnnotationType) Lookup(name string) RuntimeValue {
	return nil
}

// TypeConstantId implements RuntimeValue.
func (at *AnnotationType) TypeConstantId() TypeId {
	return TypeId(*at.Symbol.ConstantId)
}

func (at *AnnotationType) MakeValue(values []RuntimeValue) *AnnotationValue {
	return MakeAnnotationValue(at, values)
}
