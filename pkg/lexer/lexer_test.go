package lexer_test

import (
	"fmt"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

func TestLexer(t *testing.T) {
	input := `
#!/usr/bin/env zirric
mod example

import tests {
	test
}
import tests.tests_t

test "any in unions matches all types", { fail ->
	union AnyUnion {
		Int
		Any
	}

	const isCorrect = with "should be any", type AnyUnion {
		Int: { _ -> False },
		Any: { _ -> True }
	}
	unless isCorrect, fail "should not be the case"
}
`
	l, err := lexer.New(staticmodule.NewSourceString("testing:///test/test.zirr", input))
	if err != nil {
		t.Fatal(err)
	}

	expect := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		// {token.COMMENT, "!/usr/bin/env zirric"},
		{token.MODULE, "mod"},
		{token.IDENT, "example"},
		{token.IMPORT, "import"},
		{token.IDENT, "tests"},
		{token.LBRACE, "{"},
		{token.IDENT, "test"},
		{token.RBRACE, "}"},
		{token.IMPORT, "import"},
		{token.IDENT, "tests"},
		{token.DOT, "."},
		{token.IDENT, "tests_t"},
		{token.IDENT, "test"},
		{token.STRING, "any in unions matches all types"},
		{token.COMMA, ","},
		{token.LBRACE, "{"},
		{token.IDENT, "fail"},
		{token.RIGHT_ARROW, "->"},
		{token.UNION, "union"},
		{token.IDENT, "AnyUnion"},
		{token.LBRACE, "{"},
		{token.IDENT, "Int"},
		{token.IDENT, "Any"},
		{token.RBRACE, "}"},
		{token.CONST, "const"},
		{token.IDENT, "isCorrect"},
		{token.ASSIGN, "="},
		{token.IDENT, "with"},
		{token.STRING, "should be any"},
		{token.COMMA, ","},
		{token.TYPE, "type"},
		{token.IDENT, "AnyUnion"},
		{token.LBRACE, "{"},
		{token.IDENT, "Int"},
		{token.COLON, ":"},
		{token.LBRACE, "{"},
		{token.BLANK, "_"},
		{token.RIGHT_ARROW, "->"},
		{token.IDENT, "False"},
		{token.RBRACE, "}"},
		{token.COMMA, ","},
		{token.IDENT, "Any"},
		{token.COLON, ":"},
		{token.LBRACE, "{"},
		{token.BLANK, "_"},
		{token.RIGHT_ARROW, "->"},
		{token.IDENT, "True"},
		{token.RBRACE, "}"},
		{token.RBRACE, "}"},
		{token.IDENT, "unless"},
		{token.IDENT, "isCorrect"},
		{token.COMMA, ","},
		{token.IDENT, "fail"},
		{token.STRING, "should not be the case"},
		{token.RBRACE, "}"},
		{token.EOF, ""},
	}

	for i, tt := range expect {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestAllTokens(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []struct {
			expectedType    token.TokenType
			expectedLiteral string
		}
	}{
		{
			name:  "spaces",
			input: `    `,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.EOF, ""},
			},
		},
		{
			name:  "tabs",
			input: "\t\t\t\t",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.EOF, ""},
			},
		},
		{
			name:  "newlines",
			input: "\n\n\n\n",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.EOF, ""},
			},
		},
		{
			name:  "general whitespace",
			input: "  \t\n\r\n\t  \n\t",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.EOF, ""},
			},
		},
		{
			name:  "bang",
			input: `!`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.BANG, "!"},
				{token.EOF, ""},
			},
		},
		{
			name:  "neq",
			input: `!=`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.NEQ, "!="},
				{token.EOF, ""},
			},
		},
		{
			name:  "lt",
			input: `<`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.LT, "<"},
				{token.EOF, ""},
			},
		},
		{
			name:  "lte",
			input: `<=`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.LTE, "<="},
				{token.EOF, ""},
			},
		},
		{
			name:  "left arrow",
			input: `<-`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.LEFT_ARROW, "<-"},
				{token.EOF, ""},
			},
		},

		{
			name:  "gt",
			input: `>`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.GT, ">"},
				{token.EOF, ""},
			},
		},
		{
			name:  "gte",
			input: `>=`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.GTE, ">="},
				{token.EOF, ""},
			},
		},
		{
			name:  "plus",
			input: `+`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.PLUS, "+"},
				{token.EOF, ""},
			},
		},
		{
			name:  "minus",
			input: `-`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.MINUS, "-"},
				{token.EOF, ""},
			},
		},
		{
			name:  "asterisk",
			input: `*`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.ASTERISK, "*"},
				{token.EOF, ""},
			},
		},
		{
			name:  "slash",
			input: `/`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.SLASH, "/"},
				{token.EOF, ""},
			},
		},
		{
			name:  "percent",
			input: `%`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.PERCENT, "%"},
				{token.EOF, ""},
			},
		},
		{
			name:  "plus_assign",
			input: `+=`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.PLUS_ASSIGN, "+="},
				{token.EOF, ""},
			},
		},
		{
			name:  "minus_assign",
			input: `-=`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.MINUS_ASSIGN, "-="},
				{token.EOF, ""},
			},
		},
		{
			name:  "star_assign",
			input: `*=`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.STAR_ASSIGN, "*="},
				{token.EOF, ""},
			},
		},
		{
			name:  "slash_assign",
			input: `/=`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.SLASH_ASSIGN, "/="},
				{token.EOF, ""},
			},
		},
		{
			name:  "percent_assign",
			input: `%=`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.PERCENT_ASSIGN, "%="},
				{token.EOF, ""},
			},
		},
		{
			name:  "plus is not confused with plus_assign",
			input: `+ =`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.PLUS, "+"},
				{token.ASSIGN, "="},
				{token.EOF, ""},
			},
		},
		{
			name:  "comma",
			input: `,`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.COMMA, ","},
				{token.EOF, ""},
			},
		},
		{
			name:  "eq",
			input: `==`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.EQ, "=="},
				{token.EOF, ""},
			},
		},
		{
			name:  "assign",
			input: `=`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.ASSIGN, "="},
				{token.EOF, ""},
			},
		},
		{
			name:  "fat arrow",
			input: `->`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.RIGHT_ARROW, "->"},
				{token.EOF, ""},
			},
		},
		{
			name:  "thin arrow",
			input: `->`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.RIGHT_ARROW, "->"},
				{token.EOF, ""},
			},
		},
		{
			name:  "and",
			input: `&&`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.AND, "&&"},
				{token.EOF, ""},
			},
		},
		{
			name:  "or",
			input: `||`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.OR, "||"},
				{token.EOF, ""},
			},
		},
		{
			name:  "colon",
			input: `:`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.COLON, ":"},
				{token.EOF, ""},
			},
		},
		{
			name:  "dot",
			input: `.`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.DOT, "."},
				{token.EOF, ""},
			},
		},
		{
			name:  "comma",
			input: `,`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.COMMA, ","},
				{token.EOF, ""},
			},
		},
		{
			name:  "lparen",
			input: `(`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.LPAREN, "("},
				{token.EOF, ""},
			},
		},
		{
			name:  "rparen",
			input: `)`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.RPAREN, ")"},
				{token.EOF, ""},
			},
		},
		{
			name:  "lbrace",
			input: `{`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.LBRACE, "{"},
				{token.EOF, ""},
			},
		},
		{
			name:  "rbrace",
			input: `}`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.RBRACE, "}"},
				{token.EOF, ""},
			},
		},
		{
			name:  "lbracket",
			input: `[`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.LBRACKET, "["},
				{token.EOF, ""},
			},
		},
		{
			name:  "rbracket",
			input: `]`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.RBRACKET, "]"},
				{token.EOF, ""},
			},
		},
		{
			name:  "at",
			input: `@`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.AT, "@"},
				{token.EOF, ""},
			},
		},
		{
			name:  "hash comment",
			input: `# abc def`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				// {token.COMMENT, " abc def"},
				{token.EOF, ""},
			},
		},
		{
			name:  "line comment",
			input: `// abc def`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				// {token.COMMENT, " abc def"},
				{token.EOF, ""},
			},
		},
		{
			name:  "string",
			input: `"abc"`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.STRING, "abc"},
				{token.EOF, ""},
			},
		},
		{
			name:  "string escaped newline",
			input: "\"\\n\"",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.STRING, "\n"},
				{token.EOF, ""},
			},
		},
		{
			name:  "string escaped tab",
			input: "\"\\t\"",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.STRING, "\t"},
				{token.EOF, ""},
			},
		},
		{
			name:  "string escaped quote",
			input: "\"\\\"\"",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.STRING, "\""},
				{token.EOF, ""},
			},
		},
		{
			name:  "string escaped backslash",
			input: "\"\\\\\"",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.STRING, "\\"},
				{token.EOF, ""},
			},
		},
		{
			name:  "char",
			input: `'a'`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.CHAR, "a"},
				{token.EOF, ""},
			},
		},
		{
			name:  "char escaped newline",
			input: `'\n'`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.CHAR, "\\n"},
				{token.EOF, ""},
			},
		},
		{
			name:  "char escaped tab",
			input: `'\t'`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.CHAR, "\\t"},
				{token.EOF, ""},
			},
		},
		{
			name:  "char escaped quote",
			input: `'\''`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.CHAR, "\\'"},
				{token.EOF, ""},
			},
		},
		{
			name:  "char escaped backslash",
			input: `'\\'`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.CHAR, "\\\\"},
				{token.EOF, ""},
			},
		},
		{
			name:  "EOF",
			input: ``,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.EOF, ""},
			},
		},
		{
			name:  "identifier",
			input: `abc`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "abc"},
				{token.EOF, ""},
			},
		},
		{
			name:  "identifier with underscore",
			input: `abc_def`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "abc_def"},
				{token.EOF, ""},
			},
		},
		{
			name:  "identifier with number",
			input: `abc123`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "abc123"},
				{token.EOF, ""},
			},
		},
		{
			name:  "identifier with number and underscore",
			input: `abc_123`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "abc_123"},
				{token.EOF, ""},
			},
		},
		{
			name:  "identifier with camel case",
			input: `abcDef`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "abcDef"},
				{token.EOF, ""},
			},
		},
		{
			name:  "simple integer",
			input: `42`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.INT, "42"},
				{token.EOF, ""},
			},
		},
		{
			name:  "simple float",
			input: `123.0`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.FLOAT, "123.0"},
				{token.EOF, ""},
			},
		},
		{
			name:  "hexadecimal integer",
			input: `0xFFF`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.INT, "0xFFF"},
				{token.EOF, ""},
			},
		},
		{
			name:  "hexadecimal integer lowercase",
			input: `0x8899aa`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.INT, "0x8899aa"},
				{token.EOF, ""},
			},
		},
		{
			name:  "octal integer",
			input: `0777`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.INT, "0777"},
				{token.EOF, ""},
			},
		},
		{
			name:  "binary integer",
			input: `0b101010`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.INT, "0b101010"},
				{token.EOF, ""},
			},
		},
		{
			name:  "binary integer uppercase",
			input: `0B100011`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.INT, "0B100011"},
				{token.EOF, ""},
			},
		},
		{
			name:  "scientific float positive exponent",
			input: `2e10`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.FLOAT, "2e10"},
				{token.EOF, ""},
			},
		},
		{
			name:  "scientific float negative exponent",
			input: `1.5e-3`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.FLOAT, "1.5e-3"},
				{token.EOF, ""},
			},
		},
		{
			name:  "scientific float uppercase E",
			input: `3.14E+2`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.FLOAT, "3.14E+2"},
				{token.EOF, ""},
			},
		},
		{
			name:  "integer followed by dot",
			input: `123.`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.INT, "123"},
				{token.DOT, "."},
				{token.EOF, ""},
			},
		},
		{
			name:  "keyword mod",
			input: `mod`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.MODULE, "mod"},
				{token.EOF, ""},
			},
		},
		{
			name:  "keyword import",
			input: `import`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IMPORT, "import"},
				{token.EOF, ""},
			},
		},
		{
			name:  "keyword union",
			input: `union`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.UNION, "union"},
				{token.EOF, ""},
			},
		},
		{
			name:  "keyword data",
			input: `data`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.DATA, "data"},
				{token.EOF, ""},
			},
		},
		{
			name:  "keyword extern",
			input: `extern`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.EXTERN, "extern"},
				{token.EOF, ""},
			},
		},
		{
			name:  "keyword fn",
			input: `fn`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.FUNCTION, "fn"},
				{token.EOF, ""},
			},
		},
		{
			name:  "keyword const",
			input: `const`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.CONST, "const"},
				{token.EOF, ""},
			},
		},
		{
			name:  "keyword var",
			input: `var`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.VAR, "var"},
				{token.EOF, ""},
			},
		},
		{
			name:  "keyword type",
			input: `type`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.TYPE, "type"},
				{token.EOF, ""},
			},
		},
		{
			name:  "keyword return",
			input: `return`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.RETURN, "return"},
				{token.EOF, ""},
			},
		},
		{
			name:  "keyword if",
			input: `if`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IF, "if"},
				{token.EOF, ""},
			},
		},
		{
			name:  "keyword else",
			input: `else`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.ELSE, "else"},
				{token.EOF, ""},
			},
		},
		{
			name:  "keyword for",
			input: `for`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.FOR, "for"},
				{token.EOF, ""},
			},
		},
		{
			name:  "illegal and",
			input: `&`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.ILLEGAL, "&"},
				{token.EOF, ""},
			},
		},
		{
			name:  "illegal or",
			input: `|`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.ILLEGAL, "|"},
				{token.EOF, ""},
			},
		},
		{
			name:  "emoji",
			input: `🦜`,
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.ILLEGAL, string("🦜"[0])},
				{token.ILLEGAL, string("🦜"[1])},
				{token.ILLEGAL, string("🦜"[2])},
				{token.ILLEGAL, string("🦜"[3])},
				{token.EOF, ""},
			},
		},
	}

	for _, tt := range testCases {
		l, err := lexer.New(staticmodule.NewSourceString("testing:///test/test.zirr", tt.input))
		if err != nil {
			t.Fatal(err)
		}

		for i, expect := range tt.expected {
			tok := l.NextToken()

			if tok.Type != expect.expectedType && tok.Literal != expect.expectedLiteral {
				t.Errorf("%s[%d] - wrong token and literal. expected=%v, got=%v",
					tt.name, i, expect, tok)
				break
			}

			if tok.Type != expect.expectedType {
				t.Errorf("%s[%d] - tokentype wrong. expected=%q, got=%q",
					tt.name, i, expect.expectedType, tok.Type)
				break
			}

			if tok.Literal != expect.expectedLiteral {
				t.Errorf("%s[%d] - literal wrong. expected=%q, got=%q",
					tt.name, i, expect.expectedType, tok.Literal)
				break
			}
		}
	}
}

