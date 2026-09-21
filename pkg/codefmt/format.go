// Package codefmt implements Zirric's canonical source layout.
//
// Formatting never changes whether two adjacent tokens share a line: it normalizes indentation, spacing, blank line runs, trailing whitespace and the final newline, and nothing else.
package codefmt

import (
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// UnformattableError reports source that cannot be formatted without risking a change in meaning.
type UnformattableError struct {
	Filename string
	Reason   string
	Offset   int
}

func (e *UnformattableError) Error() string {
	return fmt.Sprintf("%s: cannot format at offset %d: %s", e.Filename, e.Offset, e.Reason)
}

// Source formats src, returning it unchanged with an *UnformattableError when it cannot be formatted losslessly.
func Source(filename string, src []byte, opts Options) ([]byte, error) {
	out, err := String(filename, string(src), opts)
	if err != nil {
		return src, err
	}
	return []byte(out), nil
}

// String is the string-valued form of Source.
func String(filename, src string, opts Options) (string, error) {
	items, err := scan(filename, src)
	if err != nil {
		return src, err
	}
	out := render(items, opts)
	if err := Equivalent(filename, src, out); err != nil {
		return src, err
	}
	return out, nil
}

// Equivalent reports whether a and b hold the same tokens and comments, the post-condition every format must satisfy.
func Equivalent(filename, a, b string) error {
	aItems, err := scan(filename, a)
	if err != nil {
		return err
	}
	bItems, err := scan(filename, b)
	if err != nil {
		return err
	}

	aSig, aComments := significant(aItems)
	bSig, bComments := significant(bItems)

	if len(aSig) != len(bSig) {
		return &UnformattableError{
			Filename: filename,
			Reason:   fmt.Sprintf("token count changed: %d became %d", len(aSig), len(bSig)),
		}
	}
	for i := range aSig {
		if aSig[i] != bSig[i] {
			return &UnformattableError{
				Filename: filename,
				Reason:   fmt.Sprintf("token %d changed: %q became %q", i, aSig[i], bSig[i]),
			}
		}
	}
	if len(aComments) != len(bComments) {
		return &UnformattableError{
			Filename: filename,
			Reason:   fmt.Sprintf("comment count changed: %d became %d", len(aComments), len(bComments)),
		}
	}
	for i := range aComments {
		if aComments[i] != bComments[i] {
			return &UnformattableError{
				Filename: filename,
				Reason:   fmt.Sprintf("comment %d changed: %q became %q", i, aComments[i], bComments[i]),
			}
		}
	}
	return nil
}

func significant(items []item) (tokens, comments []string) {
	for _, it := range items {
		for _, dec := range it.leading {
			if dec.Type == token.DECO_COMMENT {
				// Trailing whitespace inside a comment is noise the formatter strips; it cannot carry meaning.
				comments = append(comments, strings.TrimRight(dec.Literal, " \t"))
			}
		}
		if it.text != "" {
			tokens = append(tokens, canonicalText(it))
		}
	}
	return tokens, comments
}
