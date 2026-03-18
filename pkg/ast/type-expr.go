package ast

import "code.knabel.dev/zirric-lang/zirric/pkg/token"

// TypeExpr represents a type expression in the AST.
// Type expressions appear in type hint positions (field types,
// parameter types, return types, is/switch patterns) and are distinct
// from value expressions (Expr).
type TypeExpr interface {
	Node
	TypeExpression() string
}

// TypeExprRef is a named type reference, either simple (String) or
// qualified (prelude.String). It wraps a StaticReference.
type TypeExprRef struct {
	Reference StaticReference
}

var _ TypeExpr = TypeExprRef{}

func MakeTypeExprRef(ref StaticReference) TypeExprRef {
	return TypeExprRef{Reference: ref}
}

func (e TypeExprRef) TokenLiteral() token.Token {
	return e.Reference.TokenLiteral()
}

func (e TypeExprRef) EnumerateChildNodes(action func(Node)) {
	e.Reference.EnumerateChildNodes(action)
}

func (e TypeExprRef) TypeExpression() string {
	return e.Reference.String()
}
