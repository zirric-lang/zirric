package runtime

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ RuntimeValue = SimpleType{}

type SimpleType struct {
	Decl       *ast.Symbol
	Attributes map[TypeId]int
	// builtinTypeId overrides the TypeConstantId for prelude builtin types whose runtime values use hardcoded iota-based TypeIds.
	builtinTypeId *TypeId
}

// Inspect implements runtime.RuntimeValue.
func (i SimpleType) Inspect() string {
	return "extern " + i.Decl.Name
}

// Lookup implements runtime.RuntimeValue.
func (i SimpleType) Lookup(name string) RuntimeValue {
	return nil
}

// TypeConstantId implements runtime.RuntimeValue.
func (i SimpleType) TypeConstantId() TypeId {
	if i.builtinTypeId != nil {
		return *i.builtinTypeId
	}
	return TypeId(*i.Decl.ConstantId)
}

// MakeBuiltinSimpleType creates a SimpleType for a prelude builtin type, using the hardcoded TypeId that matches the runtime value types.
func MakeBuiltinSimpleType(decl *ast.Symbol, builtinTypeId TypeId) SimpleType {
	return SimpleType{Decl: decl, builtinTypeId: &builtinTypeId}
}

func (i SimpleType) TypeAttributes() map[TypeId]int {
	return i.Attributes
}
