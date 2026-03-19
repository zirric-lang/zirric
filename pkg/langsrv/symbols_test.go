package langsrv

import (
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentDocumentSymbol(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		wantNames  []string
		wantAbsent []string
	}{
		{
			name:      "func declaration",
			src:       "fn greet() {}",
			wantNames: []string{"greet"},
		},
		{
			name:      "data declaration",
			src:       "data Point { x }",
			wantNames: []string{"Point"},
		},
		{
			name:      "attribute declaration",
			src:       "attr Numeric { toNumber }",
			wantNames: []string{"Numeric"},
		},
		{
			name:      "multiple declarations",
			src:       "fn greet() {}\ndata Point { x }",
			wantNames: []string{"greet", "Point"},
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

			params := &protocol.DocumentSymbolParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			}
			result, err := ls.textDocumentDocumentSymbol(nil, params)
			if err != nil {
				t.Fatalf("textDocumentDocumentSymbol: %v", err)
			}

			symbols, ok := result.([]protocol.DocumentSymbol)
			if !ok && result != nil {
				t.Fatalf("expected []DocumentSymbol, got %T", result)
			}
			if result == nil {
				symbols = nil
			}

			nameSet := make(map[string]bool)
			for _, s := range symbols {
				nameSet[s.Name] = true
			}
			for _, want := range tt.wantNames {
				if !nameSet[want] {
					t.Errorf("missing symbol %q; got %v", want, symbolNames(symbols))
				}
			}
			for _, absent := range tt.wantAbsent {
				if nameSet[absent] {
					t.Errorf("unexpected symbol %q in results", absent)
				}
			}
		})
	}
}

func TestDocumentSymbolOnlyCurrentFile(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "types.zirr", "data Point { x }")
	writeFile(t, base, "main.zirr", "fn greet() {}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
	}
	result, err := ls.textDocumentDocumentSymbol(nil, params)
	if err != nil {
		t.Fatalf("textDocumentDocumentSymbol: %v", err)
	}
	symbols, _ := result.([]protocol.DocumentSymbol)
	nameSet := make(map[string]bool)
	for _, s := range symbols {
		nameSet[s.Name] = true
	}
	if !nameSet["greet"] {
		t.Errorf("missing greet from main.zirr")
	}
	if nameSet["Point"] {
		t.Errorf("Point from types.zirr should not appear in main.zirr symbols")
	}
}

func TestWorkspaceSymbol(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "fn greet() {}")
	writeFile(t, base, "types.zirr", "data Point { x }")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	tests := []struct {
		query     string
		wantNames []string
	}{
		{query: "", wantNames: []string{"greet", "Point"}},
		{query: "gr", wantNames: []string{"greet"}},
		{query: "po", wantNames: []string{"Point"}},
		{query: "xyz", wantNames: []string{}},
	}

	for _, tt := range tests {
		t.Run("query="+tt.query, func(t *testing.T) {
			params := &protocol.WorkspaceSymbolParams{Query: tt.query}
			results, err := ls.workspaceSymbol(nil, params)
			if err != nil {
				t.Fatalf("workspaceSymbol: %v", err)
			}
			nameSet := make(map[string]bool)
			for _, s := range results {
				nameSet[s.Name] = true
			}
			for _, want := range tt.wantNames {
				if !nameSet[want] {
					names := make([]string, 0, len(results))
					for _, s := range results {
						names = append(names, s.Name)
					}
					t.Errorf("missing symbol %q; got %v", want, names)
				}
			}
		})
	}
}

func TestWorkspaceSymbolLocation(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "fn greet() {}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	results, err := ls.workspaceSymbol(nil, &protocol.WorkspaceSymbolParams{Query: "greet"})
	if err != nil {
		t.Fatalf("workspaceSymbol: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
		return
	}
	sym := results[0]
	if sym.Name != "greet" {
		t.Errorf("name = %q, want greet", sym.Name)
	}
	if sym.Kind != protocol.SymbolKindFunction {
		t.Errorf("kind = %v, want Function", sym.Kind)
	}
	if !strings.HasSuffix(string(sym.Location.URI), "main.zirr") {
		t.Errorf("URI = %q, want .../main.zirr", sym.Location.URI)
	}
	// "greet" starts at col 3 in "fn greet() {}"
	if sym.Location.Range.Start.Character != 3 {
		t.Errorf("start.character = %d, want 3", sym.Location.Range.Start.Character)
	}
}

func TestWorkspaceSymbolAcrossModules(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "fn greet() {}")
	if err := base.MkdirAll("mymod", 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, base, "mymod/types.zirr", "data Point { x }\nfn helper() {}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	results, err := ls.workspaceSymbol(nil, &protocol.WorkspaceSymbolParams{Query: ""})
	if err != nil {
		t.Fatalf("workspaceSymbol: %v", err)
	}
	nameSet := make(map[string]bool)
	containerByName := make(map[string]string)
	for _, s := range results {
		nameSet[s.Name] = true
		if s.ContainerName != nil {
			containerByName[s.Name] = *s.ContainerName
		}
	}
	for _, want := range []string{"greet", "Point", "helper"} {
		if !nameSet[want] {
			names := make([]string, 0, len(results))
			for _, s := range results {
				names = append(names, s.Name)
			}
			t.Errorf("missing symbol %q across modules; got %v", want, names)
		}
	}
	// Root module symbols should have empty container; submodule symbols get "mymod".
	if c := containerByName["greet"]; c != "" {
		t.Errorf("greet container = %q, want empty", c)
	}
	if c := containerByName["Point"]; c != "mymod" {
		t.Errorf("Point container = %q, want %q", c, "mymod")
	}
}

func TestWorkspaceSymbolSkipsInvalidDirs(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "fn greet() {}")
	// ".hidden" and "node_modules" style dirs should be skipped.
	if err := base.MkdirAll(".hidden", 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, base, ".hidden/secret.zirr", "fn secret() {}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	results, err := ls.workspaceSymbol(nil, &protocol.WorkspaceSymbolParams{Query: ""})
	if err != nil {
		t.Fatalf("workspaceSymbol: %v", err)
	}
	for _, s := range results {
		if s.Name == "secret" {
			t.Error("workspace symbols should not include files from invalid module dirs like .hidden")
		}
	}
}

func symbolNames(syms []protocol.DocumentSymbol) []string {
	names := make([]string, len(syms))
	for i, s := range syms {
		names[i] = s.Name
	}
	return names
}
