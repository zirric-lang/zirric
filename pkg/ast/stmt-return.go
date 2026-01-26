package ast

import "code.knabel.dev/zirric-lang/zirric/pkg/token"

var _ Statement = &StmtReturn{}

type StmtReturn struct {
	Token token.Token
	Expr  Expr // an optional expr, if omitted `None` is assumed
}

func MakeStmtReturn(t token.Token, expr Expr) *StmtReturn {
	return &StmtReturn{
		Token: t,
		Expr:  expr,
	}
}

// EnumerateChildNodes implements Statement.
func (s *StmtReturn) EnumerateChildNodes(action func(child Node)) {
	if s.Expr != nil {
		action(s.Expr)
	}
}

// TokenLiteral implements Statement.
func (s *StmtReturn) TokenLiteral() token.Token {
	return s.Token
}

// statementNode implements Statement.
func (s *StmtReturn) statementNode() {}
