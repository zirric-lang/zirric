package lexer

// StringSegment is one piece of a string literal: a run of literal source, or an interpolated expression.
type StringSegment struct {
	// Text is a literal run's raw source, escapes left undecoded. Empty for an interpolation.
	Text string
	// Expr is an interpolation's expression source, without its `\(` and `)`.
	Expr string
	// Offset is where Expr begins inside the literal's content, which is what turns a position in the expression into one in the file.
	Offset int
	// Interpolation tells the two kinds apart, since either field may legitimately be empty.
	Interpolation bool
	// Unclosed marks an interpolation whose `)` never arrived, which the parser reports.
	Unclosed bool
}

// SplitString cuts a string literal's content into its literal runs and interpolations.
// It is the same walk the lexer uses to find where the literal ends, so a segment's Offset always lands where the parser expects it in the file.
func SplitString(content string) []StringSegment {
	segments, _ := scanStringContent(content, 0)
	return segments
}

// scanStringContent walks a string literal's content starting at input[start] and returns its segments together with the index of the byte that ends the literal — its closing quote, or the end of what could be scanned.
//
// An interpolation is scanned along with the literal so that the quote ending it is the one outside every `\( … )`, which is what lets a nested literal carry interpolations of its own.
// A line break inside an interpolation ends the literal there: nothing closes it any more, and swallowing the rest of the file over one stray `\(` would move the error far from its cause.
func scanStringContent(input string, start int) ([]StringSegment, int) {
	var segments []StringSegment
	text := start
	i := start
	for i < len(input) {
		switch input[i] {
		case '"':
			if text < i {
				segments = append(segments, StringSegment{Text: input[text:i]})
			}
			return segments, i
		case '\\':
			if i+1 >= len(input) {
				i = len(input)
				continue
			}
			if input[i+1] != '(' {
				i += 2
				continue
			}
			if text < i {
				segments = append(segments, StringSegment{Text: input[text:i]})
			}
			exprStart := i + 2
			exprEnd, closed := scanInterpolation(input, exprStart)
			segments = append(segments, StringSegment{
				Expr:          input[exprStart:exprEnd],
				Offset:        exprStart - start,
				Interpolation: true,
				Unclosed:      !closed,
			})
			if !closed {
				return segments, exprEnd
			}
			i = exprEnd + 1
			text = i
		default:
			i++
		}
	}
	if text < len(input) {
		segments = append(segments, StringSegment{Text: input[text:]})
	}
	return segments, len(input)
}

// scanInterpolation finds the `)` closing an interpolation that opened just before start, returning its index and whether it was found at all.
// Nested strings are scanned by scanStringContent, so a `)` inside one of them does not close the interpolation around it.
func scanInterpolation(input string, start int) (int, bool) {
	depth := 1
	for i := start; i < len(input); i++ {
		switch input[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, true
			}
		case '"':
			_, end := scanStringContent(input, i+1)
			if end >= len(input) {
				return len(input), false
			}
			i = end
		case '\'':
			end, ok := scanCharLiteral(input, i+1)
			if !ok {
				return end, false
			}
			i = end
		case '\n':
			return i, false
		}
	}
	return len(input), false
}

// scanCharLiteral finds the `'` closing a char literal that opened just before start.
func scanCharLiteral(input string, start int) (int, bool) {
	for i := start; i < len(input); i++ {
		switch input[i] {
		case '\\':
			i++
		case '\'':
			return i, true
		case '\n':
			return i, false
		}
	}
	return len(input), false
}

// SplitLiteral cuts a whole string literal, its quotes included, into its segments.
// It reports false for anything that is not one closed literal — an unterminated string, or an interpolation with no `)` — which is what a caller needs to know before rewriting the literal it was given.
func SplitLiteral(text string) ([]StringSegment, bool) {
	if len(text) < 2 || text[0] != '"' {
		return nil, false
	}
	segments, end := scanStringContent(text, 1)
	if end != len(text)-1 {
		return nil, false
	}
	return segments, true
}
