package codefmt

import (
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// Text is the verbatim source slice; tok.Literal is lossy, dropping string quotes and rewriting "=>" as "->".
type item struct {
	tok     token.Token
	text    string
	leading []token.DecorativeToken
}

// scan lexes src and fails unless the result accounts for every byte, since what it cannot see it must not rewrite.
func scan(filename, src string) ([]item, error) {
	lex, err := lexer.New(staticmodule.NewSourceString(registry.LogicalURI(filename), src))
	if err != nil {
		return nil, err
	}

	var toks []token.Token
	for {
		tok := lex.NextToken()
		if tok.Type == token.ILLEGAL {
			return nil, &UnformattableError{
				Filename: filename,
				Reason:   "illegal token",
				Offset:   offsetOf(tok),
			}
		}
		toks = append(toks, tok)
		if tok.Type == token.EOF {
			break
		}
	}

	items := make([]item, 0, len(toks))
	for i, tok := range toks {
		it := item{tok: tok, leading: tok.Leading}
		if tok.Type != token.EOF {
			start := offsetOf(tok)
			end := len(src)
			if i+1 < len(toks) {
				end = offsetOf(toks[i+1]) - leadingLen(toks[i+1])
			}
			if start < 0 || end > len(src) || start > end {
				return nil, &UnformattableError{
					Filename: filename,
					Reason:   "token span out of range",
					Offset:   start,
				}
			}
			it.text = src[start:end]
		}
		items = append(items, it)
	}

	if got := reconstruct(items); got != src {
		return nil, &UnformattableError{
			Filename: filename,
			Reason:   "source does not reconstruct from tokens",
			Offset:   firstDiff(src, got),
		}
	}
	return items, nil
}

// canonicalText is the spelling to emit: verbatim source, except the deprecated "=>" which becomes "->" and a string literal whose interpolations are respelled.
func canonicalText(it item) string {
	switch it.tok.Type {
	case token.RIGHT_ARROW:
		return "->"
	case token.STRING:
		return canonicalString(it.text)
	}
	return it.text
}

// canonicalString respells the `\( … )` of a string literal, leaving every other byte of it exactly as written.
// A literal the scan cannot account for whole — unterminated, or holding an interpolation with no `)` — is left alone: it is being typed, not formatted.
func canonicalString(text string) string {
	segments, ok := lexer.SplitLiteral(text)
	if !ok {
		return text
	}
	interpolated := false
	for _, segment := range segments {
		interpolated = interpolated || segment.Interpolation
	}
	if !interpolated {
		return text
	}

	var out strings.Builder
	out.WriteByte('"')
	for _, segment := range segments {
		if !segment.Interpolation {
			out.WriteString(segment.Text)
			continue
		}
		out.WriteString("\\(")
		out.WriteString(canonicalExpr(segment.Expr))
		out.WriteString(")")
	}
	out.WriteByte('"')
	return out.String()
}

// canonicalExpr spells an interpolated expression with the same spacing rules as any other, on the one line it is allowed to occupy.
func canonicalExpr(src string) string {
	items, err := scan("", src)
	if err != nil {
		return src
	}

	var out strings.Builder
	var prev token.Token
	hasPrev, prevWasPrefix := false, false
	for _, it := range items {
		if it.tok.Type == token.EOF {
			continue
		}
		text := canonicalText(it)
		glued := len(it.leading) == 0
		if hasPrev && (needSpace(prev, prevWasPrefix, it.tok, glued) || wouldGlue(out.String(), text)) {
			out.WriteString(" ")
		}
		out.WriteString(text)
		prevWasPrefix = isPrefixOperator(it.tok, prev, hasPrev, glued)
		prev = it.tok
		hasPrev = true
	}
	return out.String()
}

func reconstruct(items []item) string {
	var sb strings.Builder
	for _, it := range items {
		for _, dec := range it.leading {
			sb.WriteString(dec.Literal)
		}
		sb.WriteString(it.text)
	}
	return sb.String()
}

func offsetOf(tok token.Token) int {
	if tok.Source == nil {
		return -1
	}
	return tok.Source.Offset
}

func leadingLen(tok token.Token) int {
	n := 0
	for _, dec := range tok.Leading {
		n += len(dec.Literal)
	}
	return n
}

func firstDiff(a, b string) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return min(len(a), len(b))
}
