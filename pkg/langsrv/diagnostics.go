package langsrv

import (
	"fmt"
	"io"
	"net/url"
	"path/filepath"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"github.com/go-git/go-billy/v5"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const parserDiagnosticSource = "zirric-parser"

func (ls *zirricLangserver) refreshDiagnostics(ctx *glsp.Context) error {
	if ctx == nil || ls.fs == nil {
		return nil
	}
	current := make(map[protocol.DocumentUri]struct{}, len(ls.openDocs))
	for path, uri := range ls.openDocs {
		current[uri] = struct{}{}

		diagnostics, version, err := ls.parseDiagnosticsForFile(path)
		if err != nil {
			diagnostics = []protocol.Diagnostic{diagnosticForError(err)}
			ls.logMessage(ctx, "parse error %q: %v", path, err)
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

	for uri := range ls.diagURIs {
		if _, ok := current[uri]; ok {
			continue
		}
		ctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: nil,
		})
	}

	ls.diagURIs = current
	return nil
}

func (ls *zirricLangserver) parseDiagnosticsForFile(path string) ([]protocol.Diagnostic, *protocol.UInteger, error) {
	text, err := readFileText(ls.fs, path)
	if err != nil {
		return nil, nil, err
	}
	moduleURI := registry.JoinModuleURI("", filepath.Dir(path))
	sourceURI := registry.JoinModuleURI("", path)
	module := ast.MakeContextModule(moduleURI)
	src := overlaySource{uri: sourceURI, text: []byte(text)}
	lex, err := lexer.New(src)
	if err != nil {
		return nil, nil, err
	}
	prs := parser.NewSourceParser(lex, module.Decls, string(src.URI()))
	prs.ParseSourceFile()
	errs := prs.Errors()

	diagnostics := make([]protocol.Diagnostic, 0, len(errs))
	for _, parseErr := range errs {
		diagnostics = append(diagnostics, diagnosticForParseError(parseErr, text))
	}

	if snapshot, ok := ls.docs.Snapshot(path); ok {
		version := protocol.UInteger(snapshot.Version)
		return diagnostics, &version, nil
	}
	return diagnostics, nil, nil
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
