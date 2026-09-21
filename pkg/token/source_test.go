package token_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

func TestMakeSource(t *testing.T) {
	src := token.MakeSource("foo", 42, 3, 7)
	if src.File != "foo" {
		t.Errorf("expected %q, got %q", "foo", src.File)
	}
	if src.Offset != 42 {
		t.Errorf("expected %d, got %d", 42, src.Offset)
	}
	if src.Line != 3 || src.Column != 7 {
		t.Errorf("expected 3:7, got %d:%d", src.Line, src.Column)
	}
}

func TestSourceStringReadsAsAPosition(t *testing.T) {
	if got := token.MakeSource("main.zirr", 42, 3, 7).String(); got != "main.zirr:3:7" {
		t.Errorf("expected %q, got %q", "main.zirr:3:7", got)
	}
	// A source with no line recorded must not print a zero that reads like a real position.
	if got := (&token.Source{File: "main.zirr", Offset: 42}).String(); got != "main.zirr" {
		t.Errorf("expected %q, got %q", "main.zirr", got)
	}
	var missing *token.Source
	if got := missing.String(); got != "" {
		t.Errorf("expected an empty string, got %q", got)
	}
}
