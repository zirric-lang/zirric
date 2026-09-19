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

	module, _, _, err := ls.parseModuleFilesForPath(path)
	if err != nil {
		return nil, err
	}

	var (
		atPos, inAttribute = attributeContextStart(text, pos)
		cursorOffset       = offsetForPosition(text, pos)
		sourceURI          = string(registry.JoinModuleURI("", path))
		currentSF          = findSourceFile(module, sourceURI)
		scope              = detectCompletionScope(text, pos, module, sourceURI, cursorOffset)
	)
	scope.dirName = sanitizeModuleName(filepath.Base(filepath.Dir(path)))

	// Import statement's dotted module path ("import <cursor>"), distinct from scope.isImportBlock's member list.
	if startPos, ok := importNameContext(text, pos); ok {
		return ls.importModuleNameCompletions(startPos, pos), nil
	}

	// Handle dot-chain context: "alias.member", "modname.member", or "expr.field.subfield"
	if segments, afterDot, isDotChain := dotChainContext(text, pos); isDotChain {
		// Single-segment chains: try module/import alias first (existing behavior).
		if len(segments) == 1 {
			alias := segments[0]
			if imp, ok := findImportDecl(currentSF, alias); ok {
				if importedMod, _, ok := ls.loadImportedModule(imp); ok {
					return importedModuleCompletions(importedMod, inAttribute, scope.isTypeExpr, afterDot, pos), nil
				}
			}
			if isModuleDecl(module, alias) {
				return importedModuleCompletions(module, inAttribute, scope.isTypeExpr, afterDot, pos), nil
			}
		}

		// Multi-segment chains or single-segment non-module: type-aware resolution.
		result := ls.resolveDotChain(module, currentSF, path, cursorOffset, segments)
		if result != nil {
			if result.isModuleScope {
				return importedModuleCompletions(result.module, inAttribute, scope.isTypeExpr, afterDot, pos), nil
			}
			if len(result.fields) > 0 {
				return fieldCompletions(result.fields, afterDot, pos, scope.isTypeExpr), nil
			}
		}
		return nil, nil
	}

	// Import block context: only complete members not yet imported.
	if scope.isImportBlock {
		return ls.importBlockCompletions(text, pos, currentSF), nil
	}

	var items []protocol.CompletionItem
	for _, sym := range module.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}

		if inAttribute {
			// Skip attrs already present in the current @-chain.
			if scope.usedAttrs[sym.Name] {
				continue
			}
			if scope.isTypeExpr {
				// In type expression @: only attribute types without call syntax.
				if item, ok := typeExprAttributeItem(sym, atPos, pos); ok {
					items = append(items, item)
				}
			} else {
				// Declaration/global @: attribute with call syntax.
				if item, ok := attributeContextCompletionItem(sym, atPos, pos); ok {
					items = append(items, item)
				}
			}
		} else {
			item := contextAwareCompletionItem(sym, scope)
			items = append(items, item)
		}
	}

	// Add import aliases and directly imported members from the current file's local scope.
	if currentSF != nil {
		for _, sym := range currentSF.Decls.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}

			if inAttribute {
				// Skip attrs already present in the current @-chain.
				if scope.usedAttrs[sym.Name] {
					continue
				}
				// For imported members, resolve to actual declaration and check if it's an attr.
				switch dim := sym.Decl.(type) {
				case ast.DeclImportMember:
					resolved := ls.resolveImportMemberDecl(dim)
					if resolved == nil {
						continue
					}
					if scope.isTypeExpr {
						if isAttrDecl(resolved) {
							kind := protocol.CompletionItemKindInterface
							label := "@" + sym.Name
							filterText := sym.Name
							editRange := protocol.Range{Start: atPos, End: pos}
							items = append(items, protocol.CompletionItem{
								Label: label, FilterText: &filterText, Kind: &kind,
								TextEdit: protocol.TextEdit{Range: editRange, NewText: "@" + sym.Name},
							})
						}
					} else {
						if attr, ok := asAttrDecl(resolved); ok {
							editRange := protocol.Range{Start: atPos, End: pos}
							items = append(items, attributeSnippetItem(sym.Name, attr.Fields, editRange))
						}
					}
				case *ast.DeclImportMember:
					resolved := ls.resolveImportMemberDecl(*dim)
					if resolved == nil {
						continue
					}
					if scope.isTypeExpr {
						if isAttrDecl(resolved) {
							kind := protocol.CompletionItemKindInterface
							label := "@" + sym.Name
							filterText := sym.Name
							editRange := protocol.Range{Start: atPos, End: pos}
							items = append(items, protocol.CompletionItem{
								Label: label, FilterText: &filterText, Kind: &kind,
								TextEdit: protocol.TextEdit{Range: editRange, NewText: "@" + sym.Name},
							})
						}
					} else {
						if attr, ok := asAttrDecl(resolved); ok {
							editRange := protocol.Range{Start: atPos, End: pos}
							items = append(items, attributeSnippetItem(sym.Name, attr.Fields, editRange))
						}
					}
				}
			} else {
				switch sym.Decl.(type) {
				case ast.DeclImport, *ast.DeclImport, ast.DeclImportMember, *ast.DeclImportMember,
					*ast.DeclModule, ast.DeclModule:
					items = append(items, completionItemForDeclSymbol(sym))
				}
			}
		}

		// Also offer qualified @alias.Attr completions from imported modules.
		for _, sym := range currentSF.Decls.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}
			imp, isImport := extractImportDecl(sym.Decl)
			if !isImport || imp == nil {
				continue
			}
			alias := sym.Name
			importedMod, _, ok := ls.loadImportedModule(imp)
			if !ok {
				continue
			}
			for _, msym := range importedMod.Decls.Symbols {
				if msym == nil || msym.Decl == nil {
					continue
				}
				if !isDeclaredInModule(msym.Decl, importedMod) {
					continue
				}
				qualName := alias + "." + msym.Name
				if scope.usedAttrs[qualName] {
					continue
				}
				switch decl := msym.Decl.(type) {
				case *ast.DeclAttr:
					editRange := protocol.Range{Start: atPos, End: pos}
					if scope.isTypeExpr {
						kind := protocol.CompletionItemKindInterface
						label := "@" + qualName
						filterText := qualName
						items = append(items, protocol.CompletionItem{
							Label: label, FilterText: &filterText, Kind: &kind,
							TextEdit: protocol.TextEdit{Range: editRange, NewText: "@" + qualName},
						})
					} else {
						items = append(items, qualifiedAttributeSnippetItem(alias, msym.Name, decl.Fields, editRange))
					}
				}
			}
		}
	}

	// Add context-aware keyword completions in non-attribute context.
	if !inAttribute {
		items = append(items, contextKeywordItems(scope)...)
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

// importedModuleCompletions returns completion items for public symbols in an imported module.
// In attribute context, only attribute declarations are included with snippet TextEdits.
// In non-attribute context, all module-declared symbols are included (no prelude leak).
// Attrs in module.Member context are shown without @ prefix.
func importedModuleCompletions(
	mod *ast.ContextModule,
	inAttribute bool,
	isTypeExpr bool,
	afterDot, cursorPos protocol.Position,
) []protocol.CompletionItem {
	memberRange := protocol.Range{Start: afterDot, End: cursorPos}
	var items []protocol.CompletionItem
	for _, sym := range mod.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		// Skip prelude symbols — only include symbols declared in this module's own files.
		if !isDeclaredInModule(sym.Decl, mod) {
			continue
		}
		// Skip the module declaration itself — it's not a member.
		switch sym.Decl.(type) {
		case *ast.DeclModule, ast.DeclModule:
			continue
		}
		if inAttribute {
			switch decl := sym.Decl.(type) {
			case *ast.DeclAttr:
				if isTypeExpr {
					items = append(items, typeAttributeMemberItem(sym.Name, memberRange))
				} else {
					items = append(items, attributeMemberSnippetItem(sym.Name, decl.Fields, memberRange))
				}
			}
		} else {
			// In module.Member context, show all symbols with context-aware formatting.
			kind := completionKindForDecl(sym.Decl)
			format := protocol.InsertTextFormatPlainText
			newText := sym.Name
			label := sym.Name

			if !isTypeExpr {
				switch decl := sym.Decl.(type) {
				case *ast.DeclFunc, ast.DeclFunc, *ast.DeclExternFunc, ast.DeclExternFunc:
					format = protocol.InsertTextFormatSnippet
					newText = sym.Name + "($0)"
					kind = protocol.CompletionItemKindFunction
				case *ast.DeclData:
					format = protocol.InsertTextFormatSnippet
					newText = sym.Name + "($0)"
					kind = protocol.CompletionItemKindConstructor
					_ = decl
				case ast.DeclData:
					format = protocol.InsertTextFormatSnippet
					newText = sym.Name + "($0)"
					kind = protocol.CompletionItemKindConstructor
				case *ast.DeclAttr, ast.DeclAttr:
					// Attrs in non-@ module context are just plain names.
				}
			}

			item := protocol.CompletionItem{
				Label:            label,
				FilterText:       &label,
				Kind:             &kind,
				InsertTextFormat: &format,
				TextEdit:         protocol.TextEdit{Range: memberRange, NewText: newText},
			}
			if ov, ok := sym.Decl.(ast.Overviewable); ok {
				detail := ov.DeclOverview()
				if detail != "" {
					item.Detail = &detail
				}
			}
			items = append(items, item)
		}
	}
	return items
}

