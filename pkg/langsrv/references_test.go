package langsrv

import (
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestFindReferencesInFile(t *testing.T) {
	// "greet" appears 3 times: declaration, call in main, call in other func.
	src := "fn greet(name) {}\nfn main() { greet(\"world\") }\nfn run() { greet(\"hi\") }"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor on the "greet" declaration
	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	}
	locs, err := ls.textDocumentReferences(nil, params)
	if err != nil {
		t.Fatalf("textDocumentReferences: %v", err)
	}
	if len(locs) < 3 {
		t.Errorf("expected at least 3 references (decl + 2 calls), got %d", len(locs))
	}
}

func TestFindReferencesExcludeDeclaration(t *testing.T) {
	src := "fn greet(name) {}\nfn main() { greet(\"world\") }"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	}
	locs, err := ls.textDocumentReferences(nil, params)
	if err != nil {
		t.Fatalf("textDocumentReferences: %v", err)
	}
	// Should have 1 reference (the call), not the declaration.
	if len(locs) != 1 {
		t.Errorf("expected 1 reference (call only), got %d", len(locs))
	}
	// The call is on line 1.
	if locs[0].Range.Start.Line != 1 {
		t.Errorf("reference should be on line 1, got line %d", locs[0].Range.Start.Line)
	}
}

func TestFindReferencesAcrossFiles(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "types.zirr", "fn helper() {}")
	writeFile(t, base, "main.zirr", "fn main() { helper() }")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///types.zirr"},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	}
	locs, err := ls.textDocumentReferences(nil, params)
	if err != nil {
		t.Fatalf("textDocumentReferences: %v", err)
	}
	// Should find reference in types.zirr (decl) and main.zirr (call).
	if len(locs) < 2 {
		t.Errorf("expected at least 2 cross-file references, got %d", len(locs))
	}
	uris := make(map[string]bool)
	for _, loc := range locs {
		uris[loc.URI] = true
	}
	if !uris["file:///types.zirr"] {
		t.Error("expected reference in types.zirr")
	}
	if !uris["file:///main.zirr"] {
		t.Error("expected reference in main.zirr")
	}
}

func TestFindReferencesNoDuplicatesMultiFile(t *testing.T) {
	// 3-file module: 1 declaration + 3 calls — must return exactly 4 locations
	// (1 decl + 3 refs) regardless of the number of files. Regression for the
	// bug where SourceFile.EnumerateChildNodes visiting parent symbols caused
	// each cross-file reference to be emitted N times.
	base := memfs.New()
	writeFile(t, base, "lib.zirr", "fn helper() {}")
	writeFile(t, base, "a.zirr", "fn fa() { helper() }")
	writeFile(t, base, "b.zirr", "fn fb() { helper() }")
	writeFile(t, base, "c.zirr", "fn fc() { helper() }")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///lib.zirr"},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	}
	locs, err := ls.textDocumentReferences(nil, params)
	if err != nil {
		t.Fatalf("textDocumentReferences: %v", err)
	}
	// 1 decl (lib.zirr) + 3 calls (a, b, c) = 4 total; no duplicates.
	if len(locs) != 4 {
		t.Errorf("expected exactly 4 locations (1 decl + 3 calls), got %d", len(locs))
	}
}
