package langsrv

import (
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// preludeShim is the minimal prelude needed for type resolution tests.
const preludeShim = `mod prelude
extern type Any {}
extern type String {
	length: Int
}
extern type Int {}
extern type Bool {
	toggle() -> Bool
}
extern type Array {
	length: Int
}
extern type Func {
	arity: Int
}
`

func TestDotChainContext(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pos      protocol.Position
		wantSegs []string
		wantOk   bool
	}{
		{
			name:     "single segment",
			text:     "alias.member",
			pos:      protocol.Position{Line: 0, Character: 12},
			wantSegs: []string{"alias"},
			wantOk:   true,
		},
		{
			name:     "two segments",
			text:     "person.name.length",
			pos:      protocol.Position{Line: 0, Character: 18},
			wantSegs: []string{"person", "name"},
			wantOk:   true,
		},
		{
			name:     "three segments",
			text:     "a.b.c.d",
			pos:      protocol.Position{Line: 0, Character: 7},
			wantSegs: []string{"a", "b", "c"},
			wantOk:   true,
		},
		{
			name:     "partial member after dot",
			text:     "person.na",
			pos:      protocol.Position{Line: 0, Character: 9},
			wantSegs: []string{"person"},
			wantOk:   true,
		},
		{
			name:     "cursor right after dot",
			text:     "person.",
			pos:      protocol.Position{Line: 0, Character: 7},
			wantSegs: []string{"person"},
			wantOk:   true,
		},
		{
			name:   "no dot",
			text:   "person",
			pos:    protocol.Position{Line: 0, Character: 6},
			wantOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segs, _, ok := dotChainContext(tt.text, tt.pos)
			if ok != tt.wantOk {
				t.Fatalf("ok=%v want %v", ok, tt.wantOk)
			}
			if !ok {
				return
			}
			if len(segs) != len(tt.wantSegs) {
				t.Fatalf("segments=%v want %v", segs, tt.wantSegs)
			}
			for i := range segs {
				if segs[i] != tt.wantSegs[i] {
					t.Errorf("segments[%d]=%q want %q", i, segs[i], tt.wantSegs[i])
				}
			}
		})
	}
}

func TestTypedMemberCompletion(t *testing.T) {
	// person.  should complete with "name" and "age"
	src := preludeShim
	base := memfs.New()
	writeFile(t, base, "prelude/shim.zirr", src)
	writeFile(t, base, "mymod/main.zirr", `mod mymod

data Person {
	name: String
	age: Int
}

const peter: Person = Person("Peter", 42)
peter.
`)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor at "peter." — line 8, after the dot
	items, err := ls.completionItemsForFile("mymod/main.zirr", protocol.Position{Line: 8, Character: 6})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}
	if !labels["name"] {
		t.Errorf("expected 'name' in completions, got: %v", labelList(items))
	}
	if !labels["age"] {
		t.Errorf("expected 'age' in completions, got: %v", labelList(items))
	}
}

func TestTypedMemberCompletionNested(t *testing.T) {
	// peter.name.  should complete with "length" (String field)
	base := memfs.New()
	writeFile(t, base, "prelude/shim.zirr", preludeShim)
	writeFile(t, base, "mymod/main.zirr", `mod mymod

extern type String {
	length: Int
}
extern type Int {}

data Person {
	name: String
	age: Int
}

const peter: Person = Person("Peter", 42)
peter.name.
`)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor at "peter.name." — line 13, character 11
	items, err := ls.completionItemsForFile("mymod/main.zirr", protocol.Position{Line: 13, Character: 11})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}
	if !labels["length"] {
		t.Errorf("expected 'length' in completions, got: %v", labelList(items))
	}
}

