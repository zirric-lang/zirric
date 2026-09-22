package langsrv

import (
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestImportModuleNameCompletion(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "Cavefile", `mod proj

import cave

@cave.Package()
data Dependencies {
  @cave.Local("../helper-package")
  helper
}
`)
	writeFile(t, base, "main.zirr", "mod main\nimport \n")
	writeFile(t, base, "utils/greet.zirr", "mod utils\nconst greeting = \"hi\"\n")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 1, Character: 7})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labels := make(map[string]bool, len(items))
	for _, item := range items {
		labels[item.Label] = true
	}

	for _, want := range []string{"io", "fmt", "os", "future", "cave", "tasks", "prelude", "utils", "helper"} {
		if !labels[want] {
			t.Errorf("expected import completion to include %q, got %v", want, labels)
		}
	}
}

func TestImportModuleNameCompletionNestedPrefix(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "mod main\nimport future.\n")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 1, Character: 14})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	var found *protocol.CompletionItem
	for i := range items {
		if items[i].Label == "future.reflect" {
			found = &items[i]
		}
	}
	if found == nil {
		t.Fatalf("expected completion item 'future.reflect', got %v", labelsOf(items))
	}
	if found.TextEdit == nil {
		t.Fatal("expected a TextEdit on the completion item")
	}
	edit, ok := found.TextEdit.(protocol.TextEdit)
	if !ok {
		t.Fatalf("expected protocol.TextEdit, got %T", found.TextEdit)
	}
	if edit.NewText != "future.reflect" {
		t.Errorf("expected NewText 'future.reflect', got %q", edit.NewText)
	}
	// Must replace everything typed since "import ", not just insert at the cursor, or selecting it would leave "future.future.reflect".
	if edit.Range.Start.Character != 7 {
		t.Errorf("expected edit range to start right after 'import ' (col 7), got %d", edit.Range.Start.Character)
	}
}

func labelsOf(items []protocol.CompletionItem) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.Label
	}
	return out
}
