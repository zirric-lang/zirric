package langsrv

import (
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestCallContext(t *testing.T) {
	tests := []struct {
		name            string
		text            string
		offset          int
		wantFuncName    string
		wantQualAlias   string
		wantActiveParam int
		wantFound       bool
	}{
		{
			name:            "simple call, first param",
			text:            "greet(name",
			offset:          10,
			wantFuncName:    "greet",
			wantActiveParam: 0,
			wantFound:       true,
		},
		{
			name:            "call, second param",
			text:            "add(a, b",
			offset:          8,
			wantFuncName:    "add",
			wantActiveParam: 1,
			wantFound:       true,
		},
		{
			name:            "nested call, outer param",
			text:            "outer(inner(x), b",
			offset:          17,
			wantFuncName:    "outer",
			wantActiveParam: 1,
			wantFound:       true,
		},
		{
			name:          "qualified call",
			text:          "mymod.helper(arg",
			offset:        16,
			wantFuncName:  "helper",
			wantQualAlias: "mymod",
			wantFound:     true,
		},
		{
			name:      "not in a call",
			text:      "const x = 1",
			offset:    9,
			wantFound: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			funcName, qualAlias, activeParam, found := callContext(tt.text, tt.offset)
			if found != tt.wantFound {
				t.Fatalf("found=%v want %v", found, tt.wantFound)
			}
			if !found {
				return
			}
			if funcName != tt.wantFuncName {
				t.Errorf("funcName=%q want %q", funcName, tt.wantFuncName)
			}
			if qualAlias != tt.wantQualAlias {
				t.Errorf("qualAlias=%q want %q", qualAlias, tt.wantQualAlias)
			}
			if activeParam != tt.wantActiveParam {
				t.Errorf("activeParam=%d want %d", activeParam, tt.wantActiveParam)
			}
		})
	}
}

func TestSignatureHelp(t *testing.T) {
	src := "fn greet(name) {}\nfn add(x, y) {}\ngreet("
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor just after "greet(" on line 2
	params := &protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 2, Character: 6},
		},
	}
	result, err := ls.textDocumentSignatureHelp(nil, params)
	if err != nil {
		t.Fatalf("textDocumentSignatureHelp: %v", err)
	}
	if result == nil {
		t.Fatal("expected SignatureHelp, got nil")
		return
	}
	if len(result.Signatures) == 0 {
		t.Fatal("expected at least one signature")
		return
	}
	sig := result.Signatures[0]
	wantLabel := "fn greet(name)"
	if sig.Label != wantLabel {
		t.Errorf("signature label = %q, want %q", sig.Label, wantLabel)
	}
	if len(sig.Parameters) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(sig.Parameters))
	}
	if result.ActiveParameter == nil || *result.ActiveParameter != 0 {
		t.Errorf("activeParameter should be 0")
	}
}

func TestSignatureHelpActiveParam(t *testing.T) {
	src := "fn add(x, y) {}\nadd(1, "
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Cursor after "1, " — second parameter position
	params := &protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 1, Character: 7},
		},
	}
	result, err := ls.textDocumentSignatureHelp(nil, params)
	if err != nil {
		t.Fatalf("textDocumentSignatureHelp: %v", err)
	}
	if result == nil {
		t.Fatal("expected SignatureHelp, got nil")
		return
	}
	if result.ActiveParameter == nil || *result.ActiveParameter != 1 {
		var got protocol.UInteger
		if result.ActiveParameter != nil {
			got = *result.ActiveParameter
		}
		t.Errorf("activeParameter = %d, want 1", got)
	}
}
