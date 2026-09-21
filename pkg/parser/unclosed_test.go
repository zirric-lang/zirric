package parser_test

import (
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
)

// parseErrors parses input and returns the errors it produced.
func parseErrors(t *testing.T, input string) []parser.ParseError {
	t.Helper()
	module := ast.MakeContextModule(registry.LogicalURI("test"))
	l, err := lexer.New(staticmodule.NewSourceString("testing:///test.zirr", input))
	if err != nil {
		t.Fatal(err)
	}
	p := parser.NewSourceParser(l, module.Decls, "test.zirr")
	p.ParseSourceFile()
	return p.Errors()
}

func TestAnUnclosedGroupIsReportedWhereItOpened(t *testing.T) {
	// "unexpected }" asks a reader to find the matching bracket themselves, which is the whole difficulty.
	errs := parseErrors(t, "fn f() {\n\tconst y = (1\n}")
	var unclosed *parser.ParseError
	for i := range errs {
		if strings.HasPrefix(errs[i].Summary, "unclosed") {
			unclosed = &errs[i]
			break
		}
	}
	if unclosed == nil {
		t.Fatalf("expected an unclosed error, got %v", errs)
	}
	// The position is the opening paren on line 2, not the brace that gave it away.
	if unclosed.Token.Source == nil || unclosed.Token.Source.Line != 2 {
		t.Errorf("expected the error at the opening paren on line 2, got %v", unclosed.Token.Source)
	}
	if !strings.Contains(unclosed.Details, "expected )") {
		t.Errorf("unexpected details: %s", unclosed.Details)
	}
	// It still says where the trouble showed up.
	if !strings.Contains(unclosed.Details, "3:1") {
		t.Errorf("expected the details to name where it was noticed: %s", unclosed.Details)
	}
}

func TestEveryUnclosedGroupIsReportedAtItsOwnPosition(t *testing.T) {
	// Four unclosed parentheses are four mistakes, and each is at a column of its own.
	errs := parseErrors(t, "fn f() {\n\tconst y = ((((\n}")
	columns := map[int]bool{}
	for _, err := range errs {
		if strings.HasPrefix(err.Summary, "unclosed") && err.Token.Source != nil {
			columns[err.Token.Source.Column] = true
		}
	}
	if len(columns) != 4 {
		t.Errorf("expected four distinct positions, got %d: %v", len(columns), columns)
	}
}

func TestAMissingExpressionNamesWhatWasWanted(t *testing.T) {
	// Listing every token that could have started an expression is a list nobody reads.
	errs := parseErrors(t, "fn f() {\n\tconst y = (1\n}")
	for _, err := range errs {
		if strings.Contains(err.Details, "want one of [") && strings.Count(err.Details, ",") > 4 {
			t.Errorf("a long token list is still reported: %s", err.Error())
		}
	}
}

func TestAnUnclosedIndexIsReportedWhereItOpened(t *testing.T) {
	errs := parseErrors(t, "fn f(xs) {\n\treturn xs[0\n}")
	for _, err := range errs {
		if strings.HasPrefix(err.Summary, "unclosed") {
			if err.Token.Source != nil && err.Token.Source.Line == 2 {
				return
			}
		}
	}
	t.Errorf("expected an unclosed index error on line 2, got %v", errs)
}
