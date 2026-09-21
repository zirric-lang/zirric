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

// TestRefreshDiagnosticsReportsTypeErrors pins that what the type checker finds reaches an editor.
// The checks run inside Analyze, so they arrive as ordinary analysis diagnostics with a range of their own rather than needing a path of their own.
func TestRefreshDiagnosticsReportsTypeErrors(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		message string
	}{
		{"wrong argument count", "fn two(a, b) { a }\nfn main() { two(1) }", "wrong number of arguments"},
		{"wrong argument type", "extern type Int {}\nfn takesInt(x: Int) { x }\nfn main() { takesInt(\"hi\") }", "wrong argument type"},
		{"unsupported operator", "data P { a }\nfn main() { P(1) + 3 }", "unsupported operator"},
		{"unknown field", "data P { a }\nfn main() { P(1).nope }", "unknown field"},
		{"wrong return type", "extern type Int {}\nfn wrong() -> Int {\n\treturn \"hello\"\n}", "wrong return type"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			base := memfs.New()
			writeFile(t, base, "main.zirr", tt.source)

			ls := zirricLangserver{
				docs:     newDocumentStore(),
				diagURIs: make(map[protocol.DocumentUri]struct{}),
				openDocs: make(map[string]protocol.DocumentUri),
			}
			ls.setFilesystem(base, "/")
			ls.openDocs["main.zirr"] = ls.fileURI("main.zirr")

			rec := &diagnosticRecorder{}
			if err := ls.refreshDiagnosticsSync(&glsp.Context{Notify: rec.Notify}); err != nil {
				t.Fatalf("refresh diagnostics: %v", err)
			}

			uri := ls.fileURI("main.zirr")
			diagnostics := rec.DiagnosticsFor(uri)
			for _, d := range diagnostics {
				if !strings.Contains(d.Message, tt.message) {
					continue
				}
				// An editor places the squiggle itself, so the range has to point at something rather than at the start of the file.
				if d.Range.End.Line == 0 && d.Range.End.Character == 0 {
					t.Errorf("diagnostic %q has an empty range", d.Message)
				}
				return
			}
			t.Fatalf("expected a diagnostic mentioning %q, got %v", tt.message, diagnostics)
		})
	}
}

// TestRefreshDiagnosticsReportsUndefinedNames pins that a typo is shown while it is being typed, rather than only once a program is built.
func TestRefreshDiagnosticsReportsUndefinedNames(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "fn main() {\n\tconst x = nosuchthing\n}")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")
	ls.openDocs["main.zirr"] = ls.fileURI("main.zirr")

	rec := &diagnosticRecorder{}
	if err := ls.refreshDiagnosticsSync(&glsp.Context{Notify: rec.Notify}); err != nil {
		t.Fatalf("refresh diagnostics: %v", err)
	}

	for _, d := range rec.DiagnosticsFor(ls.fileURI("main.zirr")) {
		if strings.Contains(d.Message, "nosuchthing") {
			// The squiggle has to land on the name itself, on the second line.
			if d.Range.Start.Line != 1 {
				t.Errorf("expected the diagnostic on line 2, got line %d", d.Range.Start.Line+1)
			}
			return
		}
	}
	t.Fatalf("expected a diagnostic naming the undefined identifier, got %v", rec.DiagnosticsFor(ls.fileURI("main.zirr")))
}
