package ast

import "code.knabel.dev/zirric-lang/zirric/token"

var _ Expr = ExprTypeSwitch{}

type ExprTypeSwitch struct {
	Token     token.Token
	Type      Expr
	CaseOrder []Identifier
	Cases     map[string]Expr
}

func MakeExprTypeSwitch(type_ Expr, token token.Token) *ExprTypeSwitch {
	return &ExprTypeSwitch{
		Token:     token,
		Type:      type_,
		CaseOrder: make([]Identifier, 0),
		Cases:     make(map[string]Expr),
	}
}

func (e *ExprTypeSwitch) AddCase(key Identifier, value Expr) {
	e.CaseOrder = append(e.CaseOrder, key)
	e.Cases[key.Value] = value
}

func (e ExprTypeSwitch) TokenLiteral() token.Token {
	return e.Token
}

func (e ExprTypeSwitch) EnumerateChildNodes(enumerate func(Node)) {
	enumerate(e.Type)
	for _, key := range e.CaseOrder {
		enumerate(key)
		enumerate(e.Cases[key.Value])
	}
}

// Expression implements Expr.
func (ExprTypeSwitch) Expression() string {
	panic("unimplemented") // TODO
}