func TestTypedMemberCompletionModuleQualified(t *testing.T) {
	// mymod.peter.  should complete with Person fields
	base := memfs.New()
	writeFile(t, base, "prelude/shim.zirr", preludeShim)
	writeFile(t, base, "mymod/main.zirr", `mod mymod

data Person {
	name: String
	age: Int
}

const peter: Person = Person("Peter", 42)
mymod.peter.
`)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor at "mymod.peter." — line 8, character 12
	items, err := ls.completionItemsForFile("mymod/main.zirr", protocol.Position{Line: 8, Character: 12})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}
	if !labels["name"] {
		t.Errorf("expected 'name' in completions, got: %v", labelList(items))
	}
	if !labels["age"] {
		t.Errorf("expected 'age' in completions, got: %v", labelList(items))
	}
}

func TestTypedMemberCompletionModuleQualifiedNested(t *testing.T) {
	// mymod.peter.name.  should complete with String fields
	base := memfs.New()
	writeFile(t, base, "prelude/shim.zirr", preludeShim)
	writeFile(t, base, "mymod/main.zirr", `mod mymod

extern type String {
	length: Int
}
extern type Int {}

data Person {
	name: String
	age: Int
}

const peter: Person = Person("Peter", 42)
mymod.peter.name.
`)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor at "mymod.peter.name." — line 13, character 17
	items, err := ls.completionItemsForFile("mymod/main.zirr", protocol.Position{Line: 13, Character: 17})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}
	if !labels["length"] {
		t.Errorf("expected 'length' in completions, got: %v", labelList(items))
	}
}

func TestTypedMemberCompletionLocalVar(t *testing.T) {
	// Local var with type hint should also support member access
	base := memfs.New()
	writeFile(t, base, "prelude/shim.zirr", preludeShim)
	writeFile(t, base, "mymod/main.zirr", `mod mymod

data Person {
	name: String
	age: Int
}

fn greet() {
	var p: Person = Person("P", 1)
	p.
}
`)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor at "p." inside greet — line 9, character 3
	items, err := ls.completionItemsForFile("mymod/main.zirr", protocol.Position{Line: 9, Character: 3})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}
	if !labels["name"] {
		t.Errorf("expected 'name' in completions, got: %v", labelList(items))
	}
	if !labels["age"] {
		t.Errorf("expected 'age' in completions, got: %v", labelList(items))
	}
}

func TestTypedMemberCompletionParameter(t *testing.T) {
	// Parameter with type hint should support member access
	base := memfs.New()
	writeFile(t, base, "prelude/shim.zirr", preludeShim)
	writeFile(t, base, "mymod/main.zirr", `mod mymod

data Person {
	name: String
	age: Int
}

fn greet(p: Person) {
	p.
}
`)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor at "p." — line 8, character 3
	items, err := ls.completionItemsForFile("mymod/main.zirr", protocol.Position{Line: 8, Character: 3})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}
	if !labels["name"] {
		t.Errorf("expected 'name' in completions, got: %v", labelList(items))
	}
}

func TestTypedMemberCompletionMethodField(t *testing.T) {
	// Bool's toggle() should show as a completion
	base := memfs.New()
	writeFile(t, base, "prelude/shim.zirr", preludeShim)
	writeFile(t, base, "mymod/main.zirr", `mod mymod

extern type Bool {
	toggle() -> Bool
}

const flag: Bool = true
flag.
`)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor at "flag." — line 7, character 5
	items, err := ls.completionItemsForFile("mymod/main.zirr", protocol.Position{Line: 7, Character: 5})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	var found bool
	for _, item := range items {
		if item.Label == "toggle" {
			found = true
			// Field kind — the parser doesn't distinguish 0-param methods from value fields.
			if item.Kind == nil || *item.Kind != protocol.CompletionItemKindField {
				t.Errorf("expected toggle to be Field kind, got %v", *item.Kind)
			}
		}
	}
	if !found {
		t.Errorf("expected 'toggle' in completions, got: %v", labelList(items))
	}
}

