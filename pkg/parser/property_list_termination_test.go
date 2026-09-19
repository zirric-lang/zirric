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

// TestParsePropertyDeclarationListTerminatesOnMalformedField is a regression test for an infinite loop: Parser.expect() doesn't advance on a mismatch, so a malformed field used to make parsePropertyDeclarationList call parseDataDeclField on the same token forever.
func TestParsePropertyDeclarationListTerminatesOnMalformedField(t *testing.T) {
	inputs := []string{
		"data Foo { , }",
		"attr Cmd { , }",
		"data Foo { 123 }",
		"data Foo { , , , }",
	}
	for _, input := range inputs {
		input := input
		t.Run(input, func(t *testing.T) {
			done := make(chan int)
			go func() {
				module := ast.MakeContextModule(registry.LogicalURI("test"))
				l, err := lexer.New(staticmodule.NewSourceString("testing:///test.zirr", input))
				if err != nil {
					t.Error(err)
					done <- 0
					return
				}
				p := parser.NewSourceParser(l, module.Decls, "test.zirr")
				p.ParseSourceFile()
				done <- len(p.Errors())
			}()
			select {
			case errCount := <-done:
				if errCount == 0 {
					t.Errorf("expected parse errors for malformed input %q, got none", input)
				}
			case <-time.After(3 * time.Second):
				t.Fatalf("parsing %q did not terminate within 3s (infinite loop)", input)
			}
		})
	}
}
