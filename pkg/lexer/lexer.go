package lexer

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

type Lexer struct {
	src       registry.Source
	input     string
	startPos  int  // start position of this token
	peekPos   int  // current reading position in input (after current char)
	currPos   int  // current position in input (points to current char)
	ch        byte // current char under examination
	line      int  // 1-based line of currPos
	lineStart int  // offset of the first byte of that line, which is what turns an offset into a column
}

func New(src registry.Source) (*Lexer, error) {
	raw, err := src.Read()
	if err != nil {
		return nil, err
	}
	l := &Lexer{
		src:   src,
		input: string(raw),
		line:  1,
	}
	l.advance()
	return l, nil
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	tok.Leading = l.parseLeadingDecorations()
	l.startPos = l.currPos
	tokLine, tokColumn := l.position()
	tok.Source = token.MakeSource(string(l.src.URI()), l.currPos, tokLine, tokColumn)
	// Several branches below replace tok wholesale with a composite literal, dropping these; restore them after the switch.
	leading, src := tok.Leading, tok.Source

	switch l.ch {
	case '!':
		switch l.peekChar() {
		case '=':
			tok = token.Token{Type: token.NEQ, Literal: "!="}
			l.advance()
		case '.':
			tok = token.Token{Type: token.BANG_DOT, Literal: "!."}
			l.advance()
		case '!':
			tok = token.Token{Type: token.BANG_BANG, Literal: "!!"}
			l.advance()
		default:
			tok = l.newToken(token.BANG, l.ch)
		}
	case '?':
		switch l.peekChar() {
		case '.':
			tok = token.Token{Type: token.QUESTION_DOT, Literal: "?."}
			l.advance()
		case '?':
			tok = token.Token{Type: token.QUESTION_QUESTION, Literal: "??"}
			l.advance()
		default:
			tok = l.newToken(token.QUESTION, l.ch)
		}
	case '+':
		if l.peekChar() == '=' {
			tok = token.Token{Type: token.PLUS_ASSIGN, Literal: "+="}
			l.advance()
		} else {
			tok = l.newToken(token.PLUS, l.ch)
		}
	case '-':
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
	case '*':
		if l.peekChar() == '=' {
			tok = token.Token{Type: token.STAR_ASSIGN, Literal: "*="}
			l.advance()
		} else {
			tok = l.newToken(token.ASTERISK, l.ch)
		}
	case '/':
		if l.peekChar() == '=' {
			tok = token.Token{Type: token.SLASH_ASSIGN, Literal: "/="}
			l.advance()
		} else {
			tok = l.newToken(token.SLASH, l.ch)
		}
	case '%':
		if l.peekChar() == '=' {
			tok = token.Token{Type: token.PERCENT_ASSIGN, Literal: "%="}
			l.advance()
		} else {
			tok = l.newToken(token.PERCENT, l.ch)
		}

	case '<':
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
	case '>':
		if l.peekChar() == '=' {
			tok = token.Token{Type: token.GTE, Literal: ">="}
			l.advance()
		} else {
			tok = l.newToken(token.GT, l.ch)
		}
	case '=':
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
	case '&':
		if l.peekChar() == '&' {
			tok = token.Token{Type: token.AND, Literal: "&&"}
			l.advance()
		} else {
			tok = l.newToken(token.ILLEGAL, l.ch)
		}
	case '|':
		if l.peekChar() == '|' {
			tok = token.Token{Type: token.OR, Literal: "||"}
			l.advance()
		} else {
			tok = l.newToken(token.ILLEGAL, l.ch)
		}

	case ':':
		tok = l.newToken(token.COLON, l.ch)
	case '.':
		tok = l.newToken(token.DOT, l.ch)
	case ',':
		tok = l.newToken(token.COMMA, l.ch)
	case '(':
		tok = l.newToken(token.LPAREN, l.ch)
	case ')':
		tok = l.newToken(token.RPAREN, l.ch)
	case '{':
		tok = l.newToken(token.LBRACE, l.ch)
	case '}':
		tok = l.newToken(token.RBRACE, l.ch)
	case '[':
		tok = l.newToken(token.LBRACKET, l.ch)
	case ']':
		tok = l.newToken(token.RBRACKET, l.ch)
	case '@':
		tok = l.newToken(token.AT, l.ch)

	case '"':
		tok.Type = token.STRING
		tok.Literal = l.parseString()
	case '\'':
		tok.Type = token.CHAR
		literal, ok := l.parseChar()
		if !ok {
			tok.Type = token.ILLEGAL
			tok.Literal = l.input[l.startPos:l.currPos]
			break
		}
		tok.Literal = literal
	case 0:
		tok.Type = token.EOF
	default:
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

	tok.Leading = leading
	if tok.Source == nil {
		tok.Source = src
	}
	l.advance()
	return tok
}

// parseString scans a double-quoted string literal and returns its raw source, escapes and interpolations included. Decoding happens in the parser, the same way char literals are handled, so both literal forms accept exactly one set of escapes.
// An unterminated literal ends at EOF rather than being rejected, which keeps a half-typed line usable while it is being edited.
//
// Where the literal ends is decided by scanStringContent, the same walk the parser splits the literal with, so the two can never disagree about which quote closed it.
// The cursor is then moved there one byte at a time rather than jumped, because every line the literal spans is counted on the way.
func (l *Lexer) parseString() string {
	position := l.currPos + 1
	_, end := scanStringContent(l.input, position)
	for l.currPos < end {
		l.advance()
	}
	return l.input[position:end]
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
	// Every cursor move goes through here, one byte at a time, so counting newlines here is what keeps the line and column of any offset known without rescanning the file.
	if l.ch == '\n' {
		l.line += 1
		l.lineStart = l.peekPos
	}
	if l.peekPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.peekPos]
	}
	l.currPos = l.peekPos
	l.peekPos += 1
}

