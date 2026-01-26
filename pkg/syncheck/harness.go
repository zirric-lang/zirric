package syncheck

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type SyntaxMatcher func(lineOffset int, line string, assert Assertion) bool

type Harness struct {
	match SyntaxMatcher
}

func NewHarness(matcher SyntaxMatcher) Harness {
	return Harness{matcher}
}

func (h *Harness) Test(doc string) error {
	asserts := ParseAssertions(doc)
	lines := strings.Split(doc, "\n")
	offset := 0
	var failures []Assertion
	for _, a := range asserts {
		line := lines[a.Line-1] // this might be wrong
		matched := h.match(offset, line, a)
		if matched == a.Negated {
			failures = append(failures, a)
		}
		offset += len(line) + 1
	}

	if len(failures) == 0 {
		return nil
	}
	var out strings.Builder
	out.WriteString(fmt.Sprintf("failed assertions %d:\n", len(failures)))

	for i, a := range failures {
		out.WriteString("\n")
		out.WriteString(fmt.Sprintf("%d. error:\n", i+1))
		out.WriteString(lines[a.Line-1] + "\n")
		out.WriteString(lines[a.SourceLine-1] + "\n")
	}
	return errors.New(out.String())
}
