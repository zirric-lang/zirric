package langsrv

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestNormalizeContentChanges(t *testing.T) {
	changes, err := normalizeContentChanges([]any{
		protocol.TextDocumentContentChangeEvent{Text: "full"},
		protocol.TextDocumentContentChangeEventWhole{Text: "whole"},
	})
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}
	if changes[0].Text != "full" || changes[0].Range != nil {
		t.Fatalf("unexpected first change: %+v", changes[0])
	}
	if changes[1].Text != "whole" || changes[1].Range != nil {
		t.Fatalf("unexpected second change: %+v", changes[1])
	}
}
