package langsrv

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestDocumentStoreLifecycle(t *testing.T) {
	store := newDocumentStore()
	store.Open("module/main.zirr", 1, "hello world")

	snapshot, ok := store.Snapshot("module/main.zirr")
	if !ok {
		t.Fatalf("expected snapshot to exist")
	}
	if snapshot.Text != "hello world" {
		t.Fatalf("unexpected text: %q", snapshot.Text)
	}
	if !snapshot.Dirty {
		t.Fatalf("expected document to be dirty after open")
	}

	change := protocol.TextDocumentContentChangeEvent{
		Range: &protocol.Range{
			Start: protocol.Position{Line: 0, Character: 6},
			End:   protocol.Position{Line: 0, Character: 11},
		},
		Text: "zirr",
	}
	if err := store.ApplyChanges("module/main.zirr", 2, []protocol.TextDocumentContentChangeEvent{change}); err != nil {
		t.Fatalf("apply changes failed: %v", err)
	}

	snapshot, ok = store.Snapshot("module/main.zirr")
	if !ok {
		t.Fatalf("expected snapshot to exist after change")
	}
	if snapshot.Text != "hello zirr" {
		t.Fatalf("unexpected updated text: %q", snapshot.Text)
	}
	if snapshot.Version != 2 {
		t.Fatalf("expected version 2, got %d", snapshot.Version)
	}

	store.Save("module/main.zirr", nil)
	snapshot, ok = store.Snapshot("module/main.zirr")
	if !ok {
		t.Fatalf("expected snapshot to exist after save")
	}
	if snapshot.Dirty {
		t.Fatalf("expected document to be clean after save")
	}

	store.Close("module/main.zirr")
	if _, ok := store.Snapshot("module/main.zirr"); ok {
		t.Fatalf("expected document to be removed after close")
	}
}

func TestApplyContentChangesWhole(t *testing.T) {
	updated, err := applyContentChanges("old", []protocol.TextDocumentContentChangeEvent{{Text: "new"}})
	if err != nil {
		t.Fatalf("apply changes failed: %v", err)
	}
	if updated != "new" {
		t.Fatalf("unexpected updated text: %q", updated)
	}
}
