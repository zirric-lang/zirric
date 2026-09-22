package langsrv

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// dotChainContext extracts the full dot-separated chain of identifiers before
// the cursor position. For text like "example.person.name.le" with cursor after
// "le", it returns segments=["example", "person", "name"], afterDot is the
// position just after the last '.', and ok=true.
//
// For "alias.member" it returns segments=["alias"], afterDot after the dot.
// Returns ok=false if no dot-qualified access is present.
func dotChainContext(text string, pos protocol.Position) (segments []string, afterDot protocol.Position, ok bool) {
	line := lineAtPosition(text, pos)
	col := int(pos.Character)
	if col > len(line) {
		col = len(line)
	}

	// Scan backward past the current (partial) member name.
	memberStart := col
	for memberStart > 0 && isSimpleIdentByte(line[memberStart-1]) {
		memberStart--
	}
	// Must have '.' immediately before the member.
	if memberStart == 0 || line[memberStart-1] != '.' {
		return nil, protocol.Position{}, false
	}

	afterDot = protocol.Position{Line: pos.Line, Character: uint32(memberStart)}

	// Scan backward collecting "ident." segments.
	cursor := memberStart - 1 // pointing at the '.'
	for {
		// We're pointing at a '.'. Scan backward past the ident before it.
		identEnd := cursor
		identStart := identEnd
		for identStart > 0 && isSimpleIdentByte(line[identStart-1]) {
			identStart--
		}
		if identStart == identEnd {
			// No identifier before the dot — invalid chain.
			return nil, protocol.Position{}, false
		}
		segments = append(segments, line[identStart:identEnd])

		// Check if there's another '.' before this identifier.
		if identStart == 0 || line[identStart-1] != '.' {
			break
		}
		cursor = identStart - 1
	}

	// Segments were collected in reverse order.
	for i, j := 0, len(segments)-1; i < j; i, j = i+1, j-1 {
		segments[i], segments[j] = segments[j], segments[i]
	}
	return segments, afterDot, true
}

// resolvedChain holds the result of resolving a dot-separated member chain.
type resolvedChain struct {
	// fields contains the members available on the resolved type
	// (for completion after the final dot).
	fields []ast.DeclField

	// lastDecl is the declaration of the last resolved segment.
	// For "person.name", lastDecl is the DeclField "name".
	lastDecl ast.Decl

	// typeDecl is the type declaration that lastDecl's TypeHint resolves to.
	// For "person.name" (name: String), typeDecl is DeclExternType String.
	typeDecl ast.Decl

	// module is the module context used for symbol resolution.
	module *ast.ContextModule

	// isModuleScope is true when the chain resolved entirely as a module
	// reference (import alias or mod declaration) with no type-based steps.
	isModuleScope bool
}

// resolveDotChain resolves a chain of dot-separated segments through
// type-aware member access. The segments are everything BEFORE the final dot.
//
// For example, for "example.person.name." the segments are ["example", "person", "name"].
// The function resolves each segment through type hints and returns the fields
// available at the end of the chain.
//
// The first segment can be:
//   - An import alias → load the imported module
//   - A mod declaration name → use the current module
//   - A local/global variable, constant, or parameter → resolve its TypeHint
//
// Subsequent segments resolve through the type of the previous segment.
func (ls *zirricLangserver) resolveDotChain(
	module *ast.ContextModule,
	currentSF *ast.SourceFile,
	path string,
	cursorOffset int,
	segments []string,
) *resolvedChain {
	if len(segments) == 0 {
		return nil
	}

	first := segments[0]
	rest := segments[1:]

	// Try import alias.
	if imp, ok := findImportDecl(currentSF, first); ok {
		if importedMod, _, ok := ls.loadImportedModule(imp); ok {
			return ls.resolveDotChainInModule(importedMod, module, rest)
		}
	}

	// Try current module's mod declaration.
	if isModuleDecl(module, first) {
		return ls.resolveDotChainInModule(module, module, rest)
	}

	// Try local/global symbol with a type hint.
	return ls.resolveDotChainFromSymbol(module, currentSF, path, cursorOffset, first, rest)
}

