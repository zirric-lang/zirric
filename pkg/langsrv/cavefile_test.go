package langsrv

import (
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// cavefileTestSrc mirrors a real Cavefile declaring a Git dependency via attributes, exercising the "import cave" alias, the "@cave.Dependencies()" decorator, and a nested "@cave.Git(...)"/"@cave.Version(...)" pair.
const cavefileTestSrc = "mod ui\n\nimport cave\n\n@cave.Dependencies()\ndata UI {\n\t@cave.Git(\"https://code.knabel.dev/zirric-lang/colors\")\n\t@cave.Version(\"main\")\n\tcolors\n}\n"

func newCavefileTestServer(t *testing.T) *zirricLangserver {
	t.Helper()
	base := memfs.New()
	writeFile(t, base, "Cavefile", cavefileTestSrc)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")
	return ls
}

// TestCavefileHover is a regression test: parseModuleFiles' directory scan only ever globs *.zirr files, so it silently skipped the Cavefile itself (no such extension), leaving hover blind to its own content — hovering "cave" in "@cave.Dependencies()" used to return nil.
func TestCavefileHover(t *testing.T) {
	ls := newCavefileTestServer(t)

	params := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///Cavefile"},
			Position:     protocol.Position{Line: 4, Character: 2}, // on "cave" in "@cave.Dependencies()"
		},
	}
	result, err := ls.textDocumentHover(nil, params)
	if err != nil {
		t.Fatalf("textDocumentHover: %v", err)
	}
	if result == nil {
		t.Fatal("expected hover content for \"cave\" inside the Cavefile, got nil")
	}
	mc, ok := result.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected MarkupContent, got %T", result.Contents)
	}
	if !strings.Contains(mc.Value, "cave") {
		t.Errorf("hover content %q does not contain 'cave'", mc.Value)
	}
}

// TestCavefileDefinition is a regression test for the same gap as TestCavefileHover: go-to-definition on "Dependencies" in "@cave.Dependencies()" used to return nil since the Cavefile was never parsed as part of any module.
func TestCavefileDefinition(t *testing.T) {
	ls := newCavefileTestServer(t)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///Cavefile"},
			Position:     protocol.Position{Line: 4, Character: 10}, // inside "Dependencies" in "@cave.Dependencies()"
		},
	}
	result, err := ls.textDocumentDefinition(nil, params)
	if err != nil {
		t.Fatalf("textDocumentDefinition: %v", err)
	}
	if result == nil {
		t.Fatal("expected a definition location for cave.Dependencies, got nil")
	}
	loc, ok := result.(*protocol.Location)
	if !ok {
		t.Fatalf("expected *protocol.Location, got %T", result)
	}
	if !strings.HasSuffix(string(loc.URI), "manifest.zirr") {
		t.Errorf("expected jump target in cave's manifest.zirr, got %s", loc.URI)
	}
}

// TestCavefileImportNameCompletion checks that completing right after "import " inside the Cavefile still offers module names — this already worked before the fix (it doesn't depend on the Cavefile's own module), kept here as a baseline alongside the attribute-completion regression test below.
func TestCavefileImportNameCompletion(t *testing.T) {
	ls := newCavefileTestServer(t)

	items, err := ls.completionItemsForFile("Cavefile", protocol.Position{Line: 2, Character: 7})
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}
	labels := make(map[string]bool, len(items))
	for _, item := range items {
		labels[item.Label] = true
	}
	if !labels["cave"] {
		t.Errorf("expected import completion to include %q, got %v", "cave", labels)
	}
}

// TestCavefileAttributeCompletion is a regression test: completing after "@cave." inside the Cavefile used to return nothing, since currentSF (the Cavefile's own SourceFile, needed to resolve the "cave" import alias) was never populated.
func TestCavefileAttributeCompletion(t *testing.T) {
	ls := newCavefileTestServer(t)

	items, err := ls.completionItemsForFile("Cavefile", protocol.Position{Line: 4, Character: 6}) // right after "@cave."
	if err != nil {
		t.Fatalf("completionItemsForFile: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected completion items after \"@cave.\", got none")
	}
	labels := make(map[string]bool, len(items))
	for _, item := range items {
		labels[item.Label] = true
	}
	for _, want := range []string{"Dependencies", "Git", "Version"} {
		if !labels[want] {
			t.Errorf("expected \"@cave.\" completion to include %q, got %v", want, labels)
		}
	}
}