// isDeclaredInModule returns true if the declaration was defined in one of the module's own source files.
func isDeclaredInModule(decl ast.Decl, mod *ast.ContextModule) bool {
	tok := decl.DeclName().Token
	if tok.Source == nil {
		return false
	}
	for _, sf := range mod.Files {
		if sf.Path == tok.Source.File {
			return true
		}
	}
	return false
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
// Only attribute declarations get completions; data, union, and extern types
// are excluded from @ context.
// The TextEdit replaces from the '@' character up to the cursor so that the
// resulting text always contains the leading '@'.
func attributeContextCompletionItem(sym *ast.DeclSymbol, atPos, cursorPos protocol.Position) (protocol.CompletionItem, bool) {
	editRange := protocol.Range{Start: atPos, End: cursorPos}

	switch decl := sym.Decl.(type) {
	case *ast.DeclAttr:
		return attributeSnippetItem(sym.Name, decl.Fields, editRange), true
	case ast.DeclAttr:
		return attributeSnippetItem(sym.Name, decl.Fields, editRange), true
	default:
		return protocol.CompletionItem{}, false
	}
}

// attributeSnippetItem builds a CompletionItem for an attribute declaration.
// Always includes () syntax — even with zero fields — because attributes before
// declarations use call syntax. The TextEdit replaces from '@' to the cursor.
func attributeSnippetItem(name string, fields []ast.DeclField, editRange protocol.Range) protocol.CompletionItem {
	var (
		kind       = protocol.CompletionItemKindInterface
		label      = "@" + name
		filterText = name
		format     = protocol.InsertTextFormatSnippet
	)

	newText := fmt.Sprintf("@%s($0)", name)
	return protocol.CompletionItem{
		Label:            label,
		FilterText:       &filterText,
		Kind:             &kind,
		InsertTextFormat: &format,
		TextEdit:         protocol.TextEdit{Range: editRange, NewText: newText},
	}
}

// qualifiedAttributeSnippetItem builds an attribute completion for a qualified @alias.Name in
// declaration context (with call syntax).
func qualifiedAttributeSnippetItem(alias, name string, fields []ast.DeclField, editRange protocol.Range) protocol.CompletionItem {
	qualName := alias + "." + name
	var (
		kind       = protocol.CompletionItemKindInterface
		label      = "@" + qualName
		filterText = qualName
		format     = protocol.InsertTextFormatSnippet
	)

	newText := fmt.Sprintf("@%s($0)", qualName)
	return protocol.CompletionItem{
		Label:            label,
		FilterText:       &filterText,
		Kind:             &kind,
		InsertTextFormat: &format,
		TextEdit:         protocol.TextEdit{Range: editRange, NewText: newText},
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
		insertText = fmt.Sprintf("@%s($0)", name)
	)

	return protocol.CompletionItem{
		Label:            label,
		Kind:             &kind,
		InsertText:       &insertText,
		InsertTextFormat: &format,
	}
}

// contextKeywordItems returns context-aware keyword completions based on cursor scope.
func contextKeywordItems(scope completionScope) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindKeyword
	snippetFmt := protocol.InsertTextFormatSnippet

	var items []protocol.CompletionItem
	addKw := func(label string) {
		items = append(items, protocol.CompletionItem{Label: label, Kind: &kind})
	}
	addSnippet := func(label, insert string) {
		items = append(items, protocol.CompletionItem{
			Label:            label,
			Kind:             &kind,
			InsertText:       &insert,
			InsertTextFormat: &snippetFmt,
		})
	}

	// 'mod' only at first top-level position.
	if scope.isFirstStmt && scope.isTopLevel {
		if scope.dirName != "" && scope.dirName != "." {
			addSnippet("mod", "mod ${1:"+scope.dirName+"}")
		} else {
			addSnippet("mod", "mod ${1:name}")
		}
	}

	// Top-level only declarations.
	if scope.isTopLevel {
		addKw("data")
		addSnippet("extern type", "extern type ${1:Name}")
		addSnippet("extern fn", "extern fn ${1:name}($0)")
		addSnippet("extern const", "extern const ${1:name}")
		addKw("union")
		addKw("import")
		addKw("attr")
	}

	// Available at top-level, as statement, or as expression.
	addKw("fn")
	addKw("if")
	addKw("switch")
	addKw("for")

	// var/const available at top-level, as statement, or inside for loops.
	addKw("var")
	addKw("const")

	// 'break'/'continue' only inside for loops.
	if scope.inFor {
		addKw("break")
		addKw("continue")
	}

	// 'return' only inside functions.
	if scope.inFunc {
		addKw("return")
	}

	// Always-available value keywords.
	addKw("true")
	addKw("false")
	addKw("void")
	addKw("is")

	return items
}

