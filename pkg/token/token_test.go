package token_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

func TestLookupIdent(t *testing.T) {
	testCases := []struct {
		input    string
		expected token.TokenType
	}{
		{"foo", token.IDENT},
		{"bar", token.IDENT},
		{"true", token.TRUE},
		{"false", token.FALSE},
		{"mod", token.MODULE},
		{"import", token.IMPORT},
		{"data", token.DATA},
		{"attr", token.ANNOTATION},
		{"extern", token.EXTERN},
		{"fn", token.FUNCTION},
		{"let", token.LET},
		{"type", token.TYPE},
		{"return", token.RETURN},
		{"if", token.IF},
		{"else", token.ELSE},
		{"for", token.FOR},
		{"_", token.BLANK},
	}

	for _, tc := range testCases {
		if tok := token.LookupIdent(tc.input); tok != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, tok)
		}
	}
}