// position is the location of the cursor, as a reader would count it.
func (l *Lexer) position() (line int, column int) {
	return l.line, l.currPos - l.lineStart + 1
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

	if l.ch == '0' && l.peekChar() == 'x' {
		l.advance()
		l.advance()
		for isHexDigit(l.ch) {
			l.advance()
		}
		return l.input[position:l.currPos], token.INT
	}

	if l.ch == '0' && (l.peekChar() == 'b' || l.peekChar() == 'B') {
		l.advance()
		l.advance()
		for isBinaryDigit(l.ch) {
			l.advance()
		}
		return l.input[position:l.currPos], token.INT
	}

	for isDigit(l.ch) {
		l.advance()
	}

	if l.ch == '.' && isDigit(l.peekChar()) {
		l.advance()
		for isDigit(l.ch) {
			l.advance()
		}
		if l.ch == 'e' || l.ch == 'E' {
			l.advance()
			if l.ch == '+' || l.ch == '-' {
				l.advance()
			}
			for isDigit(l.ch) {
				l.advance()
			}
		}
		return l.input[position:l.currPos], token.FLOAT
	}

	if l.ch == 'e' || l.ch == 'E' {
		l.advance()
		if l.ch == '+' || l.ch == '-' {
			l.advance()
		}
		for isDigit(l.ch) {
			l.advance()
		}
		return l.input[position:l.currPos], token.FLOAT
	}

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
		Source:  l.sourceHere(),
	}
}

// sourceHere is the location of the cursor, for a token built away from NextToken's own bookkeeping.
func (l *Lexer) sourceHere() *token.Source {
	line, column := l.position()
	return token.MakeSource(string(l.src.URI()), l.currPos, line, column)
}

// Region returns a lexer over input[start:end], reading the same file and numbering its lines and columns as they sit in the whole of it.
// It is how an interpolated expression is lexed where it stands, so a diagnostic inside `\( … )` points at the source and not at a detached fragment of it.
func (l *Lexer) Region(start, end int) *Lexer {
	line, lineStart := 1, 0
	for i := 0; i < start && i < len(l.input); i++ {
		if l.input[i] == '\n' {
			line += 1
			lineStart = i + 1
		}
	}
	sub := &Lexer{
		src:       l.src,
		input:     l.input[:min(end, len(l.input))],
		peekPos:   start,
		line:      line,
		lineStart: lineStart,
	}
	sub.advance()
	return sub
}