func completionKindForDecl(decl ast.Decl) protocol.CompletionItemKind {
	switch decl.(type) {
	case *ast.DeclFunc, ast.DeclFunc, *ast.DeclExternFunc, ast.DeclExternFunc:
		return protocol.CompletionItemKindFunction

	case *ast.DeclVariable, ast.DeclVariable:
		return protocol.CompletionItemKindVariable

	case *ast.DeclConstant, ast.DeclConstant, *ast.DeclExternValue, ast.DeclExternValue:
		return protocol.CompletionItemKindConstant

	case *ast.DeclData, ast.DeclData:
		return protocol.CompletionItemKindStruct

	case *ast.DeclExternType, ast.DeclExternType:
		return protocol.CompletionItemKindClass

	case *ast.DeclUnion, ast.DeclUnion:
		return protocol.CompletionItemKindEnum

	case *ast.DeclAttr, ast.DeclAttr:
		return protocol.CompletionItemKindInterface

	case *ast.DeclModule, ast.DeclModule:
		return protocol.CompletionItemKindModule

	case *ast.DeclImport, ast.DeclImport:
		return protocol.CompletionItemKindFolder

	case *ast.DeclImportMember, ast.DeclImportMember:
		return protocol.CompletionItemKindReference

	default:
		return protocol.CompletionItemKindText
	}
}

