package langsrv

import (
	"strings"
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
			wantKinds:  map[string]protocol.CompletionItemKind{"Point": protocol.CompletionItemKindConstructor},
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

	attr, ok := labelSet["@Numeric"]
	if !ok {
		t.Fatalf("expected @Numeric attribute in completion, got: %v", labelKeys(labelSet))
	}
	textEdit, ok := attr.TextEdit.(protocol.TextEdit)
	if !ok {
		t.Fatalf("@Numeric: expected TextEdit, got %T", attr.TextEdit)
	}
	wantNewText := "@Numeric($0)"
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

	// Data and union types must NOT appear in @ attribute context.
	if _, ok := labelSet["@Foo"]; ok {
		t.Errorf("data type @Foo must not appear in attribute completion (got: %v)", labelKeys(labelSet))
	}
	if _, ok := labelSet["@Bar"]; ok {
		t.Errorf("union type @Bar must not appear in attribute completion (got: %v)", labelKeys(labelSet))
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

	attr, ok := labelSet["@Numeric"]
	if !ok {
		t.Fatalf("expected @Numeric in non-attribute completion, got: %v", labelKeys(labelSet))
	}
	wantInsert := "@Numeric($0)"
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

func TestForLoopBindingCompletion(t *testing.T) {
	// for pers <- [peter] { pers should complete here }
	src := "fn test() {\n  for pers <- [1] {\n    \n  }\n}"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor inside the for body (line 2, col 4)
	pos := protocol.Position{Line: 2, Character: 4}
	items, err := ls.completionItemsForFile("main.zirr", pos)
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}
	labelSet := make(map[string]bool, len(items))
	for _, item := range items {
		labelSet[item.Label] = true
	}
	if !labelSet["pers"] {
		t.Errorf("expected for-loop binding 'pers' in completion, got: %v", labelKeys2(items))
	}
}

func TestLocalVarInsideIfBlockCompletion(t *testing.T) {
	// Locals declared inside an if-block should be visible within it.
	src := "fn test() {\n  if true {\n    var x = 1\n    \n  }\n}"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor on line 3, inside if body, after var x
	pos := protocol.Position{Line: 3, Character: 4}
	items, err := ls.completionItemsForFile("main.zirr", pos)
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}
	labelSet := make(map[string]bool, len(items))
	for _, item := range items {
		labelSet[item.Label] = true
	}
	if !labelSet["x"] {
		t.Errorf("expected local 'x' inside if-block in completion, got: %v", labelKeys2(items))
	}
}

func TestKeywordCompletions(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "fn greet() {}")

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

	// Top-level keywords should include declaration keywords but NOT return/break/continue.
	wantKeywords := []string{"fn", "var", "const", "data", "union", "if", "for", "switch", "true", "false", "void", "import", "extern type", "extern fn", "extern const", "attr"}
	for _, kw := range wantKeywords {
		item, ok := labelSet[kw]
		if !ok {
			t.Errorf("expected keyword %q in completion, got labels: %v", kw, labelKeys(labelSet))
			continue
		}
		if item.Kind == nil || *item.Kind != protocol.CompletionItemKindKeyword {
			var got protocol.CompletionItemKind
			if item.Kind != nil {
				got = *item.Kind
			}
			t.Errorf("keyword %q: got kind %v, want Keyword (%v)", kw, got, protocol.CompletionItemKindKeyword)
		}
	}

	// return/break/continue should NOT appear at top level.
	notWantAtTopLevel := []string{"return", "break", "continue"}
	for _, kw := range notWantAtTopLevel {
		if _, ok := labelSet[kw]; ok {
			t.Errorf("keyword %q should NOT appear at top level", kw)
		}
	}
}

func TestKeywordCompletionsInsideFunc(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "fn greet() {\n  \n}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Inside function body.
	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 1, Character: 2})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labelSet := make(map[string]bool, len(items))
	for _, item := range items {
		labelSet[item.Label] = true
	}

	// return should appear inside functions.
	if !labelSet["return"] {
		t.Error("expected 'return' keyword inside function body")
	}
	// break/continue should NOT appear (not inside for loop).
	if labelSet["break"] {
		t.Error("'break' should NOT appear in function without for loop")
	}
	// Declaration-only keywords should NOT appear inside functions.
	if labelSet["data"] {
		t.Error("'data' should NOT appear inside function body")
	}
}

