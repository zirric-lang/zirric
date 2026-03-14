package langsrv

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"github.com/go-git/go-billy/v5"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const (
	parserDiagnosticSource   = "zirric-parser"
	analyzerDiagnosticSource = "zirric-analyzer"
)

func (ls *zirricLangserver) refreshDiagnostics(ctx *glsp.Context) error {
	if ctx == nil || ls.fs == nil {
		return nil
	}

	// Cancel any in-progress diagnostics pass and snapshot shared state atomically.
	ls.mu.Lock()
	if ls.diagCancel != nil {
		ls.diagCancel()
	}

	var (
		diagCtx, cancel = context.WithCancel(context.Background())
		openDocs        = make(map[string]protocol.DocumentUri, len(ls.openDocs))
		current         = make(map[protocol.DocumentUri]struct{}, len(openDocs))
		prevDiagURIs    = ls.diagURIs
	)
	ls.diagCancel = cancel

	for k, v := range ls.openDocs {
		openDocs[k] = v
	}

	for _, uri := range openDocs {
		current[uri] = struct{}{}
	}
	ls.diagURIs = current
	ls.mu.Unlock()

	// Invalidate the module cache so the fresh pass sees up-to-date content.
	ls.moduleCacheMu.Lock()
	ls.moduleCache = make(map[string]*moduleCacheEntry)
	ls.moduleCacheMu.Unlock()

	// Run the slow parse/analyze work in the background so that other LSP
	// requests (hover, completion, …) are not blocked while diagnostics compute.
	go ls.runDiagnosticsPass(diagCtx, ctx, openDocs, prevDiagURIs, current)

	return nil
}

// refreshDiagnosticsSync runs the full diagnostics pass synchronously. Used in tests.
func (ls *zirricLangserver) refreshDiagnosticsSync(ctx *glsp.Context) error {
	if ctx == nil || ls.fs == nil {
		return nil
	}

	ls.mu.Lock()
	openDocs := make(map[string]protocol.DocumentUri, len(ls.openDocs))
	for k, v := range ls.openDocs {
		openDocs[k] = v
	}

	prevDiagURIs := ls.diagURIs
	current := make(map[protocol.DocumentUri]struct{}, len(openDocs))
	for _, uri := range openDocs {
		current[uri] = struct{}{}
	}

	ls.diagURIs = current
	ls.mu.Unlock()

	// Invalidate the module cache for a fresh pass.
	ls.moduleCacheMu.Lock()
	ls.moduleCache = make(map[string]*moduleCacheEntry)
	ls.moduleCacheMu.Unlock()

	ls.runDiagnosticsPass(context.Background(), ctx, openDocs, prevDiagURIs, current)
	return nil
}

func (ls *zirricLangserver) runDiagnosticsPass(
	diagCtx context.Context,
	ctx *glsp.Context,
	openDocs map[string]protocol.DocumentUri,
	prevDiagURIs map[protocol.DocumentUri]struct{},
	current map[protocol.DocumentUri]struct{},
) {
	for path, uri := range openDocs {
		if diagCtx.Err() != nil {
			return
		}
		diagnostics, version, err := ls.parseDiagnosticsForFile(path)
		if err != nil {
			ls.logMessage(ctx, "diagnostics error for %q: %v", path, err)
			diagnostics = []protocol.Diagnostic{diagnosticForError(err)}
		}

		params := protocol.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: diagnostics,
		}
		if version != nil {
			params.Version = version
		}
		ctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, &params)
	}

	for uri := range prevDiagURIs {
		if _, ok := current[uri]; ok {
			continue
		}
		ctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: nil,
		})
	}
}

func (ls *zirricLangserver) parseDiagnosticsForFile(path string) (diags []protocol.Diagnostic, version *protocol.UInteger, retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("internal parser error: %v", r)
		}
	}()
	return ls.parseDiagnosticsForFileInner(path)
}

func (ls *zirricLangserver) parseDiagnosticsForFileInner(path string) ([]protocol.Diagnostic, *protocol.UInteger, error) {
	sourceURI := string(registry.JoinModuleURI("", path))
	module, parseErrsByFile, _, err := ls.parseModuleFiles(filepath.Dir(path))
	if err != nil {
		return nil, nil, err
	}

	text, err := readFileText(ls.fs, path)
	if err != nil {
		return nil, nil, err
	}

	errs := parseErrsByFile[sourceURI]
	diagnostics := make([]protocol.Diagnostic, 0, len(errs))
	for _, parseErr := range errs {
		diagnostics = append(diagnostics, diagnosticForParseError(parseErr, text))
	}

	if len(errs) == 0 {
		an := analyzer.New(nil)
		analysisErrs, _ := an.Analyze(module, false)

		for _, analysisErr := range analysisErrs {
			if analysisErr.Token.Source != nil && analysisErr.Token.Source.File == sourceURI {
				diagnostics = append(diagnostics, diagnosticForAnalysisError(analysisErr, text))
			}
		}
	}

	if snapshot, ok := ls.docs.Snapshot(path); ok {
		version := protocol.UInteger(snapshot.Version)
		return diagnostics, &version, nil
	}
	return diagnostics, nil, nil
}

