package codefmt

import "code.knabel.dev/zirric-lang/zirric/pkg/token"

// valueEnders can end an expression, so a "-" or "+" directly after one is binary rather than prefix.
var valueEnders = map[token.TokenType]bool{
	token.IDENT:    true,
	token.INT:      true,
	token.FLOAT:    true,
	token.STRING:   true,
	token.CHAR:     true,
	token.TRUE:     true,
	token.FALSE:    true,
	token.VOID:     true,
	token.BLANK:    true,
	token.RPAREN:   true,
	token.RBRACKET: true,
	token.RBRACE:   true,
}

// callTargets precede a "(" that calls rather than groups, and a "[" that indexes rather than opening a literal.
var callTargets = map[token.TokenType]bool{
	token.IDENT:    true,
	token.RPAREN:   true,
	token.RBRACKET: true,
}

var noSpaceBefore = map[token.TokenType]bool{
	token.RPAREN:       true,
	token.RBRACKET:     true,
	token.COMMA:        true,
	token.COLON:        true,
	token.DOT:          true,
	token.QUESTION_DOT: true,
	token.BANG_DOT:     true,
}

var noSpaceAfter = map[token.TokenType]bool{
	token.LPAREN:       true,
	token.LBRACKET:     true,
	token.DOT:          true,
	token.QUESTION_DOT: true,
	token.BANG_DOT:     true,
	token.AT:           true,
}

// isTypeSuffix reports whether a "!" or "?" writes one of the type shorthands, "String!" or "User?", rather than starting an expression of its own.
// What tells them apart is that a suffix sits directly on the type it qualifies, with nothing between the two — the same rule the parser applies.
func isTypeSuffix(tok token.Token, prev token.Token, hasPrev bool, glued bool) bool {
	if tok.Type != token.BANG && tok.Type != token.QUESTION {
		return false
	}
	return glued && hasPrev && valueEnders[prev.Type]
}

// isPrefixOperator resolves the only ambiguity there is: "*", "/" and "%" are always binary, leaving "-", "+" and a "!" that is not a type suffix.
func isPrefixOperator(tok token.Token, prev token.Token, hasPrev bool, glued bool) bool {
	switch tok.Type {
	case token.MINUS, token.PLUS:
		return !hasPrev || !valueEnders[prev.Type]
	case token.BANG:
		return !isTypeSuffix(tok, prev, hasPrev, glued)
	}
	return false
}

func needSpace(prev token.Token, prevWasPrefix bool, next token.Token, nextGlued bool) bool {
	if prevWasPrefix {
		return false
	}
	if isTypeSuffix(next, prev, true, nextGlued) {
		return false
	}
	if noSpaceAfter[prev.Type] {
		return false
	}
	if prev.Type == token.LBRACE && next.Type == token.RBRACE {
		return false // empty body, as in "attr Marker {}"
	}
	if noSpaceBefore[next.Type] {
		return false
	}
	if next.Type == token.LPAREN && (callTargets[prev.Type] || prev.Type == token.FUNCTION) {
		return false
	}
	if next.Type == token.LBRACKET && (callTargets[prev.Type] || prev.Type == token.STRING) {
		return false
	}
	return true
}

// glued lists two-character sequences that lex as one token; keeping a space where they would meet makes that corruption impossible.
var glued = map[string]bool{
	"->": true, "<-": true, "<=": true, ">=": true, "==": true, "!=": true,
	"&&": true, "||": true, "+=": true, "-=": true, "*=": true, "/=": true,
	"%=": true, "//": true, "/*": true, "?.": true, "!.": true,
	"??": true, "!!": true,
}

func wouldGlue(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return glued[string(left[len(left)-1])+string(right[0])]
}
