package langsrv

import (
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentHover(t *testing.T) {
	tests := []struct {
		name        string
		src         string
		pos         protocol.Position
		wantNil     bool
		wantContain string
	}{
		{
			name:        "hover over func name",
			src:         "fn greet() {}",
			pos:         protocol.Position{Line: 0, Character: 6},
			wantContain: "fn greet",
		},
		{
			name:        "hover over data name",
			src:         "data Point { x }",
			pos:         protocol.Position{Line: 0, Character: 6},
			wantContain: "data Point",
		},
		{
			name:        "hover over attribute name",
			src:         "attr Numeric { toNumber }",
			pos:         protocol.Position{Line: 0, Character: 12},
			wantContain: "attr Numeric",
		},
		{
			name:    "hover over whitespace returns nil",
			src:     "fn greet() {}",
			pos:     protocol.Position{Line: 0, Character: 2},
			wantNil: true,
		},
		{
			name:    "hover over unknown word returns nil",
			src:     "fn greet() {}",
			pos:     protocol.Position{Line: 0, Character: 14},
			wantNil: true,
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

			params := &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
					Position:     tt.pos,
				},
			}
			result, err := ls.textDocumentHover(nil, params)
			if err != nil {
				t.Fatalf("textDocumentHover: %v", err)
			}

			if result == nil {
				if !tt.wantNil {
					t.Fatalf("expected non-nil hover for %q at %+v", tt.src, tt.pos)
				}
				return
			}
			if tt.wantNil {
				t.Errorf("expected nil hover, got %+v", result)
				return
			}

			mc, ok := result.Contents.(protocol.MarkupContent)
			if !ok {
				t.Fatalf("expected MarkupContent, got %T", result.Contents)
			}
			if !strings.Contains(mc.Value, tt.wantContain) {
				t.Errorf("hover content %q does not contain %q", mc.Value, tt.wantContain)
			}
			if mc.Kind != protocol.MarkupKindMarkdown {
				t.Errorf("hover kind = %q, want markdown", mc.Kind)
			}
		})
	}
}

func TestHoverAcrossFiles(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "types.zirr", "data Point { x }")
	writeFile(t, base, "main.zirr", "const p = Point")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Hover over "Point" in main.zirr (col 10)
	params := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	}
	result, err := ls.textDocumentHover(nil, params)
	if err != nil {
		t.Fatalf("textDocumentHover: %v", err)
	}
	if result == nil {
		t.Fatal("expected hover for cross-file symbol Point, got nil")
		return
	}
	mc, ok := result.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected MarkupContent, got %T", result.Contents)
		return
	}
	if !strings.Contains(mc.Value, "Point") {
		t.Errorf("hover content %q does not contain 'Point'", mc.Value)
	}
}

func TestWordAtPosition(t *testing.T) {
	tests := []struct {
		text     string
		pos      protocol.Position
		wantWord string
		wantNone bool
	}{
		{"fn greet() {}", protocol.Position{Line: 0, Character: 6}, "greet", false},
		{"fn greet() {}", protocol.Position{Line: 0, Character: 14}, "", true}, // space before '{'
		{"fn greet() {}", protocol.Position{Line: 0, Character: 0}, "fn", false},
		{"hello\nworld", protocol.Position{Line: 1, Character: 2}, "world", false},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			word, _ := wordAtPosition(tt.text, tt.pos)
			if tt.wantNone {
				if word != "" {
					t.Errorf("wordAtPosition: got %q, want empty", word)
				}
				return
			}
			if word != tt.wantWord {
				t.Errorf("wordAtPosition: got %q, want %q", word, tt.wantWord)
			}
		})
	}
}
