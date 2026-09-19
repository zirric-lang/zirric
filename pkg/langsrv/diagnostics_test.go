package langsrv

import (
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestRefreshDiagnosticsUsesOverlay(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "fn example() { const x = 4 }")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")
	ls.docs.Open("main.zirr", 1, "fn example() { const = }")
	ls.openDocs["main.zirr"] = ls.fileURI("main.zirr")

	rec := &diagnosticRecorder{}
	ctx := &glsp.Context{Notify: rec.Notify}

	if err := ls.refreshDiagnosticsSync(ctx); err != nil {
		t.Fatalf("refresh diagnostics: %v", err)
	}

	uri := ls.fileURI("main.zirr")
	diagnostics := rec.DiagnosticsFor(uri)
	if len(diagnostics) == 0 {
		t.Fatalf("expected diagnostics for %s", uri)
	}
}

func TestRefreshDiagnosticsReportsAnalyzerErrors(t *testing.T) {
	base := memfs.New()
	src := "union Optional { Missing }"
	writeFile(t, base, "main.zirr", src)

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")
	ls.openDocs["main.zirr"] = ls.fileURI("main.zirr")

	rec := &diagnosticRecorder{}
	ctx := &glsp.Context{Notify: rec.Notify}

	if err := ls.refreshDiagnosticsSync(ctx); err != nil {
		t.Fatalf("refresh diagnostics: %v", err)
	}

	uri := ls.fileURI("main.zirr")
	diagnostics := rec.DiagnosticsFor(uri)
	if len(diagnostics) == 0 {
		t.Fatalf("expected analyzer diagnostics for %s", uri)
	}
	for _, d := range diagnostics {
		if d.Source != nil && *d.Source == analyzerDiagnosticSource {
			return
		}
	}
	t.Fatalf("expected at least one diagnostic with source %q, got %v", analyzerDiagnosticSource, diagnostics)
}

// TestRefreshDiagnosticsWarnsOnMissingDependency is a regression test: a missing Cavefile dependency used to break diagnostics for the whole project, not just files that import it.
func TestRefreshDiagnosticsWarnsOnMissingDependency(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "Cavefile", `import cave

@cave.Dependencies()
data Dependencies {
  @cave.Local("../does-not-exist")
  missing
}
`)
	writeFile(t, base, "main.zirr", "mod main\nimport missing\n")
	writeFile(t, base, "other/other.zirr", "mod other\nconst x = 1\n")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")
	ls.openDocs["main.zirr"] = ls.fileURI("main.zirr")
	ls.openDocs["other/other.zirr"] = ls.fileURI("other/other.zirr")

	rec := &diagnosticRecorder{}
	ctx := &glsp.Context{Notify: rec.Notify}

	if err := ls.refreshDiagnosticsSync(ctx); err != nil {
		t.Fatalf("refresh diagnostics: %v", err)
	}

	mainDiags := rec.DiagnosticsFor(ls.fileURI("main.zirr"))
	if len(mainDiags) == 0 {
		t.Fatal("expected a diagnostic on main.zirr for the missing dependency import")
	}
	found := false
	for _, d := range mainDiags {
		if d.Severity != nil && *d.Severity == protocol.DiagnosticSeverityWarning {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a warning-severity diagnostic, got %+v", mainDiags)
	}

	otherDiags := rec.DiagnosticsFor(ls.fileURI("other/other.zirr"))
	if len(otherDiags) != 0 {
		t.Fatalf("expected no diagnostics for unrelated file, got %+v", otherDiags)
	}
}

// TestRefreshDiagnosticsSkipsAnalysisWhenSiblingFileHasParseErrors is a regression test: opening a Cavefile (which trivially always has zero parse errors of its own) alongside a sibling .zirr file with a genuine parse error used to run the analyzer over that broken sibling's AST anyway, since the "skip if parse errors" check only looked at the requested file's own bucket. Error recovery from `f(, 1)` leaves a nil in ExprInvocation.Arguments that then panics the analyzer's walker.
func TestRefreshDiagnosticsSkipsAnalysisWhenSiblingFileHasParseErrors(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "Cavefile", `mod cavefile

import cave {
	Dependencies
}

@cave.Dependencies()
data UI {}
`)
	writeFile(t, base, "broken.zirr", "mod ui\nconst x = f(, 1)\n")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")
	ls.openDocs["Cavefile"] = ls.fileURI("Cavefile")
	ls.openDocs["broken.zirr"] = ls.fileURI("broken.zirr")

	rec := &diagnosticRecorder{}
	ctx := &glsp.Context{Notify: rec.Notify}

	if err := ls.refreshDiagnosticsSync(ctx); err != nil {
		t.Fatalf("refresh diagnostics: %v", err)
	}

	for _, d := range rec.DiagnosticsFor(ls.fileURI("Cavefile")) {
		if strings.Contains(d.Message, "internal parser error") {
			t.Fatalf("Cavefile diagnostics should not surface an internal parser error, got: %+v", d)
		}
	}

	brokenDiags := rec.DiagnosticsFor(ls.fileURI("broken.zirr"))
	if len(brokenDiags) == 0 {
		t.Fatal("expected a parse-error diagnostic on broken.zirr")
	}
}

type diagnosticRecorder struct {
	entries map[protocol.DocumentUri][]protocol.Diagnostic
}

func (r *diagnosticRecorder) Notify(method string, params any) {
	if method != protocol.ServerTextDocumentPublishDiagnostics {
		return
	}
	payload, ok := params.(*protocol.PublishDiagnosticsParams)
	if !ok {
		return
	}
	if r.entries == nil {
		r.entries = make(map[protocol.DocumentUri][]protocol.Diagnostic)
	}
	r.entries[payload.URI] = payload.Diagnostics
}

func (r *diagnosticRecorder) DiagnosticsFor(uri protocol.DocumentUri) []protocol.Diagnostic {
	if r.entries == nil {
		return nil
	}
	return r.entries[uri]
}
