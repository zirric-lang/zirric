package ast

import (
	"bytes"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Expr = ExprIf{}

type ExprIf struct {
	Token     token.Token
	Condition Expr
	ThenExpr  Expr
	ElseIf    []ExprElseIf
	ElseExpr  Expr
}

func MakeExprIf(t token.Token, cond Expr, body Expr) ExprIf {
	return ExprIf{
		Token:     t,
		Condition: cond,
		ThenExpr:  body,
	}
}

func (s *ExprIf) AddElseIf(elseif ExprElseIf) {
	s.ElseIf = append(s.ElseIf, elseif)
}
func (s *ExprIf) SetElse(elseBody Expr) {
	s.ElseExpr = elseBody
}

// EnumerateChildNodes implements Statement.
func (s ExprIf) EnumerateChildNodes(action func(child Node)) {
	action(s.Condition)
	s.Condition.EnumerateChildNodes(action)

	action(s.ThenExpr)
	s.ThenExpr.EnumerateChildNodes(action)

	for _, n := range s.ElseIf {
		action(n)
		n.EnumerateChildNodes(action)
	}

	action(s.ElseExpr)
	s.ElseExpr.EnumerateChildNodes(action)
}

// TokenLiteral implements Statement.
func (s ExprIf) TokenLiteral() token.Token {
	return s.Token
}

// Expression implements Expr.
func (e ExprIf) Expression() string {
	var out bytes.Buffer

	out.WriteString("(if ")
	out.WriteString(e.Condition.Expression())
	out.WriteString(" { ")
	out.WriteString(e.ThenExpr.Expression())

	for _, elsif := range e.ElseIf {
		out.WriteString(" } else if ")
		out.WriteString(elsif.Condition.Expression())
		out.WriteString(" { ")
		out.WriteString(elsif.Then.Expression())
	}

	out.WriteString(" } else { ")
	out.WriteString(e.ElseExpr.Expression())
	out.WriteString(" })")

	return out.String()
}
