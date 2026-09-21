package langsrv

import (
	"context"
	"sync"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"github.com/go-git/go-billy/v5"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

var lsName = "zirric"
var debug = true
var handler protocol.Handler

// moduleCacheEntry holds the result of parsing a directory of .zirr files.
type moduleCacheEntry struct {
	module          *ast.ContextModule
	parseErrsByFile map[string][]parser.ParseError
	sourceURIToPath map[string]string
}

type zirricLangserver struct {
	server   *server.Server
	docs     *documentStore
	fs       billy.Filesystem
	rootPath string

	// mu protects diagCancel, diagURIs, and openDocs.
	mu         sync.Mutex
	diagCancel context.CancelFunc
	diagURIs   map[protocol.DocumentUri]struct{}
	openDocs   map[string]protocol.DocumentUri

	// diagRunMu serializes runDiagnosticsPass invocations (refreshDiagnostics's background goroutine and refreshDiagnosticsSync's inline call can otherwise overlap) — without it, two passes' Analyze() calls can concurrently mutate the same shared, cached AST (e.g. the prelude module), racing.
	diagRunMu sync.Mutex

	// moduleCache caches parseModuleFiles results keyed by directory.
	// It is invalidated (cleared) whenever documents change.
	moduleCache   map[string]*moduleCacheEntry
	moduleCacheMu sync.Mutex

	// orch/resolver: never call RunFile, RunModulePath, Compile, or runBytecode on these — only ParseModule/ParseFile/ParseModulePath and analyzer. Exception: "zirric.task.*" (commands.go), which shells out to a subprocess instead of touching these fields.
	orch     *orchestra.Orchestra
	resolver *orchestra.ModuleResolver

	// externalSources caches materialized read-only copies of sources outside the workspace root (embedded stdlib, local/git deps), keyed by logical URI, valued by absolute OS path.
	externalSourcesMu  sync.Mutex
	externalSources    map[string]string
	externalSourcesDir string
}

var ls zirricLangserver = zirricLangserver{}

func init() {
	commonlog.Configure(2, nil)
	ls.docs = newDocumentStore()
	ls.diagURIs = make(map[protocol.DocumentUri]struct{})
	ls.openDocs = make(map[string]protocol.DocumentUri)
	ls.moduleCache = make(map[string]*moduleCacheEntry)
	ls.externalSources = make(map[string]string)

	handler = protocol.Handler{
		Initialize:                 ls.initialize,
		Initialized:                ls.initialized,
		Shutdown:                   ls.shutdown,
		Exit:                       ls.exit,
		TextDocumentDidOpen:        ls.didOpen,
		TextDocumentDidChange:      ls.didChange,
		TextDocumentDidSave:        ls.didSave,
		TextDocumentDidClose:       ls.didClose,
		TextDocumentCompletion:     ls.textDocumentCompletion,
		TextDocumentHover:          ls.textDocumentHover,
		TextDocumentDefinition:     ls.textDocumentDefinition,
		TextDocumentDocumentSymbol: ls.textDocumentDocumentSymbol,
		TextDocumentFormatting:     ls.textDocumentFormatting,
		WorkspaceSymbol:            ls.workspaceSymbol,
		TextDocumentSignatureHelp:  ls.textDocumentSignatureHelp,
		TextDocumentReferences:     ls.textDocumentReferences,
		TextDocumentRename:         ls.textDocumentRename,
		TextDocumentPrepareRename:  ls.textDocumentPrepareRename,
		WorkspaceExecuteCommand:    ls.workspaceExecuteCommand,
	}
}

func RunStdio() error {
	ls.server = server.NewServer(&handler, lsName, debug)
	return ls.server.RunStdio()
}

func RunIPC() error {
	ls.server = server.NewServer(&handler, lsName, debug)
	return ls.server.RunNodeJs()
}

func RunSocket(address string) error {
	ls.server = server.NewServer(&handler, lsName, debug)
	return ls.server.RunWebSocket(address)
}

func RunTCP(address string) error {
	ls.server = server.NewServer(&handler, lsName, debug)
	return ls.server.RunTCP(address)
}
