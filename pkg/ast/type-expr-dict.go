package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// TypeExprDict represents a dict type expression: [KeyType: ValueType].
type TypeExprDict struct {
	Token token.Token // the `[` token
	Key   TypeExpr
	Value TypeExpr
}

var _ TypeExpr = TypeExprDict{}

func MakeTypeExprDict(tok token.Token, key TypeExpr, value TypeExpr) TypeExprDict {
	return TypeExprDict{Token: tok, Key: key, Value: value}
}

func (e TypeExprDict) TokenLiteral() token.Token { return e.Token }

func (e TypeExprDict) EnumerateChildNodes(action func(Node)) {
	action(e.Key)
	e.Key.EnumerateChildNodes(action)
	action(e.Value)
	e.Value.EnumerateChildNodes(action)
}

func (e TypeExprDict) TypeExpression() string {
	return fmt.Sprintf("[%s: %s]", e.Key.TypeExpression(), e.Value.TypeExpression())
}
