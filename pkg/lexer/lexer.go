package lexer

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

type Lexer struct {
	src      registry.Source
	input    string
	startPos int  // start position of this token
	peekPos  int  // current reading position in input (after current char)
	currPos  int  // current position in input (points to current char)
	ch       byte // current char under examination
}

func New(src registry.Source) (*Lexer, error) {
	raw, err := src.Read()
	if err != nil {
		return nil, err
	}
	l := &Lexer{
		src:   src,
		input: string(raw),
	}
	l.advance()
	return l, nil
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	tok.Leading = l.parseLeadingDecorations()
	l.startPos = l.currPos
	tok.Source = token.MakeSource(string(l.src.URI()), l.currPos)

	switch l.ch {
	case '!': // BANG, NEQ
		if l.peekChar() == '=' {
			tok = token.Token{Type: token.NEQ, Literal: "!="}
			l.advance()
		} else {
			tok = l.newToken(token.BANG, l.ch)
		}
	case '+': // PLUS, PLUS_ASSIGN
		if l.peekChar() == '=' {
			tok = token.Token{Type: token.PLUS_ASSIGN, Literal: "+="}
			l.advance()
		} else {
			tok = l.newToken(token.PLUS, l.ch)
		}
	case '-': // MINUS, MINUS_ASSIGN, RIGHT_ARROW
		switch l.peekChar() {
		case '>':
			tok = token.Token{Type: token.RIGHT_ARROW, Literal: "->"}
			l.advance()
		case '=':
			tok = token.Token{Type: token.MINUS_ASSIGN, Literal: "-="}
			l.advance()
		default:
			tok = l.newToken(token.MINUS, l.ch)
		}
	case '*': // ASTERISK, STAR_ASSIGN
		if l.peekChar() == '=' {
			tok = token.Token{Type: token.STAR_ASSIGN, Literal: "*="}
			l.advance()
		} else {
			tok = l.newToken(token.ASTERISK, l.ch)
		}
	case '/': // SLASH, SLASH_ASSIGN
		if l.peekChar() == '=' {
			tok = token.Token{Type: token.SLASH_ASSIGN, Literal: "/="}
			l.advance()
		} else {
			tok = l.newToken(token.SLASH, l.ch)
		}
	case '%': // PERCENT, PERCENT_ASSIGN
		if l.peekChar() == '=' {
			tok = token.Token{Type: token.PERCENT_ASSIGN, Literal: "%="}
			l.advance()
		} else {
			tok = l.newToken(token.PERCENT, l.ch)
		}

	case '<': // LT, LTE
		switch l.peekChar() {
		case '=':
			tok = token.Token{Type: token.LTE, Literal: "<="}
			l.advance()
		case '-':
			tok = token.Token{Type: token.LEFT_ARROW, Literal: "<-"}
			l.advance()
		default:
			tok = l.newToken(token.LT, l.ch)
		}
	case '>': // GT, GTE
		if l.peekChar() == '=' {
			tok = token.Token{Type: token.GTE, Literal: ">="}
			l.advance()
		} else {
			tok = l.newToken(token.GT, l.ch)
		}
	case '=': // ASSIGN, EQ, ARROW
		switch l.peekChar() {
		case '=':
			tok = token.Token{Type: token.EQ, Literal: "=="}
			l.advance()
		case '>':
			tok = token.Token{Type: token.RIGHT_ARROW, Literal: "->"}
			l.advance()
		default:
			tok = l.newToken(token.ASSIGN, l.ch)
		}
	case '&': // AND
		if l.peekChar() == '&' {
			tok = token.Token{Type: token.AND, Literal: "&&"}
			l.advance()
		} else {
			tok = l.newToken(token.ILLEGAL, l.ch)
		}
	case '|': // OR
		if l.peekChar() == '|' {
			tok = token.Token{Type: token.OR, Literal: "||"}
			l.advance()
		} else {
			tok = l.newToken(token.ILLEGAL, l.ch)
		}

	case ':': // COLON
		tok = l.newToken(token.COLON, l.ch)
	case '.': // DOT
		tok = l.newToken(token.DOT, l.ch)
	case ',': // COMMA
		tok = l.newToken(token.COMMA, l.ch)
	case '(': // LPAREN
		tok = l.newToken(token.LPAREN, l.ch)
	case ')': // RPAREN
		tok = l.newToken(token.RPAREN, l.ch)
	case '{': // LBRACE
		tok = l.newToken(token.LBRACE, l.ch)
	case '}': // RBRACE
		tok = l.newToken(token.RBRACE, l.ch)
	case '[': // LBRACKET
		tok = l.newToken(token.LBRACKET, l.ch)
	case ']': // RBRACKET
		tok = l.newToken(token.RBRACKET, l.ch)
	case '@': // AT
		tok = l.newToken(token.AT, l.ch)

	case '"': // STRING
		tok.Type = token.STRING
		tok.Literal = l.parseString()
	case '\'': // CHAR
		tok.Type = token.CHAR
		literal, ok := l.parseChar()
		if !ok {
			tok.Type = token.ILLEGAL
			tok.Literal = l.input[l.startPos:l.currPos]
			break
		}
		tok.Literal = literal
	case 0: // EOF
		tok.Type = token.EOF
	default: // IDENT, INT, FLOAT
		if isLetter(l.ch) {
			tok.Literal = l.parseIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Literal, tok.Type = l.parseNumber()
			return tok
		} else {
			tok = l.newToken(token.ILLEGAL, l.ch)
		}
	}

	l.advance()
	return tok
}