func TestKeywordCompletionsInsideFor(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "fn test() {\n  for x <- [1] {\n    \n  }\n}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Inside for loop body.
	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 2, Character: 4})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labelSet := make(map[string]bool, len(items))
	for _, item := range items {
		labelSet[item.Label] = true
	}

	// break/continue should appear inside for loop.
	if !labelSet["break"] {
		t.Error("expected 'break' keyword inside for loop")
	}
	if !labelSet["continue"] {
		t.Error("expected 'continue' keyword inside for loop")
	}
	// return should also appear (inside function).
	if !labelSet["return"] {
		t.Error("expected 'return' keyword inside for loop (which is inside a function)")
	}
}

func TestModKeywordOnlyAtFirstPosition(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Empty file, cursor at start — first position: 'mod' should appear.
	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}
	labelSet := make(map[string]bool, len(items))
	for _, item := range items {
		labelSet[item.Label] = true
	}
	if !labelSet["mod"] {
		t.Error("expected 'mod' keyword at first position in empty file")
	}

	// After a declaration, 'mod' should NOT appear.
	base2 := memfs.New()
	writeFile(t, base2, "main.zirr", "fn greet() {}\n")
	ls2 := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls2.setFilesystem(base2, "/")

	items2, err := ls2.completionItemsForFile("main.zirr", protocol.Position{Line: 1, Character: 0})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}
	labelSet2 := make(map[string]bool, len(items2))
	for _, item := range items2 {
		labelSet2[item.Label] = true
	}
	if labelSet2["mod"] {
		t.Error("'mod' should NOT appear after declarations")
	}
}

func TestDataConstructorCompletion(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "data Point { x y }\nfn test() {\n  \n}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Inside function body — data type should appear as constructor.
	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 2, Character: 2})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "Point" {
			if item.Kind == nil || *item.Kind != protocol.CompletionItemKindConstructor {
				t.Errorf("Point inside function should be Constructor kind")
			}
			if item.InsertText == nil || !strings.Contains(*item.InsertText, "Point(") {
				t.Errorf("Point constructor should have parens in insert text, got: %v", item.InsertText)
			}
			return
		}
	}
	t.Error("expected 'Point' in completions inside function body")
}

func TestFuncCompletionWithParens(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "fn greet() {}\nfn test() {\n  \n}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Inside function body — functions should complete with parens.
	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 2, Character: 2})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "greet" {
			if item.InsertText == nil || !strings.Contains(*item.InsertText, "greet(") {
				t.Errorf("function completion should have parens, got: %v", item.InsertText)
			}
			return
		}
	}
	t.Error("expected 'greet' in completions")
}

func TestKeywordCompletionsNotInAttributeContext(t *testing.T) {
	src := "attr Test {}\n@T"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Position after '@T' — attribute context
	pos := protocol.Position{Line: 1, Character: 2}
	items, err := ls.completionItemsForFile("main.zirr", pos)
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Kind != nil && *item.Kind == protocol.CompletionItemKindKeyword {
			t.Errorf("keyword %q should not appear in attribute context", item.Label)
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

// TestPreludeSymbolCompletion verifies that prelude symbols like String, Int, Bool appear in completions when Orchestra is available.
func TestPreludeSymbolCompletion(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "const x = 1\n")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	if ls.orch == nil {
		t.Fatal("expected Orchestra to be initialized")
	}

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labelSet := make(map[string]bool, len(items))
	for _, item := range items {
		labelSet[item.Label] = true
	}

	// Prelude types should be present
	for _, name := range []string{"String", "Int", "Bool", "Array"} {
		if !labelSet[name] {
			t.Errorf("expected prelude symbol %q in completions, got: %v", name, labelKeys2(items))
		}
	}

	// User-declared symbol should also be present
	if !labelSet["x"] {
		t.Errorf("expected user symbol 'x' in completions")
	}
}

// TestPreludeSymbolsNotDuplicated verifies that prelude symbols don't appear twice when the user has an explicit prelude import.
func TestPreludeSymbolsNotDuplicated(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "const x = String\n")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	count := 0
	for _, item := range items {
		if item.Label == "String" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected String to appear exactly once, got %d", count)
	}
}

