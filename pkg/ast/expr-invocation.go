package ast

import (
	"bytes"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Expr = ExprInvocation{}

type ExprInvocation struct {
	Function  Expr
	Arguments []Expr
}

func MakeExprInvocation(function Expr) *ExprInvocation {
	return &ExprInvocation{
		Function: function,
	}
}

func (e *ExprInvocation) AddArgument(argument Expr) {
	e.Arguments = append(e.Arguments, argument)
}

// EnumerateChildNodes implements Expr.
func (n ExprInvocation) EnumerateChildNodes(action func(child Node)) {
	action(n.Function)
	for _, argument := range n.Arguments {
		action(argument)
	}
}

// TokenLiteral implements Expr.
func (n ExprInvocation) TokenLiteral() token.Token {
	return n.Function.TokenLiteral()
}

// Expression implements Expr.
func (e ExprInvocation) Expression() string {
	var out bytes.Buffer

	out.WriteString(e.Function.Expression())
	out.WriteString("(")
	for i, arg := range e.Arguments {
		out.WriteString(arg.Expression())

		if i+1 < len(e.Arguments) {
			out.WriteString(", ")
		}
	}
	out.WriteString(e.Function.Expression())
	out.WriteString(")")

	return out.String()
}
