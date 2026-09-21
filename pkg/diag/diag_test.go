package diag_test

import (
	"fmt"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/diag"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// positionedError stands in for a parse, analysis or compile error, which is all this package knows about them.
type positionedError struct {
	message string
	source  *token.Source
}

func (e positionedError) Error() string           { return e.message }
func (e positionedError) Position() *token.Source { return e.source }

func TestExcerptPointsAtTheColumn(t *testing.T) {
	content := []byte("mod a\n\nfn greet() {\n\tconst x = 1\n}\n")
	got := diag.Excerpt(content, token.MakeSource("main.zirr", 0, 3, 4))
	want := "3 | fn greet() {\n  |    ^"
	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestExcerptKeepsTabsSoTheCaretAligns(t *testing.T) {
	// Expanding a tab to a guessed number of spaces would put the caret somewhere else in a terminal that draws tabs differently.
	content := []byte("fn f() {\n\treturn 1\n}")
	got := diag.Excerpt(content, token.MakeSource("main.zirr", 0, 2, 2))
	want := "2 | \treturn 1\n  | \t^"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestExcerptDeclinesAPositionOutsideTheContent(t *testing.T) {
	// A file that changed since the error was made must not produce a wrong excerpt.
	if got := diag.Excerpt([]byte("one line\n"), token.MakeSource("main.zirr", 0, 9, 1)); got != "" {
		t.Errorf("expected no excerpt, got %q", got)
	}
	if got := diag.Excerpt([]byte("one line\n"), nil); got != "" {
		t.Errorf("expected no excerpt for a missing source, got %q", got)
	}
}

func TestRenderAddsTheLineTheErrorRefersTo(t *testing.T) {
	err := positionedError{
		message: "main.zirr:2:2: undefined type: Thnig",
		source:  token.MakeSource("main.zirr", 0, 2, 2),
	}
	read := func(string) ([]byte, error) { return []byte("fn f() {\n\tx is Thnig\n}"), nil }
	got := diag.Render(err, read)
	if !strings.HasPrefix(got, "main.zirr:2:2: undefined type: Thnig\n") {
		t.Errorf("the message is not kept first: %q", got)
	}
	if !strings.Contains(got, "x is Thnig") {
		t.Errorf("the excerpt is missing: %q", got)
	}
}

func TestRenderFallsBackToTheMessageAlone(t *testing.T) {
	positioned := positionedError{
		message: "main.zirr:2:2: undefined type: Thnig",
		source:  token.MakeSource("main.zirr", 0, 2, 2),
	}
	unreadable := func(string) ([]byte, error) { return nil, fmt.Errorf("no such file") }
	if got := diag.Render(positioned, unreadable); got != positioned.message {
		t.Errorf("expected the message alone when the file cannot be read, got %q", got)
	}
	if got := diag.Render(positioned, nil); got != positioned.message {
		t.Errorf("expected the message alone with no reader, got %q", got)
	}

	plain := fmt.Errorf("something went wrong")
	if got := diag.Render(plain, unreadable); got != plain.Error() {
		t.Errorf("expected the message alone for an error with no position, got %q", got)
	}
	if got := diag.Render(nil, unreadable); got != "" {
		t.Errorf("expected an empty rendering for no error, got %q", got)
	}
}

func TestPositionReachesThroughAWrappedError(t *testing.T) {
	source := token.MakeSource("main.zirr", 0, 4, 9)
	wrapped := fmt.Errorf("while compiling: %w", positionedError{message: "boom", source: source})
	got := diag.Position(wrapped)
	if got == nil {
		t.Fatal("expected a position through the wrapper")
	}
	if got.Line != 4 || got.Column != 9 {
		t.Errorf("expected 4:9, got %d:%d", got.Line, got.Column)
	}
	if diag.Position(fmt.Errorf("no position here")) != nil {
		t.Error("expected no position for a plain error")
	}
}

// multiError stands in for a ParseErrors or CompileErrors, which hold several errors the way Go's convention expects.
type multiError []error

func (e multiError) Error() string {
	parts := make([]string, len(e))
	for i := range e {
		parts[i] = e[i].Error()
	}
	return strings.Join(parts, "\n")
}
func (e multiError) Unwrap() []error { return e }

func TestRenderShowsEveryHeldError(t *testing.T) {
	read := func(string) ([]byte, error) { return []byte("one\ntwo\nthree\n"), nil }
	held := multiError{
		positionedError{message: "main.zirr:1:1: first", source: token.MakeSource("main.zirr", 0, 1, 1)},
		positionedError{message: "main.zirr:3:2: second", source: token.MakeSource("main.zirr", 0, 3, 2)},
	}

	got := diag.Render(held, read)
	want := "main.zirr:1:1: first\n1 | one\n  | ^\n\nmain.zirr:3:2: second\n3 | three\n  |  ^"
	if got != want {
		t.Errorf("expected:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestRenderShowsEachDistinctProblemOnce(t *testing.T) {
	// Recovering from one mistake can make a parser report the same thing repeatedly; repeating the excerpt adds nothing.
	read := func(string) ([]byte, error) { return []byte("one\ntwo\n"), nil }
	repeated := positionedError{message: "main.zirr:1:1: same", source: token.MakeSource("main.zirr", 0, 1, 1)}
	held := multiError{
		repeated,
		repeated,
		positionedError{message: "main.zirr:2:1: different", source: token.MakeSource("main.zirr", 0, 2, 1)},
		repeated,
	}

	got := diag.Render(held, read)
	if strings.Count(got, "same") != 1 {
		t.Errorf("expected the repeated problem once, got:\n%s", got)
	}
	if strings.Count(got, "different") != 1 {
		t.Errorf("expected the distinct problem once, got:\n%s", got)
	}
	// Order is the order the problems were found in, not the order they were deduplicated.
	if strings.Index(got, "same") > strings.Index(got, "different") {
		t.Errorf("expected the first problem first, got:\n%s", got)
	}
}

func TestRenderKeepsTheWrapperOfAWrappedCollection(t *testing.T) {
	// Taking a wrapped collection apart would drop the context the wrapper added, so it renders as one message.
	held := multiError{
		positionedError{message: "first", source: nil},
		positionedError{message: "second", source: nil},
	}
	wrapped := fmt.Errorf("while compiling: %w", held)
	got := diag.Render(wrapped, nil)
	if !strings.HasPrefix(got, "while compiling: ") {
		t.Errorf("expected the wrapper to be kept, got %q", got)
	}
}

func TestRenderOfASingleHeldErrorIsUnchanged(t *testing.T) {
	read := func(string) ([]byte, error) { return []byte("one\n"), nil }
	held := multiError{positionedError{message: "main.zirr:1:1: only", source: token.MakeSource("main.zirr", 0, 1, 1)}}
	if got := diag.Render(held, read); !strings.Contains(got, "1 | one") {
		t.Errorf("expected a single held error to render with its excerpt, got:\n%s", got)
	}
}
