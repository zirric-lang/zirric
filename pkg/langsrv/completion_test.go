package langsrv

import (
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestCompletionItemsForFile(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		wantLabels []string
		wantKinds  map[string]protocol.CompletionItemKind
	}{
		{
			name:       "func declaration",
			src:        "fn greet() {}",
			wantLabels: []string{"greet"},
			wantKinds:  map[string]protocol.CompletionItemKind{"greet": protocol.CompletionItemKindFunction},
		},
		{
			name:       "data declaration",
			src:        "data Point { x }",
			wantLabels: []string{"Point"},
			wantKinds:  map[string]protocol.CompletionItemKind{"Point": protocol.CompletionItemKindStruct},
		},
		{
			name:       "union declaration",
			src:        "data Foo {}\ndata Bar {}\nunion Shape {\n\tFoo\n\tBar\n}",
			wantLabels: []string{"Foo", "Bar", "Shape"},
			wantKinds:  map[string]protocol.CompletionItemKind{"Shape": protocol.CompletionItemKindEnum},
		},
		{
			name:       "multiple globals",
			src:        "fn add() {}\nfn count() {}",
			wantLabels: []string{"add", "count"},
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

			items, err := ls.completionItemsForFile("main.zirr", protocol.Position{})
			if err != nil {
				t.Fatalf("completionItemsForFile: %v", err)
			}

			labelSet := make(map[string]protocol.CompletionItem, len(items))
			for _, item := range items {
				labelSet[item.Label] = item
			}

			for _, wantLabel := range tt.wantLabels {
				item, ok := labelSet[wantLabel]
				if !ok {
					t.Errorf("expected completion item %q, got labels: %v", wantLabel, labelKeys(labelSet))
					continue
				}
				if tt.wantKinds != nil {
					if wantKind, hasKind := tt.wantKinds[wantLabel]; hasKind {
						if item.Kind == nil || *item.Kind != wantKind {
							var got protocol.CompletionItemKind
							if item.Kind != nil {
								got = *item.Kind
							}
							t.Errorf("item %q: got kind %v, want %v", wantLabel, got, wantKind)
						}
					}
				}
			}
		})
	}
}

func TestCompletionItemsWithParseErrors(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "fn ok() {}\nfn bad( {}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "ok" {
			return
		}
	}
	t.Errorf("expected completion item %q even with parse errors, got: %v", "ok", labelKeys2(items))
}

func TestCompletionAcrossFiles(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "types.zirr", "data Point { x }\nfn helper() {}")
	writeFile(t, base, "main.zirr", "fn main() {}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labelSet := make(map[string]protocol.CompletionItem, len(items))
	for _, item := range items {
		labelSet[item.Label] = item
	}

	for _, want := range []string{"Point", "helper", "main"} {
		if _, ok := labelSet[want]; !ok {
			t.Errorf("expected cross-file completion item %q, got labels: %v", want, labelKeys(labelSet))
		}
	}
}

func TestAttributeContextCompletion(t *testing.T) {
	src := "attr Numeric { toNumber }\nfn add() {}\ndata Foo {}\nunion Bar {\n\tFoo\n}"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Position after '@' on a new line: simulate "@N" at col 2
	pos := protocol.Position{Line: 6, Character: 2}
	// Append the attribute line to the source so the position exists.
	// We use a separate source string that has an @-prefixed line at line 6.
	src2 := src + "\n@N"
	base2 := memfs.New()
	writeFile(t, base2, "main.zirr", src2)
	ls2 := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls2.setFilesystem(base2, "/")

	items, err := ls2.completionItemsForFile("main.zirr", pos)
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labelSet := make(map[string]protocol.CompletionItem, len(items))
	for _, item := range items {
		labelSet[item.Label] = item
	}

	// Attribute should be present with '@' label and TextEdit snippet that includes '@'.
	attr, ok := labelSet["@Numeric"]
	if !ok {
		t.Fatalf("expected @Numeric attribute in completion, got: %v", labelKeys(labelSet))
	}
	textEdit, ok := attr.TextEdit.(protocol.TextEdit)
	if !ok {
		t.Fatalf("@Numeric: expected TextEdit, got %T", attr.TextEdit)
	}
	wantNewText := "@Numeric(${1:toNumber})"
	if textEdit.NewText != wantNewText {
		t.Errorf("@Numeric TextEdit.NewText = %q, want %q", textEdit.NewText, wantNewText)
	}
	// TextEdit range should start at the '@' (col 0) and end at cursor (col 2).
	if textEdit.Range.Start.Character != 0 || textEdit.Range.End.Character != 2 {
		t.Errorf("@Numeric TextEdit.Range = %+v, want start.char=0 end.char=2", textEdit.Range)
	}
	if attr.InsertTextFormat == nil || *attr.InsertTextFormat != protocol.InsertTextFormatSnippet {
		t.Errorf("@Numeric insertTextFormat should be Snippet")
	}
	// FilterText should be plain name so the user can type without '@'.
	if attr.FilterText == nil || *attr.FilterText != "Numeric" {
		t.Errorf("@Numeric filterText = %v, want %q", attr.FilterText, "Numeric")
	}

	// Type declarations should be present with '@' prefix (data, union).
	if _, ok := labelSet["@Foo"]; !ok {
		t.Errorf("expected @Foo in attribute completion, got: %v", labelKeys(labelSet))
	}
	if _, ok := labelSet["@Bar"]; !ok {
		t.Errorf("expected @Bar in attribute completion, got: %v", labelKeys(labelSet))
	}

	// Functions and values must NOT be present
	for label := range labelSet {
		if label == "add" || label == "@add" {
			t.Errorf("func add must not appear in attribute completion (got %q)", label)
		}
	}
}

