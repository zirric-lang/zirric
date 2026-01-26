package ast

import (
	"bytes"
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Expr = ExprFor{}

type ExprForBody struct {
	Decls      []*DeclVariable
	Stmts      Block
	DeclsTable *DeclTable
	Symbols    *SymbolTable
}

type ExprFor struct {
	Token           token.Token
	Condition       Expr
	CollectionIdent *Identifier
	CollectionExpr  Expr
	Body            ExprForBody
}

func MakeExprFor(t token.Token, cond Expr, collectionIdent *Identifier, collectionExpr Expr, body ExprForBody) ExprFor {
	return ExprFor{
		Token:           t,
		Condition:       cond,
		CollectionIdent: collectionIdent,
		CollectionExpr:  collectionExpr,
		Body:            body,
	}
}

func (s ExprFor) EnumerateChildNodes(action func(child Node)) {
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
	for _, decl := range s.Body.Decls {
		action(decl)
		decl.EnumerateChildNodes(action)
	}
	for _, stmt := range s.Body.Stmts {
		action(stmt)
		stmt.EnumerateChildNodes(action)
	}
}

func (s ExprFor) TokenLiteral() token.Token {
	return s.Token
}

func (e ExprFor) Expression() string {
	var out bytes.Buffer

	out.WriteString("for ")
	if e.CollectionIdent != nil && e.CollectionExpr != nil {
		out.WriteString(e.CollectionIdent.String())
		out.WriteString(" <- ")
		out.WriteString(e.CollectionExpr.Expression())
	} else if e.Condition != nil {
		out.WriteString(e.Condition.Expression())
	}
	out.WriteString(" { ")
	out.WriteString("/* ")
	out.WriteString(fmt.Sprintf("%d decls, %d stmts", len(e.Body.Decls), len(e.Body.Stmts)))
	out.WriteString(" */ ")
	out.WriteString("}")
	return out.String()
}
