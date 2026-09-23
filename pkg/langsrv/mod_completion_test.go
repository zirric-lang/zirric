package langsrv

import (
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func modCompletionServer(t *testing.T) (*zirricLangserver, billy.Filesystem) {
	t.Helper()
	base := memfs.New()
	writeFile(t, base, "Cavefile", "mod proj\n\nimport cave\n\n@cave.Package()\ndata Package {}\n")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	return ls, base
}

// The name a file's `mod` has to carry is already known — the Cavefile fixes the base and the directory names the rest — so it is offered rather than left to be typed out.
func TestModModuleNameCompletion(t *testing.T) {
	ls, base := modCompletionServer(t)
	writeFile(t, base, "utils/greet.zirr", "mod \n")
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("utils/greet.zirr", protocol.Position{Line: 0, Character: 4})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}
	if len(items) != 1 || items[0].Label != "proj.utils" {
		t.Fatalf("expected the one candidate proj.utils, got %v", labelsOf(items))
	}
	if items[0].TextEdit == nil {
		t.Fatal("expected a replacing TextEdit")
	}
}

// A file at the package root belongs to the base itself.
func TestModModuleNameCompletionAtProjectRoot(t *testing.T) {
	ls, base := modCompletionServer(t)
	writeFile(t, base, "main.zirr", "mod \n")
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 0, Character: 4})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}
	if len(items) != 1 || items[0].Label != "proj" {
		t.Fatalf("expected the one candidate proj, got %v", labelsOf(items))
	}
}

// Completing a partially typed name replaces all of it, rather than appending to it.
func TestModModuleNameCompletionReplacesWhatWasTyped(t *testing.T) {
	ls, base := modCompletionServer(t)
	writeFile(t, base, "utils/greet.zirr", "mod utils\n")
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("utils/greet.zirr", protocol.Position{Line: 0, Character: 9})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one candidate, got %v", labelsOf(items))
	}
	edit := items[0].TextEdit.(protocol.TextEdit)
	if edit.Range.Start.Character != 4 || edit.NewText != "proj.utils" {
		t.Fatalf("expected the whole name replaced from column 4, got %+v", edit)
	}
}

// The Cavefile declares the base every other answer is derived from, so there is nothing to derive its own from.
func TestModModuleNameCompletionSkipsCavefile(t *testing.T) {
	ls, base := modCompletionServer(t)
	ls.setFilesystem(base, "/")

	if name := ls.impliedModuleName("Cavefile"); name != "" {
		t.Fatalf("expected no implied module name for the Cavefile, got %q", name)
	}
}

// The `mod` keyword itself completes to the whole declaration, not to a placeholder.
func TestModKeywordCompletionCarriesTheQualifiedName(t *testing.T) {
	ls, base := modCompletionServer(t)
	writeFile(t, base, "utils/greet.zirr", "\n")
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("utils/greet.zirr", protocol.Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}
	for _, item := range items {
		if item.Label != "mod" {
			continue
		}
		if item.InsertText == nil || *item.InsertText != "mod proj.utils" {
			t.Fatalf("expected the mod keyword to insert 'mod proj.utils', got %v", item.InsertText)
		}
		return
	}
	t.Fatalf("expected a mod keyword completion, got %v", labelsOf(items))
}
