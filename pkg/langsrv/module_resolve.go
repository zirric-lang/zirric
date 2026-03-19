package langsrv

import (
	"context"
	"path/filepath"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// qualifiedContext detects if the cursor is immediately after "alias." and returns
// the alias name, the position just after the dot, and true.
// For example: "mymod.Foo" with cursor on/after "Foo" → ("mymod", posAfterDot, true).
// Also works for attribute context, e.g. "@alias.Foo".
func qualifiedContext(text string, pos protocol.Position) (alias string, afterDot protocol.Position, ok bool) {
	line := lineAtPosition(text, pos)
	col := int(pos.Character)
	if col > len(line) {
		col = len(line)
	}

	// Scan backward past simple ident chars to find start of the current member name.
	memberStart := col
	for memberStart > 0 && isSimpleIdentByte(line[memberStart-1]) {
		memberStart--
	}
	// Must have '.' immediately before the member.
	if memberStart == 0 || line[memberStart-1] != '.' {
		return "", protocol.Position{}, false
	}
	dotIdx := memberStart - 1

	// Scan backward to extract the alias identifier before the dot.
	aliasEnd := dotIdx
	aliasStart := aliasEnd
	for aliasStart > 0 && isSimpleIdentByte(line[aliasStart-1]) {
		aliasStart--
	}
	if aliasStart == aliasEnd {
		return "", protocol.Position{}, false
	}

	alias = line[aliasStart:aliasEnd]
	afterDot = protocol.Position{Line: pos.Line, Character: uint32(memberStart)}
	return alias, afterDot, true
}

// lineAtPosition returns the text content of the line at the given position.
func lineAtPosition(text string, pos protocol.Position) string {
	lineNum := int(pos.Line)
	start := 0

	for i := 0; i < lineNum; i++ {
		idx := strings.IndexByte(text[start:], '\n')
		if idx < 0 {
			return ""
		}
		start += idx + 1
	}

	end := strings.IndexByte(text[start:], '\n')
	if end < 0 {
		return text[start:]
	}
	return text[start : start+end]
}

// findSourceFile returns the SourceFile matching sourceURI in the module, or nil.
func findSourceFile(module *ast.ContextModule, sourceURI string) *ast.SourceFile {
	for _, sf := range module.Files {
		if sf.Path == sourceURI {
			return sf
		}
	}
	return nil
}

// findImportDecl looks up a DeclImport by alias in a SourceFile's local symbol table.
func findImportDecl(sf *ast.SourceFile, alias string) (*ast.DeclImport, bool) {
	if sf == nil {
		return nil, false
	}

	sym := sf.Decls.Symbols[alias]
	if sym == nil || sym.Decl == nil {
		return nil, false
	}

	switch d := sym.Decl.(type) {
	case *ast.DeclImport:
		return d, true
	case ast.DeclImport:
		return &d, true
	}
	return nil, false
}

// importedModuleDir converts a DeclImport's module name to a relative directory path
// and verifies that it contains .zirr files.
func (ls *zirricLangserver) importedModuleDir(imp *ast.DeclImport) (string, bool) {
	if len(imp.ModuleName) == 0 {
		return "", false
	}

	parts := make([]string, len(imp.ModuleName))
	for i, id := range imp.ModuleName {
		parts[i] = id.Value
	}

	relPath := filepath.Join(parts...)
	entries, err := ls.fs.ReadDir(relPath)
	if err != nil {
		return "", false
	}

	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".zirr") {
			return relPath, true
		}
	}
	return "", false
}

// loadImportedModule parses the module referenced by a DeclImport.
// Returns the parsed module, a sourceURI→relPath map, and whether it succeeded.
func (ls *zirricLangserver) loadImportedModule(imp *ast.DeclImport) (*ast.ContextModule, map[string]string, bool) {
	dir, ok := ls.importedModuleDir(imp)
	if !ok {
		return nil, nil, false
	}

	mod, _, srcToPath, err := ls.parseModuleFiles(dir)
	if err != nil {
		return nil, nil, false
	}
	return mod, srcToPath, true
}

// loadImportMemberModule parses the module referenced by a DeclImportMember.
func (ls *zirricLangserver) loadImportMemberModule(dim ast.DeclImportMember) (*ast.ContextModule, map[string]string, bool) {
	synthImp := &ast.DeclImport{ModuleName: ast.ModuleName(dim.ModuleName)}
	return ls.loadImportedModule(synthImp)
}