func TestBreakContinueInGlobalFor(t *testing.T) {
	// break/continue should appear inside a top-level for loop (not inside a function).
	base := memfs.New()
	writeFile(t, base, "main.zirr", "for x <- [1] {\n  \n}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 1, Character: 2})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labelSet := make(map[string]bool, len(items))
	for _, item := range items {
		labelSet[item.Label] = true
	}

	if !labelSet["break"] {
		t.Error("expected 'break' keyword inside global for loop")
	}
	if !labelSet["continue"] {
		t.Error("expected 'continue' keyword inside global for loop")
	}
	// return should NOT appear (not inside a function).
	if labelSet["return"] {
		t.Error("'return' should NOT appear inside global for loop (no enclosing function)")
	}
}

func TestExternKeywordSnippets(t *testing.T) {
	// extern type, extern fn, extern const should appear as separate snippet items.
	base := memfs.New()
	writeFile(t, base, "main.zirr", "")

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

	labelMap := make(map[string]protocol.CompletionItem)
	for _, item := range items {
		labelMap[item.Label] = item
	}

	for _, kw := range []string{"extern type", "extern fn", "extern const"} {
		item, ok := labelMap[kw]
		if !ok {
			t.Errorf("expected %q in completions", kw)
			continue
		}
		if item.Kind == nil || *item.Kind != protocol.CompletionItemKindKeyword {
			t.Errorf("%q should be Keyword kind", kw)
		}
		if item.InsertTextFormat == nil || *item.InsertTextFormat != protocol.InsertTextFormatSnippet {
			t.Errorf("%q should be Snippet insertTextFormat", kw)
		}
		if item.InsertText == nil {
			t.Errorf("%q should have InsertText", kw)
		}
	}

	// Bare "extern" should NOT appear.
	if _, ok := labelMap["extern"]; ok {
		t.Error("bare 'extern' keyword should NOT appear; use 'extern type', 'extern fn', 'extern const'")
	}
}

func TestImportBlockCompletions(t *testing.T) {
	// Completions inside import { } should only show members of the imported module.
	base := memfs.New()
	writeFile(t, base, "mymod/greet.zirr", "mod mymod\nfn greet() {}\ndata Person { name }")

	writeFile(t, base, "main.zirr", "import mymod {\n  greet\n  \n}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor on line 2, inside the import { } block.
	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 2, Character: 2})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labelSet := make(map[string]bool, len(items))
	for _, item := range items {
		labelSet[item.Label] = true
	}

	// Person should be offered (not yet imported).
	if !labelSet["Person"] {
		t.Error("expected 'Person' in import block completions (not yet imported)")
	}
	// greet should NOT be offered (already imported).
	if labelSet["greet"] {
		t.Error("'greet' should NOT appear in import block (already imported)")
	}
}

func TestAttrOnlyInAttributeContext(t *testing.T) {
	// In @ context, only attr declarations should appear — NOT data or union.
	src := "attr Validated { }\ndata Point { x }\nunion Shape { Point }\n@V"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor after "@V" on line 3.
	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 3, Character: 2})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		label := item.Label
		if label == "Point" || label == "@Point" || label == "Shape" || label == "@Shape" {
			t.Errorf("data/union %q should NOT appear in @ attribute context", label)
		}
	}

	// @Validated should appear.
	found := false
	for _, item := range items {
		if item.Label == "@Validated" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected '@Validated' in attribute completion")
	}
}

func TestAttrAlwaysHasParensInDeclContext(t *testing.T) {
	// @Attr before a declaration should always include () even with zero fields.
	src := "attr NoArgs { }\nattr WithArgs { a b }\n@"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 2, Character: 1})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "@NoArgs" {
			te, ok := item.TextEdit.(protocol.TextEdit)
			if !ok {
				t.Fatalf("@NoArgs should have TextEdit")
			}
			if !strings.Contains(te.NewText, "(") {
				t.Errorf("@NoArgs should include () in insert text, got %q", te.NewText)
			}
		}
		if item.Label == "@WithArgs" {
			te, ok := item.TextEdit.(protocol.TextEdit)
			if !ok {
				t.Fatalf("@WithArgs should have TextEdit")
			}
			wantText := "@WithArgs($0)"
			if te.NewText != wantText {
				t.Errorf("@WithArgs TextEdit.NewText = %q, want %q", te.NewText, wantText)
			}
		}
	}
}

