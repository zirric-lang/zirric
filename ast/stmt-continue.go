package ast

import "code.knabel.dev/zirric-lang/zirric/token"

var _ Statement = StmtContinue{}

type StmtContinue struct {
	Token token.Token
}

func MakeStmtContinue(t token.Token) StmtContinue {
	return StmtContinue{Token: t}
}

func (s StmtContinue) EnumerateChildNodes(action func(child Node)) {}

func (s StmtContinue) TokenLiteral() token.Token {
	return s.Token
}

func (s StmtContinue) statementNode() {}