// attributeMemberSnippetItem builds an attribute completion for qualified context (@alias.Name).
// The TextEdit replaces only the member portion (after the dot), so "@alias." is preserved.
// Always includes () call syntax. Label shows just the member name since @alias. is already typed.
func attributeMemberSnippetItem(name string, fields []ast.DeclField, memberRange protocol.Range) protocol.CompletionItem {
	var (
		kind       = protocol.CompletionItemKindInterface
		label      = name
		filterText = name
		format     = protocol.InsertTextFormatSnippet
		newText    = fmt.Sprintf("%s($0)", name)
	)

	return protocol.CompletionItem{
		Label:            label,
		FilterText:       &filterText,
		Kind:             &kind,
		InsertTextFormat: &format,
		TextEdit:         protocol.TextEdit{Range: memberRange, NewText: newText},
	}
}

// typeAttributeMemberItem builds a type completion for qualified attribute context (@alias.Type).
// Label shows just the member name since @alias. is already typed.
func typeAttributeMemberItem(name string, memberRange protocol.Range) protocol.CompletionItem {
	var (
		kind       = protocol.CompletionItemKindInterface
		label      = name
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

// contextAwareCompletionItem returns a completion item for a symbol, adjusted
// based on the cursor's scope context.
func contextAwareCompletionItem(sym *ast.DeclSymbol, scope completionScope) protocol.CompletionItem {
	if scope.isTypeExpr {
		// In type expressions, data types appear as Struct kind.
		return completionItemForDeclSymbol(sym)
	}

	switch decl := sym.Decl.(type) {
	case *ast.DeclData:
		// In expression context, data types complete as constructors with parens.
		return dataConstructorCompletionItem(sym.Name, decl)
	case ast.DeclData:
		return dataConstructorCompletionItem(sym.Name, &decl)
	case *ast.DeclFunc, ast.DeclFunc, *ast.DeclExternFunc, ast.DeclExternFunc:
		return funcCompletionItem(sym)
	default:
		return completionItemForDeclSymbol(sym)
	}
}

// dataConstructorCompletionItem creates a snippet completion for a data type
// used as a constructor: Data($0). Parameters are shown via signature help.
func dataConstructorCompletionItem(name string, decl *ast.DeclData) protocol.CompletionItem {
	kind := protocol.CompletionItemKindConstructor
	format := protocol.InsertTextFormatSnippet
	insertText := name + "($0)"

	var detail string
	if ov, ok := ast.Decl(decl).(ast.Overviewable); ok {
		detail = ov.DeclOverview()
	}

	item := protocol.CompletionItem{
		Label:            name,
		Kind:             &kind,
		InsertText:       &insertText,
		InsertTextFormat: &format,
	}
	if detail != "" {
		item.Detail = &detail
	}
	return item
}

// funcCompletionItem creates a completion item for a function that appends parens.
func funcCompletionItem(sym *ast.DeclSymbol) protocol.CompletionItem {
	kind := completionKindForDecl(sym.Decl)
	format := protocol.InsertTextFormatSnippet

	// For functions, add parentheses snippet.
	insertText := sym.Name + "($0)"

	item := protocol.CompletionItem{
		Label:            sym.Name,
		Kind:             &kind,
		InsertText:       &insertText,
		InsertTextFormat: &format,
	}
	if ov, ok := sym.Decl.(ast.Overviewable); ok {
		detail := ov.DeclOverview()
		if detail != "" {
			item.Detail = &detail
		}
	}
	return item
}

// typeExprAttributeItem returns a simple @TypeName completion without call syntax
// for use in type expression positions (after : or ->). Only attributes are valid here.
// Uses a TextEdit from atPos so the existing '@' is replaced, preventing '@@'.
func typeExprAttributeItem(sym *ast.DeclSymbol, atPos, cursorPos protocol.Position) (protocol.CompletionItem, bool) {
	switch sym.Decl.(type) {
	case *ast.DeclAttr, ast.DeclAttr:
		kind := protocol.CompletionItemKindInterface
		label := "@" + sym.Name
		filterText := sym.Name
		editRange := protocol.Range{Start: atPos, End: cursorPos}
		return protocol.CompletionItem{
			Label:      label,
			FilterText: &filterText,
			Kind:       &kind,
			TextEdit:   protocol.TextEdit{Range: editRange, NewText: "@" + sym.Name},
		}, true
	default:
		return protocol.CompletionItem{}, false
	}
}

// isModuleDecl returns true if the given name matches the module's own mod declaration.
func isModuleDecl(module *ast.ContextModule, name string) bool {
	for _, sym := range module.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		switch sym.Decl.(type) {
		case *ast.DeclModule, ast.DeclModule:
			return sym.Name == name
		}
	}
	// Also check source file local symbols (mod has ExportScopeLocal).
	for _, sf := range module.Files {
		for _, sym := range sf.Decls.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}
			switch sym.Decl.(type) {
			case *ast.DeclModule, ast.DeclModule:
				return sym.Name == name
			}
		}
	}
	return false
}