func TestAttrNoParensInTypeExpr(t *testing.T) {
	// @Attr inside a type expression should never include ().
	// Use a completed data block with cursor after ':' on a separate line.
	src := "attr Tag { }\nfn greet(name: @"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 1, Character: 16})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "@Tag" {
			if item.InsertText != nil && strings.Contains(*item.InsertText, "(") {
				t.Errorf("@Tag in type expr should NOT have parens, got %q", *item.InsertText)
			}
			return
		}
	}
	t.Error("expected '@Tag' in type expr completions")
}

func TestGlobalForIterVarCompletion(t *testing.T) {
	// Iteration variable from a global for loop should be available inside the loop body.
	base := memfs.New()
	writeFile(t, base, "main.zirr", "for item <- [1] {\n  \n}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 1, Character: 2})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "item" {
			if item.Kind == nil || *item.Kind != protocol.CompletionItemKindVariable {
				t.Errorf("'item' should be Variable kind")
			}
			return
		}
	}
	t.Error("expected 'item' (for-loop iteration variable) in completions")
}

func TestImportBlockAttrNoAtPrefix(t *testing.T) {
	// Attrs in import block completions should appear without @ prefix.
	base := memfs.New()
	writeFile(t, base, "mymod/types.zirr", "mod mymod\nattr Tag { }\ndata Point { x }")
	writeFile(t, base, "main.zirr", "import mymod {\n  \n}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 1, Character: 2})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if strings.HasPrefix(item.Label, "@") {
			t.Errorf("import block should NOT have @ prefix, got %q", item.Label)
		}
	}

	labelSet := make(map[string]bool)
	for _, item := range items {
		labelSet[item.Label] = true
	}
	if !labelSet["Tag"] {
		t.Error("expected 'Tag' (attr) in import block completions without @ prefix")
	}
	if !labelSet["Point"] {
		t.Error("expected 'Point' in import block completions")
	}
}

func TestCompletionParensDropParams(t *testing.T) {
	// Data constructors and functions should use ($0) not (${1:param}, ...).
	base := memfs.New()
	writeFile(t, base, "main.zirr", "data Point { x y }\nfn greet(name) {}\nfn test() {\n  \n}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 3, Character: 2})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "Point" {
			want := "Point($0)"
			if item.InsertText == nil || *item.InsertText != want {
				got := ""
				if item.InsertText != nil {
					got = *item.InsertText
				}
				t.Errorf("Point insertText = %q, want %q", got, want)
			}
		}
		if item.Label == "greet" {
			want := "greet($0)"
			if item.InsertText == nil || *item.InsertText != want {
				got := ""
				if item.InsertText != nil {
					got = *item.InsertText
				}
				t.Errorf("greet insertText = %q, want %q", got, want)
			}
		}
	}
}

func TestTypeExprAttrNoDoubleAt(t *testing.T) {
	// Completing @Attr in type expression should NOT produce @@Attr.
	// The TextEdit must replace from the existing '@' position.
	src := "attr Tag { }\nfn greet(person: @"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 1, Character: 18})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "@Tag" {
			te, ok := item.TextEdit.(protocol.TextEdit)
			if !ok {
				t.Fatalf("@Tag in type expr should have TextEdit, got insertText")
			}
			if te.NewText != "@Tag" {
				t.Errorf("@Tag TextEdit.NewText = %q, want %q", te.NewText, "@Tag")
			}
			// The edit range should start at the '@' (char 17) and end at cursor (char 18).
			if te.Range.Start.Character != 17 {
				t.Errorf("TextEdit start char = %d, want 17 (position of @)", te.Range.Start.Character)
			}
			return
		}
	}
	t.Error("expected '@Tag' in type expr completions")
}

func TestTypeExprAttrChainCompletion(t *testing.T) {
	// After `: @Attr1 @`, the second @ should still be in type expr context.
	src := "attr Tag { }\nattr Validated { }\nfn greet(person: @Tag @"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 2, Character: 23})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "@Validated" {
			te, ok := item.TextEdit.(protocol.TextEdit)
			if !ok {
				t.Fatal("@Validated should have TextEdit in type expr chain")
			}
			// Must NOT contain () — this is type expr context.
			if strings.Contains(te.NewText, "(") {
				t.Errorf("@Validated in type expr chain should NOT have parens, got %q", te.NewText)
			}
			return
		}
	}
	t.Error("expected '@Validated' in type expr attr chain completions")
}