func TestAttributeNonContextCompletion(t *testing.T) {
	// Attributes should appear with '@' prefix even outside attribute context.
	src := "attr Numeric { toNumber }\nfn add() {}"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labelSet := make(map[string]protocol.CompletionItem, len(items))
	for _, item := range items {
		labelSet[item.Label] = item
	}

	// Attribute should appear with '@' prefix and snippet insertText.
	attr, ok := labelSet["@Numeric"]
	if !ok {
		t.Fatalf("expected @Numeric in non-attribute completion, got: %v", labelKeys(labelSet))
	}
	wantInsert := "@Numeric(${1:toNumber})"
	if attr.InsertText == nil || *attr.InsertText != wantInsert {
		t.Errorf("@Numeric insertText = %v, want %q", attr.InsertText, wantInsert)
	}
	if attr.InsertTextFormat == nil || *attr.InsertTextFormat != protocol.InsertTextFormatSnippet {
		t.Errorf("@Numeric insertTextFormat should be Snippet")
	}

	// Regular function should appear without '@'.
	if _, ok := labelSet["add"]; !ok {
		t.Errorf("expected func add in completion, got: %v", labelKeys(labelSet))
	}
}

func TestIsAttributeContext(t *testing.T) {
	tests := []struct {
		name string
		text string
		pos  protocol.Position
		want bool
	}{
		{"at-sign only", "@", protocol.Position{Line: 0, Character: 1}, true},
		{"at-sign with partial name", "@Num", protocol.Position{Line: 0, Character: 4}, true},
		{"plain name no at", "Num", protocol.Position{Line: 0, Character: 3}, false},
		{"space then at name", "  @Foo", protocol.Position{Line: 0, Character: 6}, true},
		{"at-sign on second line", "const x = 0\n@Bar", protocol.Position{Line: 1, Character: 4}, true},
		{"normal code on second line", "const x = 0\nfn", protocol.Position{Line: 1, Character: 2}, false},
		{"cursor at start", "@Foo", protocol.Position{Line: 0, Character: 0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAttributeContext(tt.text, tt.pos)
			if got != tt.want {
				t.Errorf("isAttributeContext(%q, {%d,%d}) = %v, want %v",
					tt.text, tt.pos.Line, tt.pos.Character, got, tt.want)
			}
		})
	}
}

func labelKeys(m map[string]protocol.CompletionItem) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func labelKeys2(items []protocol.CompletionItem) []string {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, item.Label)
	}
	return labels
}

func TestLocalParamCompletion(t *testing.T) {
	// Cursor inside a function body — params should be suggested.
	// "fn greet(name) {\n  |cursor|\n}"
	src := "fn greet(name) {\n  \n}"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Position on line 1 (inside the function body)
	pos := protocol.Position{Line: 1, Character: 2}
	items, err := ls.completionItemsForFile("main.zirr", pos)
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}
	labelSet := make(map[string]bool, len(items))
	for _, item := range items {
		labelSet[item.Label] = true
	}
	if !labelSet["name"] {
		t.Errorf("expected param 'name' in completion, got: %v", labelKeys2(items))
	}
}

