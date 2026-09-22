package ast

import (
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

type Docs struct {
	// not only text, but also parsing @type and @param, ...
	// or more general?
	Content string
}

// MakeDocs turns the comment lines written above a declaration into its documentation, stripping the comment markers and the one space conventionally written after them.
// A shebang is skipped: it addresses the host that runs the file, not a reader of the declaration below it.
func MakeDocs(comments []string) *Docs {
	lines := make([]string, 0, len(comments))
	for _, comment := range comments {
		text, ok := docCommentText(comment)
		if !ok {
			continue
		}
		lines = append(lines, text)
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return &Docs{
		Content: strings.Join(lines, "\n"),
	}
}

func docCommentText(comment string) (string, bool) {
	trimmed := strings.TrimSpace(comment)
	switch {
	case strings.HasPrefix(trimmed, "#!"):
		return "", false
	case strings.HasPrefix(trimmed, "//"):
		return strings.TrimPrefix(strings.TrimPrefix(trimmed, "//"), " "), true
	case strings.HasPrefix(trimmed, "#"):
		return strings.TrimPrefix(strings.TrimPrefix(trimmed, "#"), " "), true
	default:
		return "", false
	}
}

// LeadingDocComments returns the block of comments written directly above tok, in the order they were written.
// A comment that follows code on the same line documents that line, and a comment separated from tok by a blank line documents whatever it stood next to, so neither belongs to the declaration and both end the block.
func LeadingDocComments(tok token.Token) []string {
	var reversed []string
	for i := len(tok.Leading) - 1; i >= 0; i-- {
		decoration := tok.Leading[i]
		switch decoration.Type {
		case token.DECO_COMMENT:
			if i > 0 && tok.Leading[i-1].Type == token.DECO_INLINE {
				return reverseStrings(reversed)
			}
			reversed = append(reversed, decoration.Literal)
		case token.DECO_MULTI:
			if strings.Count(decoration.Literal, "\n") > 1 {
				return reverseStrings(reversed)
			}
		}
	}
	return reverseStrings(reversed)
}

// DocsOf returns the documentation written above a declaration, or the empty string when it carries none.
func DocsOf(decl Decl) string {
	documented, ok := decl.(Documented)
	if !ok {
		return ""
	}
	docs := documented.ProvidedDocs()
	if docs == nil {
		return ""
	}
	return docs.Content
}

func reverseStrings(values []string) []string {
	for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
		values[i], values[j] = values[j], values[i]
	}
	return values
}
