package lexer_test

import (
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

func lexAll(t *testing.T, input string) []token.Token {
	t.Helper()
	l, err := lexer.New(staticmodule.NewSourceString("testing:///trivia/test.zirr", input))
	if err != nil {
		t.Fatal(err)
	}
	var toks []token.Token
	for {
		tok := l.NextToken()
		toks = append(toks, tok)
		if tok.Type == token.EOF {
			return toks
		}
	}
}

// findToken returns the first token of the given type.
func findToken(t *testing.T, toks []token.Token, typ token.TokenType) token.Token {
	t.Helper()
	for _, tok := range toks {
		if tok.Type == typ {
			return tok
		}
	}
	t.Fatalf("no %s token in stream", typ)
	return token.Token{}
}

// Every token must carry the trivia before it; delimiters and operators once dropped it, silently deleting comments.
func TestLeadingTriviaSurvivesOnDelimiters(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		typ   token.TokenType
	}{
		{"before RBRACE", "data D {\n\t// trailing note\n}", token.RBRACE},
		{"before LBRACE", "data D\n\t// note\n{}", token.LBRACE},
		{"before RPAREN", "f(a\n\t// note\n)", token.RPAREN},
		{"before COMMA", "f(a\n\t// note\n, b)", token.COMMA},
		{"before AT", "// note\n@Attr()", token.AT},
		{"before RBRACKET", "[1\n\t// note\n]", token.RBRACKET},
		{"before COLON", "x\n\t// note\n: Int", token.COLON},
		{"before DOT", "x\n\t// note\n.y", token.DOT},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tok := findToken(t, lexAll(t, tt.input), tt.typ)
			if !hasComment(tok.Leading) {
				t.Errorf("token %s lost its leading comment: Leading = %q", tt.typ, tok.Leading)
			}
		})
	}
}

func TestLeadingTriviaSurvivesOnTwoCharOperators(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		typ   token.TokenType
	}{
		{"NEQ", "a\n// note\n!= b", token.NEQ},
		{"EQ", "a\n// note\n== b", token.EQ},
		{"LTE", "a\n// note\n<= b", token.LTE},
		{"GTE", "a\n// note\n>= b", token.GTE},
		{"AND", "a\n// note\n&& b", token.AND},
		{"OR", "a\n// note\n|| b", token.OR},
		{"RIGHT_ARROW", "fn f()\n// note\n-> Int", token.RIGHT_ARROW},
		{"PLUS_ASSIGN", "a\n// note\n+= b", token.PLUS_ASSIGN},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tok := findToken(t, lexAll(t, tt.input), tt.typ)
			if !hasComment(tok.Leading) {
				t.Errorf("token %s lost its leading comment: Leading = %q", tt.typ, tok.Leading)
			}
		})
	}
}

// Two-character operators once carried no Source at all, so every offset-based consumer silently skipped them.
func TestTwoCharOperatorsCarrySource(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		typ   token.TokenType
		want  int
	}{
		{"NEQ", "a != b", token.NEQ, 2},
		{"EQ", "a == b", token.EQ, 2},
		{"LTE", "a <= b", token.LTE, 2},
		{"GTE", "a >= b", token.GTE, 2},
		{"AND", "a && b", token.AND, 2},
		{"OR", "a || b", token.OR, 2},
		{"RIGHT_ARROW", "a -> b", token.RIGHT_ARROW, 2},
		{"LEFT_ARROW", "a <- b", token.LEFT_ARROW, 2},
		{"PLUS_ASSIGN", "a += b", token.PLUS_ASSIGN, 2},
		{"MINUS_ASSIGN", "a -= b", token.MINUS_ASSIGN, 2},
		{"STAR_ASSIGN", "a *= b", token.STAR_ASSIGN, 2},
		{"SLASH_ASSIGN", "a /= b", token.SLASH_ASSIGN, 2},
		{"PERCENT_ASSIGN", "a %= b", token.PERCENT_ASSIGN, 2},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tok := findToken(t, lexAll(t, tt.input), tt.typ)
			if tok.Source == nil {
				t.Fatalf("token %s has no Source", tt.typ)
			}
			if tok.Source.Offset != tt.want {
				t.Errorf("token %s: want offset %d, got %d", tt.typ, tt.want, tok.Source.Offset)
			}
		})
	}
}

// Comment literals must be verbatim, marker included, or "//" and "#" become indistinguishable and shebangs are destroyed.
func TestCommentLiteralsAreVerbatim(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{"line comment", "// hello\nx", "// hello"},
		{"line comment no space", "//hello\nx", "//hello"},
		{"doc comment", "/// docs\nx", "/// docs"},
		{"hash comment", "# hello\nx", "# hello"},
		{"shebang", "#!/usr/bin/env zirric\nx", "#!/usr/bin/env zirric"},
		{"comment at EOF without newline", "x\n// end", "// end"},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			for _, tok := range lexAll(t, tt.input) {
				for _, d := range tok.Leading {
					if d.Type == token.DECO_COMMENT {
						got = d.Literal
					}
				}
			}
			if got != tt.want {
				t.Errorf("want comment literal %q, got %q", tt.want, got)
			}
		})
	}
}

// Trivia must account for every byte between tokens, which is what lets the formatter slice tokens verbatim.
func TestTriviaReconstructsSource(t *testing.T) {
	inputs := []string{
		"mod prelude\n\n// a comment\nfn f() {\n\tconst x = 1\n}\n",
		"data D {\n\ta: Int // trailing\n\t// own line\n\tb: String\n}\n",
		"#!/usr/bin/env zirric\nmod main\n",
		"a != b && c <= d\n",
		"switch v {\ncase is String:\n\tx\ncase _:\n\ty\n}\n",
		"",
		"// only a comment",
		"f(a\n\t// note\n, b)\n",
	}

	for _, input := range inputs {
		t.Run(strings.SplitN(input, "\n", 2)[0], func(t *testing.T) {
			var sb strings.Builder
			for _, tok := range lexAll(t, input) {
				for _, d := range tok.Leading {
					sb.WriteString(d.Literal)
				}
				if tok.Type == token.EOF {
					continue
				}
				sb.WriteString(rawLiteral(tok))
			}
			if sb.String() != input {
				t.Errorf("source did not reconstruct:\n want %q\n  got %q", input, sb.String())
			}
		})
	}
}

func hasComment(decos []token.DecorativeToken) bool {
	for _, d := range decos {
		if d.Type == token.DECO_COMMENT {
			return true
		}
	}
	return false
}

// rawLiteral re-spells a token as it appeared, adding back the quotes that string and char tokens drop.
func rawLiteral(tok token.Token) string {
	switch tok.Type {
	case token.STRING:
		return `"` + tok.Literal + `"`
	case token.CHAR:
		return "'" + tok.Literal + "'"
	default:
		return tok.Literal
	}
}
