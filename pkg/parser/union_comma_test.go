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

// parseUnionNamed parses input and returns the union declared under name, which lands in the module's table rather than on the file.
func parseUnionNamed(t *testing.T, input string, name string) *ast.DeclUnion {
	t.Helper()
	module := ast.MakeContextModule(registry.LogicalURI("test"))
	l, err := lexer.New(staticmodule.NewSourceString("testing:///test.zirr", input))
	if err != nil {
		t.Fatal(err)
	}
	p := parser.NewSourceParser(l, module.Decls, "test.zirr")
	p.ParseSourceFile()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors for %q: %v", input, p.Errors())
	}
	sym, ok := module.Decls.Symbols[name]
	if !ok {
		t.Fatalf("no declaration %q parsed from %q", name, input)
	}
	union, ok := sym.Decl.(*ast.DeclUnion)
	if !ok {
		t.Fatalf("%q is a %T, not a union", name, sym.Decl)
	}
	return union
}

func TestUnionMembersMayBeSeparatedByCommas(t *testing.T) {
	for _, input := range []string{
		"data A { x }\ndata B { y }\nunion U { A, B }",
		"data A { x }\ndata B { y }\nunion U { A, B, }",
		"data A { x }\ndata B { y }\nunion U {\nA\nB\n}",
		"data A { x }\ndata B { y }\nunion U {\nA,\nB,\n}",
	} {
		union := parseUnionNamed(t, input, "U")
		if len(union.Members) != 2 {
			t.Errorf("expected 2 members from %q, got %d", input, len(union.Members))
		}
	}
}

func TestInlineUnionMembersMayBeSeparatedByCommas(t *testing.T) {
	union := parseUnionNamed(t, "union Shape {\ndata Circle { r },\ndata Square { s },\n}", "Shape")
	if len(union.Members) != 2 {
		t.Errorf("expected 2 inline members, got %d", len(union.Members))
	}
}

func TestAUnionMemberThatCannotBeParsedDoesNotHang(t *testing.T) {
	// Reporting an error without consuming anything would offer the same token to the same parse forever.
	done := make(chan struct{})
	go func() {
		defer close(done)
		l, err := lexer.New(staticmodule.NewSourceString("testing:///test/test.zirr", "union U { 123 }"))
		if err != nil {
			return
		}
		module := ast.MakeContextModule(registry.LogicalURI("test"))
		parser.NewSourceParser(l, module.Decls, "test.zirr").ParseSourceFile()
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("parsing a malformed union member did not finish")
	}
}
