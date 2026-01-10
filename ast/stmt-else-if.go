package ast

import "code.knabel.dev/zirric-lang/zirric/token"

var _ Node = StmtElseIf{}

type StmtElseIf struct {
	Token     token.Token
	Condition Expr
	Block     Block
}

func MakeStmtIfElse(t token.Token, cond Expr, body Block) StmtElseIf {
	return StmtElseIf{
		Token:     t,
		Condition: cond,
		Block:     body,
	}
}

// EnumerateChildNodes implements Statement.
func (s StmtElseIf) EnumerateChildNodes(action func(child Node)) {
	action(s.Condition)
	s.Condition.EnumerateChildNodes(action)

	for _, n := range s.Block {
		action(n)
		n.EnumerateChildNodes(action)
	}
}

// TokenLiteral implements Statement.
func (s StmtElseIf) TokenLiteral() token.Token {
	return s.Token
}
