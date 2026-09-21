package lexer

import (
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

func (l *Lexer) parseLeadingDecorations() []token.DecorativeToken {
	var decors []token.DecorativeToken
	for {
		tok := l.parseDecorativeToken()
		if tok == nil {
			return decors
		}
		decors = append(decors, *tok)
	}
}

func (l *Lexer) parseDecorativeToken() *token.DecorativeToken {
	var tok token.DecorativeToken
	switch {
	case l.ch == '#': // COMMENT
		tok.Type = token.DECO_COMMENT
		tok.Literal = l.parseInlineComment()
	case l.ch == '/': // eventually COMMENT
		if l.peekChar() == '/' {
			tok.Type = token.DECO_COMMENT
			tok.Literal = l.parseInlineComment()
		} else {
			return nil
		}
	case isWhitespace(l.ch):
		tok.Type, tok.Literal = l.skipWhitespace()
	default:
		return nil
	}
	return &tok
}

// parseInlineComment returns the comment verbatim, marker included, stopping before the newline so every byte belongs to exactly one token or decoration.
func (l *Lexer) parseInlineComment() string {
	position := l.currPos
	for {
		if l.ch == '\n' || l.ch == 0 {
			return l.input[position:l.currPos]
		}
		l.advance()
	}
}

func (l *Lexer) skipWhitespace() (token.DecorativeTokenType, string) {
	tok := token.DECO_INLINE
	ws := strings.Builder{}
	for isWhitespace(l.ch) {
		if isNewline(l.ch) {
			tok = token.DECO_MULTI
		}
		ws.WriteByte(l.ch)
		l.advance()
	}
	return tok, ws.String()
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || isNewline(ch)
}

func isNewline(ch byte) bool {
	return ch == '\n'
}