// sanitizeModuleName converts a directory name into a valid Zirric identifier.
// Non-identifier characters (spaces, hyphens, etc.) are replaced with underscores.
// A leading digit gets a '_' prefix.
func sanitizeModuleName(name string) string {
	if name == "" {
		return ""
	}
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' {
			result = append(result, ch)
		} else if '0' <= ch && ch <= '9' {
			if i == 0 || len(result) == 0 {
				result = append(result, '_')
			}
			result = append(result, ch)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}

// isAttrDecl returns true if the declaration is a DeclAttr.
func isAttrDecl(decl ast.Decl) bool {
	switch decl.(type) {
	case *ast.DeclAttr, ast.DeclAttr:
		return true
	}
	return false
}

// extractImportDecl returns the DeclImport if the declaration is one.
func extractImportDecl(decl ast.Decl) (*ast.DeclImport, bool) {
	switch d := decl.(type) {
	case *ast.DeclImport:
		return d, true
	case ast.DeclImport:
		return &d, true
	}
	return nil, false
}

// asAttrDecl returns the DeclAttr if the declaration is one.
func asAttrDecl(decl ast.Decl) (*ast.DeclAttr, bool) {
	switch d := decl.(type) {
	case *ast.DeclAttr:
		return d, true
	case ast.DeclAttr:
		return &d, true
	}
	return nil, false
}

