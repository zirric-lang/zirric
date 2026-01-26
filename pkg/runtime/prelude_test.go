package runtime

import "testing"

func TestPreludeBool(t *testing.T) {
	var p Prelude
	b := p.Bool(true)
	if b.Inspect() != "true" {
		t.Fatalf("expected Inspect true, got %s", b.Inspect())
	}
	if b.TypeConstantId() != typeIdBool {
		t.Fatalf("expected typeIdBool, got %d", b.TypeConstantId())
	}
}
