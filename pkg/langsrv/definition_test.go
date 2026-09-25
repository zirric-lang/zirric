package langsrv

import (
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentDefinition(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		pos      protocol.Position
		wantNil  bool
		wantLine uint32
		wantChar uint32
	}{
		{
			name:     "definition of func from its own name",
			src:      "fn greet() {}",
			pos:      protocol.Position{Line: 0, Character: 6},
			wantLine: 0,
			wantChar: 3, // 'g' in 'greet'
		},
		{
			name:     "definition of data from its own name",
			src:      "data Point { x }",
			pos:      protocol.Position{Line: 0, Character: 6},
			wantLine: 0,
			wantChar: 5, // 'P' in 'Point'
		},
		{
			name:    "definition of unknown word returns nil",
			src:     "fn greet() {}",
			pos:     protocol.Position{Line: 0, Character: 14},
			wantNil: true,
		},
		{
			name:    "definition of whitespace returns nil",
			src:     "fn greet() {}",
			pos:     protocol.Position{Line: 0, Character: 2},
			wantNil: true,
		},
		{
			name:     "definition of type ref in parameter",
			src:      "data Point { x }\nfn move(p: Point) {}",
			pos:      protocol.Position{Line: 1, Character: 12},
			wantLine: 0,
			wantChar: 5, // 'P' in 'data Point'
		},
		{
			name:     "definition of type ref in return type",
			src:      "data Point { x }\nfn origin() -> Point {}",
			pos:      protocol.Position{Line: 1, Character: 16},
			wantLine: 0,
			wantChar: 5, // 'P' in 'data Point'
		},
		{
			name:     "definition of type ref in const type hint",
			src:      "data Point { x }\nconst p: Point = Point",
			pos:      protocol.Position{Line: 1, Character: 10},
			wantLine: 0,
			wantChar: 5,
		},
		{
			name:     "definition of parameter from function body",
			src:      "fn greet(name) {\n  name\n}",
			pos:      protocol.Position{Line: 1, Character: 3},
			wantLine: 0,
			wantChar: 9, // 'n' in 'name' parameter
		},
		{
			name:     "definition of local var from function body",
			src:      "fn test() {\n  var x = 1\n  x\n}",
			pos:      protocol.Position{Line: 2, Character: 2},
			wantLine: 1,
			wantChar: 6, // 'x' in 'var x'
		},
		{
			name:     "definition of local const from function body",
			src:      "fn test() {\n  const y = 2\n  y\n}",
			pos:      protocol.Position{Line: 2, Character: 2},
			wantLine: 1,
			wantChar: 8, // 'y' in 'const y'
		},
		{
			name:     "definition of a name inside a string interpolation",
			src:      "fn greet(name) {\n  \"hi \\(name)\"\n}",
			pos:      protocol.Position{Line: 1, Character: 9},
			wantLine: 0,
			wantChar: 9, // 'n' in the 'name' parameter
		},
		{
			name:     "definition of for-loop binding",
			src:      "fn test() {\n  for item <- [1] {\n    item\n  }\n}",
			pos:      protocol.Position{Line: 2, Character: 5},
			wantLine: 1,
			wantChar: 6, // 'i' in 'item' after 'for '
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := memfs.New()
			writeFile(t, base, "main.zirr", tt.src)

			ls := zirricLangserver{
				docs:     newDocumentStore(),
				diagURIs: make(map[protocol.DocumentUri]struct{}),
				openDocs: make(map[string]protocol.DocumentUri),
			}
			ls.setFilesystem(base, "/")

			params := &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
					Position:     tt.pos,
				},
			}
			result, err := ls.textDocumentDefinition(nil, params)
			if err != nil {
				t.Fatalf("textDocumentDefinition: %v", err)
			}

			if result == nil {
				if !tt.wantNil {
					t.Fatalf("expected non-nil definition for %q at %+v", tt.src, tt.pos)
				}
				return
			}
			if tt.wantNil {
				t.Errorf("expected nil definition, got %+v", result)
				return
			}

			loc, ok := result.(*protocol.Location)
			if !ok {
				t.Fatalf("expected *protocol.Location, got %T", result)
				return
			}
			if loc.Range.Start.Line != tt.wantLine {
				t.Errorf("definition line = %d, want %d", loc.Range.Start.Line, tt.wantLine)
			}
			if loc.Range.Start.Character != tt.wantChar {
				t.Errorf("definition char = %d, want %d", loc.Range.Start.Character, tt.wantChar)
			}
		})
	}
}

