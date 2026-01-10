package ast

import (
	"code.knabel.dev/zirric-lang/zirric/token"
)

var _ Expr = ExprIdentifier{}

type ExprIdentifier struct {
	Name Identifier
}

func MakeExprIdentifier(name Identifier) *ExprIdentifier {
	return &ExprIdentifier{
		Name: name,
	}
}

// EnumerateChildNodes implements Expr.
func (ExprIdentifier) EnumerateChildNodes(func(child Node)) {
	// No child nodes.
}

// TokenLiteral implements Expr.
func (n ExprIdentifier) TokenLiteral() token.Token {
	return n.Name.TokenLiteral()
}

// Expression implements Expr.
func (e ExprIdentifier) Expression() string {
	return e.Name.Value
}