func TestAttrChainExcludesUsedAttrs(t *testing.T) {
	// When completing after @Tag @, Tag should not appear again.
	src := "attr Tag { }\nattr Validated { }\nattr Required { }\nfn greet(person: @Tag @"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 3, Character: 23})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "@Tag" {
			t.Error("@Tag should NOT appear — it is already in the chain")
		}
	}

	// Validated and Required should still appear.
	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}
	if !labels["@Validated"] {
		t.Error("expected @Validated in attr chain completions")
	}
	if !labels["@Required"] {
		t.Error("expected @Required in attr chain completions")
	}
}

func TestAttrChainExcludesMultipleUsed(t *testing.T) {
	// When completing after @Tag @Validated @, both Tag and Validated should be excluded.
	src := "attr Tag { }\nattr Validated { }\nattr Required { }\nfn greet(person: @Tag @Validated @"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 3, Character: 35})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "@Tag" {
			t.Error("@Tag should NOT appear — already used in chain")
		}
		if item.Label == "@Validated" {
			t.Error("@Validated should NOT appear — already used in chain")
		}
	}

	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}
	if !labels["@Required"] {
		t.Error("expected @Required — not yet used in chain")
	}
}

func TestDeclAttrChainExcludesUsed(t *testing.T) {
	// Declaration-level attrs on separate lines: @Tag() and @Validated() already used.
	src := "attr Tag { }\nattr Validated { }\nattr Required { }\n@Tag()\n@Validated()\n@"
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 5, Character: 1})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "@Tag" {
			t.Error("@Tag should NOT appear — already used in declaration chain")
		}
		if item.Label == "@Validated" {
			t.Error("@Validated should NOT appear — already used in declaration chain")
		}
	}

	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}
	if !labels["@Required"] {
		t.Error("expected @Required — not yet used in declaration chain")
	}
}

func TestQualifiedAttrNoParensInTypeExpr(t *testing.T) {
	// @alias.Attr in a type expression should NOT get parens.
	base := memfs.New()
	writeFile(t, base, "lib/lib.zirr", "attr AnyOption { }")
	writeFile(t, base, "main.zirr", "import xprelude = lib\nfn greet(person: @xprelude.")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 1, Character: 27})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "AnyOption" {
			te, ok := item.TextEdit.(protocol.TextEdit)
			if !ok {
				t.Fatal("AnyOption should have TextEdit")
			}
			if strings.Contains(te.NewText, "(") {
				t.Errorf("qualified attr in type expr should NOT have parens, got %q", te.NewText)
			}
			return
		}
	}
	t.Error("expected AnyOption in qualified module attr completions")
}

func TestAtSignCompletesQualifiedAttrsFromImports(t *testing.T) {
	// Typing @ should also suggest @lib.SomeAttr for attrs in imported modules.
	base := memfs.New()
	writeFile(t, base, "lib/lib.zirr", "attr SomeAttr { }\nattr OtherAttr { }\ndata NotAnAttr { }")
	writeFile(t, base, "main.zirr", "import mylib = lib\n@")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 1, Character: 1})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}

	if !labels["@mylib.SomeAttr"] {
		t.Error("expected @mylib.SomeAttr in qualified attr completions")
	}
	if !labels["@mylib.OtherAttr"] {
		t.Error("expected @mylib.OtherAttr in qualified attr completions")
	}
	if labels["@mylib.NotAnAttr"] {
		t.Error("@mylib.NotAnAttr should NOT appear — it's a data type, not an attr")
	}

	for _, item := range items {
		if item.Label == "@mylib.SomeAttr" {
			te, ok := item.TextEdit.(protocol.TextEdit)
			if !ok {
				t.Fatal("@mylib.SomeAttr should have TextEdit")
			}
			if !strings.Contains(te.NewText, "(") {
				t.Errorf("qualified attr in decl context should have parens, got %q", te.NewText)
			}
		}
	}
}

