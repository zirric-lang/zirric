package token_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/token"
)

func TestMakeSource(t *testing.T) {
	src := token.MakeSource("foo", 42)
	if src.File != "foo" {
		t.Errorf("expected %q, got %q", "foo", src.File)
	}
	if src.Offset != 42 {
		t.Errorf("expected %d, got %d", 42, src.Offset)
	}
}
