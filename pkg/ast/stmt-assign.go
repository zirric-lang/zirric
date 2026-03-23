package ast

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Statement = &StmtAssign{}

// StmtAssign represents an assignment statement: `target = value` or `target op= value`.
// Target must be an ExprIdentifier, ExprMemberAccess, or ExprIndexAccess.
// Op is empty for plain assignment, or PLUS/MINUS/ASTERISK/SLASH/PERCENT for compound assignment.
type StmtAssign struct {
	Token  token.Token     // the '=' or augmented op token
	Target Expr            // lvalue: ExprIdentifier, ExprMemberAccess, or ExprIndexAccess
	Op     token.TokenType // "" for plain '='; PLUS/MINUS/ASTERISK/SLASH/PERCENT for compound
	Value  Expr
}

func MakeStmtAssign(tok token.Token, target Expr, op token.TokenType, value Expr) *StmtAssign {
	return &StmtAssign{
		Token:  tok,
		Target: target,
		Op:     op,
		Value:  value,
	}
}

func (s *StmtAssign) statementNode()            {}
func (s *StmtAssign) TokenLiteral() token.Token { return s.Token }

func (s *StmtAssign) EnumerateChildNodes(cb func(Node)) {
	cb(s.Target)
	s.Target.EnumerateChildNodes(cb)
	cb(s.Value)
	s.Value.EnumerateChildNodes(cb)
}
