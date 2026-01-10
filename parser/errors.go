package parser

import (
	"bytes"
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/token"
)

type ParseError struct {
	Token   token.Token
	Summary string
	Details string
}

// Error implements error.
func (e ParseError) Error() string {
	return fmt.Sprintf("syntax error: %s, %s", e.Summary, e.Details)
}

func (p *Parser) errUnexpectedToken(want ...token.TokenType) {
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

func (p *Parser) errUnexpectedPeekToken(want ...token.TokenType) {
	var wanted bytes.Buffer
	for i, t := range want {
		wanted.WriteString(strings.ToLower(string(t)))
		if i < len(want)-1 {
			wanted.WriteString(", ")
		}
	}
	p.detectError(ParseError{
		Token:   p.peekToken,
		Summary: fmt.Sprintf("unexpected %s %q", p.peekToken.Type, p.peekToken.Literal),
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
		summary = "module may only appear first"
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
	case IN_ENUM:
		details = "not allowed inside enum"
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
		Summary: fmt.Sprintf("%s cannot be annotated", strings.ToLower(string(p.curToken.Type))),
	})
}
