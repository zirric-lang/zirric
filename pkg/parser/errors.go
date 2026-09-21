package parser

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

type ParseError struct {
	Token   token.Token
	Summary string
	Details string
}

// Error implements error.
func (e ParseError) Error() string {
	if e.Token.Source == nil {
		return fmt.Sprintf("%s: %s", e.Summary, e.Details)
	}
	if e.Details == "" {
		return fmt.Sprintf("%s: %s", e.Token.Source.String(), e.Summary)
	}
	return fmt.Sprintf("%s: %s: %s", e.Token.Source.String(), e.Summary, e.Details)
}

// Position implements diag.Positioned, so a renderer can show the line this refers to.
func (e ParseError) Position() *token.Source {
	return e.Token.Source
}

// ParseErrors is a collection of parse errors that implements the error
// interface. Callers can unwrap it via errors.As to access individual errors.
type ParseErrors []ParseError

// Unwrap implements the convention for an error holding several errors, which is what lets errors.As reach an individual ParseError and what a renderer walks to show each one.
func (e ParseErrors) Unwrap() []error {
	errs := make([]error, len(e))
	for i := range e {
		errs[i] = e[i]
	}
	return errs
}

func (e ParseErrors) Error() string {
	msgs := make([]string, len(e))
	for i, pe := range e {
		msgs[i] = pe.Error()
	}
	return strings.Join(msgs, "\n")
}

func (p *Parser) errUnexpectedToken(want ...token.TokenType) {
	slices.SortFunc(want, func(a, b token.TokenType) int {
		return strings.Compare(string(a), string(b))
	})

	var wanted bytes.Buffer

	for i, t := range want {
		wanted.WriteString(strings.ToLower(string(t)))
		if i < len(want)-1 {
			wanted.WriteString(", ")
		}
	}
	p.detectError(ParseError{
		Token:   p.curToken,
		Summary: fmt.Sprintf("unexpected %q", p.curToken.Literal),
		Details: fmt.Sprintf("want one of [%s]", wanted.String()),
	})
}

func (p *Parser) errUnderlyingErrorf(err error, format string, a ...any) {
	p.detectError(ParseError{
		Token:   p.curToken,
		Summary: fmt.Sprintf(format, a...),
		Details: err.Error(),
	})
}

func (p *Parser) errStatementMisplaced(pos StatementPosition) {
	summary := fmt.Sprintf("statement %s misplaced", strings.ToLower(string(p.curToken.Type)))
	switch p.curToken.Type {
	case token.RETURN:
		summary = "return must be inside function"
	case token.IMPORT:
		summary = "imports must be global"
	case token.EXTERN:
		summary = "extern must be global"
	case token.MODULE:
		summary = "mod may only appear first"
	}

	details := "here"
	switch pos {
	case IN_INITIAL:
		details = "not allowed as first global statement"
	case IN_GLOBAL:
		if p.curToken.Type == token.MODULE {
			details = "another statement precedes it"
		} else {
			details = "not allowed as global statement"
		}
	case IN_UNION:
		details = "not allowed inside union"
	case IN_DATA:
		details = "not allowed as part of data"
	case IN_EXTERN:
		details = "not allowed as part of extern"
	case IN_FUNC:
		details = "not allowed inside function"
	case IN_FOR:
		details = "not allowed in for loop"
	case IN_SWITCH:
		details = "not allowed in switch statement"
	}
	p.detectError(ParseError{
		Token:   p.curToken,
		Summary: summary,
		Details: details,
	})
}

func (p *Parser) errCannotBeAnnotated() {
	p.detectError(ParseError{
		Token:   p.curToken,
		Summary: fmt.Sprintf("%s cannot have attributes", strings.ToLower(string(p.curToken.Type))),
	})
}

// errUnclosed reports a group that was opened and never closed.
//
// It is anchored at the opening delimiter rather than at the token that finally gave the game away, because that is where the fix goes: a reader told only "unexpected }" has to find the matching bracket themselves, which is the whole difficulty.
func (p *Parser) errUnclosed(open token.Token, want token.TokenType) {
	found := p.curToken.Literal
	if p.curToken.Type == token.EOF {
		found = "end of file"
	} else {
		found = fmt.Sprintf("%q", found)
	}

	details := fmt.Sprintf("expected %s, found %s", strings.ToLower(string(want)), found)
	if source := p.curToken.Source; source != nil && source.Line > 0 {
		details = fmt.Sprintf("%s at %d:%d", details, source.Line, source.Column)
	}
	p.detectError(ParseError{
		Token:   open,
		Summary: fmt.Sprintf("unclosed %s", open.Literal),
		Details: details,
	})
}

// expectClosing consumes the delimiter that closes a group, naming where the group was opened when it is missing.
func (p *Parser) expectClosing(open token.Token, want token.TokenType) (token.Token, bool) {
	if !p.curIs(want) {
		p.errUnclosed(open, want)
		return p.errorToken(), false
	}
	cur := p.curToken
	p.nextToken()
	return cur, true
}

// errExpected reports a token that cannot begin what was being parsed, naming what was wanted rather than every token that could have started it.
//
// Listing them all tells a reader less, not more: seventeen tokens is a list nobody reads, and "expected a statement" is the same information in a form they can act on.
func (p *Parser) errExpected(what string) {
	p.detectError(ParseError{
		Token:   p.curToken,
		Summary: fmt.Sprintf("unexpected %q", p.curToken.Literal),
		Details: fmt.Sprintf("expected %s", what),
	})
}
