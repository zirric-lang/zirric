package ast

import (
	"strconv"

	"code.knabel.dev/zirric-lang/zirric/token"
)

var _ Expr = ExprString{}

type ExprString struct {
	Token   token.Token
	Literal string
}

func MakeExprString(token token.Token, literal string) *ExprString {
	return &ExprString{
		Literal: literal,
		Token:   token,
	}
}

func (e ExprString) TokenLiteral() token.Token {
	return e.Token
}

func (e ExprString) EnumerateChildNodes(enumerate func(Node)) {
	// No child nodes.
}

// Expression implements Expr.
func (e ExprString) Expression() string {
	return strconv.Quote(e.Literal)
}
