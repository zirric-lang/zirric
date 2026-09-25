package ast

import (
	"bytes"
	"strconv"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Expr = ExprStringInterpolation{}

// StringPart is one piece of an interpolated string literal: either a run of literal text or an embedded expression.
type StringPart struct {
	// Literal is the decoded text of a literal run. Read only when Expr is nil.
	Literal string
	// Expr is the embedded expression of a `\( … )`, already parsed.
	Expr Expr
}

// ExprStringInterpolation is a string literal that embeds expressions, as in "n = \(n)".
// Every part renders the way fmt.sprint renders it and the pieces are joined, so the whole is a String.
type ExprStringInterpolation struct {
	Token token.Token
	Parts []StringPart
}

func MakeExprStringInterpolation(token token.Token, parts []StringPart) *ExprStringInterpolation {
	return &ExprStringInterpolation{
		Token: token,
		Parts: parts,
	}
}

func (e ExprStringInterpolation) TokenLiteral() token.Token {
	return e.Token
}

func (e ExprStringInterpolation) EnumerateChildNodes(enumerate func(Node)) {
	for _, part := range e.Parts {
		if part.Expr == nil {
			continue
		}
		enumerate(part.Expr)
		part.Expr.EnumerateChildNodes(enumerate)
	}
}

// Expression implements Expr.
func (e ExprStringInterpolation) Expression() string {
	var out bytes.Buffer

	out.WriteString("\"")
	for _, part := range e.Parts {
		if part.Expr != nil {
			out.WriteString("\\(")
			out.WriteString(part.Expr.Expression())
			out.WriteString(")")
			continue
		}
		// Quoted and then unwrapped, so the text is escaped exactly as a plain string literal escapes it.
		quoted := strconv.Quote(part.Literal)
		out.WriteString(strings.TrimSuffix(strings.TrimPrefix(quoted, "\""), "\""))
	}
	out.WriteString("\"")

	return out.String()
}
