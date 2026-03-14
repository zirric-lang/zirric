package langsrv

import (
	"path/filepath"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
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

	text, err := readFileText(ls.fs, path)
	if err != nil {
		return nil, nil
	}

	module, _, sourceURIToPath, err := ls.parseModuleFiles(filepath.Dir(path))
	if err != nil {
		return nil, nil
	}

	// Run the analyzer to populate ExprIdentifier.Symbol across the whole module.
	an := analyzer.New(nil)
	an.Analyze(module, false)

	word, _ := wordAtPosition(text, params.Position)
	if word == "" {
		return nil, nil
	}
	cursorOffset := offsetForPosition(text, params.Position)

	sourceURI := string(registry.JoinModuleURI("", path))
	currentSF := findSourceFile(module, sourceURI)

	// Find the symbol at the cursor: prefer the resolved ExprIdentifier (handles
	// shadowed names correctly), then fall back to a top-level symbol table lookup
	// for when the cursor is on a declaration name rather than a reference.
	var targetSym *ast.Symbol
	if currentSF != nil {
		targetSym = symbolAtOffset(currentSF, cursorOffset)
	}
	if targetSym == nil && module.Symbols != nil {
		targetSym = module.Symbols.Symbols[word]
	}
	if targetSym == nil && currentSF != nil && currentSF.Symbols != nil {
		targetSym = currentSF.Symbols.Symbols[word]
	}
	if targetSym == nil {
		return nil, nil
	}

	original := targetSym.Original()

	// Walk the analyzed AST of every file in the module. Declaration names are
	// ast.Identifier (not *ast.ExprIdentifier) so they will not appear here;
	// this loop collects only genuine references.
	//
	// SourceFile.EnumerateChildNodes also visits sf.Decls.Parent.Symbols
	// (module-level declarations from all files), so a reference that lives
	// in file B is reachable from file A's walk as well. Deduplicate by
	// (source file, byte offset) to avoid emitting each location N times.
	type refKey struct {
		file   string
		offset int
	}
	seen := make(map[refKey]struct{})
	var locations []protocol.Location
	for _, sf := range module.Files {
		if sf == nil {
			continue
		}

		sfFilePath := sourceURIToPath[sf.Path]
		if sfFilePath == "" {
			continue
		}

		walkASTNode(sf, func(node ast.Node) {
			expr, ok := node.(*ast.ExprIdentifier)
			if !ok || expr.Symbol == nil {
				return
			}

			if expr.Symbol.Original() != original {
				return
			}

			tok := expr.Name.Token
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

			refText, terr := readFileText(ls.fs, filePath)
			if terr != nil {
				return
			}

			end := tok.Source.Offset + len(tok.Literal)
			locations = append(locations, protocol.Location{
				URI:   ls.fileURI(filePath),
				Range: rangeForOffsets(refText, tok.Source.Offset, end),
			})
		})
	}

	// Declaration names (ast.Identifier) are not ExprIdentifier nodes, so they
	// are not included in the walk above. Add the declaration site explicitly
	// when the client requests it.
	if !params.Context.IncludeDeclaration || original.Decl == nil {
		return locations, nil
	}

	nameTok := original.Decl.DeclName().Token
	if nameTok.Source == nil {
		return locations, nil
	}

	declFilePath := sourceURIToPath[nameTok.Source.File]
	if declFilePath == "" {
		return locations, nil
	}

	declText, ferr := readFileText(ls.fs, declFilePath)
	if ferr == nil {
		end := nameTok.Source.Offset + len(nameTok.Literal)
		locations = append(locations, protocol.Location{
			URI:   ls.fileURI(declFilePath),
			Range: rangeForOffsets(declText, nameTok.Source.Offset, end),
		})
	}

	return locations, nil
}

// symbolAtOffset walks the analyzed AST of sf and returns the Symbol of the
// ExprIdentifier whose name token spans the given byte offset, or nil.
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
// EnumerateChildNodes already performs a full deep traversal (calling action on
// every descendant, not just direct children), so we only need to call it once.
func walkASTNode(node ast.Node, visit func(ast.Node)) {
	if node == nil {
		return
	}

	visit(node)
	node.EnumerateChildNodes(visit)
}
