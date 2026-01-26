package ast

import (
	"bytes"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Expr = ExprIndexAccess{}

type ExprIndexAccess struct {
	Token     token.Token
	Target    Expr
	IndexExpr Expr
}

func MakeExprIndexAccess(tok token.Token, target Expr, indexExpr Expr) *ExprIndexAccess {
	return &ExprIndexAccess{
		Token:     tok,
		Target:    target,
		IndexExpr: indexExpr,
	}
}

// EnumerateChildNodes implements Expr.
func (n ExprIndexAccess) EnumerateChildNodes(action func(child Node)) {
	action(n.Target)
	n.Target.EnumerateChildNodes(action)

	action(n.IndexExpr)
	n.IndexExpr.EnumerateChildNodes(action)
}

// TokenLiteral implements Expr.
func (n ExprIndexAccess) TokenLiteral() token.Token {
	return n.Token
}

// Expression implements Expr.
func (e ExprIndexAccess) Expression() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(e.Target.Expression())
	out.WriteString("[")
	out.WriteString(e.IndexExpr.Expression())
	out.WriteString("]")
	out.WriteString(")")

	return out.String()
}
