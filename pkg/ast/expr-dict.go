package ast

import (
	"bytes"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Expr = ExprDict{}

type ExprDict struct {
	Token   token.Token
	Entries []ExprDictEntry
}

// TokenLiteral implements Expr.
// The opening bracket is the dict's position; returning an empty token would leave anything reported about a dict literal with nowhere to point.
func (e ExprDict) TokenLiteral() token.Token {
	return e.Token
}

// EnumerateChildNodes implements Expr.
func (e ExprDict) EnumerateChildNodes(enumerate func(Node)) {
	for _, entry := range e.Entries {
		enumerate(entry.Key)
		entry.Key.EnumerateChildNodes(enumerate)
		enumerate(entry.Value)
		entry.Value.EnumerateChildNodes(enumerate)
	}
}

func MakeExprDict(entries []ExprDictEntry, token token.Token) *ExprDict {
	return &ExprDict{
		Token:   token,
		Entries: entries,
	}
}

type ExprDictEntry struct {
	Key   Expr
	Value Expr
}

func MakeExprDictEntry(key Expr, value Expr) ExprDictEntry {
	return ExprDictEntry{
		Key:   key,
		Value: value,
	}
}

// Expression implements Expr.
func (e ExprDict) Expression() string {
	var out bytes.Buffer

	out.WriteString("[")
	for i, v := range e.Entries {
		out.WriteString(v.Key.Expression())
		out.WriteString(": ")
		out.WriteString(v.Value.Expression())

		if i+1 < len(e.Entries) {
			out.WriteString(", ")
		}
	}
	out.WriteString("]")

	return out.String()
}