// resolveDotChainInModule resolves remaining segments starting from a module scope.
// If rest is empty, returns the module scope (isModuleScope=true).
// Otherwise, resolves the first element in rest as a module member, then follows
// the type chain for remaining segments.
func (ls *zirricLangserver) resolveDotChainInModule(
	targetMod *ast.ContextModule,
	localMod *ast.ContextModule,
	rest []string,
) *resolvedChain {
	if len(rest) == 0 {
		return &resolvedChain{module: targetMod, isModuleScope: true}
	}

	sym, ok := targetMod.Decls.Resolve(rest[0])
	if !ok || sym.Decl == nil {
		return nil
	}

	if len(rest) == 1 {
		// One more segment — resolve its type to get available fields.
		return resolvedChainForDecl(sym.Decl, localMod)
	}

	// Multiple remaining segments — follow the type chain.
	return resolveChainThroughType(sym.Decl, localMod, rest[1:])
}

// resolveDotChainFromSymbol resolves a chain starting from a local/global symbol.
// The first segment is the symbol name; rest are the subsequent dot-separated members.
func (ls *zirricLangserver) resolveDotChainFromSymbol(
	module *ast.ContextModule,
	currentSF *ast.SourceFile,
	path string,
	cursorOffset int,
	symbolName string,
	rest []string,
) *resolvedChain {
	sourceURI := string(registry.JoinModuleURI("", path))

	// Check local scope first.
	decl := findLocalDecl(module, sourceURI, cursorOffset, symbolName)
	if decl == nil {
		// Try module-level.
		if sym, ok := module.Decls.Resolve(symbolName); ok && sym.Decl != nil {
			decl = sym.Decl
		}
	}
	if decl == nil {
		return nil
	}

	if len(rest) == 0 {
		return resolvedChainForDecl(decl, module)
	}
	return resolveChainThroughType(decl, module, rest)
}

// resolvedChainForDecl creates a resolvedChain from a single declaration,
// resolving its TypeHint to determine available fields.
func resolvedChainForDecl(decl ast.Decl, module *ast.ContextModule) *resolvedChain {
	typeHint := typeHintFromDecl(decl)
	if typeHint == nil {
		return &resolvedChain{lastDecl: decl, module: module}
	}
	typeDecl := resolveTypeExpr(module, typeHint)
	if typeDecl == nil {
		return &resolvedChain{lastDecl: decl, module: module}
	}
	fields := fieldsForTypeDecl(typeDecl)
	return &resolvedChain{
		fields:   fields,
		lastDecl: decl,
		typeDecl: typeDecl,
		module:   module,
	}
}

// resolveChainThroughType walks a chain of field names through the type system.
// decl is the starting declaration, rest are the remaining segment names.
func resolveChainThroughType(decl ast.Decl, module *ast.ContextModule, rest []string) *resolvedChain {
	currentDecl := decl
	for _, seg := range rest {
		typeHint := typeHintFromDecl(currentDecl)
		if typeHint == nil {
			return nil
		}
		typeDecl := resolveTypeExpr(module, typeHint)
		if typeDecl == nil {
			return nil
		}
		field := findFieldByName(typeDecl, seg)
		if field == nil {
			return nil
		}
		currentDecl = *field
	}
	// Resolve the final declaration's type for available fields.
	return resolvedChainForDecl(currentDecl, module)
}

// typeHintFromDecl extracts the TypeExpr from a declaration that carries a type hint.
func typeHintFromDecl(decl ast.Decl) ast.TypeExpr {
	switch d := decl.(type) {
	case ast.DeclVariable:
		return d.TypeHint
	case *ast.DeclVariable:
		return d.TypeHint
	case ast.DeclConstant:
		return d.TypeHint
	case *ast.DeclConstant:
		return d.TypeHint
	case ast.DeclParameter:
		return d.TypeHint
	case *ast.DeclParameter:
		return d.TypeHint
	case ast.DeclField:
		return d.TypeHint
	case *ast.DeclField:
		return d.TypeHint
	}
	return nil
}

