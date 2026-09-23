package langsrv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
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
	diagCtx, openDocs, prevDiagURIs, current := ls.beginDiagnosticsPass()

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
	diagCtx, openDocs, prevDiagURIs, current := ls.beginDiagnosticsPass()
	ls.runDiagnosticsPass(diagCtx, ctx, openDocs, prevDiagURIs, current)
	return nil
}

// beginDiagnosticsPass cancels any in-progress pass, snapshots the state a new one needs, and invalidates cached parse results — shared setup for both refreshDiagnostics and refreshDiagnosticsSync, so a sync call started while an async pass is still running properly supersedes it instead of running concurrently against the same shared, mutable AST (see runDiagnosticsPass's diagRunMu for the other half of that guarantee).
func (ls *zirricLangserver) beginDiagnosticsPass() (diagCtx context.Context, openDocs map[string]protocol.DocumentUri, prevDiagURIs map[protocol.DocumentUri]struct{}, current map[protocol.DocumentUri]struct{}) {
	ls.mu.Lock()
	if ls.diagCancel != nil {
		ls.diagCancel()
	}

	var cancel context.CancelFunc
	diagCtx, cancel = context.WithCancel(context.Background())
	openDocs = make(map[string]protocol.DocumentUri, len(ls.openDocs))
	current = make(map[protocol.DocumentUri]struct{}, len(ls.openDocs))
	prevDiagURIs = ls.diagURIs
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

	if ls.resolver != nil {
		ls.resolver.InvalidateModules()
	}

	return diagCtx, openDocs, prevDiagURIs, current
}

func (ls *zirricLangserver) runDiagnosticsPass(
	diagCtx context.Context,
	ctx *glsp.Context,
	openDocs map[string]protocol.DocumentUri,
	prevDiagURIs map[protocol.DocumentUri]struct{},
	current map[protocol.DocumentUri]struct{},
) {
	// Serializes against any other in-flight pass (refreshDiagnostics's background goroutine vs. refreshDiagnosticsSync's inline call, or two overlapping refreshDiagnostics calls): only one Analyze() over the shared, mutable AST may run at a time. beginDiagnosticsPass already cancelled whichever pass was running before this one, so a pass currently holding this lock will notice diagCtx.Err() on its next file and release it promptly.
	ls.diagRunMu.Lock()
	defer ls.diagRunMu.Unlock()

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
	module, parseErrsByFile, _, err := ls.parseModuleFilesForPath(path)
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

	if !moduleHasParseErrors(parseErrsByFile) {
		an := analyzer.New(ls.resolver)
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
//
// When an Orchestra is available, the module gets prelude injection and the
// resolver is used for proper symbol resolution.
func (ls *zirricLangserver) parseModuleFiles(moduleDir string) (*ast.ContextModule, map[string][]parser.ParseError, map[string]string, error) {
	ls.moduleCacheMu.Lock()
	defer ls.moduleCacheMu.Unlock()

	if entry, ok := ls.moduleCache[moduleDir]; ok {
		return entry.module, entry.parseErrsByFile, entry.sourceURIToPath, nil
	}

	var (
		// Under the base the Cavefile fixed, so that the module carries the name its location implies — the same name the CLI gives it, and the one `mod` declarations are held to. Source URIs stay workspace-relative below, since diagnostics are matched against them by path.
		moduleURI       = registry.JoinModuleURI(ls.declaredPackageBase(), moduleDir)
		parseErrsByFile = make(map[string][]parser.ParseError)
		sourceURIToPath = make(map[string]string)
	)

	entries, err := ls.fs.ReadDir(moduleDir)
	if err != nil {
		module := ast.MakeContextModule(moduleURI)
		return module, parseErrsByFile, sourceURIToPath, err
	}

	// Build sources from directory
	var sources []registry.Source
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".zirr") {
			continue
		}

		filePath := filepath.Join(moduleDir, entry.Name())
		fileText, err := readFileText(ls.fs, filePath)
		if err != nil {
			continue
		}

		sourceURI := registry.JoinModuleURI("", filePath)
		sources = append(sources, overlaySource{uri: sourceURI, text: []byte(fileText)})
		sourceURIToPath[string(sourceURI)] = filePath
	}

	var module *ast.ContextModule

	if ls.orch != nil && ls.resolver != nil {
		// Use Orchestra for prelude injection and proper symbol resolution.
		mod := staticmodule.NewModule(moduleURI, sources)
		module, err = ls.orch.ParseModule(context.Background(), mod, ls.resolver)

		var parseErrs parser.ParseErrors
		if errors.As(err, &parseErrs) {
			for _, pe := range parseErrs {
				file := ""
				if pe.Token.Source != nil {
					file = pe.Token.Source.File
				}
				parseErrsByFile[file] = append(parseErrsByFile[file], pe)
			}
		} else if err != nil {
			// Truly fatal error (not parse errors)
			if module == nil {
				module = ast.MakeContextModule(moduleURI)
			}
			return module, parseErrsByFile, sourceURIToPath, err
		}
	} else {
		// Fallback: manual parsing without Orchestra.
		module = ast.MakeContextModule(moduleURI)
		for _, src := range sources {
			lex, err := lexer.New(src)
			if err != nil {
				continue
			}
			prs := parser.NewSourceParser(lex, module.Decls, string(src.URI()))
			tree := prs.ParseSourceFile()
			module.AddSourceFile(tree)
			parseErrsByFile[string(src.URI())] = prs.Errors()
		}
	}

	if module == nil {
		module = ast.MakeContextModule(moduleURI)
	}

	ls.moduleCache[moduleDir] = &moduleCacheEntry{
		module:          module,
		parseErrsByFile: parseErrsByFile,
		sourceURIToPath: sourceURIToPath,
	}

	return module, parseErrsByFile, sourceURIToPath, nil
}

