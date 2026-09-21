package codefmt

import (
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

type frameKind int

const (
	frameBrace frameKind = iota
	frameParen
	frameBracket
	// frameSwitchBrace adds no indent of its own, so `case` sits level with its `switch`.
	frameSwitchBrace
	// frameCase is virtual: case bodies end at the next `case` or the closing brace, not at a bracket.
	frameCase
)

func (k frameKind) indents() bool { return k != frameSwitchBrace }

type frame struct {
	kind     frameKind
	openLine int
}

// continuationStarters can only continue the previous line, so indenting them keeps a hand-written operator chain off the left margin.
var continuationStarters = map[token.TokenType]bool{
	token.DOT: true, token.PLUS: true, token.MINUS: true, token.ASTERISK: true,
	token.SLASH: true, token.PERCENT: true, token.EQ: true, token.NEQ: true,
	token.LT: true, token.GT: true, token.LTE: true, token.GTE: true,
	token.AND: true, token.OR: true, token.RIGHT_ARROW: true, token.LEFT_ARROW: true,
}

type printer struct {
	opts   Options
	indent string

	out []string
	cur strings.Builder

	lineNo       int
	frames       []frame
	switchDepths []int

	prev          token.Token
	hasPrev       bool
	prevWasPrefix bool

	// prevLineFrames is len(frames) when the current line started, read only by the continuation-line rule.
	prevLineFrames int

	// prePopped counts closers on this line whose frames were already popped.
	prePopped int

	// casePopped records that the upcoming `case` closed its predecessor early, so an introducing comment aligns with it.
	casePopped bool
}

func newPrinter(opts Options) *printer {
	return &printer{opts: opts, indent: opts.indentUnit()}
}

// indentLevel counts distinct lines that opened a frame, which is what indents the body of `@Error(fn(err) {` once rather than twice.
func (p *printer) indentLevel() int {
	level, last := 0, -1
	for _, f := range p.frames {
		if !f.kind.indents() || f.openLine >= p.lineNo {
			continue
		}
		if f.openLine != last {
			level++
			last = f.openLine
		}
	}
	return level
}

func (p *printer) flushLine() {
	p.out = append(p.out, strings.TrimRight(p.cur.String(), " \t"))
	p.cur.Reset()
}

func (p *printer) startLine(blanks int, first token.Token, hasFirst bool) {
	p.flushLine()
	for range blanks {
		p.out = append(p.out, "")
	}
	p.lineNo = len(p.out)
	p.prevLineFrames = len(p.frames)

	level := p.indentLevel()
	if hasFirst && continuationStarters[first.Type] && len(p.frames) == p.prevLineFrames && p.hasPrev {
		level++
	}
	p.cur.WriteString(strings.Repeat(p.indent, level))
}

func (p *printer) lineHasContent() bool {
	return strings.TrimSpace(p.cur.String()) != ""
}

// blankLinesFor bounds a run of newlines, suppressing blanks where they only add noise.
func (p *printer) blankLinesFor(newlines int, next token.Token, hasNext bool) int {
	blanks := newlines - 1
	if blanks > p.opts.MaxBlankLines {
		blanks = p.opts.MaxBlankLines
	}
	if blanks < 0 {
		blanks = 0
	}
	if len(p.out) == 0 && !p.lineHasContent() {
		return 0 // start of file
	}
	if p.hasPrev && p.prev.Type == token.LBRACE {
		return 0
	}
	if hasNext && next.Type == token.RBRACE {
		return 0
	}
	return blanks
}

func (p *printer) writeComment(text string, newlines int) {
	if newlines == 0 && p.lineHasContent() {
		p.cur.WriteString(" ")
		p.cur.WriteString(text)
		return
	}
	if p.lineHasContent() || len(p.out) > 0 {
		p.startLine(p.blankLinesFor(newlines, token.Token{}, false), token.Token{}, false)
	} else {
		p.cur.WriteString(strings.Repeat(p.indent, p.indentLevel()))
	}
	p.cur.WriteString(text)
}

func (p *printer) writeToken(items []item, i int, newlines int) {
	it := items[i]
	tok := it.tok
	onNewLine := newlines > 0 || !p.lineHasContent()

	if onNewLine {
		// A line may close several frames at once, as in "})"; all must pop before its indent is computed or it hangs too deep.
		p.prePopped = 0
		for j := i; j < len(items) && isCloser(items[j].tok.Type); j++ {
			if j > i && startsNewLine(items[j]) {
				break
			}
			p.frames = popCloser(p.frames, items[j].tok.Type)
			p.prePopped++
		}
		p.popCaseFor(tok)
		if isCloser(tok.Type) && p.prePopped > 0 {
			p.prePopped--
		}
		if p.lineHasContent() || len(p.out) > 0 {
			p.startLine(p.blankLinesFor(newlines, tok, true), tok, true)
		} else {
			p.cur.WriteString(strings.Repeat(p.indent, p.indentLevel()))
		}
	} else {
		if p.prePopped > 0 && isCloser(tok.Type) {
			p.prePopped--
		} else {
			p.frames = popCloser(p.frames, tok.Type)
		}
		p.popCaseFor(tok)
		if p.hasPrev && (needSpace(p.prev, p.prevWasPrefix, tok) ||
			wouldGlue(p.cur.String(), canonicalText(it))) {
			p.cur.WriteString(" ")
		}
	}

	p.cur.WriteString(canonicalText(it))
	p.prevWasPrefix = isPrefixOperator(tok, p.prev, p.hasPrev)
	p.prev = tok
	p.hasPrev = true
	p.pushForOpener(tok)
}

// popCaseFor closes the preceding case body so a sibling `case` lands at the switch's own level.
func (p *printer) popCaseFor(tok token.Token) {
	if tok.Type != token.CASE || !p.inSwitchBody() {
		return
	}
	if p.casePopped {
		p.casePopped = false
		return
	}
	if n := len(p.frames); n > 0 && p.frames[n-1].kind == frameCase {
		p.frames = p.frames[:n-1]
	}
}

// prepareForCase closes the previous case body before this item's comments, so a comment documenting a `case` aligns with it.
func (p *printer) prepareForCase(it item) {
	if it.tok.Type != token.CASE || !startsNewLine(it) || !p.inSwitchBody() {
		return
	}
	if n := len(p.frames); n > 0 && p.frames[n-1].kind == frameCase {
		p.frames = p.frames[:n-1]
		p.casePopped = true
	}
}

func (p *printer) inSwitchBody() bool {
	for i := len(p.frames) - 1; i >= 0; i-- {
		if p.frames[i].kind == frameCase {
			continue
		}
		return p.frames[i].kind == frameSwitchBrace
	}
	return false
}

func isCloser(t token.TokenType) bool {
	return t == token.RBRACE || t == token.RPAREN || t == token.RBRACKET
}

func popCloser(frames []frame, tokType token.TokenType) []frame {
	var want frameKind
	switch tokType {
	case token.RBRACE:
		if n := len(frames); n > 0 && frames[n-1].kind == frameCase {
			frames = frames[:n-1]
		}
		if n := len(frames); n > 0 && frames[n-1].kind == frameSwitchBrace {
			return frames[:n-1]
		}
		want = frameBrace
	case token.RPAREN:
		want = frameParen
	case token.RBRACKET:
		want = frameBracket
	default:
		return frames
	}
	if n := len(frames); n > 0 && frames[n-1].kind == want {
		return frames[:n-1]
	}
	return frames
}

func startsNewLine(it item) bool {
	for _, dec := range it.leading {
		if dec.Type == token.DECO_MULTI || dec.Type == token.DECO_COMMENT {
			return true
		}
	}
	return false
}

func (p *printer) pushForOpener(tok token.Token) {
	switch tok.Type {
	case token.SWITCH:
		p.switchDepths = append(p.switchDepths, len(p.frames))
	case token.LBRACE:
		// Drop stale entries from a `switch` that never reached its brace.
		for len(p.switchDepths) > 0 && p.switchDepths[len(p.switchDepths)-1] > len(p.frames) {
			p.switchDepths = p.switchDepths[:len(p.switchDepths)-1]
		}
		if n := len(p.switchDepths); n > 0 && p.switchDepths[n-1] == len(p.frames) {
			p.switchDepths = p.switchDepths[:n-1]
			p.frames = append(p.frames, frame{frameSwitchBrace, p.lineNo})
			return
		}
		p.frames = append(p.frames, frame{frameBrace, p.lineNo})
	case token.LPAREN:
		p.frames = append(p.frames, frame{frameParen, p.lineNo})
	case token.LBRACKET:
		p.frames = append(p.frames, frame{frameBracket, p.lineNo})
	case token.COLON:
		if n := len(p.frames); n > 0 && p.frames[n-1].kind == frameCase {
			// The case pattern's colon opens the body; the frame is already open.
			return
		}
	case token.CASE:
		if p.inSwitchBody() {
			p.frames = append(p.frames, frame{frameCase, p.lineNo})
		}
	}
}

func (p *printer) result() string {
	p.flushLine()
	for len(p.out) > 0 && p.out[len(p.out)-1] == "" {
		p.out = p.out[:len(p.out)-1]
	}
	if len(p.out) == 0 {
		return ""
	}
	text := strings.Join(p.out, "\n")
	if p.opts.InsertFinalNewline {
		text += "\n"
	}
	return text
}

func render(items []item, opts Options) string {
	p := newPrinter(opts)
	for i, it := range items {
		p.prepareForCase(it)
		newlines := 0
		for _, dec := range it.leading {
			switch dec.Type {
			case token.DECO_MULTI:
				newlines += strings.Count(dec.Literal, "\n")
			case token.DECO_COMMENT:
				p.writeComment(dec.Literal, newlines)
				newlines = 0
			}
		}
		if it.tok.Type == token.EOF {
			continue
		}
		p.writeToken(items, i, newlines)
	}
	return p.result()
}
