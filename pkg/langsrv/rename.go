package langsrv

import (
	"errors"
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const errNotRenamable = "cannot rename: declared in an embedded stdlib package or an external dependency, not in this workspace"

// textDocumentPrepareRename answers whether the symbol at the cursor can be renamed, letting the client show an error before the user finishes typing a new name.
func (ls *zirricLangserver) textDocumentPrepareRename(
	ctx *glsp.Context,
	params *protocol.PrepareRenameParams,
) (any, error) {
	defer func() {
		if r := recover(); r != nil {
			ls.logMessage(ctx, "panic in textDocumentPrepareRename: %v", r)
		}
	}()

	path, ok := ls.pathForURI(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	if imp, _, text, ok := ls.importAliasAtPosition(path, params.Position); ok {
		tok := imp.Alias.Token
		start := tok.Source.Offset
		return rangeForOffsets(text, start, start+len(tok.Literal)), nil
	}

	result := ls.resolveSymbolReferences(path, params.Position)
	if result == nil {
		return nil, nil
	}
	if !result.editable {
		return nil, errors.New(errNotRenamable)
	}
	return result.wordRange, nil
}

// textDocumentRename renames every reference to the symbol at the cursor (see resolveSymbolReferences), rejecting symbols declared outside the workspace.
func (ls *zirricLangserver) textDocumentRename(
	ctx *glsp.Context,
	params *protocol.RenameParams,
) (*protocol.WorkspaceEdit, error) {
	defer func() {
		if r := recover(); r != nil {
			ls.logMessage(ctx, "panic in textDocumentRename: %v", r)
		}
	}()

	if !isValidIdentifierName(params.NewName) {
		return nil, fmt.Errorf("%q is not a valid identifier name", params.NewName)
	}

	path, ok := ls.pathForURI(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	if imp, currentSF, text, ok := ls.importAliasAtPosition(path, params.Position); ok {
		return ls.renameImportAlias(path, imp, currentSF, text, params.NewName)
	}

	result := ls.resolveSymbolReferences(path, params.Position)
	if result == nil {
		return nil, nil
	}
	if !result.editable {
		return nil, errors.New(errNotRenamable)
	}

	changes := make(map[protocol.DocumentUri][]protocol.TextEdit)
	addEdit := func(loc protocol.Location) {
		changes[loc.URI] = append(changes[loc.URI], protocol.TextEdit{
			Range:   loc.Range,
			NewText: params.NewName,
		})
	}
	if result.declLocation != nil {
		addEdit(*result.declLocation)
	}
	for _, loc := range result.locations {
		addEdit(loc)
	}

	return &protocol.WorkspaceEdit{Changes: changes}, nil
}

// importAliasAtPosition reports whether pos is on an import statement's own alias token, as opposed to a dot-chain usage elsewhere (handled by resolveSymbolReferences instead). For a bare import, the alias token is the module path's own last segment; for an explicit alias, it's a separate token.
func (ls *zirricLangserver) importAliasAtPosition(path string, pos protocol.Position) (imp *ast.DeclImport, currentSF *ast.SourceFile, text string, ok bool) {
	text, err := readFileText(ls.fs, path)
	if err != nil {
		return nil, nil, "", false
	}
	module, _, _, err := ls.parseModuleFilesForPath(path)
	if err != nil {
		return nil, nil, "", false
	}
	sourceURI := string(registry.JoinModuleURI("", path))
	currentSF = findSourceFile(module, sourceURI)
	if currentSF == nil || currentSF.Decls == nil {
		return nil, nil, "", false
	}
	offset := offsetForPosition(text, pos)
	for _, sym := range currentSF.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		candidate, isImp := extractImportDecl(sym.Decl)
		if !isImp || candidate == nil {
			continue
		}
		tok := candidate.Alias.Token
		if tok.Source == nil {
			continue
		}
		start := tok.Source.Offset
		end := start + len(tok.Literal)
		if offset >= start && offset < end {
			return candidate, currentSF, text, true
		}
	}
	return nil, nil, "", false
}

// renameImportAlias renames an import's own alias: file-scoped (only the one file is touched), and a bare import needs "newName = " inserted since there's no separate alias token to replace.
func (ls *zirricLangserver) renameImportAlias(
	path string,
	imp *ast.DeclImport,
	currentSF *ast.SourceFile,
	text string,
	newName string,
) (*protocol.WorkspaceEdit, error) {
	if !isValidIdentifierName(newName) {
		return nil, fmt.Errorf("%q is not a valid identifier name", newName)
	}
	oldAlias := imp.Alias.Value
	if oldAlias == newName {
		return &protocol.WorkspaceEdit{}, nil
	}

	var edits []protocol.TextEdit

	lastSegment := imp.ModuleName[len(imp.ModuleName)-1]
	bare := imp.Alias.Token.Source != nil && lastSegment.Token.Source != nil &&
		imp.Alias.Token.Source.Offset == lastSegment.Token.Source.Offset

	if bare {
		firstSegment := imp.ModuleName[0]
		if firstSegment.Token.Source == nil {
			return nil, errors.New("cannot rename: import has no source position")
		}
		insertPos := positionForOffset(text, firstSegment.Token.Source.Offset)
		edits = append(edits, protocol.TextEdit{
			Range:   protocol.Range{Start: insertPos, End: insertPos},
			NewText: newName + " = ",
		})
	} else {
		if imp.Alias.Token.Source == nil {
			return nil, errors.New("cannot rename: import alias has no source position")
		}
		start := imp.Alias.Token.Source.Offset
		edits = append(edits, protocol.TextEdit{
			Range:   rangeForOffsets(text, start, start+len(imp.Alias.Token.Literal)),
			NewText: newName,
		})
	}

	// Matched by name rather than resolved Symbol identity, since bare identifier resolution through an import doesn't always populate a usable Decl; a file-scoped name match is unambiguous regardless, since a file has at most one binding per alias.
	seenOffsets := make(map[int]bool)
	walkASTNode(currentSF, func(node ast.Node) {
		ma, ok := node.(*ast.ExprMemberAccess)
		if !ok {
			return
		}
		target, ok := ma.Target.(*ast.ExprIdentifier)
		if !ok || target.Name.Value != oldAlias {
			return
		}
		tok := target.Name.Token
		if tok.Source == nil || seenOffsets[tok.Source.Offset] {
			return
		}
		seenOffsets[tok.Source.Offset] = true
		edits = append(edits, protocol.TextEdit{
			Range:   rangeForOffsets(text, tok.Source.Offset, tok.Source.Offset+len(tok.Literal)),
			NewText: newName,
		})
	})

	uri := ls.fileURI(path)
	return &protocol.WorkspaceEdit{Changes: map[protocol.DocumentUri][]protocol.TextEdit{uri: edits}}, nil
}

// isValidIdentifierName reports whether name is a syntactically valid, non-reserved Zirric identifier.
func isValidIdentifierName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		isLetter := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_'
		isDigit := c >= '0' && c <= '9'
		if i == 0 && !isLetter {
			return false
		}
		if i > 0 && !isLetter && !isDigit {
			return false
		}
	}
	return token.LookupIdent(name) == token.IDENT
}