// resolveTypeExpr resolves a type expression to a declaration in the module's scope.
// It walks the DeclTable parent chain, so prelude types (String, Int, etc.) are found.
func resolveTypeExpr(module *ast.ContextModule, typeExpr ast.TypeExpr) ast.Decl {
	if module == nil || module.Decls == nil {
		return nil
	}
	switch te := typeExpr.(type) {
	case ast.TypeExprRef:
		name := te.Reference.Name().Value
		if sym, ok := module.Decls.Resolve(name); ok && sym.Decl != nil {
			return sym.Decl
		}
	case ast.TypeExprFunc:
		if sym, ok := module.Decls.Resolve("Func"); ok && sym.Decl != nil {
			return sym.Decl
		}
	case ast.TypeExprArray:
		if sym, ok := module.Decls.Resolve("Array"); ok && sym.Decl != nil {
			return sym.Decl
		}
	case ast.TypeExprDict:
		if sym, ok := module.Decls.Resolve("Dict"); ok && sym.Decl != nil {
			return sym.Decl
		}
	}
	return nil
}

// fieldsForTypeDecl returns the fields/members of a type declaration.
func fieldsForTypeDecl(decl ast.Decl) []ast.DeclField {
	switch d := decl.(type) {
	case *ast.DeclData:
		return d.Fields
	case ast.DeclData:
		return d.Fields
	case *ast.DeclAttr:
		return d.Fields
	case ast.DeclAttr:
		return d.Fields
	case *ast.DeclExternType:
		return d.Fields
	case ast.DeclExternType:
		return d.Fields
	}
	return nil
}

// findFieldByName looks up a field by name in a type declaration.
func findFieldByName(typeDecl ast.Decl, name string) *ast.DeclField {
	switch d := typeDecl.(type) {
	case *ast.DeclData:
		for i := range d.Fields {
			if d.Fields[i].Name.Value == name {
				return &d.Fields[i]
			}
		}
	case ast.DeclData:
		for i := range d.Fields {
			if d.Fields[i].Name.Value == name {
				return &d.Fields[i]
			}
		}
	case *ast.DeclAttr:
		for i := range d.Fields {
			if d.Fields[i].Name.Value == name {
				return &d.Fields[i]
			}
		}
	case ast.DeclAttr:
		for i := range d.Fields {
			if d.Fields[i].Name.Value == name {
				return &d.Fields[i]
			}
		}
	case *ast.DeclExternType:
		for i := range d.Fields {
			if d.Fields[i].Name.Value == name {
				return &d.Fields[i]
			}
		}
	case ast.DeclExternType:
		for i := range d.Fields {
			if d.Fields[i].Name.Value == name {
				return &d.Fields[i]
			}
		}
	}
	return nil
}

// fieldCompletions builds completion items for a set of fields (members).
func fieldCompletions(
	fields []ast.DeclField,
	afterDot, cursorPos protocol.Position,
	isTypeExpr bool,
) []protocol.CompletionItem {
	memberRange := protocol.Range{Start: afterDot, End: cursorPos}
	var items []protocol.CompletionItem
	for _, field := range fields {
		kind := protocol.CompletionItemKindField
		format := protocol.InsertTextFormatPlainText
		newText := field.Name.Value
		label := field.Name.Value

		if !isTypeExpr && len(field.Parameters) > 0 {
			format = protocol.InsertTextFormatSnippet
			newText = field.Name.Value + "($0)"
			kind = protocol.CompletionItemKindMethod
		}

		item := protocol.CompletionItem{
			Label:            label,
			Kind:             &kind,
			InsertTextFormat: &format,
			TextEdit:         protocol.TextEdit{Range: memberRange, NewText: newText},
		}
		detail := field.DeclOverview()
		if detail != "" {
			item.Detail = &detail
		}
		items = append(items, item)
	}
	return items
}
