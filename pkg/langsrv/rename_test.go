package langsrv

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"

	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// applyTextEdits applies edits from the end of the text backward, so earlier ranges/offsets stay valid as later ones are applied.
func applyTextEdits(t *testing.T, text string, edits []protocol.TextEdit) string {
	t.Helper()
	type span struct {
		start, end int
		newText    string
	}
	spans := make([]span, 0, len(edits))
	for _, e := range edits {
		spans = append(spans, span{
			start:   offsetForPosition(text, e.Range.Start),
			end:     offsetForPosition(text, e.Range.End),
			newText: e.NewText,
		})
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start > spans[j].start })
	for _, s := range spans {
		text = text[:s.start] + s.newText + text[s.end:]
	}
	return text
}

func TestRenameImportAliasBare(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "flow/types.zirr", "mod flow\nfn helper() {}\n")
	src := "mod main\nimport flow\nconst x = flow.helper()\n"
	writeFile(t, base, "main.zirr", src)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 1, Character: 9},
		},
		NewName: "myflow",
	}
	edit, err := ls.textDocumentRename(nil, params)
	if err != nil {
		t.Fatalf("textDocumentRename: %v", err)
	}
	if edit == nil {
		t.Fatal("expected a WorkspaceEdit, got nil")
	}
	edits := edit.Changes["file:///main.zirr"]
	if len(edits) != 2 {
		t.Fatalf("expected 2 edits (alias insert + usage), got %d: %+v", len(edits), edits)
	}
	got := applyTextEdits(t, src, edits)
	want := "mod main\nimport myflow = flow\nconst x = myflow.helper()\n"
	if got != want {
		t.Errorf("applied edits =\n%q\nwant\n%q", got, want)
	}
}

func TestRenameImportAliasExplicit(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "flow/types.zirr", "mod flow\nfn helper() {}\n")
	src := "mod main\nimport myflow = flow\nconst x = myflow.helper()\n"
	writeFile(t, base, "main.zirr", src)

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 1, Character: 10},
		},
		NewName: "renamed",
	}
	edit, err := ls.textDocumentRename(nil, params)
	if err != nil {
		t.Fatalf("textDocumentRename: %v", err)
	}
	if edit == nil {
		t.Fatal("expected a WorkspaceEdit, got nil")
	}
	edits := edit.Changes["file:///main.zirr"]
	if len(edits) != 2 {
		t.Fatalf("expected 2 edits (alias + usage), got %d: %+v", len(edits), edits)
	}
	got := applyTextEdits(t, src, edits)
	want := "mod main\nimport renamed = flow\nconst x = renamed.helper()\n"
	if got != want {
		t.Errorf("applied edits =\n%q\nwant\n%q", got, want)
	}
}

func TestPrepareRenameImportAlias(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "flow/types.zirr", "mod flow\nfn helper() {}\n")
	writeFile(t, base, "main.zirr", "mod main\nimport flow\nimport myflow = flow\n")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	// Bare "import flow": range should cover "flow" (col 7-11).
	bareResult, err := ls.textDocumentPrepareRename(nil, &protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 1, Character: 9},
		},
	})
	if err != nil {
		t.Fatalf("prepareRename (bare): %v", err)
	}
	bareRange, ok := bareResult.(protocol.Range)
	if !ok {
		t.Fatalf("expected protocol.Range, got %T", bareResult)
	}
	if bareRange.Start.Character != 7 || bareRange.End.Character != 11 {
		t.Errorf("bare import range = %+v, want [7,11)", bareRange)
	}

	// Explicit "import myflow = flow": range should cover "myflow" (col 7-13).
	explicitResult, err := ls.textDocumentPrepareRename(nil, &protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 2, Character: 10},
		},
	})
	if err != nil {
		t.Fatalf("prepareRename (explicit): %v", err)
	}
	explicitRange, ok := explicitResult.(protocol.Range)
	if !ok {
		t.Fatalf("expected protocol.Range, got %T", explicitResult)
	}
	if explicitRange.Start.Character != 7 || explicitRange.End.Character != 13 {
		t.Errorf("explicit alias range = %+v, want [7,13)", explicitRange)
	}
}

func TestRenameLocal(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "fn greet(name) {}\nfn main() { greet(\"world\") }\n")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 0, Character: 5}, // on "greet" decl
		},
		NewName: "salute",
	}
	edit, err := ls.textDocumentRename(nil, params)
	if err != nil {
		t.Fatalf("textDocumentRename: %v", err)
	}
	if edit == nil {
		t.Fatal("expected a WorkspaceEdit, got nil")
	}
	edits := edit.Changes["file:///main.zirr"]
	if len(edits) != 2 {
		t.Fatalf("expected 2 edits (decl + call), got %d: %+v", len(edits), edits)
	}
	for _, e := range edits {
		if e.NewText != "salute" {
			t.Errorf("expected NewText 'salute', got %q", e.NewText)
		}
	}
}

