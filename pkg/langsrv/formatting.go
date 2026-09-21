package langsrv

import (
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/codefmt"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (ls *zirricLangserver) textDocumentFormatting(
	ctx *glsp.Context,
	params *protocol.DocumentFormattingParams,
) ([]protocol.TextEdit, error) {
	defer func() {
		if r := recover(); r != nil {
			ls.logMessage(ctx, "panic in textDocumentFormatting: %v", r)
		}
	}()

	path, ok := ls.pathForURI(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	// A Cavefile that does not parse may declare excludes we cannot see, so stop rather than rewrite a file meant to be left alone.
	if cavefile, broken := ls.cavefileIsUnreadable(); broken {
		ls.logMessage(ctx, "not formatting %s: %s is malformed, so its formatting excludes cannot be read", path, cavefile)
		return nil, nil
	}

	if ls.formattingExcludes().Match(path) {
		ls.logMessage(ctx, "not formatting %s: excluded by @cave.FormattingExcludes", path)
		return nil, nil
	}

	text, ok := ls.documentText(path)
	if !ok {
		return nil, nil
	}

	opts := codefmt.DefaultOptions().FromLSP(params.Options)
	formatted, err := codefmt.String(path, text, opts)
	if err != nil {
		// Reporting this as an error would raise a dialog on every save of a half-typed buffer.
		ls.logMessage(ctx, "not formatting %s: %v", path, err)
		return nil, nil
	}
	if formatted == text {
		// Returning an empty edit would mark a clean buffer dirty.
		return nil, nil
	}

	return []protocol.TextEdit{minimalEdit(text, formatted)}, nil
}

// cavefileIsUnreadable reports a Cavefile that fails to parse; having none at all is fine.
func (ls *zirricLangserver) cavefileIsUnreadable() (string, bool) {
	path := orchestra.DefaultCavefileName
	if ls.orch != nil {
		if p := ls.orch.CavefilePath(); p != "" {
			path = p
		}
	}
	if ls.fs == nil {
		return "", false
	}
	if _, err := ls.fs.Stat(path); err != nil {
		return "", false
	}
	_, parseErrs, _, err := ls.parseCavefileModule(path)
	if err != nil {
		return path, true
	}
	for _, errs := range parseErrs {
		if len(errs) > 0 {
			return path, true
		}
	}
	return "", false
}

// formattingExcludes is empty without a usable Cavefile, so formatting still works.
func (ls *zirricLangserver) formattingExcludes() codefmt.Excludes {
	if ls.orch == nil {
		return nil
	}
	return codefmt.Excludes(ls.orch.Cavefile().FormattingExcludes)
}

// documentText prefers the open buffer, which may hold unsaved edits.
func (ls *zirricLangserver) documentText(path string) (string, bool) {
	if snapshot, ok := ls.docs.Snapshot(path); ok {
		return snapshot.Text, true
	}
	if ls.fs == nil {
		return "", false
	}
	text, err := readFileText(ls.fs, path)
	if err != nil {
		return "", false
	}
	return text, true
}

// minimalEdit replaces only the changed lines, since a whole-document edit discards folding and cursor state in several editors.
func minimalEdit(before, after string) protocol.TextEdit {
	beforeLines := strings.SplitAfter(before, "\n")
	afterLines := strings.SplitAfter(after, "\n")

	prefix := 0
	for prefix < len(beforeLines) && prefix < len(afterLines) &&
		beforeLines[prefix] == afterLines[prefix] {
		prefix++
	}

	suffix := 0
	for suffix < len(beforeLines)-prefix && suffix < len(afterLines)-prefix &&
		beforeLines[len(beforeLines)-1-suffix] == afterLines[len(afterLines)-1-suffix] {
		suffix++
	}

	start := 0
	for _, line := range beforeLines[:prefix] {
		start += len(line)
	}
	end := len(before)
	for _, line := range beforeLines[len(beforeLines)-suffix:] {
		end -= len(line)
	}

	replacement := strings.Join(afterLines[prefix:len(afterLines)-suffix], "")

	return protocol.TextEdit{
		Range:   rangeForOffsets(before, start, end),
		NewText: replacement,
	}
}
