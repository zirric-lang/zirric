package langsrv

import (
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestRefreshDiagnosticsUsesOverlay(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "func example() { let x = 4 }")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")
	ls.docs.Open("main.zirr", 1, "func example() { let = }")
	ls.openDocs["main.zirr"] = ls.fileURI("main.zirr")

	rec := &diagnosticRecorder{}
	ctx := &glsp.Context{Notify: rec.Notify}

	if err := ls.refreshDiagnostics(ctx); err != nil {
		t.Fatalf("refresh diagnostics: %v", err)
	}

	uri := ls.fileURI("main.zirr")
	diagnostics := rec.DiagnosticsFor(uri)
	if len(diagnostics) == 0 {
		t.Fatalf("expected diagnostics for %s", uri)
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
