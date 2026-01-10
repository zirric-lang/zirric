package ast

import (
	"strconv"

	"code.knabel.dev/zirric-lang/zirric/token"
)

var _ Expr = ExprChar{}

type ExprChar struct {
	Token   token.Token
	Literal rune
}

func MakeExprChar(literal rune, token token.Token) *ExprChar {
	return &ExprChar{
		Literal: literal,
		Token:   token,
	}
}

func (e ExprChar) TokenLiteral() token.Token {
	return e.Token
}

func (ExprChar) EnumerateChildNodes(func(Node)) {
	// No child nodes.
}

func (e ExprChar) Expression() string {
	return strconv.QuoteRune(e.Literal)
}
