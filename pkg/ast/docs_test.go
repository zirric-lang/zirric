package ast_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// docsOfLastToken lexes src and returns the documentation of its last real token, which is where a doc comment written above a declaration ends up.
func docsOfLastToken(t *testing.T, src string) string {
	t.Helper()
	lex, err := lexer.New(staticmodule.NewSourceString("testing:///docs/main.zirr", src))
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	var last token.Token
	for {
		tok := lex.NextToken()
		if tok.Type == token.EOF {
			break
		}
		last = tok
	}
	return ast.MakeDocs(ast.LeadingDocComments(last)).Content
}

func TestLeadingDocComments(t *testing.T) {
	tests := []struct {
		label string
		src   string
		want  string
	}{
		{"one line", "// What it is.\nfn", "What it is."},
		{"several lines", "// First.\n// Second.\nfn", "First.\nSecond."},
		{"an empty comment line is kept between paragraphs", "// First.\n//\n// Third.\nfn", "First.\n\nThird."},
		{"indented", "data X {\n\t// A field.\n\tname\n", "A field."},
		{"hash marker", "# What it is.\nfn", "What it is."},
		{"no space after the marker", "//What it is.\nfn", "What it is."},
		{"a blank line ends the block", "// Not this.\n\nfn", ""},
		{"a comment after code documents its own line", "const x = 1 // Not this.\nfn", ""},
		{"a trailing comment ends the block above it", "const x = 1 // Not this.\n// This.\nfn", "This."},
		{"a shebang is not documentation", "#!/usr/bin/env zirric\nfn", ""},
		{"a shebang above a comment is left out of it", "#!/usr/bin/env zirric\n// This.\nfn", "This."},
		{"no comment at all", "fn", ""},
		{"three slashes are not a doc marker of their own", "/// What it is.\nfn", "/ What it is."},
	}
	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := docsOfLastToken(t, tt.src); got != tt.want {
				t.Errorf("docs = %q, want %q", got, tt.want)
			}
		})
	}
}