func TestDecorativeLexer(t *testing.T) {
	type deco struct {
		decorativeType token.DecorativeTokenType
		literal        string
	}
	type tok struct {
		tokenType          token.TokenType
		literal            string
		leadingDecoratives []deco
	}
	testCases := []struct {
		name  string
		input string
		want  []tok
	}{
		{
			"empty",
			"",
			[]tok{{token.EOF, "", nil}},
		},
		{
			"empty with trailing ws",
			"  \t\n ",
			[]tok{
				{
					token.EOF, "", []deco{
						{token.DECO_MULTI, "  \t\n "},
					},
				},
			},
		},
		{
			"empty with trailing ws and comment",
			"\t\n// hello\n\t",
			[]tok{
				{
					token.EOF, "", []deco{
						{token.DECO_MULTI, "\t\n"},
						{token.DECO_COMMENT, "hello"},
						{token.DECO_INLINE, "\t"},
					},
				},
			},
		},
		{
			"empty shebang",
			"#!/usr/bin/env zirric",
			[]tok{
				{token.EOF, "", []deco{
					{token.DECO_COMMENT, "!/usr/bin/env zirric"},
				}},
			},
		},
		{
			"comment before data",
			"// cool stuff\ndata",
			[]tok{
				{
					token.DATA, "data", []deco{
						{token.DECO_COMMENT, "cool stuff"},
					},
				},
			},
		},
		{
			"comment after data",
			"data // cool stuff",
			[]tok{
				{token.DATA, "data", nil},
				{token.EOF, "", []deco{
					{token.DECO_INLINE, " "},
					{token.DECO_COMMENT, "cool stuff"},
				}},
			},
		},
		{
			"comment after data with trailing ws",
			"data // cool stuff\n\t",
			[]tok{
				{token.DATA, "data", nil},
				{token.EOF, "", []deco{
					{token.DECO_INLINE, " "},
					{token.DECO_COMMENT, "cool stuff"},
					{token.DECO_INLINE, "\t"},
				}},
			},
		},
		{
			"comment after data with trailing ws and comment",
			"data // cool stuff\n\n\t// hello",
			[]tok{
				{token.DATA, "data", nil},
				{token.EOF, "", []deco{
					{token.DECO_INLINE, " "},
					{token.DECO_COMMENT, "cool stuff"},
					{token.DECO_MULTI, "\n\t"},
					{token.DECO_COMMENT, "hello"},
				}},
			},
		},
		{
			"comment after data with trailing ws and comment and trailing ws",
			"data // cool stuff\n\n\t// hello\n\t",
			[]tok{
				{token.DATA, "data", nil},
				{token.EOF, "", []deco{
					{token.DECO_INLINE, " "},
					{token.DECO_COMMENT, "cool stuff"},
					{token.DECO_MULTI, "\n\t"},
					{token.DECO_COMMENT, "hello"},
					{token.DECO_INLINE, "\t"},
				}},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			l, err := lexer.New(staticmodule.NewSourceString("testing:///decorative/test.zirr", tt.input))
			if err != nil {
				t.Fatal(err)
			}

			for i, want := range tt.want {
				t.Run(fmt.Sprintf("[%d] %s", i, want.tokenType), func(t *testing.T) {
					got := l.NextToken()
					if got.Type != want.tokenType {
						t.Errorf("want token type %s, got %s", want.tokenType, got.Type)
					}
					if got.Literal != want.literal {
						t.Errorf("want literal %q, got %q", want.literal, got.Literal)
					}
					if len(got.Leading) != len(want.leadingDecoratives) {
						t.Errorf(
							"want leading tokens %d, got %d: (\n\twant %q\n\n\tgot %q\n)",
							len(want.leadingDecoratives),
							len(got.Leading),
							want.leadingDecoratives,
							got.Leading,
						)
					}

					for j, wantdeco := range want.leadingDecoratives {
						t.Run(fmt.Sprintf("[%d] %s", j, wantdeco.decorativeType), func(t *testing.T) {
							gotdeco := got.Leading[j]

							if gotdeco.Type != wantdeco.decorativeType {
								t.Errorf("want token type %s, got %s", wantdeco.decorativeType, gotdeco.Type)
							}
							if gotdeco.Literal != wantdeco.literal {
								t.Errorf("want literal %q, got %q", wantdeco.literal, gotdeco.Literal)
							}
						})
					}
				})
			}
		})
	}
}
