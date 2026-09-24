package langsrv

import (
	"path/filepath"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (ls *zirricLangserver) textDocumentReferences(
	ctx *glsp.Context,
	params *protocol.ReferenceParams,
) ([]protocol.Location, error) {
	defer func() {
		if r := recover(); r != nil {
			ls.logMessage(ctx, "panic in textDocumentReferences: %v", r)
		}
	}()

	path, ok := ls.pathForURI(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	result := ls.resolveSymbolReferences(path, params.Position)
	if result == nil {
		return nil, nil
	}

	locations := result.locations
	if params.Context.IncludeDeclaration && result.declLocation != nil {
		locations = append(locations, *result.declLocation)
	}
	return locations, nil
}

// symbolReferences holds everything needed to answer both textDocument/references and textDocument/rename for the symbol at a cursor position.
type symbolReferences struct {
	original *ast.Symbol
	// wordRange is the range a rename would replace: the identifier at the cursor itself, which may be a usage, not the declaration.
	wordRange protocol.Range
	// locations are usage sites only, not including the declaration.
	locations []protocol.Location
	// declLocation is the declaration site, if resolvable.
	declLocation *protocol.Location
	// editable is false for a declaration outside the workspace (embedded stdlib, external dependency) — never safe to rename.
	editable bool
}

// resolveSymbolReferences resolves the symbol at pos in path and collects every reference across the workspace. Returns nil if there's no symbol at pos.
func (ls *zirricLangserver) resolveSymbolReferences(path string, pos protocol.Position) *symbolReferences {
	text, err := readFileText(ls.fs, path)
	if err != nil {
		return nil
	}

	module, parseErrsByFile, sourceURIToPath, err := ls.parseModuleFilesForPath(path)
	if err != nil {
		return nil
	}

	// Skip if any file in the module has parse errors: error recovery can leave malformed nodes the analyzer isn't safe to walk.
	if !moduleHasParseErrors(parseErrsByFile) {
		an := analyzer.New(ls.resolver)
		an.Analyze(module, false)
	}

	word, wordRange := wordAtPosition(text, pos)
	if word == "" {
		return nil
	}
	cursorOffset := offsetForPosition(text, pos)

	sourceURI := string(registry.JoinModuleURI("", path))
	currentSF := findSourceFile(module, sourceURI)

	// Prefer the resolved ExprIdentifier (handles shadowing), falling back to a top-level symbol table lookup for a cursor on a declaration name.
	var targetSym *ast.Symbol
	if currentSF != nil {
		targetSym = symbolAtOffset(currentSF, cursorOffset)
	}

	// For a qualified member (alias.Member, modname.Member) track declaringModule so cross-module search starts from where the symbol is actually declared.
	declaringModule := module
	var importURI registry.LogicalURI // set only when declaringModule != module
	if targetSym == nil {
		if segments, _, isDotChain := dotChainContext(text, pos); isDotChain && len(segments) > 0 {
			alias := segments[0]
			if imp, ok := findImportDecl(currentSF, alias); ok {
				if importedMod, _, ok := ls.loadImportedModule(imp); ok && importedMod.Symbols != nil {
					if sym := importedMod.Symbols.Symbols[word]; sym != nil {
						targetSym = sym
						declaringModule = importedMod
						importURI = imp.ModuleName.URI()
					}
				}
			}
			if targetSym == nil && isModuleDecl(module, alias) && module.Symbols != nil {
				targetSym = module.Symbols.Symbols[word]
			}
		}
	}

	if targetSym == nil && module.Symbols != nil {
		targetSym = module.Symbols.Symbols[word]
	}
	if targetSym == nil && currentSF != nil && currentSF.Symbols != nil {
		targetSym = currentSF.Symbols.Symbols[word]
	}
	if targetSym == nil {
		return nil
	}

	original := targetSym.Original()
	result := &symbolReferences{original: original, wordRange: wordRange}

	if original.Decl == nil {
		return result
	}
	nameTok := original.Decl.DeclName().Token
	if nameTok.Source == nil {
		return result
	}

	// Source.File is a canonicalized logical URI (e.g. "flow/types.zirr" becomes "flow.types.zirr"), not reversible by string manipulation, so resolve it via the same sourceURIToPath map the parser built. Not found here means the declaration lives outside the workspace and isn't safe to rename.
	declaringSrcToPath := sourceURIToPath
	if declaringModule != module {
		declaringSrcToPath = ls.srcToPathForModule(importURI, declaringModule)
	}
	declPath, ok := declaringSrcToPath[nameTok.Source.File]
	if !ok {
		return result
	}
	declDir := filepath.Dir(declPath)

	// declaringModule/original may come from ls.resolver's cache, independent of parseModuleFiles' own — the same directory can end up parsed into two pointer-distinct modules, so a symbol from one wouldn't pointer-equal a reference from the other. Re-parse declDir and re-find the equivalent Symbol by position so everything downstream goes through parseModuleFiles' cache consistently.
	declMod, declParseErrs, declSrcToPath, err := ls.parseModuleFiles(declDir)
	if err != nil || moduleHasParseErrors(declParseErrs) {
		return result
	}
	if canonical := findSymbolAtDeclPosition(declMod, declSrcToPath, declPath, nameTok.Source.Offset); canonical != nil {
		original = canonical.Original()
		declaringModule = declMod
	}
	result.original = original
	result.editable = true

	an := analyzer.New(ls.resolver)
	an.Analyze(declMod, false)

	if declText, ferr := ls.readSourceText(declPath); ferr == nil {
		end := nameTok.Source.Offset + len(nameTok.Literal)
		loc := protocol.Location{
			URI:   ls.fileURI(declPath),
			Range: rangeForOffsets(declText, nameTok.Source.Offset, end),
		}
		result.declLocation = &loc
	}

	// Search declDir (where the symbol is actually declared) plus every other directory-module that imports it, so a reference is found regardless of which file the request originated from.
	seen := make(map[refKey]struct{})
	targetURIs := currentModuleURIs(ls, declDir)
	result.locations = ls.collectReferencesInModule(declMod, declSrcToPath, declaringModule, original, targetURIs, seen)

	for _, dir := range ls.collectModuleDirs(".") {
		if dir == declDir {
			continue
		}
		otherMod, otherParseErrs, otherSrcToPath, err := ls.parseModuleFiles(dir)
		if err != nil || moduleHasParseErrors(otherParseErrs) {
			continue
		}
		if !moduleImportsAny(otherMod, targetURIs) {
			continue
		}
		an := analyzer.New(ls.resolver)
		an.Analyze(otherMod, false)
		result.locations = append(result.locations, ls.collectReferencesInModule(otherMod, otherSrcToPath, declaringModule, original, targetURIs, seen)...)
	}

	return result
}

// findSymbolAtDeclPosition finds the declaration whose name token, mapped through sourceURIToPath, resolves to declPath at offset — re-resolving a symbol from one parse cache into the equivalent Symbol from another (see resolveSymbolReferences for why raw Source.File strings aren't comparable across caches).
func findSymbolAtDeclPosition(mod *ast.ContextModule, sourceURIToPath map[string]string, declPath string, offset int) *ast.Symbol {
	tables := make([]*ast.SymbolTable, 0, len(mod.Files)+1)
	if mod.Symbols != nil {
		tables = append(tables, mod.Symbols)
	}
	for _, sf := range mod.Files {
		if sf != nil && sf.Symbols != nil {
			tables = append(tables, sf.Symbols)
		}
	}
	for _, table := range tables {
		for _, sym := range table.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}
			tok := sym.Decl.DeclName().Token
			if tok.Source == nil || tok.Source.Offset != offset {
				continue
			}
			if sourceURIToPath[tok.Source.File] == declPath {
				return sym
			}
		}
	}
	return nil
}

