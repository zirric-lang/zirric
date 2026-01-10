package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/token"
)

var _ Expr = ExprBool{}

type ExprBool struct {
	Token   token.Token
	Literal bool
}

func MakeExprBool(literal bool, token token.Token) *ExprBool {
	return &ExprBool{
		Literal: literal,
		Token:   token,
	}
}

// EnumerateChildNodes implements Expr.
func (ExprBool) EnumerateChildNodes(func(child Node)) {
	// No child nodes.
}

// TokenLiteral implements Expr.
func (n ExprBool) TokenLiteral() token.Token {
	return n.Token
}

// Expression implements Expr.
func (e ExprBool) Expression() string {
	return fmt.Sprintf("%t", e.Literal)
}
