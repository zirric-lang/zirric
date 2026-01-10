package ast

import "code.knabel.dev/zirric-lang/zirric/token"

var _ Statement = StmtIf{}

type StmtIf struct {
	Token     token.Token
	Condition Expr
	IfBlock   Block
	ElseIf    []StmtElseIf
	ElseBlock Block
}

func MakeStmtIf(t token.Token, cond Expr, body Block) StmtIf {
	return StmtIf{
		Token:     t,
		Condition: cond,
		IfBlock:   body,
	}
}

func (s *StmtIf) AddElseIf(elseif StmtElseIf) {
	s.ElseIf = append(s.ElseIf, elseif)
}
func (s *StmtIf) SetElse(elseBody Block) {
	s.ElseBlock = elseBody
}

// EnumerateChildNodes implements Statement.
func (s StmtIf) EnumerateChildNodes(action func(child Node)) {
	action(s.Condition)
	s.Condition.EnumerateChildNodes(action)

	for _, n := range s.IfBlock {
		action(n)
		n.EnumerateChildNodes(action)
	}
	for _, n := range s.ElseIf {
		action(n)
		n.EnumerateChildNodes(action)
	}
	for _, n := range s.ElseBlock {
		action(n)
		n.EnumerateChildNodes(action)
	}
}

// TokenLiteral implements Statement.
func (s StmtIf) TokenLiteral() token.Token {
	return s.Token
}

// statementNode implements Statement.
func (s StmtIf) statementNode() {}
