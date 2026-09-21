package analyzer_test

import (
	"errors"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

func TestAnalysisErrorsReachTheIndividualErrors(t *testing.T) {
	errs := analyzer.AnalysisErrors{
		{Token: token.Token{Source: token.MakeSource("main.zirr", 0, 2, 3)}, Summary: "first"},
		{Token: token.Token{Source: token.MakeSource("main.zirr", 0, 5, 1)}, Summary: "second"},
	}

	// Analysis finds everything in one pass, so a caller must be able to reach all of it rather than only the first.
	var one analyzer.AnalysisError
	if !errors.As(error(errs), &one) {
		t.Fatal("expected errors.As to reach an AnalysisError")
	}
	if one.Summary != "first" {
		t.Errorf("expected the first error, got %q", one.Summary)
	}
	if got := len(errs.Unwrap()); got != 2 {
		t.Errorf("expected 2 held errors, got %d", got)
	}
	message := errs.Error()
	if !strings.Contains(message, "first") || !strings.Contains(message, "second") {
		t.Errorf("expected both errors in the message, got %q", message)
	}
}

func TestAnalysisErrorPositionIsALineAndColumn(t *testing.T) {
	err := analyzer.AnalysisError{Token: token.Token{Source: token.MakeSource("main.zirr", 42, 3, 7)}, Summary: "boom"}
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

func TestAnalysisErrorsKeepWarningsAlongsideErrors(t *testing.T) {
	// Analysis reports both in one slice, and reporting all of them must not lose the severity that tells them apart.
	errs := analyzer.AnalysisErrors{
		{Token: token.Token{Source: token.MakeSource("main.zirr", 0, 1, 1)}, Summary: "dependency not installed", Severity: analyzer.AnalysisSeverityWarning},
		{Token: token.Token{Source: token.MakeSource("main.zirr", 0, 4, 2)}, Summary: "unknown reference"},
	}

	message := errs.Error()
	if !strings.Contains(message, "dependency not installed") || !strings.Contains(message, "unknown reference") {
		t.Errorf("expected both diagnostics in the message, got %q", message)
	}

	held := errs.Unwrap()
	if len(held) != 2 {
		t.Fatalf("expected 2 held diagnostics, got %d", len(held))
	}
	first, ok := held[0].(analyzer.AnalysisError)
	if !ok {
		t.Fatalf("expected an AnalysisError, got %T", held[0])
	}
	if first.Severity != analyzer.AnalysisSeverityWarning {
		t.Error("the warning lost its severity when held in a collection")
	}
}
