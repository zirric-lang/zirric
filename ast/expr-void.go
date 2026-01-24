package ast

import (
	"code.knabel.dev/zirric-lang/zirric/token"
)

var _ Expr = ExprVoid{}

type ExprVoid struct {
	Token token.Token
}

func MakeExprVoid(token token.Token) *ExprVoid {
	return &ExprVoid{
		Token: token,
	}
}

// EnumerateChildNodes implements Expr.
func (ExprVoid) EnumerateChildNodes(func(child Node)) {
	// No child nodes.
}

// TokenLiteral implements Expr.
func (n ExprVoid) TokenLiteral() token.Token {
	return n.Token
}

// Expression implements Expr.
func (e ExprVoid) Expression() string {
	return "void"
}
