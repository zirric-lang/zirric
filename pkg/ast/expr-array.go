package ast

import (
	"bytes"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Expr = ExprArray{}

type ExprArray struct {
	Token    token.Token
	Elements []Expr
}

// TokenLiteral implements Expr.
func (e ExprArray) TokenLiteral() token.Token {
	return e.Token
}

func MakeExprArray(elements []Expr, token token.Token) *ExprArray {
	return &ExprArray{
		Elements: elements,
		Token:    token,
	}
}

func (e ExprArray) EnumerateChildNodes(enumerate func(Node)) {
	for _, el := range e.Elements {
		enumerate(el)
		el.EnumerateChildNodes(enumerate)
	}
}

// Expression implements Expr.
func (e ExprArray) Expression() string {
	var out bytes.Buffer

	out.WriteString("[")
	for i, el := range e.Elements {
		out.WriteString(el.Expression())
		if i+1 < len(e.Elements) {
			out.WriteString(", ")
		}
	}
	out.WriteString("]")

	return out.String()
}