// parseModuleFilesForPath is what every "operate on the file this request is about" handler should call instead of parseModuleFiles(filepath.Dir(path)) directly: the Cavefile has no .zirr extension, so parseModuleFiles' directory scan always skips it, leaving hover/definition/completion/references/rename blind to its own content (import cave, @cave.* attributes, declared dependencies) even though diagnostics and go-to-definition into it from other files work fine via the resolver. Every other file is unaffected.
func (ls *zirricLangserver) parseModuleFilesForPath(path string) (*ast.ContextModule, map[string][]parser.ParseError, map[string]string, error) {
	if ls.isCavefilePath(path) {
		return ls.parseCavefileModule(path)
	}
	return ls.parseModuleFiles(filepath.Dir(path))
}

// declaredPackageBase is the module path the project's Cavefile declares with its own `mod`, or "" when there is no Cavefile yet — the base every module of this package sits under.
func (ls *zirricLangserver) declaredPackageBase() registry.LogicalURI {
	if ls.orch == nil {
		return ""
	}
	return registry.LogicalURI(ls.orch.Cavefile().ModulePath)
}

// isCavefilePath reports whether path is the project's Cavefile, per ls.orch's own resolved Cavefile path (defaults to "Cavefile" at the workspace root when no Orchestra is set up yet).
func (ls *zirricLangserver) isCavefilePath(path string) bool {
	cavefilePath := orchestra.DefaultCavefileName
	if ls.orch != nil {
		if p := ls.orch.CavefilePath(); p != "" {
			cavefilePath = p
		}
	}
	return filepath.Clean(path) == filepath.Clean(cavefilePath)
}

// cavefileCacheKey namespaces parseCavefileModule's moduleCache entries away from parseModuleFiles' directory keys — belt-and-suspenders, since a directory literally named "Cavefile" would otherwise theoretically collide.
func cavefileCacheKey(path string) string {
	return "\x00cavefile:" + path
}

// parseCavefileModule parses path (the Cavefile) as its own standalone, single-file module — mirroring how the CLI itself compiles it (Orchestra.compileCavefileForTasks), and unlike parseModuleFiles, which only ever globs *.zirr files in a directory and so always skips it. Uses the same URI and caching conventions as parseModuleFiles so every existing caller (findSourceFile, sourceURIToPath lookups, …) works unchanged.
func (ls *zirricLangserver) parseCavefileModule(path string) (*ast.ContextModule, map[string][]parser.ParseError, map[string]string, error) {
	ls.moduleCacheMu.Lock()
	defer ls.moduleCacheMu.Unlock()

	cacheKey := cavefileCacheKey(path)
	if entry, ok := ls.moduleCache[cacheKey]; ok {
		return entry.module, entry.parseErrsByFile, entry.sourceURIToPath, nil
	}

	var (
		moduleURI       = registry.JoinModuleURI("", path)
		parseErrsByFile = make(map[string][]parser.ParseError)
		sourceURIToPath = make(map[string]string)
	)

	fileText, err := readFileText(ls.fs, path)
	if err != nil {
		module := ast.MakeContextModule(moduleURI)
		return module, parseErrsByFile, sourceURIToPath, err
	}

	sourceURI := moduleURI // single-file module: its own URI equals the module's URI
	sourceURIToPath[string(sourceURI)] = path
	sources := []registry.Source{overlaySource{uri: sourceURI, text: []byte(fileText)}}

	var module *ast.ContextModule

	if ls.orch != nil && ls.resolver != nil {
		mod := staticmodule.NewModule(moduleURI, sources)
		module, err = ls.orch.ParseModule(context.Background(), mod, ls.resolver)

		var parseErrs parser.ParseErrors
		if errors.As(err, &parseErrs) {
			for _, pe := range parseErrs {
				file := ""
				if pe.Token.Source != nil {
					file = pe.Token.Source.File
				}
				parseErrsByFile[file] = append(parseErrsByFile[file], pe)
			}
		} else if err != nil {
			if module == nil {
				module = ast.MakeContextModule(moduleURI)
			}
			return module, parseErrsByFile, sourceURIToPath, err
		}
	} else {
		module = ast.MakeContextModule(moduleURI)
		lex, lexErr := lexer.New(sources[0])
		if lexErr == nil {
			prs := parser.NewSourceParser(lex, module.Decls, string(sourceURI))
			tree := prs.ParseSourceFile()
			module.AddSourceFile(tree)
			parseErrsByFile[string(sourceURI)] = prs.Errors()
		}
	}

	if module == nil {
		module = ast.MakeContextModule(moduleURI)
	}

	ls.moduleCache[cacheKey] = &moduleCacheEntry{
		module:          module,
		parseErrsByFile: parseErrsByFile,
		sourceURIToPath: sourceURIToPath,
	}

	return module, parseErrsByFile, sourceURIToPath, nil
}

// moduleHasParseErrors checks every file in the module, not just the requested one, since error-recovery can leave malformed nodes the analyzer isn't safe to walk.
func moduleHasParseErrors(parseErrsByFile map[string][]parser.ParseError) bool {
	for _, errs := range parseErrsByFile {
		if len(errs) > 0 {
			return true
		}
	}
	return false
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
	if ls.rootPath != "" && !filepath.IsAbs(path) {
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
	if err.Severity == analyzer.AnalysisSeverityWarning {
		severity = protocol.DiagnosticSeverityWarning
	}

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
