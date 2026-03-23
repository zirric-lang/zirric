package ast

import (
	"bytes"
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// SwitchCaseKind identifies the pattern kind in a switch case arm.
type SwitchCaseKind int

const (
	SwitchCaseIsType  SwitchCaseKind = iota // case is TypeExpr:
	SwitchCaseDefault                       // case _:
	SwitchCaseValue                         // case expr:
)

// ExprSwitchCase is a single case arm in a switch expression.
type ExprSwitchCase struct {
	Token   token.Token    // the `case` token
	Kind    SwitchCaseKind //
	TypeRef TypeExpr       // for SwitchCaseIsType: the type expression (named, composite, or attrs)
	Pattern Expr           // for SwitchCaseValue: the expression to compare
	Body    Expr           // the expression value for this case
}

var _ Expr = ExprSwitch{}

// ExprSwitch is a switch used in expression position.
// Each case body is a single expression. A default case (case _:) is required.
type ExprSwitch struct {
	Token token.Token // the `switch` token
	Value Expr        // the value being switched on
	Cases []ExprSwitchCase
}

func MakeExprSwitch(tok token.Token, value Expr) *ExprSwitch {
	return &ExprSwitch{
		Token: tok,
		Value: value,
		Cases: make([]ExprSwitchCase, 0),
	}
}

func (e *ExprSwitch) AddCase(c ExprSwitchCase) {
	e.Cases = append(e.Cases, c)
}

func (e ExprSwitch) TokenLiteral() token.Token {
	return e.Token
}

func (e ExprSwitch) EnumerateChildNodes(action func(Node)) {
	action(e.Value)
	e.Value.EnumerateChildNodes(action)
	for _, c := range e.Cases {
		if c.TypeRef != nil {
			action(c.TypeRef)
			c.TypeRef.EnumerateChildNodes(action)
		}
		if c.Pattern != nil {
			action(c.Pattern)
			c.Pattern.EnumerateChildNodes(action)
		}
		action(c.Body)
		c.Body.EnumerateChildNodes(action)
	}
}

func (e ExprSwitch) Expression() string {
	var out bytes.Buffer
	out.WriteString("(switch ")
	out.WriteString(e.Value.Expression())
	out.WriteString(" { ")
	for i, c := range e.Cases {
		if i > 0 {
			out.WriteString(" ")
		}
		switch c.Kind {
		case SwitchCaseIsType:
			fmt.Fprintf(&out, "case is %s: ", c.TypeRef.TypeExpression())
		case SwitchCaseDefault:
			out.WriteString("case _: ")
		case SwitchCaseValue:
			fmt.Fprintf(&out, "case %s: ", c.Pattern.Expression())
		}
		out.WriteString(c.Body.Expression())
	}
	out.WriteString(" })")
	return out.String()
}
