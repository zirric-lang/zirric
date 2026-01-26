package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Expr = ExprInt{}

type ExprInt struct {
	Token   token.Token
	Literal int64
}

func MakeExprInt(literal int64, token token.Token) *ExprInt {
	return &ExprInt{
		Literal: literal,
		Token:   token,
	}
}

// EnumerateChildNodes implements Expr.
func (ExprInt) EnumerateChildNodes(func(child Node)) {
	// No child nodes.
}

// TokenLiteral implements Expr.
func (n ExprInt) TokenLiteral() token.Token {
	return n.Token
}

// Expression implements Expr.
func (e ExprInt) Expression() string {
	return fmt.Sprintf("%d", e.Literal)
}
