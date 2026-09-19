package langsrv

import (
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestFindReferencesInFile(t *testing.T) {
	src := "fn greet(name) {}\nfn main() { greet(\"world\") }\nfn run() { greet(\"hi\") }"
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

// TestFindReferencesCrossModule is a regression test: textDocumentReferences used to only ever search filepath.Dir(path), reporting zero references for a symbol used from a different directory that imports it.
func TestFindReferencesCrossModule(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "flow/types.zirr", "mod flow\nfn helper() {}\n")
	writeFile(t, base, "main.zirr", "mod main\nimport flow\nconst x = flow.helper()\n")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///flow/types.zirr"},
			Position:     protocol.Position{Line: 1, Character: 4},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	}
	locs, err := ls.textDocumentReferences(nil, params)
	if err != nil {
		t.Fatalf("textDocumentReferences: %v", err)
	}
	if len(locs) != 1 {
		t.Fatalf("expected exactly 1 cross-module reference, got %d: %+v", len(locs), locs)
	}
	if locs[0].URI != "file:///main.zirr" {
		t.Errorf("expected reference in main.zirr, got %s", locs[0].URI)
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
	if len(locs) != 1 {
		t.Errorf("expected 1 reference (call only), got %d", len(locs))
	}
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

// TestFindReferencesNoDuplicatesMultiFile is a regression test: SourceFile.EnumerateChildNodes visiting parent symbols used to emit each cross-file reference N times.
func TestFindReferencesNoDuplicatesMultiFile(t *testing.T) {
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
	if len(locs) != 4 {
		t.Errorf("expected exactly 4 locations (1 decl + 3 calls), got %d", len(locs))
	}
}

// TestFindReferencesInAttributeDecorators is a regression test: a decorator usage (@Widget(...)) is a DeclAttrInstance node, different from the TypeExprRef the walk already handled for type-hint positions, so it was silently missed.
func TestFindReferencesInAttributeDecorators(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "attr Widget {}\n\n@Widget()\ndata Model {}\n")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 0, Character: 6},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	}
	locs, err := ls.textDocumentReferences(nil, params)
	if err != nil {
		t.Fatalf("textDocumentReferences: %v", err)
	}
	if len(locs) != 1 {
		t.Fatalf("expected exactly 1 reference (the @Widget() decorator), got %d: %+v", len(locs), locs)
	}
	if locs[0].Range.Start.Line != 2 {
		t.Errorf("expected reference on line 2 (0-indexed), got line %d", locs[0].Range.Start.Line)
	}
}

func TestFindReferencesInAttributeDecoratorsCrossModule(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "flow/types.zirr", "mod flow\nattr Widget {}\n")
	writeFile(t, base, "main.zirr", "mod main\nimport flow { Widget }\n\n@Widget()\ndata Model {}\n")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///flow/types.zirr"},
			Position:     protocol.Position{Line: 1, Character: 6},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	}
	locs, err := ls.textDocumentReferences(nil, params)
	if err != nil {
		t.Fatalf("textDocumentReferences: %v", err)
	}
	// Both the member-import list entry and the @Widget() decorator usage must be found.
	if len(locs) != 2 {
		t.Fatalf("expected exactly 2 cross-module references, got %d: %+v", len(locs), locs)
	}
	for _, loc := range locs {
		if loc.URI != "file:///main.zirr" {
			t.Errorf("expected reference in main.zirr, got %s", loc.URI)
		}
	}
}

func TestFindReferencesInTypeExpressions(t *testing.T) {
	src := "data Point { x }\nfn move(p: Point) -> Point { Point }"
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
			Position:     protocol.Position{Line: 0, Character: 6},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	}
	locs, err := ls.textDocumentReferences(nil, params)
	if err != nil {
		t.Fatalf("textDocumentReferences: %v", err)
	}
	if len(locs) != 4 {
		t.Errorf("expected 4 references (decl + 2 type refs + 1 value ref), got %d", len(locs))
		for i, loc := range locs {
			t.Logf("  [%d] %s line=%d char=%d", i, loc.URI, loc.Range.Start.Line, loc.Range.Start.Character)
		}
	}
}

