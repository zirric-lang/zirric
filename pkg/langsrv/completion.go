package langsrv

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (ls *zirricLangserver) textDocumentCompletion(
	ctx *glsp.Context,
	params *protocol.CompletionParams,
) (any, error) {
	defer func() {
		if r := recover(); r != nil {
			ls.logMessage(ctx, "panic in textDocumentCompletion: %v", r)
		}
	}()

	path, ok := ls.pathForURI(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	items, err := ls.completionItemsForFile(path, params.Position)
	if err != nil {
		return nil, nil
	}

	return items, nil
}

func (ls *zirricLangserver) completionItemsForFile(path string, pos protocol.Position) ([]protocol.CompletionItem, error) {
	text, err := readFileText(ls.fs, path)
	if err != nil {
		return nil, err
	}

	module, _, _, err := ls.parseModuleFiles(filepath.Dir(path))
	if err != nil {
		return nil, err
	}

	var (
		atPos, inAttribute = attributeContextStart(text, pos)
		cursorOffset       = offsetForPosition(text, pos)
		sourceURI          = string(registry.JoinModuleURI("", path))
		currentSF          = findSourceFile(module, sourceURI)
	)

	// Handle qualified context: "alias.member" or "@alias.member"
	if alias, afterDot, isQualified := qualifiedContext(text, pos); isQualified {
		if imp, ok := findImportDecl(currentSF, alias); ok {
			if importedMod, _, ok := ls.loadImportedModule(imp); ok {
				return importedModuleCompletions(importedMod, inAttribute, afterDot, pos), nil
			}
		}
		// Unknown module alias — return empty rather than irrelevant globals.
		return nil, nil
	}

	var items []protocol.CompletionItem
	for _, sym := range module.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}

		if inAttribute {
			if item, ok := attributeContextCompletionItem(sym, atPos, pos); ok {
				items = append(items, item)
			}
		} else {
			items = append(items, completionItemForDeclSymbol(sym))
		}
	}

	// Add import aliases and directly imported members from the current file's local scope.
	if currentSF != nil {
		for _, sym := range currentSF.Decls.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}
			switch sym.Decl.(type) {
			case ast.DeclImport, *ast.DeclImport, ast.DeclImportMember, *ast.DeclImportMember:
				if !inAttribute {
					items = append(items, completionItemForDeclSymbol(sym))
				}
			}
		}
	}

	// Add local bindings (params, local lets, for bindings) in non-attribute context.
	if !inAttribute {
		for _, decl := range collectLocalsBeforeCursor(module, sourceURI, cursorOffset) {
			kind := protocol.CompletionItemKindVariable
			name := decl.DeclName().Value
			if name == "" {
				continue
			}
			item := protocol.CompletionItem{Label: name, Kind: &kind}
			if ov, ok := decl.(ast.Overviewable); ok {
				detail := ov.DeclOverview()
				if detail != "" {
					item.Detail = &detail
				}
			}
			items = append(items, item)
		}
	}

	return items, nil
}

// importedModuleCompletions returns completion items for all public symbols in an imported module.
// In attribute context, only attribute/type declarations are included with snippet TextEdits.
func importedModuleCompletions(
	mod *ast.ContextModule,
	inAttribute bool,
	afterDot, cursorPos protocol.Position,
) []protocol.CompletionItem {
	memberRange := protocol.Range{Start: afterDot, End: cursorPos}
	var items []protocol.CompletionItem
	for _, sym := range mod.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		if inAttribute {
			switch decl := sym.Decl.(type) {
			case *ast.DeclAttr:
				items = append(items, attributeMemberSnippetItem(sym.Name, decl.Fields, memberRange))
			case *ast.DeclData, *ast.DeclUnion, *ast.DeclExternType:
				_ = decl
				items = append(items, typeAttributeMemberItem(sym.Name, memberRange))
			}
		} else {
			items = append(items, completionItemForDeclSymbol(sym))
		}
	}
	return items
}

