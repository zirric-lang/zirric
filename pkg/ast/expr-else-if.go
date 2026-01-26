package ast

import "code.knabel.dev/zirric-lang/zirric/pkg/token"

var _ Node = ExprElseIf{}

type ExprElseIf struct {
	Token     token.Token
	Condition Expr
	Then      Expr
}

func MakeExprElseIf(t token.Token, cond Expr, then Expr) ExprElseIf {
	return ExprElseIf{
		Token:     t,
		Condition: cond,
		Then:      then,
	}
}

// EnumerateChildNodes implements Statement.
func (s ExprElseIf) EnumerateChildNodes(action func(child Node)) {
	action(s.Condition)
	s.Condition.EnumerateChildNodes(action)

	action(s.Then)
	s.Then.EnumerateChildNodes(action)
}

// TokenLiteral implements Statement.
func (s ExprElseIf) TokenLiteral() token.Token {
	return s.Token
}