func TestFindReferencesFromTypePosition(t *testing.T) {
	src := "data Point { x }\nfn move(p: Point) { Point }"
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
			Position:     protocol.Position{Line: 1, Character: 12},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	}
	locs, err := ls.textDocumentReferences(nil, params)
	if err != nil {
		t.Fatalf("textDocumentReferences: %v", err)
	}
	if len(locs) < 3 {
		t.Errorf("expected at least 3 references, got %d", len(locs))
		for i, loc := range locs {
			t.Logf("  [%d] %s line=%d char=%d", i, loc.URI, loc.Range.Start.Line, loc.Range.Start.Character)
		}
	}
}

func TestFindReferencesQualifiedModName(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "mymod/main.zirr", "mod mymod\nfn helper() {}\nmymod.helper()\nhelper()")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

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
	if len(locs) < 2 {
		t.Errorf("expected at least 2 references, got %d", len(locs))
		for i, loc := range locs {
			t.Logf("  [%d] %s line=%d char=%d", i, loc.URI, loc.Range.Start.Line, loc.Range.Start.Character)
		}
	}
}

// TestFindReferencesUnionCase is a regression test: DeclUnionMember is a different AST shape than TypeExprRef/ExprIdentifier, so a union case (union Shape { Circle }) was invisible to references (and rename) entirely.
func TestFindReferencesUnionCase(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "data Circle { radius }\nunion Shape {\n\tCircle\n}\n")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 0, Character: 7},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	}
	locs, err := ls.textDocumentReferences(nil, params)
	if err != nil {
		t.Fatalf("textDocumentReferences: %v", err)
	}
	if len(locs) != 1 {
		t.Fatalf("expected exactly 1 reference (the union case), got %d: %+v", len(locs), locs)
	}
	if locs[0].Range.Start.Line != 2 {
		t.Errorf("expected the union case reference on line 2, got line %d", locs[0].Range.Start.Line)
	}
}

// TestFindReferencesImportMemberList is a regression test: DeclImportMember is a different AST shape than ExprIdentifier/ExprMemberAccess, so a member-import list entry (import mod { Foo }) was invisible to references (and rename), even though the name resolved fine once used.
func TestFindReferencesImportMemberList(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "flow/types.zirr", "mod flow\nfn helper() {}\n")
	// Uses qualified flow.helper() rather than a bare helper() call: bare calls of a member-imported name from another directory hit a separate, pre-existing VM panic (confirmed via orchestra.RunFile), out of scope here.
	writeFile(t, base, "main.zirr", "mod main\nimport flow { helper }\nconst x = flow.helper()\n")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///flow/types.zirr"},
			Position:     protocol.Position{Line: 1, Character: 4},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	}
	locs, err := ls.textDocumentReferences(nil, params)
	if err != nil {
		t.Fatalf("textDocumentReferences: %v", err)
	}
	if len(locs) != 2 {
		t.Fatalf("expected exactly 2 references, got %d: %+v", len(locs), locs)
	}
	foundImportEntry := false
	for _, loc := range locs {
		if loc.URI != "file:///main.zirr" {
			t.Errorf("expected reference in main.zirr, got %s", loc.URI)
		}
		if loc.Range.Start.Line == 1 {
			foundImportEntry = true
		}
	}
	if !foundImportEntry {
		t.Error("expected a reference on line 1 (the import { helper } list entry)")
	}
}

// TestFindReferencesFromUsageSiteInDifferentSubmodule is a regression test: the cross-module search used to always search the current file's own directory plus its importers, which is wrong when the cursor is on a symbol imported from elsewhere — it searched importers of the wrong (current, not declaring) module.
func TestFindReferencesFromUsageSiteInDifferentSubmodule(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "moda/types.zirr", "mod moda\nfn shared() {}\n")
	writeFile(t, base, "modb/usage.zirr", "mod modb\nimport moda\nconst x = moda.shared()\n")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///modb/usage.zirr"},
			Position:     protocol.Position{Line: 2, Character: 18},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	}
	locs, err := ls.textDocumentReferences(nil, params)
	if err != nil {
		t.Fatalf("textDocumentReferences: %v", err)
	}
	if len(locs) != 2 {
		t.Fatalf("expected exactly 2 locations (decl + usage), got %d: %+v", len(locs), locs)
	}
	var sawDecl, sawUsage bool
	for _, loc := range locs {
		switch loc.URI {
		case "file:///moda/types.zirr":
			sawDecl = true
		case "file:///modb/usage.zirr":
			sawUsage = true
		}
	}
	if !sawDecl {
		t.Error("expected to find the declaration in moda/types.zirr")
	}
	if !sawUsage {
		t.Error("expected to find the usage in modb/usage.zirr")
	}
}
