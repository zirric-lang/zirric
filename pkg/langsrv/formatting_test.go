package langsrv

import (
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const fmtUnformatted = "fn f() {\n      const x = 1+2\n}\n"
const fmtFormatted = "fn f() {\n\tconst x = 1 + 2\n}\n"

func newFormattingServer(t *testing.T, path, contents string) *zirricLangserver {
	t.Helper()
	base := memfs.New()
	writeFile(t, base, path, contents)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")
	return ls
}

func formatDoc(t *testing.T, ls *zirricLangserver, path string, opts protocol.FormattingOptions) []protocol.TextEdit {
	t.Helper()
	edits, err := ls.textDocumentFormatting(&glsp.Context{}, &protocol.DocumentFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///" + path},
		Options:      opts,
	})
	if err != nil {
		t.Fatalf("textDocumentFormatting: %v", err)
	}
	return edits
}

// applyEdit applies a single TextEdit the way a client would, by line/character.
func applyEdit(t *testing.T, text string, edit protocol.TextEdit) string {
	t.Helper()
	start := offsetForPosition(text, edit.Range.Start)
	end := offsetForPosition(text, edit.Range.End)
	return text[:start] + edit.NewText + text[end:]
}

func TestFormattingProducesOneApplicableEdit(t *testing.T) {
	ls := newFormattingServer(t, "main.zirr", fmtUnformatted)
	ls.docs.Open("main.zirr", 1, fmtUnformatted)

	edits := formatDoc(t, ls, "main.zirr", protocol.FormattingOptions{})
	if len(edits) != 1 {
		t.Fatalf("want exactly one edit, got %d", len(edits))
	}
	if got := applyEdit(t, fmtUnformatted, edits[0]); got != fmtFormatted {
		t.Errorf("applying the edit gave %q, want %q", got, fmtFormatted)
	}
}

// Format-on-save must not mark an already formatted buffer dirty.
func TestFormattingReturnsNoEditsWhenClean(t *testing.T) {
	ls := newFormattingServer(t, "main.zirr", fmtFormatted)
	ls.docs.Open("main.zirr", 1, fmtFormatted)

	if edits := formatDoc(t, ls, "main.zirr", protocol.FormattingOptions{}); len(edits) != 0 {
		t.Errorf("want no edits for formatted source, got %d: %+v", len(edits), edits)
	}
}

// The edit must cover only the changed region, not the whole document.
func TestFormattingEditIsMinimal(t *testing.T) {
	src := "mod sample\n\nfn a() {\n\tconst x = 1\n}\n\nfn b() {\n      const y = 2\n}\n"
	ls := newFormattingServer(t, "main.zirr", src)
	ls.docs.Open("main.zirr", 1, src)

	edits := formatDoc(t, ls, "main.zirr", protocol.FormattingOptions{})
	if len(edits) != 1 {
		t.Fatalf("want one edit, got %d", len(edits))
	}
	if edits[0].Range.Start.Line == 0 {
		t.Errorf("edit should not start at line 0, got %+v", edits[0].Range)
	}
	if strings.Contains(edits[0].NewText, "mod sample") {
		t.Errorf("edit should not resend the unchanged prefix: %q", edits[0].NewText)
	}
}

func TestFormattingHonoursInsertSpaces(t *testing.T) {
	ls := newFormattingServer(t, "main.zirr", fmtUnformatted)
	ls.docs.Open("main.zirr", 1, fmtUnformatted)

	edits := formatDoc(t, ls, "main.zirr", protocol.FormattingOptions{
		"insertSpaces": true,
		"tabSize":      float64(2),
	})
	if len(edits) != 1 {
		t.Fatalf("want one edit, got %d", len(edits))
	}
	got := applyEdit(t, fmtUnformatted, edits[0])
	if want := "fn f() {\n  const x = 1 + 2\n}\n"; got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

// The unsaved buffer is the source of truth, not what is on disk.
func TestFormattingUsesUnsavedBuffer(t *testing.T) {
	ls := newFormattingServer(t, "main.zirr", fmtFormatted)
	ls.docs.Open("main.zirr", 1, fmtUnformatted)

	edits := formatDoc(t, ls, "main.zirr", protocol.FormattingOptions{})
	if len(edits) != 1 {
		t.Fatalf("want one edit for the dirty buffer, got %d", len(edits))
	}
}

// A buffer that cannot be formatted yields no edits and no error, so a half-typed file never raises a dialog on save.
func TestFormattingDeclinesOnUnformattableSource(t *testing.T) {
	broken := "fn f() { const c = 'unterminated\n}\n"
	ls := newFormattingServer(t, "main.zirr", broken)
	ls.docs.Open("main.zirr", 1, broken)

	edits, err := ls.textDocumentFormatting(&glsp.Context{}, &protocol.DocumentFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
		Options:      protocol.FormattingOptions{},
	})
	if err != nil {
		t.Fatalf("declining must not be an LSP error, got %v", err)
	}
	if len(edits) != 0 {
		t.Errorf("want no edits, got %+v", edits)
	}
}

