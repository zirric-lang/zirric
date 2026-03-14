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

	// Check qualified context first: "alias.member"
	if alias, _, isQualified := qualifiedContext(text, params.Position); isQualified {
		if imp, ok := findImportDecl(currentSF, alias); ok {
			if importedMod, srcToPath, ok := ls.loadImportedModule(imp); ok {
				if loc, ok := ls.symbolLocation(importedMod, word, srcToPath); ok {
					return loc, nil
				}
			}
		}
		return nil, nil
	}

	// Try global symbols first.
	sym := module.Decls.Symbols[word]
	// Also check current file's local scope (for imports with ExportScopeLocal).
	if (sym == nil || sym.Decl == nil) && currentSF != nil {
		sym = currentSF.Decls.Symbols[word]
	}
	if sym == nil || sym.Decl == nil {
		return nil, nil
	}

	// DeclImportMember: navigate to its actual definition in the imported module.
	switch dim := sym.Decl.(type) {
	case ast.DeclImportMember:
		if importedMod, srcToPath, ok := ls.loadImportMemberModule(dim); ok {
			if loc, ok := ls.symbolLocation(importedMod, word, srcToPath); ok {
				return loc, nil
			}
		}
		return nil, nil
	case *ast.DeclImportMember:
		if importedMod, srcToPath, ok := ls.loadImportMemberModule(*dim); ok {
			if loc, ok := ls.symbolLocation(importedMod, word, srcToPath); ok {
				return loc, nil
			}
		}
		return nil, nil
	}

	nameToken := sym.Decl.DeclName().Token
	if nameToken.Source == nil {
		return nil, nil
	}

	defFilePath, ok := sourceURIToPath[nameToken.Source.File]
	if !ok {
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
