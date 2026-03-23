package ast

import (
	"bytes"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Expr = ExprIs{}

// ExprIs represents a type-check expression: `value is TypeExpr`.
// TypeRef holds the type expression which can be a named type, composite type,
// or attribute constraint (TypeExprAttrs for @A @B @C).
type ExprIs struct {
	Token   token.Token // the `is` token
	Value   Expr
	TypeRef TypeExpr // the type expression to check against
}

func MakeExprIs(tok token.Token, value Expr, typeRef TypeExpr) ExprIs {
	return ExprIs{
		Token:   tok,
		Value:   value,
		TypeRef: typeRef,
	}
}

func (e ExprIs) TokenLiteral() token.Token {
	return e.Token
}

func (e ExprIs) EnumerateChildNodes(action func(Node)) {
	action(e.Value)
	e.Value.EnumerateChildNodes(action)
	if e.TypeRef != nil {
		action(e.TypeRef)
		e.TypeRef.EnumerateChildNodes(action)
	}
}

func (e ExprIs) Expression() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(e.Value.Expression())
	out.WriteString(" is ")
	out.WriteString(e.TypeRef.TypeExpression())
	out.WriteString(")")
	return out.String()
}