func TestFormattingFallsBackToDisk(t *testing.T) {
	ls := newFormattingServer(t, "main.zirr", fmtUnformatted)
	// Deliberately not opened in the document store.

	if edits := formatDoc(t, ls, "main.zirr", protocol.FormattingOptions{}); len(edits) != 1 {
		t.Errorf("want one edit from the on-disk file, got %d", len(edits))
	}
}

// A file excluded by the Cavefile must not be formatted by the editor either, or the two would disagree about the project's sources.
func TestFormattingHonoursCavefileExcludes(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "Cavefile", "mod proj\n\nimport cave\n\n"+
		"@cave.FormattingExcludes([\"vendor/**\"])\n"+
		"data Formatting {}\n")
	writeFile(t, base, "vendor/dep.zirr", fmtUnformatted)
	writeFile(t, base, "main.zirr", fmtUnformatted)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	if got := ls.formattingExcludes(); len(got) != 1 || got[0] != "vendor/**" {
		t.Fatalf("excludes not read from the Cavefile: %v", got)
	}

	ls.docs.Open("vendor/dep.zirr", 1, fmtUnformatted)
	if edits := formatDoc(t, ls, "vendor/dep.zirr", protocol.FormattingOptions{}); len(edits) != 0 {
		t.Errorf("excluded file should produce no edits, got %+v", edits)
	}

	ls.docs.Open("main.zirr", 1, fmtUnformatted)
	if edits := formatDoc(t, ls, "main.zirr", protocol.FormattingOptions{}); len(edits) != 1 {
		t.Errorf("non-excluded file should still format, got %d edits", len(edits))
	}
}

// With no Cavefile there are no excludes and formatting still works.
func TestFormattingWithoutCavefileHasNoExcludes(t *testing.T) {
	ls := newFormattingServer(t, "main.zirr", fmtUnformatted)
	if got := ls.formattingExcludes(); len(got) != 0 {
		t.Errorf("want no excludes, got %v", got)
	}
	ls.docs.Open("main.zirr", 1, fmtUnformatted)
	if edits := formatDoc(t, ls, "main.zirr", protocol.FormattingOptions{}); len(edits) != 1 {
		t.Errorf("want one edit, got %d", len(edits))
	}
}

// A Cavefile that does not parse may declare excludes the server cannot see, so it stops rather than rewrite an excluded file.
func TestFormattingStopsWhenCavefileIsMalformed(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "Cavefile", "mod proj\n\nimport cave\n\n@cave.Package(\ndata Broken {\n")
	writeFile(t, base, "main.zirr", fmtUnformatted)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	if _, broken := ls.cavefileIsUnreadable(); !broken {
		t.Fatal("a malformed Cavefile should be reported as unreadable")
	}

	ls.docs.Open("main.zirr", 1, fmtUnformatted)
	edits, err := ls.textDocumentFormatting(&glsp.Context{}, &protocol.DocumentFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
		Options:      protocol.FormattingOptions{},
	})
	if err != nil {
		t.Fatalf("declining must not be an LSP error, got %v", err)
	}
	if len(edits) != 0 {
		t.Errorf("want no edits while the Cavefile is malformed, got %+v", edits)
	}
}

// A well-formed Cavefile does not block formatting.
func TestFormattingProceedsWithValidCavefile(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "Cavefile", "mod proj\n\nimport cave\n\n@cave.Package()\ndata Dependencies {}\n")
	writeFile(t, base, "main.zirr", fmtUnformatted)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	if path, broken := ls.cavefileIsUnreadable(); broken {
		t.Fatalf("a valid Cavefile should not block formatting (%s)", path)
	}
	ls.docs.Open("main.zirr", 1, fmtUnformatted)
	if edits := formatDoc(t, ls, "main.zirr", protocol.FormattingOptions{}); len(edits) != 1 {
		t.Errorf("want one edit, got %d", len(edits))
	}
}