// TestCavefileOwnSyntaxErrorIsDiagnosed is a regression test: since the Cavefile was never part of any parsed module, a genuine syntax error inside it (as opposed to one on an import target) never surfaced as a diagnostic.
func TestCavefileOwnSyntaxErrorIsDiagnosed(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "Cavefile", "mod ui\n\nimport cave\n\n@cave.Dependencies()\ndata UI {\n\t@cave.Git(\"x\")\n\t@cave.Version(\n\tcolors\n}\n")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")
	ls.openDocs["Cavefile"] = ls.fileURI("Cavefile")

	rec := &diagnosticRecorder{}
	ctx := &glsp.Context{Notify: rec.Notify}
	if err := ls.refreshDiagnosticsSync(ctx); err != nil {
		t.Fatalf("refresh diagnostics: %v", err)
	}

	diags := rec.DiagnosticsFor(ls.fileURI("Cavefile"))
	if len(diags) == 0 {
		t.Fatal("expected a parse-error diagnostic on the Cavefile's own malformed syntax, got none")
	}
}

// TestCavefileDoesNotShareScopeWithSiblingZirrFiles checks that parseCavefileModule's single-file module is entirely separate from the directory-scan module built for sibling *.zirr files in the same folder: distinct *ast.ContextModule instances, and neither module's own top-level declarations are visible from the other.
func TestCavefileDoesNotShareScopeWithSiblingZirrFiles(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "mod ui\nconst rootValue = 42\nfn rootHelper() {}\n")
	writeFile(t, base, "Cavefile", cavefileTestSrc)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	mainModule, _, _, err := ls.parseModuleFilesForPath("main.zirr")
	if err != nil {
		t.Fatalf("parse main.zirr: %v", err)
	}
	caveModule, _, _, err := ls.parseModuleFilesForPath("Cavefile")
	if err != nil {
		t.Fatalf("parse Cavefile: %v", err)
	}

	if mainModule == caveModule {
		t.Fatal("expected main.zirr and the Cavefile to parse into distinct *ast.ContextModule instances, got the same pointer")
	}
	if _, ok := mainModule.Decls.Symbols["UI"]; ok {
		t.Error("main.zirr's module unexpectedly sees \"UI\", which is only declared in the Cavefile")
	}
	if _, ok := caveModule.Decls.Symbols["rootValue"]; ok {
		t.Error("the Cavefile's module unexpectedly sees \"rootValue\", which is only declared in main.zirr")
	}
	if _, ok := caveModule.Decls.Symbols["rootHelper"]; ok {
		t.Error("the Cavefile's module unexpectedly sees \"rootHelper\", which is only declared in main.zirr")
	}
	if _, ok := mainModule.Decls.Symbols["rootValue"]; !ok {
		t.Error("expected main.zirr's own module to still see its own \"rootValue\" declaration")
	}
	if _, ok := caveModule.Decls.Symbols["UI"]; !ok {
		t.Error("expected the Cavefile's own module to still see its own \"UI\" declaration")
	}

	// Completion inside main.zirr must not offer the Cavefile-only "UI", and vice versa.
	mainItems, err := ls.completionItemsForFile("main.zirr", protocol.Position{Line: 2, Character: 0})
	if err != nil {
		t.Fatalf("completionItemsForFile(main.zirr): %v", err)
	}
	for _, item := range mainItems {
		if item.Label == "UI" {
			t.Error("main.zirr completion unexpectedly offers the Cavefile-only symbol \"UI\"")
		}
	}
	caveItems, err := ls.completionItemsForFile("Cavefile", protocol.Position{Line: 8, Character: 0})
	if err != nil {
		t.Fatalf("completionItemsForFile(Cavefile): %v", err)
	}
	for _, item := range caveItems {
		if item.Label == "rootValue" || item.Label == "rootHelper" {
			t.Errorf("Cavefile completion unexpectedly offers the main.zirr-only symbol %q", item.Label)
		}
	}
}