// refKey identifies a reference site by source file and byte offset, deduplicating locations across multiple module walks.
type refKey struct {
	file   string
	offset int
}

// currentModuleURIs returns dir's bare dotted form (e.g. "flow") and, when the project has a name, its <projectBaseURI>.<name> qualified form (e.g. "ui.flow"), mirroring ResolveModule's fallback.
func currentModuleURIs(ls *zirricLangserver, dir string) map[registry.LogicalURI]bool {
	uris := make(map[registry.LogicalURI]bool, 2)
	bare := registry.JoinModuleURI("", dir)
	uris[bare] = true
	if ls.orch != nil {
		uris[registry.JoinModuleURI(registry.LogicalURI(ls.orch.Cavefile().Name), string(bare))] = true
	}
	return uris
}

// moduleImportsAny reports whether mod imports any of targets. Imports have ExportScopeLocal, so they live in each file's sf.Decls.Symbols rather than the shared mod.Decls.Symbols (see isModuleDecl in completion.go).
func moduleImportsAny(mod *ast.ContextModule, targets map[registry.LogicalURI]bool) bool {
	if mod == nil {
		return false
	}
	symbolTables := make([]*ast.DeclTable, 0, len(mod.Files)+1)
	if mod.Decls != nil {
		symbolTables = append(symbolTables, mod.Decls)
	}
	for _, sf := range mod.Files {
		if sf != nil && sf.Decls != nil {
			symbolTables = append(symbolTables, sf.Decls)
		}
	}
	for _, table := range symbolTables {
		for _, sym := range table.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}
			imp, ok := extractImportDecl(sym.Decl)
			if !ok || imp == nil {
				continue
			}
			if targets[imp.ModuleName.URI()] {
				return true
			}
		}
	}
	return false
}

