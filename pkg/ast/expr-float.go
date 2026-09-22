package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Expr = ExprFloat{}

type ExprFloat struct {
	Token   token.Token
	Literal float64
}

func MakeExprFloat(literal float64, token token.Token) *ExprFloat {
	return &ExprFloat{
		Literal: literal,
		Token:   token,
	}
}

// TokenLiteral implements Expr.
func (e ExprFloat) TokenLiteral() token.Token {
	return e.Token
}

func (e ExprFloat) EnumerateChildNodes(enumerate func(Node)) {
	// No child nodes.
}

// Expression implements Expr.
func (e ExprFloat) Expression() string {
	return fmt.Sprintf("%f", e.Literal)
}