// attributeContextStart returns the position of '@' and true when the cursor
// is directly after a '@' sign (possibly followed by partial identifier chars),
// indicating the user is completing an attribute reference.
func attributeContextStart(text string, pos protocol.Position) (protocol.Position, bool) {
	lineNum := int(pos.Line)
	start := 0
	for i := 0; i < lineNum; i++ {
		idx := strings.IndexByte(text[start:], '\n')
		if idx < 0 {
			return protocol.Position{}, false
		}
		start += idx + 1
	}

	end := strings.IndexByte(text[start:], '\n')
	var line string
	if end < 0 {
		line = text[start:]
	} else {
		line = text[start : start+end]
	}

	col := int(pos.Character)
	if col > len(line) {
		col = len(line)
	}

	prefix := line[:col]

	// Walk backward past identifier chars to find the preceding non-identifier character.
	i := len(prefix) - 1
	for i >= 0 && isIdentByte(prefix[i]) {
		i--
	}
	if i < 0 || prefix[i] != '@' {
		return protocol.Position{}, false
	}

	atChar := uint32(i)
	return protocol.Position{Line: pos.Line, Character: atChar}, true
}

// isAttributeContext reports whether pos is directly after a '@' sign.
func isAttributeContext(text string, pos protocol.Position) bool {
	_, ok := attributeContextStart(text, pos)
	return ok
}

func isIdentByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '.'
}

// attributeContextCompletionItem returns a completion item for use after '@'.
// Annotation declarations get a snippet with their field names as placeholders.
// Type declarations (data, union, extern type) are included as plain names.
// Value and function declarations are excluded.
// The TextEdit replaces from the '@' character up to the cursor so that the
// resulting text always contains the leading '@'.
func attributeContextCompletionItem(sym *ast.DeclSymbol, atPos, cursorPos protocol.Position) (protocol.CompletionItem, bool) {
	editRange := protocol.Range{Start: atPos, End: cursorPos}

	switch decl := sym.Decl.(type) {
	case *ast.DeclAttr:
		return attributeSnippetItem(sym.Name, decl.Fields, editRange), true

	case *ast.DeclData, *ast.DeclUnion, *ast.DeclExternType:
		return typeAttributeItem(sym.Name, editRange), true

	default:
		_ = decl
		return protocol.CompletionItem{}, false
	}
}

// attributeSnippetItem builds a CompletionItem for an attribute declaration.
// When the attribute has fields, the insert text is a snippet with each field
// name as a tab-stop placeholder: @Name(${1:field1}, ${2:field2}).
// The TextEdit replaces from '@' to the cursor so the '@' is always present.
func attributeSnippetItem(name string, fields []ast.DeclField, editRange protocol.Range) protocol.CompletionItem {
	var (
		kind       = protocol.CompletionItemKindInterface
		label      = "@" + name
		filterText = name
		format     = protocol.InsertTextFormatSnippet
	)

	var newText string
	if len(fields) == 0 {
		newText = "@" + name
	} else {
		parts := make([]string, len(fields))
		for i, f := range fields {
			parts[i] = fmt.Sprintf("${%d:%s}", i+1, f.Name.Value)
		}
		newText = fmt.Sprintf("@%s(%s)", name, strings.Join(parts, ", "))
	}
	return protocol.CompletionItem{
		Label:            label,
		FilterText:       &filterText,
		Kind:             &kind,
		InsertTextFormat: &format,
		TextEdit:         protocol.TextEdit{Range: editRange, NewText: newText},
	}
}

// typeAttributeItem builds a CompletionItem for a type used in attribute position.
func typeAttributeItem(name string, editRange protocol.Range) protocol.CompletionItem {
	var (
		kind       = protocol.CompletionItemKindStruct
		label      = "@" + name
		filterText = name
		format     = protocol.InsertTextFormatPlainText
	)

	return protocol.CompletionItem{
		Label:            label,
		FilterText:       &filterText,
		Kind:             &kind,
		InsertTextFormat: &format,
		TextEdit:         protocol.TextEdit{Range: editRange, NewText: "@" + name},
	}
}

