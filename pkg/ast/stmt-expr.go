package ast

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Statement = &StmtExpr{}

type StmtExpr struct {
	Token token.Token
	Expr  Expr
}

func MakeStmtExpr(tok token.Token, expr Expr) *StmtExpr {
	return &StmtExpr{tok, expr}
}

// EnumerateChildNodes implements Statement.
func (s *StmtExpr) EnumerateChildNodes(action func(child Node)) {
	action(s.Expr)
	s.Expr.EnumerateChildNodes(action)
}

// TokenLiteral implements Statement.
func (s *StmtExpr) TokenLiteral() token.Token {
	return s.Token
}

// statementNode implements Statement.
func (*StmtExpr) statementNode() {}
