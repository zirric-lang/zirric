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

func TestFindReferencesInTypeExpressions(t *testing.T) {
	// "Point" is used in: declaration, parameter type, return type, value ref.
	src := "data Point { x }\nfn move(p: Point) -> Point { Point }"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor on "Point" declaration
	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 0, Character: 6},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	}
	locs, err := ls.textDocumentReferences(nil, params)
	if err != nil {
		t.Fatalf("textDocumentReferences: %v", err)
	}
	// 1 decl + 2 type refs (param + return) + 1 value ref = 4
	if len(locs) != 4 {
		t.Errorf("expected 4 references (decl + 2 type refs + 1 value ref), got %d", len(locs))
		for i, loc := range locs {
			t.Logf("  [%d] %s line=%d char=%d", i, loc.URI, loc.Range.Start.Line, loc.Range.Start.Character)
		}
	}
}

func TestFindReferencesFromTypePosition(t *testing.T) {
	// Cursor on "Point" in a type hint position — should still find all references.
	src := "data Point { x }\nfn move(p: Point) { Point }"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor on "Point" in the parameter type hint (line 1, col 11)
	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 1, Character: 12},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	}
	locs, err := ls.textDocumentReferences(nil, params)
	if err != nil {
		t.Fatalf("textDocumentReferences: %v", err)
	}
	// 1 decl + 1 type ref + 1 value ref = 3
	if len(locs) < 3 {
		t.Errorf("expected at least 3 references, got %d", len(locs))
		for i, loc := range locs {
			t.Logf("  [%d] %s line=%d char=%d", i, loc.URI, loc.Range.Start.Line, loc.Range.Start.Character)
		}
	}
}

func TestFindReferencesQualifiedModName(t *testing.T) {
	// modname.helper should find "helper" references including the qualified one.
	base := memfs.New()
	writeFile(t, base, "mymod/main.zirr", "mod mymod\nfn helper() {}\nmymod.helper()\nhelper()")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor on "helper" in "mymod.helper()" at line 2, character 6
	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///mymod/main.zirr"},
			Position:     protocol.Position{Line: 2, Character: 8},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	}
	locs, err := ls.textDocumentReferences(nil, params)
	if err != nil {
		t.Fatalf("textDocumentReferences: %v", err)
	}
	// Should find: 1 decl "fn helper" + 1 qualified ref "mymod.helper" + 1 unqualified ref "helper"
	if len(locs) < 2 {
		t.Errorf("expected at least 2 references, got %d", len(locs))
		for i, loc := range locs {
			t.Logf("  [%d] %s line=%d char=%d", i, loc.URI, loc.Range.Start.Line, loc.Range.Start.Character)
		}
	}
}