// collectReferencesInModule walks every file in mod for references to original, declared in declaringModule. targetURIs guards the DeclImportMember case against name collisions with unrelated modules. Locations in seen are skipped; new ones are added to it.
func (ls *zirricLangserver) collectReferencesInModule(
	mod *ast.ContextModule,
	sourceURIToPath map[string]string,
	declaringModule *ast.ContextModule,
	original *ast.Symbol,
	targetURIs map[registry.LogicalURI]bool,
	seen map[refKey]struct{},
) []protocol.Location {
	var locations []protocol.Location
	for _, sf := range mod.Files {
		if sf == nil {
			continue
		}

		sfFilePath := sourceURIToPath[sf.Path]
		if sfFilePath == "" {
			continue
		}

		emit := func(tok token.Token) {
			if tok.Source == nil {
				return
			}
			key := refKey{tok.Source.File, tok.Source.Offset}
			if _, dup := seen[key]; dup {
				return
			}
			seen[key] = struct{}{}
			filePath := sourceURIToPath[tok.Source.File]
			if filePath == "" {
				filePath = sfFilePath
			}
			refText, terr := ls.readSourceText(filePath)
			if terr != nil {
				return
			}
			end := tok.Source.Offset + len(tok.Literal)
			locations = append(locations, protocol.Location{
				URI:   ls.fileURI(filePath),
				Range: rangeForOffsets(refText, tok.Source.Offset, end),
			})
		}

		lookupInDeclaringModule := func(name string) bool {
			if declaringModule.Symbols == nil {
				return false
			}
			sym := declaringModule.Symbols.Symbols[name]
			return sym != nil && sym.Original() == original
		}

		walkASTNode(sf, func(node ast.Node) {
			switch n := node.(type) {
			case *ast.ExprIdentifier:
				if n.Symbol != nil && n.Symbol.Original() == original {
					emit(n.Name.Token)
				}

			case *ast.ExprMemberAccess:
				if lookupInDeclaringModule(n.Property.Value) {
					emit(n.Property.Token)
				}

			case ast.TypeExprRef:
				if len(n.Reference) == 0 {
					return
				}
				lastIdent := n.Reference[len(n.Reference)-1]
				if lookupInDeclaringModule(lastIdent.Value) {
					emit(lastIdent.Token)
				}

			case *ast.DeclAttrInstance:
				if len(n.Reference) == 0 {
					return
				}
				lastIdent := n.Reference[len(n.Reference)-1]
				if lookupInDeclaringModule(lastIdent.Value) {
					emit(lastIdent.Token)
				}

			case *ast.DeclUnionMember:
				if len(n.Member) == 0 {
					return
				}
				lastIdent := n.Member[len(n.Member)-1]
				if lookupInDeclaringModule(lastIdent.Value) {
					emit(lastIdent.Token)
				}

			case ast.DeclImportMember:
				// Also requires the import's own module to be one of targetURIs, since matching purely by name would misfire on an unrelated module exporting the same name.
				if targetURIs[n.ModuleName.URI()] && lookupInDeclaringModule(n.Name.Value) {
					emit(n.Name.Token)
				}
			}
		})
	}
	return locations
}

// symbolAtOffset returns the Symbol of the ExprIdentifier whose name token spans offset, or nil.
func symbolAtOffset(sf *ast.SourceFile, offset int) *ast.Symbol {
	var found *ast.Symbol
	walkASTNode(sf, func(node ast.Node) {
		if found != nil {
			return
		}

		expr, ok := node.(*ast.ExprIdentifier)
		if !ok || expr.Symbol == nil {
			return
		}

		tok := expr.Name.Token
		if tok.Source == nil {
			return
		}

		end := tok.Source.Offset + len(tok.Literal)
		if offset >= tok.Source.Offset && offset < end {
			found = expr.Symbol
		}
	})
	return found
}

// walkASTNode visits node and all its descendants.
// EnumerateChildNodes already performs a full deep traversal (calling action on every descendant, not just direct children), so we only need to call it once.
func walkASTNode(node ast.Node, visit func(ast.Node)) {
	if node == nil {
		return
	}

	visit(node)
	node.EnumerateChildNodes(visit)
}