func TestAtSignCompletesQualifiedAttrsTypeExprNoParens(t *testing.T) {
	// Typing : @ should suggest @lib.SomeAttr without parens.
	base := memfs.New()
	writeFile(t, base, "lib/lib.zirr", "attr SomeAttr { }")
	writeFile(t, base, "main.zirr", "import mylib = lib\nfn greet(person: @")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 1, Character: 18})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "@mylib.SomeAttr" {
			te, ok := item.TextEdit.(protocol.TextEdit)
			if !ok {
				t.Fatal("@mylib.SomeAttr should have TextEdit")
			}
			if strings.Contains(te.NewText, "(") {
				t.Errorf("qualified attr in type expr should NOT have parens, got %q", te.NewText)
			}
			return
		}
	}
	t.Error("expected @mylib.SomeAttr in qualified attr completions for type expr")
}

func TestQualifiedAttrHasParensInDeclContext(t *testing.T) {
	// @alias.Attr in a declaration context SHOULD get parens.
	base := memfs.New()
	writeFile(t, base, "lib/lib.zirr", "attr AnyOption { }")
	writeFile(t, base, "main.zirr", "import xprelude = lib\n@xprelude.")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 1, Character: 10})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "AnyOption" {
			te, ok := item.TextEdit.(protocol.TextEdit)
			if !ok {
				t.Fatal("AnyOption should have TextEdit")
			}
			if !strings.Contains(te.NewText, "(") {
				t.Errorf("qualified attr in decl context should have parens, got %q", te.NewText)
			}
			return
		}
	}
	t.Error("expected AnyOption in qualified module attr completions")
}

func TestQualifiedModuleMemberContextAwareCompletions(t *testing.T) {
	// alias.member completions should add parens for functions and data constructors.
	base := memfs.New()
	writeFile(t, base, "lib/lib.zirr", "fn helper() {}\ndata Point { x }\nunion Shape { Point }\nattr Tag { }\nconst pi = 3")
	writeFile(t, base, "main.zirr", "import mylib = lib\nmylib.")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 1, Character: 6})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	itemMap := make(map[string]protocol.CompletionItem)
	for _, item := range items {
		itemMap[item.Label] = item
	}

	// Function should have parens.
	if item, ok := itemMap["helper"]; ok {
		te, isTE := item.TextEdit.(protocol.TextEdit)
		if isTE {
			if !strings.Contains(te.NewText, "($0)") {
				t.Errorf("helper should have ($0) parens, got %q", te.NewText)
			}
		} else if item.InsertText != nil && !strings.Contains(*item.InsertText, "($0)") {
			t.Errorf("helper should have ($0) parens, got %q", *item.InsertText)
		}
	} else {
		t.Error("expected 'helper' in qualified completions")
	}

	// Data constructor should have parens.
	if item, ok := itemMap["Point"]; ok {
		te, isTE := item.TextEdit.(protocol.TextEdit)
		if isTE {
			if !strings.Contains(te.NewText, "($0)") {
				t.Errorf("Point should have ($0) parens, got %q", te.NewText)
			}
		} else if item.InsertText != nil && !strings.Contains(*item.InsertText, "($0)") {
			t.Errorf("Point should have ($0) parens, got %q", *item.InsertText)
		}
	} else {
		t.Error("expected 'Point' in qualified completions")
	}

	// Union should be plain (no parens).
	if item, ok := itemMap["Shape"]; ok {
		te, isTE := item.TextEdit.(protocol.TextEdit)
		if isTE && strings.Contains(te.NewText, "(") {
			t.Errorf("Shape (union) should NOT have parens, got %q", te.NewText)
		}
	} else {
		t.Error("expected 'Shape' in qualified completions")
	}

	// Attr should be plain (no @, no parens) in non-attribute context.
	if item, ok := itemMap["Tag"]; ok {
		te, isTE := item.TextEdit.(protocol.TextEdit)
		if isTE && strings.Contains(te.NewText, "@") {
			t.Errorf("Tag in non-@ context should not have @, got %q", te.NewText)
		}
	} else {
		t.Error("expected 'Tag' in qualified completions")
	}

	// Constant should be plain.
	if item, ok := itemMap["pi"]; ok {
		te, isTE := item.TextEdit.(protocol.TextEdit)
		if isTE && strings.Contains(te.NewText, "(") {
			t.Errorf("pi (const) should NOT have parens, got %q", te.NewText)
		}
	} else {
		t.Error("expected 'pi' in qualified completions")
	}
}

