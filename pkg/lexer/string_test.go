package lexer_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// TestStringLiteralSpansItsInterpolations checks that the quote ending a literal is the one outside every `\( … )`, which is what a nested literal depends on.
func TestStringLiteralSpansItsInterpolations(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		literal string
	}{
		{"plain", `"hello"`, `hello`},
		{"interpolation", `"n = \(n)"`, `n = \(n)`},
		{"nested literal", `"a \(f("b"))c"`, `a \(f("b"))c`},
		{"nested interpolation", `"\(f("\(g("x"))"))"`, `\(f("\(g("x"))"))`},
		{"escaped backslash", `"\\(literal)"`, `\\(literal)`},
		{"char literal inside", `"\(c == '"')"`, `\(c == '"')`},
		{"unterminated", `"a \(b`, `a \(b`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l, err := lexer.New(staticmodule.NewSourceString("testing:///t.zirr", tc.input))
			if err != nil {
				t.Fatal(err)
			}
			tok := l.NextToken()
			if tok.Type != token.STRING {
				t.Fatalf("token type = %s, want STRING", tok.Type)
			}
			if tok.Literal != tc.literal {
				t.Errorf("literal = %q, want %q", tok.Literal, tc.literal)
			}
		})
	}
}

func TestSplitString(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []lexer.StringSegment
	}{
		{"no interpolation", `plain text`, []lexer.StringSegment{{Text: `plain text`}}},
		{"only interpolation", `\(x)`, []lexer.StringSegment{{Expr: `x`, Offset: 2, Interpolation: true}}},
		{
			"text around",
			`a \(x) b`,
			[]lexer.StringSegment{
				{Text: `a `},
				{Expr: `x`, Offset: 4, Interpolation: true},
				{Text: ` b`},
			},
		},
		{
			"escaped backslash is not one",
			`\\(x)`,
			[]lexer.StringSegment{{Text: `\\(x)`}},
		},
		{
			"unclosed",
			`a \(x`,
			[]lexer.StringSegment{
				{Text: `a `},
				{Expr: `x`, Offset: 4, Interpolation: true, Unclosed: true},
			},
		},
		{
			"a line break ends it",
			"a \\(x\ny)",
			[]lexer.StringSegment{
				{Text: `a `},
				{Expr: `x`, Offset: 4, Interpolation: true, Unclosed: true},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := lexer.SplitString(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d segments (%+v), want %d", len(got), got, len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("segment %d = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestSplitLiteralRefusesWhatItCannotAccountFor(t *testing.T) {
	for _, input := range []string{`"a \(b"`, `"unterminated`, `x`, `"`} {
		if _, ok := lexer.SplitLiteral(input); ok {
			t.Errorf("SplitLiteral(%q) reported a whole literal", input)
		}
	}
	if _, ok := lexer.SplitLiteral(`"a \(b)"`); !ok {
		t.Error("SplitLiteral refused a closed literal")
	}
}