// parseString scans a double-quoted string literal and returns its raw source,
// escapes included. Decoding happens in the parser, the same way char literals
// are handled, so both literal forms accept exactly one set of escapes.
// An unterminated literal ends at EOF rather than being rejected, which keeps
// a half-typed line usable while it is being edited.
func (l *Lexer) parseString() string {
	position := l.currPos + 1
	escaped := false
	for {
		l.advance()
		if l.ch == 0 {
			break
		}
		if escaped {
			escaped = false
			continue
		}
		if l.ch == '\\' {
			escaped = true
			continue
		}
		if l.ch == '"' {
			break
		}
	}
	return l.input[position:l.currPos]
}

func (l *Lexer) parseChar() (string, bool) {
	position := l.currPos + 1
	escaped := false
	for {
		l.advance()
		if l.ch == 0 {
			l.peekPos = l.currPos
			return "", false
		}
		if l.ch == '\n' || l.ch == '\r' || l.ch == '\t' {
			l.peekPos = l.currPos
			return "", false
		}
		if escaped {
			escaped = false
			continue
		}
		if l.ch == '\\' {
			escaped = true
			continue
		}
		if l.ch == '\'' {
			break
		}
	}
	return l.input[position:l.currPos], true
}

func (l *Lexer) advance() {
	if l.peekPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.peekPos]
	}
	l.currPos = l.peekPos
	l.peekPos += 1
}

func (l *Lexer) peekChar() byte {
	if l.peekPos >= len(l.input) {
		return 0
	} else {
		return l.input[l.peekPos]
	}
}

func (l *Lexer) parseIdentifier() string {
	position := l.currPos
	for isLetter(l.ch) || isDigit(l.ch) {
		l.advance()
	}
	return l.input[position:l.currPos]
}

func (l *Lexer) parseNumber() (string, token.TokenType) {
	position := l.currPos

	// Handle special prefixes: 0x, 0b, 0B
	if l.ch == '0' && l.peekChar() == 'x' {
		// Hexadecimal: 0x...
		l.advance() // consume '0'
		l.advance() // consume 'x'
		for isHexDigit(l.ch) {
			l.advance()
		}
		return l.input[position:l.currPos], token.INT
	}

	if l.ch == '0' && (l.peekChar() == 'b' || l.peekChar() == 'B') {
		// Binary: 0b... or 0B...
		l.advance() // consume '0'
		l.advance() // consume 'b' or 'B'
		for isBinaryDigit(l.ch) {
			l.advance()
		}
		return l.input[position:l.currPos], token.INT
	}

	// Parse regular decimal number (including octal starting with 0)
	for isDigit(l.ch) {
		l.advance()
	}

	// Check for decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.advance() // consume '.'
		for isDigit(l.ch) {
			l.advance()
		}
		// Check for scientific notation in float
		if l.ch == 'e' || l.ch == 'E' {
			l.advance() // consume 'e' or 'E'
			if l.ch == '+' || l.ch == '-' {
				l.advance() // consume sign
			}
			for isDigit(l.ch) {
				l.advance()
			}
		}
		return l.input[position:l.currPos], token.FLOAT
	}

	// Check for scientific notation in integer (e.g., 2e10)
	if l.ch == 'e' || l.ch == 'E' {
		l.advance() // consume 'e' or 'E'
		if l.ch == '+' || l.ch == '-' {
			l.advance() // consume sign
		}
		for isDigit(l.ch) {
			l.advance()
		}
		return l.input[position:l.currPos], token.FLOAT
	}

	// Regular integer (decimal or octal)
	return l.input[position:l.currPos], token.INT
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isHexDigit(ch byte) bool {
	return isDigit(ch) || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}

func isBinaryDigit(ch byte) bool {
	return ch == '0' || ch == '1'
}

func (l *Lexer) newToken(tokenType token.TokenType, ch byte) token.Token {
	return token.Token{
		Type:    tokenType,
		Literal: string(ch),
		Source:  token.MakeSource(string(l.src.URI()), l.currPos),
	}
}