func TestDefinitionAcrossFiles(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "types.zirr", "data Point { x }")
	writeFile(t, base, "main.zirr", "const p = Point")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	}
	result, err := ls.textDocumentDefinition(nil, params)
	if err != nil {
		t.Fatalf("textDocumentDefinition: %v", err)
	}
	if result == nil {
		t.Fatal("expected cross-file definition for Point, got nil")
		return
	}
	loc, ok := result.(*protocol.Location)
	if !ok {
		t.Fatalf("expected *protocol.Location, got %T", result)
		return
	}
	// "Point" is at col 5 in "data Point { x }"
	if loc.Range.Start.Line != 0 || loc.Range.Start.Character != 5 {
		t.Errorf("definition location = line %d col %d, want line 0 col 5",
			loc.Range.Start.Line, loc.Range.Start.Character)
	}
	if loc.URI != "file:///types.zirr" {
		t.Errorf("definition URI = %q, want file:///types.zirr", loc.URI)
	}
}

func TestDefinitionQualifiedModule(t *testing.T) {
	base := memfs.New()
	if err := base.MkdirAll("mymod", 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, base, "mymod/types.zirr", "fn helper() {}")
	writeFile(t, base, "main.zirr", "import mymod\nmymod.helper")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor on "helper" at line 1, col 6+ (after "mymod.")
	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 1, Character: 8},
		},
	}
	result, err := ls.textDocumentDefinition(nil, params)
	if err != nil {
		t.Fatalf("textDocumentDefinition: %v", err)
	}
	if result == nil {
		t.Fatal("expected definition for qualified mymod.helper, got nil")
	}
	loc, ok := result.(*protocol.Location)
	if !ok {
		t.Fatalf("expected *protocol.Location, got %T", result)
	}
	if loc.URI != "file:///mymod/types.zirr" {
		t.Errorf("definition URI = %q, want file:///mymod/types.zirr", loc.URI)
	}
}

// TestDefinitionBareImportViaProjectBaseURI is a regression test: with an explicit Cavefile package name ("ui"), a bare import's real URI is "ui.flow" — go-to-definition worked via an old ad hoc path, but diagnostics reported "unknown module" since the resolver had no <projectBaseURI>.<name> fallback yet.
func TestDefinitionBareImportViaProjectBaseURI(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "Cavefile", "mod proj\n\nimport cave\n\n@cave.Package()\ndata App {}\n")
	writeFile(t, base, "flow/types.zirr", "mod flow\nfn helper() {}\n")
	writeFile(t, base, "main.zirr", "mod main\nimport flow\nflow.helper")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor on "helper" in "flow.helper" (line 2, after "flow.").
	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 2, Character: 6},
		},
	}
	result, err := ls.textDocumentDefinition(nil, params)
	if err != nil {
		t.Fatalf("textDocumentDefinition: %v", err)
	}
	if result == nil {
		t.Fatal("expected definition for bare-imported flow.helper, got nil")
	}
	loc, ok := result.(*protocol.Location)
	if !ok {
		t.Fatalf("expected *protocol.Location, got %T", result)
	}
	if loc.URI != "file:///flow/types.zirr" {
		t.Errorf("definition URI = %q, want file:///flow/types.zirr", loc.URI)
	}
}

func TestDefinitionModNameQualified(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "mymod/main.zirr", "mod mymod\nfn helper() {}\nmymod.helper")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///mymod/main.zirr"},
			Position:     protocol.Position{Line: 2, Character: 8},
		},
	}
	result, err := ls.textDocumentDefinition(nil, params)
	if err != nil {
		t.Fatalf("textDocumentDefinition: %v", err)
	}
	if result == nil {
		t.Fatal("expected definition for modname.helper, got nil")
	}
	loc, ok := result.(*protocol.Location)
	if !ok {
		t.Fatalf("expected *protocol.Location, got %T", result)
	}
	if loc.URI != "file:///mymod/main.zirr" {
		t.Errorf("definition URI = %q, want file:///mymod/main.zirr", loc.URI)
	}
}