func TestModKeywordCompletesWithDirName(t *testing.T) {
	// When completing 'mod' in a subfolder, the insert text should use the folder name.
	base := memfs.New()
	writeFile(t, base, "myproject/main.zirr", "")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("myproject/main.zirr", protocol.Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "mod" {
			if item.InsertText == nil {
				t.Fatal("mod should have InsertText")
			}
			if *item.InsertText != "mod ${1:myproject}" {
				t.Errorf("mod InsertText should be 'mod ${1:myproject}', got %q", *item.InsertText)
			}
			return
		}
	}
	t.Error("expected 'mod' in completions")
}

func TestModKeywordSanitizesDirName(t *testing.T) {
	// Directory names with hyphens/spaces should be sanitized to underscores.
	base := memfs.New()
	writeFile(t, base, "my-project/main.zirr", "")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("my-project/main.zirr", protocol.Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "mod" {
			if item.InsertText == nil {
				t.Fatal("mod should have InsertText")
			}
			if *item.InsertText != "mod ${1:my_project}" {
				t.Errorf("mod InsertText should be 'mod ${1:my_project}', got %q", *item.InsertText)
			}
			return
		}
	}
	t.Error("expected 'mod' in completions")
}

func TestModNameDotCompletesModuleMembers(t *testing.T) {
	// When typing "mymod.", completions should show members of the current module.
	base := memfs.New()
	writeFile(t, base, "mymod/main.zirr", "mod mymod\nfn helper() {}\ndata Point { x }\nattr Tag { }\nmymod.")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("mymod/main.zirr", protocol.Position{Line: 4, Character: 6})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}

	if !labels["helper"] {
		t.Error("expected 'helper' in modname. completions")
	}
	if !labels["Point"] {
		t.Error("expected 'Point' in modname. completions")
	}
	if !labels["Tag"] {
		t.Error("expected 'Tag' in modname. completions")
	}
	// The module declaration itself should NOT appear.
	if labels["mymod"] {
		t.Error("'mymod' (the mod decl) should NOT appear as a member")
	}
}

func TestAtModNameDotCompletesAttrsOnly(t *testing.T) {
	// When typing "@mymod.", only attrs should appear.
	base := memfs.New()
	writeFile(t, base, "mymod/main.zirr", "mod mymod\nfn helper() {}\ndata Point { x }\nattr Tag { }\n@mymod.")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("mymod/main.zirr", protocol.Position{Line: 4, Character: 7})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}

	if !labels["Tag"] {
		t.Error("expected 'Tag' in @modname. completions")
	}
	if labels["helper"] {
		t.Error("'helper' should NOT appear in @modname. context")
	}
	if labels["Point"] {
		t.Error("'Point' should NOT appear in @modname. context")
	}
}

func TestModNameCompletesAsSymbol(t *testing.T) {
	// The modname itself should appear in completions (not just modname.members).
	base := memfs.New()
	writeFile(t, base, "mymod/main.zirr", "mod mymod\nfn helper() {}\n")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("mymod/main.zirr", protocol.Position{Line: 1, Character: 16})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "mymod" {
			return
		}
	}
	t.Error("expected 'mymod' in completions")
}

func TestModuleMemberNoParensInTypeExpr(t *testing.T) {
	// In type-expr context (after :), module.Data should NOT have parens.
	base := memfs.New()
	writeFile(t, base, "lib/lib.zirr", "fn helper() {}\ndata Point { x }")
	writeFile(t, base, "main.zirr", "import mylib = lib\nfn greet(p: mylib.")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	items, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 1, Character: 18})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}

	for _, item := range items {
		if item.Label == "Point" {
			te, ok := item.TextEdit.(protocol.TextEdit)
			if !ok {
				t.Fatal("Point should have TextEdit")
			}
			if strings.Contains(te.NewText, "(") {
				t.Errorf("Point in type-expr should NOT have parens, got %q", te.NewText)
			}
		}
		if item.Label == "helper" {
			te, ok := item.TextEdit.(protocol.TextEdit)
			if !ok {
				t.Fatal("helper should have TextEdit")
			}
			if strings.Contains(te.NewText, "(") {
				t.Errorf("helper in type-expr should NOT have parens, got %q", te.NewText)
			}
		}
	}
}