func completionItemForDeclSymbol(sym *ast.DeclSymbol) protocol.CompletionItem {
	// Attributes outside attribute context are offered with '@' prefix and snippet.
	if decl, ok := sym.Decl.(*ast.DeclAttr); ok {
		return attributeNonContextItem(sym.Name, decl.Fields)
	}

	kind := completionKindForDecl(sym.Decl)
	item := protocol.CompletionItem{
		Label: sym.Name,
		Kind:  &kind,
	}

	if ov, ok := sym.Decl.(ast.Overviewable); ok {
		detail := ov.DeclOverview()
		if detail != "" {
			item.Detail = &detail
		}
	}

	return item
}

// attributeNonContextItem returns a completion item for an attribute when
// there is no '@' before the cursor. The '@' is prepended to the label and
// included in the insertText snippet so it is always part of the result.
func attributeNonContextItem(name string, fields []ast.DeclField) protocol.CompletionItem {
	var (
		kind       = protocol.CompletionItemKindInterface
		label      = "@" + name
		format     = protocol.InsertTextFormatSnippet
		insertText string
	)

	if len(fields) == 0 {
		insertText = "@" + name
	} else {
		parts := make([]string, len(fields))
		for i, f := range fields {
			parts[i] = fmt.Sprintf("${%d:%s}", i+1, f.Name.Value)
		}
		insertText = fmt.Sprintf("@%s(%s)", name, strings.Join(parts, ", "))
	}
	return protocol.CompletionItem{
		Label:            label,
		Kind:             &kind,
		InsertText:       &insertText,
		InsertTextFormat: &format,
	}
}

func completionKindForDecl(decl ast.Decl) protocol.CompletionItemKind {
	switch decl.(type) {
	case *ast.DeclFunc, ast.DeclFunc, *ast.DeclExternFunc, ast.DeclExternFunc:
		return protocol.CompletionItemKindFunction

	case *ast.DeclVariable, ast.DeclVariable, *ast.DeclConstant, ast.DeclConstant, *ast.DeclExternValue, ast.DeclExternValue:
		return protocol.CompletionItemKindVariable

	case *ast.DeclData, ast.DeclData, *ast.DeclExternType, ast.DeclExternType:
		return protocol.CompletionItemKindStruct

	case *ast.DeclUnion, ast.DeclUnion:
		return protocol.CompletionItemKindClass

	case *ast.DeclAttr, ast.DeclAttr:
		return protocol.CompletionItemKindInterface

	case *ast.DeclImport, ast.DeclImport, *ast.DeclModule, ast.DeclModule:
		return protocol.CompletionItemKindModule

	case *ast.DeclImportMember, ast.DeclImportMember:
		return protocol.CompletionItemKindModule

	default:
		return protocol.CompletionItemKindText
	}
}

// attributeMemberSnippetItem builds an attribute completion for qualified context (@alias.Name).
// The TextEdit replaces only the member portion (after the dot), so "@alias." is preserved.
func attributeMemberSnippetItem(name string, fields []ast.DeclField, memberRange protocol.Range) protocol.CompletionItem {
	var (
		kind       = protocol.CompletionItemKindInterface
		label      = "@" + name
		filterText = name
		format     = protocol.InsertTextFormatSnippet
		newText    string
	)

	if len(fields) == 0 {
		newText = name
	} else {
		parts := make([]string, len(fields))
		for i, f := range fields {
			parts[i] = fmt.Sprintf("${%d:%s}", i+1, f.Name.Value)
		}
		newText = fmt.Sprintf("%s(%s)", name, strings.Join(parts, ", "))
	}
	return protocol.CompletionItem{
		Label:            label,
		FilterText:       &filterText,
		Kind:             &kind,
		InsertTextFormat: &format,
		TextEdit:         protocol.TextEdit{Range: memberRange, NewText: newText},
	}
}

