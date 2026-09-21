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
	token.RPAREN:   true,
	token.RBRACKET: true,
	token.COMMA:    true,
	token.COLON:    true,
	token.DOT:      true,
}

var noSpaceAfter = map[token.TokenType]bool{
	token.LPAREN:   true,
	token.LBRACKET: true,
	token.DOT:      true,
	token.AT:       true,
	token.BANG:     true,
}

// isPrefixOperator resolves the only ambiguity there is: "*", "/" and "%" are always binary and "!" always prefix, leaving "-" and "+".
func isPrefixOperator(tok token.Token, prev token.Token, hasPrev bool) bool {
	if tok.Type != token.MINUS && tok.Type != token.PLUS {
		return false
	}
	if !hasPrev {
		return true
	}
	return !valueEnders[prev.Type]
}

func needSpace(prev token.Token, prevWasPrefix bool, next token.Token) bool {
	if prevWasPrefix {
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
	"%=": true, "//": true, "/*": true,
}

func wouldGlue(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return glued[string(left[len(left)-1])+string(right[0])]
}
