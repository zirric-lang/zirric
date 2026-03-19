package langsrv

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (ls *zirricLangserver) initialize(ctx *glsp.Context, params *protocol.InitializeParams) (any, error) {
	rootPath, err := resolveRootPath(params)
	if err != nil {
		ls.logMessage(ctx, "initialize error: %v", err)
		return nil, err
	}
	if rootPath == "" {
		rootPath, err = os.Getwd()
		if err != nil {
			ls.logMessage(ctx, "initialize error: %v", err)
			return nil, err
		}
	}
	ls.setRoot(rootPath)

	syncKind := protocol.TextDocumentSyncKindIncremental
	opens := protocol.True
	save := protocol.SaveOptions{IncludeText: &protocol.True}

	return protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: protocol.TextDocumentSyncOptions{
				OpenClose: &opens,
				Change:    &syncKind,
				Save:      &save,
			},
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{"@", "."},
			},
			HoverProvider:           true,
			DefinitionProvider:      &protocol.DefinitionOptions{},
			DocumentSymbolProvider:  &protocol.DocumentSymbolOptions{},
			WorkspaceSymbolProvider: &protocol.WorkspaceSymbolOptions{},
			ReferencesProvider:      &protocol.ReferenceOptions{},
			SignatureHelpProvider: &protocol.SignatureHelpOptions{
				TriggerCharacters: []string{"(", ","},
			},
		},
		ServerInfo: &protocol.InitializeResultServerInfo{Name: lsName},
	}, nil
}

func (ls *zirricLangserver) initialized(ctx *glsp.Context, _ *protocol.InitializedParams) error {
	return ls.refreshDiagnostics(ctx)
}

func (ls *zirricLangserver) shutdown(ctx *glsp.Context) error {
	return nil
}

func (ls *zirricLangserver) exit(ctx *glsp.Context) error {
	os.Exit(0)
	return nil
}

func (ls *zirricLangserver) didOpen(ctx *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	path, ok := ls.pathForURI(params.TextDocument.URI)
	if !ok {
		return nil
	}

	ls.docs.Open(path, int32(params.TextDocument.Version), params.TextDocument.Text)

	ls.mu.Lock()
	ls.openDocs[path] = params.TextDocument.URI
	ls.mu.Unlock()

	return ls.refreshDiagnostics(ctx)
}

func (ls *zirricLangserver) didChange(ctx *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	path, ok := ls.pathForURI(params.TextDocument.URI)
	if !ok {
		return nil
	}

	changes, err := normalizeContentChanges(params.ContentChanges)
	if err != nil {
		return err
	}

	if err := ls.docs.ApplyChanges(path, int32(params.TextDocument.Version), changes); err != nil {
		return err
	}
	return ls.refreshDiagnostics(ctx)
}

func (ls *zirricLangserver) didSave(ctx *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	path, ok := ls.pathForURI(params.TextDocument.URI)
	if !ok {
		return nil
	}

	ls.docs.Save(path, params.Text)
	return ls.refreshDiagnostics(ctx)
}

func (ls *zirricLangserver) didClose(ctx *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	path, ok := ls.pathForURI(params.TextDocument.URI)
	if !ok {
		return nil
	}

	ls.docs.Close(path)

	ls.mu.Lock()
	delete(ls.openDocs, path)
	ls.mu.Unlock()

	return ls.refreshDiagnostics(ctx)
}

func (ls *zirricLangserver) setRoot(rootPath string) {
	rootPath = filepath.Clean(rootPath)
	base := osfs.New(rootPath)
	ls.setFilesystem(base, rootPath)
}

func (ls *zirricLangserver) setFilesystem(base billy.Filesystem, rootPath string) {
	rootPath = filepath.Clean(rootPath)
	ls.rootPath = rootPath
	ls.fs = newOverlayFS(base, ls.docs)
	ls.moduleCache = make(map[string]*moduleCacheEntry)

	pkgName := filepath.Base(rootPath)
	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:   ls.fs,
		RegistryFS:  memfs.New(),
		PackageName: pkgName,
	})
	if err != nil {
		// Fall back to no-orchestra mode; parseModuleFiles will use manual parsing.
		ls.orch = nil
		ls.resolver = nil
		return
	}
	ls.orch = orch
	ls.resolver, _ = orch.NewResolver(orchestra.ReadOnly())
}

func (ls *zirricLangserver) pathForURI(uri protocol.DocumentUri) (string, bool) {
	absPath, err := uriToPath(uri)
	if err != nil {
		return "", false
	}

	if ls.rootPath == "" {
		return cleanPath(absPath), true
	}

	rel, err := filepath.Rel(ls.rootPath, absPath)
	if err != nil {
		return "", false
	}

	rel = filepath.Clean(rel)
	if rel == "." {
		return "", false
	}

	if strings.HasPrefix(rel, "..") {
		return "", false
	}

	return rel, true
}

func resolveRootPath(params *protocol.InitializeParams) (string, error) {
	if params == nil {
		return "", nil
	}

	if len(params.WorkspaceFolders) > 0 {
		if path, err := uriToPath(params.WorkspaceFolders[0].URI); err == nil {
			return path, nil
		}
	}

	if params.RootURI != nil {
		if path, err := uriToPath(*params.RootURI); err == nil {
			return path, nil
		}
	}

	if params.RootPath != nil && *params.RootPath != "" {
		return *params.RootPath, nil
	}

	return "", nil
}

func uriToPath(uri protocol.DocumentUri) (string, error) {
	u, err := url.Parse(string(uri))
	if err != nil {
		return "", err
	}

	if u.Scheme != "file" {
		return "", fmt.Errorf("unsupported URI scheme: %s", u.Scheme)
	}

	if u.Path == "" {
		return "", errors.New("empty URI path")
	}

	path, err := url.PathUnescape(u.Path)
	if err != nil {
		return "", err
	}

	return filepath.FromSlash(path), nil
}

func normalizeContentChanges(changes []any) ([]protocol.TextDocumentContentChangeEvent, error) {
	if len(changes) == 0 {
		return nil, nil
	}
	result := make([]protocol.TextDocumentContentChangeEvent, 0, len(changes))
	for _, change := range changes {
		switch typed := change.(type) {
		case protocol.TextDocumentContentChangeEvent:
			result = append(result, typed)

		case *protocol.TextDocumentContentChangeEvent:
			result = append(result, *typed)

		case protocol.TextDocumentContentChangeEventWhole:
			result = append(result, protocol.TextDocumentContentChangeEvent{Text: typed.Text})

		case *protocol.TextDocumentContentChangeEventWhole:
			result = append(result, protocol.TextDocumentContentChangeEvent{Text: typed.Text})

		default:
			return nil, fmt.Errorf("unsupported content change type %T", change)
		}
	}
	return result, nil
}
