package ast

import "code.knabel.dev/zirric-lang/zirric/pkg/token"

var _ Statement = StmtBreak{}

type StmtBreak struct {
	Token token.Token
}

func MakeStmtBreak(t token.Token) StmtBreak {
	return StmtBreak{Token: t}
}

func (s StmtBreak) EnumerateChildNodes(action func(child Node)) {}

func (s StmtBreak) TokenLiteral() token.Token {
	return s.Token
}

func (s StmtBreak) statementNode() {}
