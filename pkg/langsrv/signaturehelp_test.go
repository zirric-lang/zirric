package langsrv

import (
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestCallChainContext(t *testing.T) {
	tests := []struct {
		name            string
		text            string
		offset          int
		wantFuncName    string
		wantQualChain   []string
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
			wantQualChain: []string{"mymod"},
			wantFound:     true,
		},
		{
			name:      "not in a call",
			text:      "const x = 1",
			offset:    9,
			wantFound: false,
		},
		{
			name:          "at-qualified call",
			text:          "@mymod.Tag(arg",
			offset:        14,
			wantFuncName:  "Tag",
			wantQualChain: []string{"mymod"},
			wantFound:     true,
		},
		{
			name:          "multi-segment chain call",
			text:          "person.name.toggle(arg",
			offset:        22,
			wantFuncName:  "toggle",
			wantQualChain: []string{"person", "name"},
			wantFound:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			funcName, qualChain, activeParam, found := callChainContext(tt.text, tt.offset)
			if found != tt.wantFound {
				t.Fatalf("found=%v want %v", found, tt.wantFound)
			}
			if !found {
				return
			}
			if funcName != tt.wantFuncName {
				t.Errorf("funcName=%q want %q", funcName, tt.wantFuncName)
			}
			if len(qualChain) != len(tt.wantQualChain) {
				t.Errorf("qualChain=%v want %v", qualChain, tt.wantQualChain)
			} else {
				for i := range qualChain {
					if qualChain[i] != tt.wantQualChain[i] {
						t.Errorf("qualChain[%d]=%q want %q", i, qualChain[i], tt.wantQualChain[i])
					}
				}
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

func TestSignatureHelpDataConstructor(t *testing.T) {
	src := "data Point { x y }\nPoint("
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 1, Character: 6},
		},
	}
	result, err := ls.textDocumentSignatureHelp(nil, params)
	if err != nil {
		t.Fatalf("textDocumentSignatureHelp: %v", err)
	}
	if result == nil {
		t.Fatal("expected SignatureHelp for data constructor, got nil")
		return
	}
	if len(result.Signatures) == 0 {
		t.Fatal("expected at least one signature")
		return
	}
	sig := result.Signatures[0]
	wantLabel := "data Point(x, y)"
	if sig.Label != wantLabel {
		t.Errorf("signature label = %q, want %q", sig.Label, wantLabel)
	}
	if len(sig.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(sig.Parameters))
	}
}

func TestSignatureHelpDataConstructorWithTypes(t *testing.T) {
	src := "data Person { name: String age: Int }\nPerson("
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

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
		t.Fatal("expected SignatureHelp for typed data constructor, got nil")
		return
	}
	if len(result.Signatures) == 0 {
		t.Fatal("expected at least one signature")
		return
	}
	sig := result.Signatures[0]
	wantLabel := "data Person(name: String, age: Int)"
	if sig.Label != wantLabel {
		t.Errorf("signature label = %q, want %q", sig.Label, wantLabel)
	}
	if len(sig.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(sig.Parameters))
	}
	if sig.Parameters[0].Label != "name: String" {
		t.Errorf("param[0] label = %q, want %q", sig.Parameters[0].Label, "name: String")
	}
	if sig.Parameters[1].Label != "age: Int" {
		t.Errorf("param[1] label = %q, want %q", sig.Parameters[1].Label, "age: Int")
	}
}

func TestSignatureHelpAttribute(t *testing.T) {
	src := "attr Doc { summary details }\n@Doc("
	base := memfs.New()
	writeFile(t, base, "main.zirr", src)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 1, Character: 5},
		},
	}
	result, err := ls.textDocumentSignatureHelp(nil, params)
	if err != nil {
		t.Fatalf("textDocumentSignatureHelp: %v", err)
	}
	if result == nil {
		t.Fatal("expected SignatureHelp for attribute call, got nil")
		return
	}
	if len(result.Signatures) == 0 {
		t.Fatal("expected at least one signature")
		return
	}
	sig := result.Signatures[0]
	wantLabel := "attr Doc(summary, details)"
	if sig.Label != wantLabel {
		t.Errorf("signature label = %q, want %q", sig.Label, wantLabel)
	}
	if len(sig.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(sig.Parameters))
	}
}

func TestSignatureHelpModNameQualified(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "mymod/main.zirr", "mod mymod\nfn helper(x) {}\nmymod.helper(")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///mymod/main.zirr"},
			Position:     protocol.Position{Line: 2, Character: 13},
		},
	}
	result, err := ls.textDocumentSignatureHelp(nil, params)
	if err != nil {
		t.Fatalf("textDocumentSignatureHelp: %v", err)
	}
	if result == nil {
		t.Fatal("expected SignatureHelp for modname.helper(, got nil")
		return
	}
	if len(result.Signatures) == 0 {
		t.Fatal("expected at least one signature")
		return
	}
	sig := result.Signatures[0]
	wantLabel2 := "fn helper(x)"
	if sig.Label != wantLabel2 {
		t.Errorf("signature label = %q, want %q", sig.Label, wantLabel2)
	}
}

func TestSignatureHelpQualifiedAttrFromImport(t *testing.T) {
	base := memfs.New()
	if err := base.MkdirAll("mymod", 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, base, "mymod/types.zirr", "attr Tag { name }")
	writeFile(t, base, "main.zirr", "import mymod\n@mymod.Tag(")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 1, Character: 11},
		},
	}
	result, err := ls.textDocumentSignatureHelp(nil, params)
	if err != nil {
		t.Fatalf("textDocumentSignatureHelp: %v", err)
	}
	if result == nil {
		t.Fatal("expected SignatureHelp for @mymod.Tag(, got nil")
		return
	}
	if len(result.Signatures) == 0 {
		t.Fatal("expected at least one signature")
		return
	}
	sig2 := result.Signatures[0]
	wantLabel3 := "attr Tag(name)"
	if sig2.Label != wantLabel3 {
		t.Errorf("signature label = %q, want %q", sig2.Label, wantLabel3)
	}
}
