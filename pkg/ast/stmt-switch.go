package ast

import (
	"bytes"
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// StmtSwitchCase is a single case arm in a switch statement.
type StmtSwitchCase struct {
	Token   token.Token    // the `case` token
	Kind    SwitchCaseKind //
	TypeRef TypeExpr       // for SwitchCaseIsType: the type expression (named, composite, or attrs)
	Pattern Expr           // for SwitchCaseValue: the expression to compare
	Body    Block          // statement block for this case
}

var _ Statement = StmtSwitch{}

// StmtSwitch is a switch used at statement level.
// Each case body is a block of statements. A default case is optional.
type StmtSwitch struct {
	Token token.Token // the `switch` token
	Value Expr        // the value being switched on
	Cases []StmtSwitchCase
}

func MakeStmtSwitch(tok token.Token, value Expr) *StmtSwitch {
	return &StmtSwitch{
		Token: tok,
		Value: value,
		Cases: make([]StmtSwitchCase, 0),
	}
}

func (s *StmtSwitch) AddCase(c StmtSwitchCase) {
	s.Cases = append(s.Cases, c)
}

func (s StmtSwitch) TokenLiteral() token.Token {
	return s.Token
}

func (StmtSwitch) statementNode() {}

func (s StmtSwitch) EnumerateChildNodes(action func(Node)) {
	action(s.Value)
	s.Value.EnumerateChildNodes(action)
	for _, c := range s.Cases {
		if c.TypeRef != nil {
			action(c.TypeRef)
			c.TypeRef.EnumerateChildNodes(action)
		}
		if c.Pattern != nil {
			action(c.Pattern)
			c.Pattern.EnumerateChildNodes(action)
		}
		for _, stmt := range c.Body {
			action(stmt)
			stmt.EnumerateChildNodes(action)
		}
	}
}

func (s StmtSwitch) String() string {
	var out bytes.Buffer
	out.WriteString("switch ")
	out.WriteString(s.Value.Expression())
	out.WriteString(" { ")
	for _, c := range s.Cases {
		switch c.Kind {
		case SwitchCaseIsType:
			fmt.Fprintf(&out, "case is %s: ", c.TypeRef.TypeExpression())
		case SwitchCaseDefault:
			out.WriteString("case _: ")
		case SwitchCaseValue:
			fmt.Fprintf(&out, "case %s: ", c.Pattern.Expression())
		}
		fmt.Fprintf(&out, "/* %d stmts */ ", len(c.Body))
	}
	out.WriteString("}")
	return out.String()
}
