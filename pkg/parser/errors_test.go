package parser_test

import (
	"errors"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

func TestParseErrorsReachTheIndividualErrors(t *testing.T) {
	errs := parser.ParseErrors{
		{Token: token.Token{Source: token.MakeSource("main.zirr", 0, 2, 3)}, Summary: "first"},
		{Token: token.Token{Source: token.MakeSource("main.zirr", 0, 5, 1)}, Summary: "second"},
	}

	// Holding several errors the way Go expects is what lets a caller reach one without knowing the collection type.
	var one parser.ParseError
	if !errors.As(error(errs), &one) {
		t.Fatal("expected errors.As to reach a ParseError")
	}
	if one.Summary != "first" {
		t.Errorf("expected the first error, got %q", one.Summary)
	}
	if got := len(errs.Unwrap()); got != 2 {
		t.Errorf("expected 2 held errors, got %d", got)
	}
}

func TestParseErrorPositionIsALineAndColumn(t *testing.T) {
	err := parser.ParseError{Token: token.Token{Source: token.MakeSource("main.zirr", 42, 3, 7)}, Summary: "boom"}
	source := err.Position()
	if source == nil {
		t.Fatal("expected a position")
	}
	if source.Line != 3 || source.Column != 7 {
		t.Errorf("expected 3:7, got %d:%d", source.Line, source.Column)
	}
	if err.Error() != "main.zirr:3:7: boom" {
		t.Errorf("unexpected message: %q", err.Error())
	}
}