func TestLocalLetCompletion(t *testing.T) {
	// Local let binding should be suggested after it is declared.
	// "fn f() {\n  const x = 1\n  |cursor|\n}"
	src := "fn f() {\n  const x = 1\n  \n}"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor on line 2 (after let x)
	pos := protocol.Position{Line: 2, Character: 2}
	items, err := ls.completionItemsForFile("main.zirr", pos)
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}
	labelSet := make(map[string]bool, len(items))
	for _, item := range items {
		labelSet[item.Label] = true
	}
	if !labelSet["x"] {
		t.Errorf("expected local 'x' in completion, got: %v", labelKeys2(items))
	}
	// The global function 'f' should also still be present
	if !labelSet["f"] {
		t.Errorf("expected global 'f' in completion alongside locals, got: %v", labelKeys2(items))
	}
}

func TestLocalLetNotBeforeDecl(t *testing.T) {
	// A local let should NOT appear before its declaration.
	// "fn f() {\n  |cursor|\n  const x = 1\n}"
	src := "fn f() {\n  \n  const x = 1\n}"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor on line 1 (BEFORE let x)
	pos := protocol.Position{Line: 1, Character: 2}
	items, err := ls.completionItemsForFile("main.zirr", pos)
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}
	for _, item := range items {
		if item.Label == "x" {
			t.Errorf("local 'x' should not appear before its declaration")
		}
	}
}

func TestQualifiedContext(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		pos          protocol.Position
		wantAlias    string
		wantOk       bool
		wantAfterDot uint32
	}{
		{
			name:         "simple qualified",
			text:         "mymod.Foo",
			pos:          protocol.Position{Line: 0, Character: 9},
			wantAlias:    "mymod",
			wantOk:       true,
			wantAfterDot: 6,
		},
		{
			name:         "cursor at dot",
			text:         "mymod.",
			pos:          protocol.Position{Line: 0, Character: 6},
			wantAlias:    "mymod",
			wantOk:       true,
			wantAfterDot: 6,
		},
		{
			name:   "plain identifier",
			text:   "Foo",
			pos:    protocol.Position{Line: 0, Character: 3},
			wantOk: false,
		},
		{
			name:         "attribute qualified",
			text:         "@mymod.Attribute",
			pos:          protocol.Position{Line: 0, Character: 17},
			wantAlias:    "mymod",
			wantOk:       true,
			wantAfterDot: 7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alias, afterDot, ok := qualifiedContext(tt.text, tt.pos)
			if ok != tt.wantOk {
				t.Fatalf("qualifiedContext ok=%v want %v", ok, tt.wantOk)
			}
			if !ok {
				return
			}
			if alias != tt.wantAlias {
				t.Errorf("alias=%q want %q", alias, tt.wantAlias)
			}
			if afterDot.Character != tt.wantAfterDot {
				t.Errorf("afterDot.Character=%d want %d", afterDot.Character, tt.wantAfterDot)
			}
		})
	}
}

func TestImportAliasCompletion(t *testing.T) {
	// import aliases should appear in completion for the current file.
	base := memfs.New()
	if err := base.MkdirAll("mymod", 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, base, "mymod/types.zirr", "fn helper() {}")
	writeFile(t, base, "main.zirr", "import mymod\n")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 1})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}
	labelSet := make(map[string]bool, len(items))
	for _, item := range items {
		labelSet[item.Label] = true
	}
	if !labelSet["mymod"] {
		t.Errorf("expected import alias 'mymod' in completion, got: %v", labelKeys2(items))
	}
}

func TestQualifiedModuleCompletion(t *testing.T) {
	// When user types "mymod.", completions should show members of mymod.
	base := memfs.New()
	if err := base.MkdirAll("mymod", 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, base, "mymod/types.zirr", "fn helper() {}\ndata Point { x }")
	// main.zirr with "mymod." at cursor position (line 1, col 6)
	writeFile(t, base, "main.zirr", "import mymod\nmymod.")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor right after the dot
	pos := protocol.Position{Line: 1, Character: 6}
	items, err := ls.completionItemsForFile("main.zirr", pos)
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}
	labelSet := make(map[string]bool, len(items))
	for _, item := range items {
		labelSet[item.Label] = true
	}
	if !labelSet["helper"] {
		t.Errorf("expected 'helper' from imported module, got: %v", labelKeys2(items))
	}
	if !labelSet["Point"] {
		t.Errorf("expected 'Point' from imported module, got: %v", labelKeys2(items))
	}
}
