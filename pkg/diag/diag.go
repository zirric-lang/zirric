// Package diag renders errors for a person to read.
//
// An error already says where it is, as file:line:column. What it cannot do on its own is show the line it refers to, and that is what makes a mistake findable at a glance rather than by counting lines in an editor.
//
// Only a terminal needs this. A language server places a diagnostic at a range of its own and renders nothing, so it maps positions directly instead of going through here.
package diag

import (
	"errors"
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// Positioned is implemented by errors that know where they came from.
// Keeping this an interface is what lets this package render a parse, analysis or compile error without depending on any of them.
type Positioned interface {
	error
	Position() *token.Source
}

// Traced is implemented by errors that know which frames were live when they happened.
// Keeping this an interface is what lets this package show a stack trace without depending on the VM.
type Traced interface {
	error
	StackTrace() string
}

// SourceReader reads the file an error's position names.
// It returns an error when the file cannot be read, which is not itself a failure: the message is still rendered, just without the excerpt.
type SourceReader func(file string) ([]byte, error)

// Position returns where err happened, or nil when neither it nor anything it wraps carries a position.
func Position(err error) *token.Source {
	var positioned Positioned
	if errors.As(err, &positioned) {
		return positioned.Position()
	}
	return nil
}

// Render returns err's message followed by the line it refers to and a caret under the column it names.
//
// An error holding several errors — as parsing a file with more than one mistake does — renders each of them in turn, separated by a blank line so that one excerpt does not run into the next.
// An error with no position, or one whose file cannot be read, renders as its message alone, so a caller needs no fallback of its own.
func Render(err error, read SourceReader) string {
	if err == nil {
		return ""
	}
	held := heldErrors(err)
	if len(held) > 0 {
		parts := make([]string, 0, len(held))
		// Recovering from one mistake can make a parser report the same thing several times over — four unclosed parentheses give four identical messages at the same place. Repeating an excerpt for each says nothing the first did not, so each distinct problem is shown once, in the order it was found.
		seen := make(map[string]struct{}, len(held))
		for _, one := range held {
			rendered := renderOne(one, read)
			if _, repeated := seen[rendered]; repeated {
				continue
			}
			seen[rendered] = struct{}{}
			parts = append(parts, rendered)
		}
		return strings.Join(parts, "\n\n")
	}
	return renderOne(err, read)
}

// heldErrors returns the errors err holds, flattening a collection of collections, or nothing when err holds none.
// Only err itself is examined: a collection wrapped in something else keeps that wrapper's message rather than being taken apart and losing the context it added.
func heldErrors(err error) []error {
	multi, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return nil
	}
	var out []error
	for _, one := range multi.Unwrap() {
		if nested := heldErrors(one); len(nested) > 0 {
			out = append(out, nested...)
			continue
		}
		out = append(out, one)
	}
	return out
}

func renderOne(err error, read SourceReader) string {
	message := err.Error()

	source := Position(err)
	if source == nil || source.Line <= 0 || read == nil {
		return message + traceOf(err)
	}
	content, readErr := read(source.File)
	if readErr != nil {
		return message + traceOf(err)
	}
	excerpt := Excerpt(content, source)
	if excerpt == "" {
		return message + traceOf(err)
	}
	return message + "\n" + excerpt + traceOf(err)
}

// traceOf renders the frames an error was made with, when it has any.
func traceOf(err error) string {
	var traced Traced
	if !errors.As(err, &traced) {
		return ""
	}
	trace := traced.StackTrace()
	if trace == "" {
		return ""
	}
	return "\n" + trace
}

// Excerpt returns the line source names, numbered, with a caret beneath the column.
// It returns an empty string when the position falls outside the content, which happens when a file has changed since the error was made.
func Excerpt(content []byte, source *token.Source) string {
	if source == nil || source.Line <= 0 {
		return ""
	}
	lines := strings.Split(string(content), "\n")
	if source.Line > len(lines) {
		return ""
	}
	line := strings.TrimSuffix(lines[source.Line-1], "\r")

	number := fmt.Sprintf("%d", source.Line)
	gutter := strings.Repeat(" ", len(number))

	var out strings.Builder
	fmt.Fprintf(&out, "%s | %s\n", number, line)
	fmt.Fprintf(&out, "%s | %s^", gutter, caretPadding(line, source.Column))
	return out.String()
}

// caretPadding builds the run of whitespace that puts a caret under a column.
//
// Every byte before the column becomes a space, except a tab, which stays a tab. That way the caret lands in the right place whatever width the terminal draws tabs at, which expanding them to a guessed number of spaces would not.
func caretPadding(line string, column int) string {
	if column < 1 {
		column = 1
	}
	before := column - 1
	if before > len(line) {
		before = len(line)
	}
	var padding strings.Builder
	for _, ch := range line[:before] {
		if ch == '\t' {
			padding.WriteByte('\t')
			continue
		}
		padding.WriteByte(' ')
	}
	return padding.String()
}
