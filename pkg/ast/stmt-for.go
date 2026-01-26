package ast

import "code.knabel.dev/zirric-lang/zirric/pkg/token"

var _ Statement = StmtFor{}

type StmtFor struct {
	Token           token.Token
	Condition       Expr
	CollectionIdent *Identifier
	CollectionExpr  Expr
	Body            Block
}

func MakeStmtFor(t token.Token, cond Expr, collectionIdent *Identifier, collectionExpr Expr, body Block) StmtFor {
	return StmtFor{
		Token:           t,
		Condition:       cond,
		CollectionIdent: collectionIdent,
		CollectionExpr:  collectionExpr,
		Body:            body,
	}
}

func (s StmtFor) EnumerateChildNodes(action func(child Node)) {
	if s.Condition != nil {
		action(s.Condition)
		s.Condition.EnumerateChildNodes(action)
	}
	if s.CollectionIdent != nil {
		action(s.CollectionIdent)
		s.CollectionIdent.EnumerateChildNodes(action)
	}
	if s.CollectionExpr != nil {
		action(s.CollectionExpr)
		s.CollectionExpr.EnumerateChildNodes(action)
	}
	for _, n := range s.Body {
		action(n)
		n.EnumerateChildNodes(action)
	}
}

func (s StmtFor) TokenLiteral() token.Token {
	return s.Token
}

func (s StmtFor) statementNode() {}
