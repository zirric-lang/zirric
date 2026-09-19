package langsrv

import (
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (ls *zirricLangserver) textDocumentHover(
	ctx *glsp.Context,
	params *protocol.HoverParams,
) (*protocol.Hover, error) {
	defer func() {
		if r := recover(); r != nil {
			ls.logMessage(ctx, "panic in textDocumentHover: %v", r)
		}
	}()

	path, ok := ls.pathForURI(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	text, err := readFileText(ls.fs, path)
	if err != nil {
		return nil, nil
	}

	module, _, _, err := ls.parseModuleFilesForPath(path)
	if err != nil {
		return nil, nil
	}

	word, wordRange := wordAtPosition(text, params.Position)
	if word == "" {
		return nil, nil
	}

	sourceURI := string(registry.JoinModuleURI("", path))
	currentSF := findSourceFile(module, sourceURI)

	// Check qualified/dot-chain context: "alias.member" or "expr.field.subfield"
	if segments, _, isDotChain := dotChainContext(text, params.Position); isDotChain {
		// Try module/import alias first (single-segment).
		if len(segments) == 1 {
			alias := segments[0]
			if imp, ok := findImportDecl(currentSF, alias); ok {
				if importedMod, _, ok := ls.loadImportedModule(imp); ok {
					content := hoverDeclInModule(importedMod, word)
					if content != "" {
						kind := protocol.MarkupKindMarkdown
						return &protocol.Hover{
							Contents: protocol.MarkupContent{Kind: kind, Value: content},
							Range:    &wordRange,
						}, nil
					}
				}
			}
			if isModuleDecl(module, alias) {
				content := hoverDeclInModule(module, word)
				if content != "" {
					kind := protocol.MarkupKindMarkdown
					return &protocol.Hover{
						Contents: protocol.MarkupContent{Kind: kind, Value: content},
						Range:    &wordRange,
					}, nil
				}
			}
		}

		// Multi-segment or non-module single segment: type-aware resolution.
		// Resolve the chain to get available fields, then find the word among them.
		cursorOffset := offsetForPosition(text, params.Position)
		result := ls.resolveDotChain(module, currentSF, path, cursorOffset, segments)
		if result != nil && len(result.fields) > 0 {
			for _, f := range result.fields {
				if f.Name.Value == word {
					content := hoverContentForDecl(f)
					if content != "" {
						kind := protocol.MarkupKindMarkdown
						return &protocol.Hover{
							Contents: protocol.MarkupContent{Kind: kind, Value: content},
							Range:    &wordRange,
						}, nil
					}
				}
			}
		}
		return nil, nil
	}

	cursorOffset := offsetForPosition(text, params.Position)
	sym := ls.resolveWordDecl(module, currentSF, path, cursorOffset, word)
	if sym == nil || sym.Decl == nil {
		return nil, nil
	}

	// DeclImportMember: show hover from actual declaration in imported module.
	var content string
	switch dim := sym.Decl.(type) {
	case ast.DeclImportMember:
		if resolved := ls.resolveImportMemberDecl(dim); resolved != nil {
			content = hoverContentForDecl(resolved)
		}

	case *ast.DeclImportMember:
		if resolved := ls.resolveImportMemberDecl(*dim); resolved != nil {
			content = hoverContentForDecl(resolved)
		}

	default:
		content = hoverContentForDecl(sym.Decl)
	}

	if content == "" {
		return nil, nil
	}

	kind := protocol.MarkupKindMarkdown
	return &protocol.Hover{
		Contents: protocol.MarkupContent{Kind: kind, Value: content},
		Range:    &wordRange,
	}, nil
}

// wordAtPosition returns the identifier word under the cursor and its document range.
func wordAtPosition(text string, pos protocol.Position) (string, protocol.Range) {
	lineNum := int(pos.Line)
	start := 0
	for i := 0; i < lineNum; i++ {
		idx := strings.IndexByte(text[start:], '\n')
		if idx < 0 {
			return "", protocol.Range{}
		}
		start += idx + 1
	}

	var (
		line string
		end  = strings.IndexByte(text[start:], '\n')
	)

	if end < 0 {
		line = text[start:]
	} else {
		line = text[start : start+end]
	}

	col := int(pos.Character)
	if col > len(line) {
		col = len(line)
	}

	wordStart := col
	for wordStart > 0 && isSimpleIdentByte(line[wordStart-1]) {
		wordStart--
	}
	wordEnd := col
	for wordEnd < len(line) && isSimpleIdentByte(line[wordEnd]) {
		wordEnd++
	}
	if wordStart >= wordEnd {
		return "", protocol.Range{}
	}

	word := line[wordStart:wordEnd]
	wordRange := protocol.Range{
		Start: protocol.Position{Line: pos.Line, Character: uint32(wordStart)},
		End:   protocol.Position{Line: pos.Line, Character: uint32(wordEnd)},
	}
	return word, wordRange
}

// isSimpleIdentByte matches identifier characters excluding '.'.
func isSimpleIdentByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_'
}

// hoverContentForDecl builds markdown hover text for a declaration.
func hoverContentForDecl(decl ast.Decl) string {
	var sb strings.Builder

	if ov, ok := decl.(ast.Overviewable); ok {
		overview := ov.DeclOverview()
		if overview != "" {
			fmt.Fprintf(&sb, "```zirric\n%s\n```", overview)
		}
	}

	if doc, ok := decl.(ast.Documented); ok {
		if docs := doc.ProvidedDocs(); docs != nil && docs.Content != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(docs.Content)
		}
	}

	return sb.String()
}