// typeAttributeMemberItem builds a type completion for qualified attribute context (@alias.Type).
func typeAttributeMemberItem(name string, memberRange protocol.Range) protocol.CompletionItem {
	var (
		kind       = protocol.CompletionItemKindStruct
		label      = "@" + name
		filterText = name
		format     = protocol.InsertTextFormatPlainText
	)
	return protocol.CompletionItem{
		Label:            label,
		FilterText:       &filterText,
		Kind:             &kind,
		InsertTextFormat: &format,
		TextEdit:         protocol.TextEdit{Range: memberRange, NewText: name},
	}
}

// offsetForPosition converts an LSP Position (line, UTF-16 character) to a byte offset in text.
func offsetForPosition(text string, pos protocol.Position) int {
	line := int(pos.Line)
	offset := 0

	for i := 0; i < line; i++ {
		idx := strings.IndexByte(text[offset:], '\n')
		if idx < 0 {
			return len(text)
		}
		offset += idx + 1
	}

	col := int(pos.Character)
	i := offset
	for i < len(text) && col > 0 && text[i] != '\n' {
		r, size := utf8.DecodeRuneInString(text[i:])
		units := int(utf16Units(r))

		if units > col {
			break
		}

		col -= units
		i += size
	}
	return i
}

// collectLocalsBeforeCursor walks top-level declarations from the current file
// and returns all local declarations (DeclParameter, local DeclVariable,
// DeclForBinding) whose defining token starts before cursorOffset.
func collectLocalsBeforeCursor(module *ast.ContextModule, sourceURI string, cursorOffset int) []ast.Decl {
	var results []ast.Decl
	for _, sym := range module.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}

		// Only walk top-level decls from the current file.
		nameToken := sym.Decl.DeclName().Token
		if nameToken.Source == nil || nameToken.Source.File != sourceURI {
			continue
		}

		if nodeOffset(sym.Decl) >= cursorOffset {
			continue
		}

		switch d := sym.Decl.(type) {
		case ast.DeclFunc:
			if d.Impl != nil {
				results = append(results, exprFuncLocals(d.Impl, cursorOffset)...)
			}
		case *ast.DeclFunc:
			if d.Impl != nil {
				results = append(results, exprFuncLocals(d.Impl, cursorOffset)...)
			}
		}
	}

	return results
}

// exprFuncLocals collects locals visible at cursorOffset inside an ExprFunc.
func exprFuncLocals(fn *ast.ExprFunc, cursorOffset int) []ast.Decl {
	if fn == nil || nodeOffset(fn) >= cursorOffset {
		return nil
	}

	var results []ast.Decl
	for _, p := range fn.Parameters {
		results = append(results, p)
	}
	results = append(results, blockLocals(fn.Impl, cursorOffset)...)

	return results
}

// blockLocals walks a Block (slice of statements) and collects local declarations
// whose token starts before cursorOffset.
func blockLocals(block ast.Block, cursorOffset int) []ast.Decl {
	var results []ast.Decl
	for _, stmt := range block {
		if nodeOffset(stmt) >= cursorOffset {
			break
		}

		switch s := stmt.(type) {
		case *ast.DeclVariable:
			if !s.IsGlobal {
				results = append(results, s)
			}
		case *ast.DeclConstant:
			if !s.IsGlobal {
				results = append(results, s)
			}
		case ast.DeclFunc:
			if s.Impl != nil {
				results = append(results, exprFuncLocals(s.Impl, cursorOffset)...)
			}
		case *ast.DeclFunc:
			if s.Impl != nil {
				results = append(results, exprFuncLocals(s.Impl, cursorOffset)...)
			}
		case *ast.StmtFor:
			if s.CollectionIdent != nil {
				results = append(results, ast.MakeDeclForBinding(s.Token, *s.CollectionIdent))
			}
			results = append(results, blockLocals(s.Body, cursorOffset)...)
		}
	}
	return results
}

// nodeOffset returns the byte offset of a node's leading token, or -1 if unavailable.
func nodeOffset(node ast.Node) int {
	tok := node.TokenLiteral()

	if tok.Source == nil {
		return -1
	}

	return tok.Source.Offset
}
