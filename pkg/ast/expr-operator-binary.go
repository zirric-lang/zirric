package ast

import (
	"bytes"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Expr = ExprOperatorBinary{}

type ExprOperatorBinary struct {
	Operator OperatorBinary
	Left     Expr
	Right    Expr
}

func MakeExprOperatorBinary(operator OperatorBinary, left, right Expr) *ExprOperatorBinary {
	return &ExprOperatorBinary{
		Operator: operator,
		Left:     left,
		Right:    right,
	}
}

// EnumerateChildNodes implements Expr.
func (n ExprOperatorBinary) EnumerateChildNodes(action func(child Node)) {
	action(n.Left)
	action(n.Operator)
	action(n.Right)
}

// TokenLiteral implements Expr.
func (n ExprOperatorBinary) TokenLiteral() token.Token {
	return n.Left.TokenLiteral()
}

// Expression implements Expr.
func (e ExprOperatorBinary) Expression() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(e.Left.Expression())
	out.WriteString(e.Operator.Literal)
	out.WriteString(e.Right.Expression())
	out.WriteString(")")

	return out.String()
}
