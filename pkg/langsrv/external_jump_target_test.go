package langsrv

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestDefinitionStdlibJumpTarget(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "main.zirr", "mod main\nimport fmt\nconst x = fmt.sprint\n")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///main.zirr"},
			Position:     protocol.Position{Line: 2, Character: 16},
		},
	}
	result, err := ls.textDocumentDefinition(nil, params)
	if err != nil {
		t.Fatalf("textDocumentDefinition: %v", err)
	}
	if result == nil {
		t.Fatal("expected a definition location for fmt.sprint, got nil")
	}
	loc, ok := result.(*protocol.Location)
	if !ok {
		t.Fatalf("expected *protocol.Location, got %T", result)
	}
	if !strings.HasSuffix(string(loc.URI), "shim.zirr") {
		t.Errorf("expected jump target in fmt's shim.zirr, got %s", loc.URI)
	}

	rawPath := strings.TrimPrefix(string(loc.URI), "file://")
	data, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatalf("materialized jump target not readable: %v", err)
	}
	if !strings.Contains(string(data), "sprint") {
		t.Errorf("materialized file doesn't contain 'sprint': %s", data)
	}

	// Resolving again must reuse the cached temp file, not materialize a new one.
	result2, err := ls.textDocumentDefinition(nil, params)
	if err != nil {
		t.Fatalf("textDocumentDefinition (2nd call): %v", err)
	}
	loc2 := result2.(*protocol.Location)
	if loc2.URI != loc.URI {
		t.Errorf("expected the same materialized path on repeated calls, got %s then %s", loc.URI, loc2.URI)
	}
}

// initLocalGitDepRepo creates a real, local (network-free) git repository with one commit and one tag, usable as a "file://" @cave.Git dependency source.
func initLocalGitDepRepo(t *testing.T) string {
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

// TestDefinitionGitDependencyJumpTarget mirrors the real LSP flow: install non-read-only once (like `zirric install`), then read-only afterward (like the language server).
func TestDefinitionGitDependencyJumpTarget(t *testing.T) {
	remote := initLocalGitDepRepo(t)
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

	ls := zirricLangserver{
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

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: ls.fileURI("main.zirr")},
			Position:     protocol.Position{Line: 2, Character: 18},
		},
	}
	result, err := ls.textDocumentDefinition(nil, params)
	if err != nil {
		t.Fatalf("textDocumentDefinition: %v", err)
	}
	if result == nil {
		t.Fatal("expected a definition location for gitdep.greet, got nil")
	}
	loc, ok := result.(*protocol.Location)
	if !ok {
		t.Fatalf("expected *protocol.Location, got %T", result)
	}

	rawPath := strings.TrimPrefix(string(loc.URI), "file://")
	data, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatalf("materialized jump target not readable: %v", err)
	}
	if !strings.Contains(string(data), "greet") {
		t.Errorf("materialized file doesn't contain 'greet': %s", data)
	}
}