// resolveImportMemberDecl resolves a DeclImportMember to its actual declaration.
// First tries filesystem-based loading, then falls back to the Orchestra resolver
// (needed for embedded modules like the prelude).
func (ls *zirricLangserver) resolveImportMemberDecl(dim ast.DeclImportMember) ast.Decl {
	name := dim.Name.Value
	if mod, _, ok := ls.loadImportMemberModule(dim); ok {
		if sym, ok := mod.Decls.Resolve(name); ok && sym.Decl != nil {
			return sym.Decl
		}
	}
	// Fallback: resolve via Orchestra resolver (covers embedded/installed packages).
	if ls.resolver != nil {
		uri := registry.LogicalURI(ast.StaticReference(dim.ModuleName).String())
		if mod, err := ls.resolver.ResolveModule(context.Background(), uri); err == nil && mod != nil {
			if sym, ok := mod.Decls.Resolve(name); ok && sym.Decl != nil {
				return sym.Decl
			}
		}
	}
	return nil
}

// resolveImportMemberLocation resolves a DeclImportMember to the LSP Location of
// the actual declaration. Tries filesystem, then Orchestra resolver.
func (ls *zirricLangserver) resolveImportMemberLocation(dim ast.DeclImportMember, name string) *protocol.Location {
	if importedMod, srcToPath, ok := ls.loadImportMemberModule(dim); ok {
		if loc, ok := ls.symbolLocation(importedMod, name, srcToPath); ok {
			return loc
		}
	}
	// Fallback: resolve via Orchestra resolver.
	resolved := ls.resolveImportMemberDecl(dim)
	if resolved == nil {
		return nil
	}
	return ls.locationForDecl(resolved)
}

// locationForDecl returns the LSP Location of a declaration's name token,
// reading the source file to compute line/column positions.
func (ls *zirricLangserver) locationForDecl(decl ast.Decl) *protocol.Location {
	return ls.locationForDeclWithPaths(decl, nil)
}

// locationForDeclWithPaths returns the LSP Location of a declaration's name token,
// using srcToPath to translate source URIs to filesystem paths when available.
func (ls *zirricLangserver) locationForDeclWithPaths(decl ast.Decl, srcToPath map[string]string) *protocol.Location {
	nameToken := decl.DeclName().Token
	if nameToken.Source == nil {
		return nil
	}
	filePath := nameToken.Source.File
	if srcToPath != nil {
		if mapped, ok := srcToPath[filePath]; ok {
			filePath = mapped
		}
	}
	text, err := readFileText(ls.fs, filePath)
	if err != nil {
		return nil
	}
	offset := nameToken.Source.Offset
	nameLen := len(nameToken.Literal)
	if nameLen <= 0 {
		nameLen = 1
	}
	return &protocol.Location{
		URI:   ls.fileURI(filePath),
		Range: rangeForOffsets(text, offset, offset+nameLen),
	}
}

// symbolLocation resolves a symbol name in a module and returns its LSP Location.
func (ls *zirricLangserver) symbolLocation(mod *ast.ContextModule, name string, srcToPath map[string]string) (*protocol.Location, bool) {
	sym, _ := mod.Decls.Resolve(name)
	if sym == nil || sym.Decl == nil {
		return nil, false
	}

	nameToken := sym.Decl.DeclName().Token
	if nameToken.Source == nil {
		return nil, false
	}

	defFilePath, ok := srcToPath[nameToken.Source.File]
	if !ok {
		return nil, false
	}

	defText, err := readFileText(ls.fs, defFilePath)
	if err != nil {
		return nil, false
	}

	offset := nameToken.Source.Offset
	nameLen := len(nameToken.Literal)
	if nameLen <= 0 {
		nameLen = 1
	}

	return &protocol.Location{
		URI:   ls.fileURI(defFilePath),
		Range: rangeForOffsets(defText, offset, offset+nameLen),
	}, true
}

// hoverDeclInModule looks up a symbol in an imported module and returns its hover content.
func hoverDeclInModule(mod *ast.ContextModule, name string) string {
	sym, _ := mod.Decls.Resolve(name)
	if sym == nil || sym.Decl == nil {
		return ""
	}

	return hoverContentForDecl(sym.Decl)
}
