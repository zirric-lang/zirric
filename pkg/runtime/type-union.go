package runtime

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ RuntimeValue = &UnionType{}

type UnionType struct {
	Symbol        *ast.Symbol
	MemberTypeIds []TypeId
	Attributes    map[TypeId]int
}

func MakeUnionType(symbol *ast.Symbol, memberTypeIds []TypeId) *UnionType {
	return &UnionType{
		Symbol:        symbol,
		MemberTypeIds: memberTypeIds,
	}
}

// IsMember returns true if the given type ID is a direct member of this union.
func (ut *UnionType) IsMember(typeId TypeId) bool {
	for _, mid := range ut.MemberTypeIds {
		if mid == typeId {
			return true
		}
	}
	return false
}

// Inspect implements RuntimeValue.
func (ut *UnionType) Inspect() string {
	return fmt.Sprintf("union %s", ut.Symbol.Decl.DeclName())
}

// Lookup implements RuntimeValue.
func (*UnionType) Lookup(name string) RuntimeValue {
	return nil
}

// TypeConstantId implements RuntimeValue.
func (ut *UnionType) TypeConstantId() TypeId {
	return TypeId(*ut.Symbol.TypeSymbol.ConstantId)
}

func (ut *UnionType) TypeAttributes() map[TypeId]int {
	return ut.Attributes
}