// importBlockCompletions returns completions for members inside an import { } block.
// It finds the enclosing import using brace matching, loads the imported module,
// and returns only the members that haven't been imported yet.
func (ls *zirricLangserver) importBlockCompletions(
	text string,
	pos protocol.Position,
	currentSF *ast.SourceFile,
) []protocol.CompletionItem {
	if currentSF == nil {
		return nil
	}

	cursorOffset := offsetForPosition(text, pos)

	// Find the enclosing import by checking which import's braces contain the cursor.
	var enclosingImport *ast.DeclImport
	for _, sym := range currentSF.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		var imp *ast.DeclImport
		switch d := sym.Decl.(type) {
		case *ast.DeclImport:
			imp = d
		case ast.DeclImport:
			imp = &d
		default:
			continue
		}

		// Use brace matching from the import's token offset.
		importOffset := nodeOffset(sym.Decl)
		if importOffset < 0 || importOffset >= cursorOffset {
			continue
		}
		if cursorInsideBraces(text, importOffset, cursorOffset) {
			enclosingImport = imp
			break
		}
	}

	if enclosingImport == nil {
		return nil
	}

	// Load the imported module.
	importedMod, _, ok := ls.loadImportedModule(enclosingImport)
	if !ok {
		return nil
	}

	// Collect already-imported member names to exclude them.
	alreadyImported := make(map[string]bool)
	for _, m := range enclosingImport.Members {
		alreadyImported[m.Name.Value] = true
	}

	// Add completions for module-declared symbols not already imported.
	// Inside import blocks, all names are plain (no @ prefix, no call syntax).
	var items []protocol.CompletionItem
	for _, sym := range importedMod.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		if alreadyImported[sym.Name] {
			continue
		}
		if !isDeclaredInModule(sym.Decl, importedMod) {
			continue
		}
		kind := completionKindForDecl(sym.Decl)
		item := protocol.CompletionItem{Label: sym.Name, Kind: &kind}
		if ov, ok := sym.Decl.(ast.Overviewable); ok {
			detail := ov.DeclOverview()
			if detail != "" {
				item.Detail = &detail
			}
		}
		items = append(items, item)
	}
	return items
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

	// Walk top-level statements (global for-loops, if-blocks, etc.) for locals.
	for _, sf := range module.Files {
		if sf.Path != sourceURI {
			continue
		}
		results = append(results, blockLocals(sf.Statements, cursorOffset)...)
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
		case ast.StmtFor:
			if s.CollectionIdent != nil {
				results = append(results, ast.MakeDeclForBinding(s.Token, *s.CollectionIdent))
			}
			results = append(results, blockLocals(s.Body, cursorOffset)...)
		case ast.StmtIf:
			results = append(results, stmtIfLocals(s, cursorOffset)...)
		case *ast.StmtIf:
			results = append(results, stmtIfLocals(*s, cursorOffset)...)
		case ast.StmtSwitch:
			results = append(results, stmtSwitchLocals(s, cursorOffset)...)
		case *ast.StmtSwitch:
			results = append(results, stmtSwitchLocals(*s, cursorOffset)...)
		}
	}
	return results
}

// stmtIfLocals recurses into if/else-if/else blocks to find locals.
func stmtIfLocals(s ast.StmtIf, cursorOffset int) []ast.Decl {
	var results []ast.Decl
	results = append(results, blockLocals(s.IfBlock, cursorOffset)...)
	for _, ei := range s.ElseIf {
		results = append(results, blockLocals(ei.Block, cursorOffset)...)
	}
	results = append(results, blockLocals(s.ElseBlock, cursorOffset)...)
	return results
}

// stmtSwitchLocals recurses into switch case bodies to find locals.
func stmtSwitchLocals(s ast.StmtSwitch, cursorOffset int) []ast.Decl {
	var results []ast.Decl
	for _, c := range s.Cases {
		results = append(results, blockLocals(c.Body, cursorOffset)...)
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
