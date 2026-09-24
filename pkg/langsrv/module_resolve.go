package langsrv

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// qualifiedContext detects if the cursor is immediately after "alias." and returns the alias name, the position just after the dot, and true.
// For example: "mymod.Foo" with cursor on/after "Foo" → ("mymod", posAfterDot, true).
// Also works for attribute context, e.g. "@alias.Foo".
func qualifiedContext(text string, pos protocol.Position) (alias string, afterDot protocol.Position, ok bool) {
	line := lineAtPosition(text, pos)
	col := int(pos.Character)
	if col > len(line) {
		col = len(line)
	}

	memberStart := col
	for memberStart > 0 && isSimpleIdentByte(line[memberStart-1]) {
		memberStart--
	}
	if memberStart == 0 || line[memberStart-1] != '.' {
		return "", protocol.Position{}, false
	}
	dotIdx := memberStart - 1

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

// loadImportedModule resolves imp via the same orchestra.ModuleResolver diagnostics use, including its <projectBaseURI>.<name> fallback, so hover/definition/completion/references agree with what diagnostics accept.
func (ls *zirricLangserver) loadImportedModule(imp *ast.DeclImport) (*ast.ContextModule, map[string]string, bool) {
	if ls.resolver == nil {
		return nil, nil, false
	}
	uri := imp.ModuleName.URI()
	mod, err := ls.resolver.ResolveModule(context.Background(), uri)
	if err != nil || mod == nil {
		return nil, nil, false
	}
	return mod, ls.srcToPathForModule(uri, mod), true
}

// srcToPathForModule maps each source URI to a real path: project-local files strip the "<projectBaseURI>/" prefix fsmodule encodes; everything else (embedded stdlib, local/git deps) is materialized to a cached temp file, distinguished later via filepath.IsAbs.
func (ls *zirricLangserver) srcToPathForModule(uri registry.LogicalURI, mod *ast.ContextModule) map[string]string {
	srcToPath := make(map[string]string)
	if ls.orch == nil {
		return srcToPath
	}
	// LogicalURI.Join("") reproduces FSSource's exact prefix, including its edge case for a "/" project root.
	prefix := string(registry.LogicalURI(ls.orch.Cavefile().Name).Join(""))

	var external map[string]registry.Source // populated lazily, only if needed
	for _, sf := range mod.Files {
		if sf == nil {
			continue
		}
		if rel, ok := strings.CutPrefix(sf.Path, prefix); ok {
			srcToPath[sf.Path] = filepath.FromSlash(rel)
			continue
		}
		if external == nil {
			external = ls.externalModuleSources(uri)
		}
		src, ok := external[sf.Path]
		if !ok {
			continue
		}
		if path, ok := ls.materializeExternalSource(src); ok {
			srcToPath[sf.Path] = path
		}
	}
	return srcToPath
}

// externalModuleSources fetches uri's raw, unparsed sources from the resolver, keyed by their own logical URI string.
func (ls *zirricLangserver) externalModuleSources(uri registry.LogicalURI) map[string]registry.Source {
	if ls.resolver == nil {
		return nil
	}
	sources, err := ls.resolver.FindModuleSources(context.Background(), uri)
	if err != nil {
		return nil
	}
	out := make(map[string]registry.Source, len(sources))
	for _, src := range sources {
		out[string(src.URI())] = src
	}
	return out
}

// materializeExternalSource writes src's content to a stable, cached, read-only temp file and returns its absolute path, since embedded/dependency sources don't live under ls.fs's chroot.
func (ls *zirricLangserver) materializeExternalSource(src registry.Source) (string, bool) {
	key := string(src.URI())

	ls.externalSourcesMu.Lock()
	defer ls.externalSourcesMu.Unlock()

	if path, ok := ls.externalSources[key]; ok {
		return path, true
	}

	if ls.externalSourcesDir == "" {
		dir, err := os.MkdirTemp("", "zirric-lsp-sources-")
		if err != nil {
			return "", false
		}
		ls.externalSourcesDir = dir
	}

	data, err := src.Read()
	if err != nil {
		return "", false
	}

	// Preserve the source's relative structure under the temp dir, sanitizing each segment since a dependency URL can contain characters invalid in Windows paths (e.g. ':') or "." / ".." segments that could escape the temp dir.
	segments := strings.Split(key, "/")
	pathParts := make([]string, 0, len(segments)+1)
	pathParts = append(pathParts, ls.externalSourcesDir)
	for _, seg := range segments {
		pathParts = append(pathParts, sanitizePathSegment(seg))
	}
	target := filepath.Join(pathParts...)

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", false
	}
	if err := os.WriteFile(target, data, 0o444); err != nil {
		return "", false
	}

	if ls.externalSources == nil {
		ls.externalSources = make(map[string]string)
	}
	ls.externalSources[key] = target
	return target, true
}

// sanitizePathSegment replaces characters invalid in Windows paths (and control characters) with '_', and neutralizes "." / "..".
func sanitizePathSegment(seg string) string {
	if seg == "" || seg == "." || seg == ".." {
		return "_"
	}
	var b strings.Builder
	b.Grow(len(seg))
	for _, r := range seg {
		switch {
		case r < 0x20, r == '<', r == '>', r == ':', r == '"', r == '|', r == '?', r == '*', r == '\\':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// readSourceText reads a workspace-relative path via the overlay filesystem, or an absolute (materializeExternalSource) path directly from the OS.
func (ls *zirricLangserver) readSourceText(path string) (string, error) {
	if filepath.IsAbs(path) {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return readFileText(ls.fs, path)
}

// loadImportMemberModule parses the module referenced by a DeclImportMember.
func (ls *zirricLangserver) loadImportMemberModule(dim ast.DeclImportMember) (*ast.ContextModule, map[string]string, bool) {
	synthImp := &ast.DeclImport{ModuleName: ast.ModuleName(dim.ModuleName)}
	return ls.loadImportedModule(synthImp)
}

// resolveImportMemberDecl resolves a DeclImportMember to its actual declaration.
func (ls *zirricLangserver) resolveImportMemberDecl(dim ast.DeclImportMember) ast.Decl {
	name := dim.Name.Value
	mod, _, ok := ls.loadImportMemberModule(dim)
	if !ok {
		return nil
	}
	if sym, ok := mod.Decls.Resolve(name); ok && sym.Decl != nil {
		return sym.Decl
	}
	return nil
}

// resolveImportMemberLocation resolves a DeclImportMember to the LSP Location of the actual declaration.
func (ls *zirricLangserver) resolveImportMemberLocation(dim ast.DeclImportMember, name string) *protocol.Location {
	if importedMod, srcToPath, ok := ls.loadImportMemberModule(dim); ok {
		if loc, ok := ls.symbolLocation(importedMod, name, srcToPath); ok {
			return loc
		}
	}
	resolved := ls.resolveImportMemberDecl(dim)
	if resolved == nil {
		return nil
	}
	return ls.locationForDecl(resolved)
}

// locationForDecl returns the LSP Location of a declaration's name token, reading the source file to compute line/column positions.
func (ls *zirricLangserver) locationForDecl(decl ast.Decl) *protocol.Location {
	return ls.locationForDeclWithPaths(decl, nil)
}

// locationForDeclWithPaths returns the LSP Location of a declaration's name token, using srcToPath to translate source URIs to filesystem paths when available.
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
	text, err := ls.readSourceText(filePath)
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

	defText, err := ls.readSourceText(defFilePath)
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
