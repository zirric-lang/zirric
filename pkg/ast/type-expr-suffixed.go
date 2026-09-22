package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// TypeExprOption represents the optional type shorthand: Element?.
// Zirric has no generic types, so what the hint asks a value to be is an Option; Element says what a present one holds, and is compared between two written hints the way an array's element is.
type TypeExprOption struct {
	Token   token.Token // the `?` token
	Element TypeExpr
}

var _ TypeExpr = TypeExprOption{}

func MakeTypeExprOption(tok token.Token, element TypeExpr) TypeExprOption {
	return TypeExprOption{Token: tok, Element: element}
}

func (e TypeExprOption) TokenLiteral() token.Token { return e.Token }

func (e TypeExprOption) EnumerateChildNodes(action func(Node)) {
	action(e.Element)
	e.Element.EnumerateChildNodes(action)
}

func (e TypeExprOption) TypeExpression() string {
	return fmt.Sprintf("%s?", e.Element.TypeExpression())
}

// TypeExprResult represents the result type shorthand: Element!.
// Element says what a success holds, on the same terms as TypeExprOption's — what the hint asks a value to be is a Result.
type TypeExprResult struct {
	Token   token.Token // the `!` token
	Element TypeExpr
}

var _ TypeExpr = TypeExprResult{}

func MakeTypeExprResult(tok token.Token, element TypeExpr) TypeExprResult {
	return TypeExprResult{Token: tok, Element: element}
}

func (e TypeExprResult) TokenLiteral() token.Token { return e.Token }

func (e TypeExprResult) EnumerateChildNodes(action func(Node)) {
	action(e.Element)
	e.Element.EnumerateChildNodes(action)
}

func (e TypeExprResult) TypeExpression() string {
	return fmt.Sprintf("%s!", e.Element.TypeExpression())
}
