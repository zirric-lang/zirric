package ast

import (
	"bytes"
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// TypeExprAttrs represents a multi-attribute type expression: @A @B @C.
// Each attribute is a TypeExprRef (e.g., @prelude.Iterable).
type TypeExprAttrs struct {
	Token token.Token // the first `@` token
	Attrs []TypeExprRef
}

var _ TypeExpr = TypeExprAttrs{}

func MakeTypeExprAttrs(tok token.Token, attrs []TypeExprRef) TypeExprAttrs {
	return TypeExprAttrs{Token: tok, Attrs: attrs}
}

func (e TypeExprAttrs) TokenLiteral() token.Token { return e.Token }

func (e TypeExprAttrs) EnumerateChildNodes(action func(Node)) {
	for _, a := range e.Attrs {
		action(a)
	}
}

func (e TypeExprAttrs) TypeExpression() string {
	var out bytes.Buffer
	for i, a := range e.Attrs {
		if i > 0 {
			out.WriteString(" ")
		}
		fmt.Fprintf(&out, "@%s", a.TypeExpression())
	}
	return out.String()
}
