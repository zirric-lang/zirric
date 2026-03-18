package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// TypeExprArray represents an array type expression: [ElementType].
type TypeExprArray struct {
	Token   token.Token // the `[` token
	Element TypeExpr
}

var _ TypeExpr = TypeExprArray{}

func MakeTypeExprArray(tok token.Token, element TypeExpr) TypeExprArray {
	return TypeExprArray{Token: tok, Element: element}
}

func (e TypeExprArray) TokenLiteral() token.Token { return e.Token }

func (e TypeExprArray) EnumerateChildNodes(action func(Node)) {
	action(e.Element)
}

func (e TypeExprArray) TypeExpression() string {
	return fmt.Sprintf("[%s]", e.Element.TypeExpression())
}
