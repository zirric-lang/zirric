package ast

import (
	"bytes"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Expr = ExprMemberAccess{}

type ExprMemberAccess struct {
	Token    token.Token
	Target   Expr
	Property Identifier
}

func MakeExprMemberAccess(tok token.Token, target Expr, prop Identifier) *ExprMemberAccess {
	return &ExprMemberAccess{
		Token:    tok,
		Target:   target,
		Property: prop,
	}
}

// EnumerateChildNodes implements Expr.
func (n ExprMemberAccess) EnumerateChildNodes(action func(child Node)) {
	action(n.Target)
	n.Target.EnumerateChildNodes(action)

	action(n.Property)
}

// TokenLiteral implements Expr.
func (n ExprMemberAccess) TokenLiteral() token.Token {
	return n.Token
}

// Expression implements Expr.
func (e ExprMemberAccess) Expression() string {
	var out bytes.Buffer

	out.WriteString(e.Target.Expression())
	out.WriteString(".")
	out.WriteString(e.Property.Value)

	return out.String()
}