func TestTypedMemberHover(t *testing.T) {
	// Hover over "name" in "peter.name" should show the field info
	base := memfs.New()
	writeFile(t, base, "prelude/shim.zirr", preludeShim)
	writeFile(t, base, "mymod/main.zirr", `mod mymod

data Person {
	name: String
	age: Int
}

const peter: Person = Person("Peter", 42)
peter.name
`)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///mymod/main.zirr"},
			Position:     protocol.Position{Line: 8, Character: 8}, // on "name"
		},
	}
	result, err := ls.textDocumentHover(nil, params)
	if err != nil {
		t.Fatalf("textDocumentHover: %v", err)
	}
	if result == nil {
		t.Fatal("expected hover result, got nil")
		return
	}
	content := result.Contents.(protocol.MarkupContent)
	if !strings.Contains(content.Value, "name: String") {
		t.Errorf("expected hover to contain 'name: String', got: %q", content.Value)
	}
}

func TestTypedMemberHoverNested(t *testing.T) {
	// Hover over "length" in "peter.name.length"
	base := memfs.New()
	writeFile(t, base, "prelude/shim.zirr", preludeShim)
	writeFile(t, base, "mymod/main.zirr", `mod mymod

extern type String {
	length: Int
}
extern type Int {}

data Person {
	name: String
	age: Int
}

const peter: Person = Person("Peter", 42)
peter.name.length
`)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///mymod/main.zirr"},
			Position:     protocol.Position{Line: 13, Character: 14}, // on "length"
		},
	}
	result, err := ls.textDocumentHover(nil, params)
	if err != nil {
		t.Fatalf("textDocumentHover: %v", err)
	}
	if result == nil {
		t.Fatal("expected hover result, got nil")
		return
	}
	content := result.Contents.(protocol.MarkupContent)
	if !strings.Contains(content.Value, "length: Int") {
		t.Errorf("expected hover to contain 'length: Int', got: %q", content.Value)
	}
}

func TestTypedMemberDefinition(t *testing.T) {
	// Go-to-definition on "name" in "peter.name" should navigate to the field declaration
	base := memfs.New()
	writeFile(t, base, "prelude/shim.zirr", preludeShim)
	writeFile(t, base, "mymod/main.zirr", `mod mymod

data Person {
	name: String
	age: Int
}

const peter: Person = Person("Peter", 42)
peter.name
`)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///mymod/main.zirr"},
			Position:     protocol.Position{Line: 8, Character: 8}, // on "name"
		},
	}
	result, err := ls.textDocumentDefinition(nil, params)
	if err != nil {
		t.Fatalf("textDocumentDefinition: %v", err)
	}
	if result == nil {
		t.Fatal("expected definition result, got nil")
	}
	loc, ok := result.(*protocol.Location)
	if !ok {
		t.Fatalf("expected *Location, got %T", result)
	}
	// The definition should point to the "name" field in Person (line 3).
	if loc.Range.Start.Line != 3 {
		t.Errorf("expected definition on line 3 (field declaration), got line %d", loc.Range.Start.Line)
	}
}

func TestTypedMemberSignatureHelp(t *testing.T) {
	// flag.toggle( should provide signature help for the toggle method
	base := memfs.New()
	writeFile(t, base, "prelude/shim.zirr", preludeShim)
	writeFile(t, base, "mymod/main.zirr", `mod mymod

extern type Bool {
	toggle() -> Bool
}

const flag: Bool = true
flag.toggle(
`)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///mymod/main.zirr"},
			Position:     protocol.Position{Line: 7, Character: 12},
		},
	}
	result, err := ls.textDocumentSignatureHelp(nil, params)
	if err != nil {
		t.Fatalf("textDocumentSignatureHelp: %v", err)
	}
	if result == nil {
		t.Fatal("expected SignatureHelp for flag.toggle(, got nil")
		return
	}
	if len(result.Signatures) == 0 {
		t.Fatal("expected at least one signature")
	}
}

func labelList(items []protocol.CompletionItem) []string {
	var labels []string
	for _, item := range items {
		labels = append(labels, item.Label)
	}
	return labels
}
