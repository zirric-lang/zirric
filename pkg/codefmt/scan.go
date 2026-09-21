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

// canonicalText is the spelling to emit: verbatim source, except the deprecated "=>" which becomes "->".
func canonicalText(it item) string {
	if it.tok.Type == token.RIGHT_ARROW {
		return "->"
	}
	return it.text
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
