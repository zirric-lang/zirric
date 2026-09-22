package parser_test

import (
	"testing"
	"time"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
)

// An unterminated body must end the parse rather than spin at EOF, a state the editor sees on nearly every keystroke while a block is typed.
func TestParseUnterminatedBodiesTerminate(t *testing.T) {
	inputs := []string{
		"data B {",
		"data B { a",
		"data B { a: Int",
		"data B { m()",
		"union U {",
		"union U { data D {",
		"attr A {",
		"extern type T {",
		"import m {",
		"import m { a",
		"fn f() {",
		"switch x {",
		"@A()\ndata B {",
		"@A(\ndata B {",
		"mod m\n\n@cave.Package()\ndata Deps {\n\t@cave.Stdlib(\"prelude\")\n\tprelude",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			done := make(chan struct{})
			go func() {
				defer close(done)
				module := ast.MakeContextModule(registry.LogicalURI("test"))
				l, err := lexer.New(staticmodule.NewSourceString("testing:///t.zirr", input))
				if err != nil {
					return
				}
				p := parser.NewSourceParser(l, module.Decls, "t.zirr")
				p.ParseSourceFile()
				if len(p.Errors()) == 0 {
					t.Errorf("expected parse errors for unterminated input")
				}
			}()

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("parser did not terminate on unterminated input")
			}
		})
	}
}