func TestRenameCrossModule(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "flow/types.zirr", "mod flow\nfn helper() {}\n")
	writeFile(t, base, "main.zirr", "mod main\nimport flow\nconst x = flow.helper()\n")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///flow/types.zirr"},
			Position:     protocol.Position{Line: 1, Character: 4}, // on "helper" decl
		},
		NewName: "assist",
	}
	edit, err := ls.textDocumentRename(nil, params)
	if err != nil {
		t.Fatalf("textDocumentRename: %v", err)
	}
	if edit == nil {
		t.Fatal("expected a WorkspaceEdit, got nil")
	}
	if len(edit.Changes["file:///flow/types.zirr"]) != 1 {
		t.Errorf("expected 1 edit in flow/types.zirr, got %+v", edit.Changes["file:///flow/types.zirr"])
	}
	if len(edit.Changes["file:///main.zirr"]) != 1 {
		t.Errorf("expected 1 edit in main.zirr, got %+v", edit.Changes["file:///main.zirr"])
	}
}

func TestRenameRejectsInvalidIdentifier(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "fn greet(name) {}\n")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	for _, newName := range []string{"", "123abc", "not valid", "if"} {
		params := &protocol.RenameParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
				Position:     protocol.Position{Line: 0, Character: 5},
			},
			NewName: newName,
		}
		if _, err := ls.textDocumentRename(nil, params); err == nil {
			t.Errorf("expected an error renaming to invalid identifier %q", newName)
		}
	}
}

func TestRenameRejectsStdlibSymbol(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "mod main\nimport fmt\nconst x = fmt.sprint\n")

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 2, Character: 16}, // on "sprint"
		},
		NewName: "sprint2",
	}
	if _, err := ls.textDocumentRename(nil, params); err == nil {
		t.Fatal("expected renaming a stdlib symbol to be rejected")
	}

	prepareParams := &protocol.PrepareRenameParams{
		TextDocumentPositionParams: params.TextDocumentPositionParams,
	}
	if _, err := ls.textDocumentPrepareRename(nil, prepareParams); err == nil {
		t.Fatal("expected prepareRename on a stdlib symbol to be rejected")
	}
}

// initLocalGitDepRepoForRename creates a real, local (network-free) git repository with one commit and one tag, usable as a "file://" @cave.Git dependency source.
func initLocalGitDepRepoForRename(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("plain init: %v", err)
	}
	if err := os.WriteFile(dir+"/greet.zirr", []byte("mod gitdep\nfn greet() {}\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	if _, err := wt.Add("greet.zirr"); err != nil {
		t.Fatalf("add: %v", err)
	}
	sig := &object.Signature{Name: "test", Email: "test@test.com", When: time.Now()}
	hash, err := wt.Commit("init", &git.CommitOptions{Author: sig})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := repo.CreateTag("v1.0.0", hash, nil); err != nil {
		t.Fatalf("create tag: %v", err)
	}
	return dir
}

// TestRenameRejectsGitDependencySymbol mirrors the real LSP flow: install non-read-only once (like `zirric install`), then read-only afterward (like the language server).
func TestRenameRejectsGitDependencySymbol(t *testing.T) {
	remote := initLocalGitDepRepoForRename(t)
	source := "file://" + remote

	root := t.TempDir()
	registryFS := memfs.New()
	cave := `import cave

@cave.Dependencies()
data Dependencies {
  @cave.Git("` + source + `")
  @cave.Version("v1.0.0")
  gitdep
}
`
	if err := os.WriteFile(root+"/Cavefile", []byte(cave), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root+"/main.zirr", []byte("mod main\nimport gitdep\nconst x = gitdep.greet\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	installOrch, err := orchestra.New(orchestra.Config{ProjectFS: osfs.New(root), RegistryFS: registryFS, PackageName: "proj"})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}
	installResolver, err := installOrch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	if _, err := installResolver.EnsureInstalled(context.Background()); err != nil {
		t.Fatalf("ensure installed: %v", err)
	}

	ls := &zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.rootPath = root
	ls.fs = newOverlayFS(osfs.New(root), ls.docs)
	ls.moduleCache = make(map[string]*moduleCacheEntry)
	ls.orch, err = orchestra.New(orchestra.Config{ProjectFS: ls.fs, RegistryFS: registryFS, PackageName: "proj"})
	if err != nil {
		t.Fatalf("new lsp orchestra: %v", err)
	}
	ls.resolver, err = ls.orch.NewResolver(orchestra.ReadOnly())
	if err != nil {
		t.Fatalf("new read-only resolver: %v", err)
	}

	params := &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: ls.fileURI("main.zirr")},
			Position:     protocol.Position{Line: 2, Character: 18}, // on "greet"
		},
		NewName: "greet2",
	}
	if _, err := ls.textDocumentRename(nil, params); err == nil {
		t.Fatal("expected renaming a git-dependency symbol to be rejected")
	}
}