// parseModuleFiles parses all .zirr files in the given directory into a shared
// module, so that all symbols are visible across files. It returns the module,
// a map from source URI string to that file's parse errors, and a map from
// source URI string to the relative file path (for go-to-definition).
// Results are cached by directory and invalidated whenever documents change.
func (ls *zirricLangserver) parseModuleFiles(moduleDir string) (*ast.ContextModule, map[string][]parser.ParseError, map[string]string, error) {
	ls.moduleCacheMu.Lock()
	if entry, ok := ls.moduleCache[moduleDir]; ok {
		ls.moduleCacheMu.Unlock()
		return entry.module, entry.parseErrsByFile, entry.sourceURIToPath, nil
	}
	ls.moduleCacheMu.Unlock()

	var (
		moduleURI       = registry.JoinModuleURI("", moduleDir)
		module          = ast.MakeContextModule(moduleURI)
		parseErrsByFile = make(map[string][]parser.ParseError)
		sourceURIToPath = make(map[string]string)
	)

	entries, err := ls.fs.ReadDir(moduleDir)
	if err != nil {
		return module, parseErrsByFile, sourceURIToPath, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".zirr") {
			continue
		}

		filePath := filepath.Join(moduleDir, entry.Name())
		fileText, err := readFileText(ls.fs, filePath)
		if err != nil {
			continue
		}

		var (
			sourceURI = registry.JoinModuleURI("", filePath)
			src       = overlaySource{uri: sourceURI, text: []byte(fileText)}
		)

		lex, err := lexer.New(src)
		if err != nil {
			continue
		}

		prs := parser.NewSourceParser(lex, module.Decls, string(src.URI()))
		tree := prs.ParseSourceFile()

		module.AddSourceFile(tree)
		parseErrsByFile[string(sourceURI)] = prs.Errors()
		sourceURIToPath[string(sourceURI)] = filePath
	}

	ls.moduleCacheMu.Lock()
	ls.moduleCache[moduleDir] = &moduleCacheEntry{
		module:          module,
		parseErrsByFile: parseErrsByFile,
		sourceURIToPath: sourceURIToPath,
	}
	ls.moduleCacheMu.Unlock()

	return module, parseErrsByFile, sourceURIToPath, nil
}

func readFileText(fs billy.Filesystem, path string) (string, error) {
	file, err := fs.Open(path)
	if err != nil {
		return "", err
	}

	defer func() {
		_ = file.Close()
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (ls *zirricLangserver) fileURI(path string) protocol.DocumentUri {
	absPath := path
	if ls.rootPath != "" {
		absPath = filepath.Join(ls.rootPath, path)
	}

	absPath = filepath.Clean(absPath)
	uri := url.URL{Scheme: "file", Path: filepath.ToSlash(absPath)}
	return protocol.DocumentUri(uri.String())
}

func diagnosticForAnalysisError(err analyzer.AnalysisError, text string) protocol.Diagnostic {
	var (
		severity = protocol.DiagnosticSeverityError
		message  = err.Summary
	)

	if err.Details != "" {
		message = fmt.Sprintf("%s: %s", err.Summary, err.Details)
	}

	start := 0
	if err.Token.Source != nil {
		start = err.Token.Source.Offset
	}

	length := len(err.Token.Literal)
	if length <= 0 {
		length = 1
	}

	return protocol.Diagnostic{
		Range:    rangeForOffsets(text, start, start+length),
		Severity: &severity,
		Source:   ptr(analyzerDiagnosticSource),
		Message:  message,
	}
}

func diagnosticForParseError(err parser.ParseError, text string) protocol.Diagnostic {
	severity := protocol.DiagnosticSeverityError
	message := err.Summary
	if err.Details != "" {
		message = fmt.Sprintf("%s: %s", err.Summary, err.Details)
	}

	start := 0
	if err.Token.Source != nil {
		start = err.Token.Source.Offset
	}

	length := len(err.Token.Literal)
	if length <= 0 {
		length = 1
	}

	return protocol.Diagnostic{
		Range:    rangeForOffsets(text, start, start+length),
		Severity: &severity,
		Source:   ptr(parserDiagnosticSource),
		Message:  message,
	}
}

func diagnosticForError(err error) protocol.Diagnostic {
	severity := protocol.DiagnosticSeverityError
	return protocol.Diagnostic{
		Range:    rangeForOffsets("", 0, 0),
		Severity: &severity,
		Source:   ptr(parserDiagnosticSource),
		Message:  err.Error(),
	}
}

func rangeForOffsets(text string, start int, end int) protocol.Range {
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	if start > len(text) {
		start = len(text)
	}
	if end > len(text) {
		end = len(text)
	}
	return protocol.Range{
		Start: positionForOffset(text, start),
		End:   positionForOffset(text, end),
	}
}

func positionForOffset(text string, offset int) protocol.Position {
	if offset <= 0 {
		return protocol.Position{}
	}
	if offset > len(text) {
		offset = len(text)
	}
	var line protocol.UInteger
	var col protocol.UInteger
	for idx, r := range text {
		if idx >= offset {
			break
		}
		if r == '\n' {
			line++
			col = 0
			continue
		}
		col += utf16Units(r)
	}
	return protocol.Position{Line: line, Character: col}
}

func utf16Units(r rune) protocol.UInteger {
	if r > 0xFFFF {
		return 2
	}
	return 1
}

func ptr[T any](value T) *T {
	return &value
}

type overlaySource struct {
	uri  registry.LogicalURI
	text []byte
}

func (s overlaySource) URI() registry.LogicalURI {
	return s.uri
}

func (s overlaySource) Read() ([]byte, error) {
	return s.text, nil
}
