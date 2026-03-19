package langsrv

import (
	"path/filepath"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (ls *zirricLangserver) textDocumentDefinition(
	ctx *glsp.Context,
	params *protocol.DefinitionParams,
) (any, error) {
	defer func() {
		if r := recover(); r != nil {
			ls.logMessage(ctx, "panic in textDocumentDefinition: %v", r)
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

	sourceURI := string(registry.JoinModuleURI("", path))
	currentSF := findSourceFile(module, sourceURI)

	word, _ := wordAtPosition(text, params.Position)
	if word == "" {
		return nil, nil
	}

	// Check qualified/dot-chain context: "alias.member" or "expr.field.subfield"
	if segments, _, isDotChain := dotChainContext(text, params.Position); isDotChain {
		// Try module/import alias first (single-segment).
		if len(segments) == 1 {
			alias := segments[0]
			if imp, ok := findImportDecl(currentSF, alias); ok {
				if importedMod, srcToPath, ok := ls.loadImportedModule(imp); ok {
					if loc, ok := ls.symbolLocation(importedMod, word, srcToPath); ok {
						return loc, nil
					}
				}
			}
			if isModuleDecl(module, alias) {
				if loc, ok := ls.symbolLocation(module, word, sourceURIToPath); ok {
					return loc, nil
				}
			}
		}

		// Multi-segment or non-module single segment: type-aware resolution.
		cursorOffset := offsetForPosition(text, params.Position)
		result := ls.resolveDotChain(module, currentSF, path, cursorOffset, segments)
		if result != nil && len(result.fields) > 0 {
			for _, f := range result.fields {
				if f.Name.Value == word {
					if loc := ls.locationForDeclWithPaths(f, sourceURIToPath); loc != nil {
						return loc, nil
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

	// DeclImportMember: navigate to its actual definition in the imported module.
	switch dim := sym.Decl.(type) {
	case ast.DeclImportMember:
		if loc := ls.resolveImportMemberLocation(dim, word); loc != nil {
			return loc, nil
		}
		return nil, nil
	case *ast.DeclImportMember:
		if loc := ls.resolveImportMemberLocation(*dim, word); loc != nil {
			return loc, nil
		}
		return nil, nil
	}

	nameToken := sym.Decl.DeclName().Token
	if nameToken.Source == nil {
		return nil, nil
	}

	defFilePath, ok := sourceURIToPath[nameToken.Source.File]
	if !ok {
		// Local declarations may have source files not in sourceURIToPath;
		// fall back to reading the file directly.
		if loc := ls.locationForDecl(sym.Decl); loc != nil {
			return loc, nil
		}
		return nil, nil
	}

	defText, err := readFileText(ls.fs, defFilePath)
	if err != nil {
		return nil, nil
	}

	offset := nameToken.Source.Offset
	nameLen := len(nameToken.Literal)
	if nameLen <= 0 {
		nameLen = 1
	}
	return &protocol.Location{
		URI:   ls.fileURI(defFilePath),
		Range: rangeForOffsets(defText, offset, offset+nameLen),
	}, nil
}
