package langsrv

import (
	"path/filepath"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (ls *zirricLangserver) textDocumentDocumentSymbol(
	ctx *glsp.Context,
	params *protocol.DocumentSymbolParams,
) (any, error) {
	defer func() {
		if r := recover(); r != nil {
			ls.logMessage(ctx, "panic in textDocumentDocumentSymbol: %v", r)
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

	var (
		sourceURI = string(registry.JoinModuleURI("", path))
		symbols   []protocol.DocumentSymbol
	)
	for _, sym := range module.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}

		nameToken := sym.Decl.DeclName().Token
		if nameToken.Source == nil || nameToken.Source.File != sourceURI {
			continue
		}

		defFilePath := sourceURIToPath[nameToken.Source.File]
		if defFilePath == "" {
			continue
		}

		ds := documentSymbolForDecl(sym.Decl, text)
		if ds == nil {
			continue
		}

		symbols = append(symbols, *ds)
	}
	return symbols, nil
}

func (ls *zirricLangserver) workspaceSymbol(
	ctx *glsp.Context,
	params *protocol.WorkspaceSymbolParams,
) ([]protocol.SymbolInformation, error) {
	defer func() {
		if r := recover(); r != nil {
			ls.logMessage(ctx, "panic in workspaceSymbol: %v", r)
		}
	}()

	if ls.fs == nil {
		return nil, nil
	}

	var (
		query   = strings.ToLower(params.Query)
		results []protocol.SymbolInformation
	)
	for _, dir := range ls.collectModuleDirs(".") {
		module, _, sourceURIToPath, err := ls.parseModuleFiles(dir)
		if err != nil {
			continue
		}
		containerName := moduleContainerName(dir)
		for _, sym := range module.Decls.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}

			name := sym.Decl.DeclName().Value
			if query != "" && !strings.Contains(strings.ToLower(name), query) {
				continue
			}

			nameToken := sym.Decl.DeclName().Token
			if nameToken.Source == nil {
				continue
			}

			defFilePath, ok := sourceURIToPath[nameToken.Source.File]
			if !ok {
				continue
			}

			defText, err := readFileText(ls.fs, defFilePath)
			if err != nil {
				continue
			}

			offset := nameToken.Source.Offset
			nameLen := len(nameToken.Literal)
			if nameLen <= 0 {
				nameLen = 1
			}

			results = append(results, protocol.SymbolInformation{
				Name:          name,
				Kind:          symbolKindForDecl(sym.Decl),
				ContainerName: &containerName,
				Location: protocol.Location{
					URI:   ls.fileURI(defFilePath),
					Range: rangeForOffsets(defText, offset, offset+nameLen),
				},
			})
		}
	}
	return results, nil
}

// collectModuleDirs returns all directories (including root) that contain .zirr
// files, found by recursively walking from the given root via the overlay
// filesystem. The root should be a relative path (e.g. ".") since the overlay
// filesystem is chrooted to the project root. Only directories with valid
// module names (identifier characters) are traversed, matching the convention
// used by fsmodule.DiscoverModules.
func (ls *zirricLangserver) collectModuleDirs(root string) []string {
	entries, err := ls.fs.ReadDir(root)
	if err != nil {
		return nil
	}

	var dirs []string
	hasZirr := false
	for _, e := range entries {
		if e.IsDir() {
			// Only descend into directories with valid module names.
			if !isValidModuleDirName(e.Name()) {
				continue
			}
			sub := filepath.Join(root, e.Name())
			dirs = append(dirs, ls.collectModuleDirs(sub)...)
		} else if strings.HasSuffix(e.Name(), ".zirr") {
			hasZirr = true
		}
	}
	if hasZirr {
		dirs = append([]string{root}, dirs...)
	}
	return dirs
}

// isValidModuleDirName returns true if the directory name is a valid module
// identifier: starts with a letter or underscore, followed by letters, digits,
// underscores, or hyphens. This matches fsmodule's recursive discovery rules.
func isValidModuleDirName(name string) bool {
	if name == "" {
		return false
	}
	for i, c := range name {
		if i == 0 {
			if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && c != '_' {
				return false
			}
		} else {
			if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' && c != '-' {
				return false
			}
		}
	}
	return true
}

// moduleContainerName returns a human-readable module identifier for use as
// ContainerName in workspace symbol results. It converts the relative directory
// path to a dot-separated module path (e.g. "examples/project" → "examples.project").
// The root module (dir ".") returns an empty string.
func moduleContainerName(dir string) string {
	if dir == "." || dir == "" {
		return ""
	}
	return strings.ReplaceAll(filepath.ToSlash(dir), "/", ".")
}

// documentSymbolForDecl builds a DocumentSymbol from a declaration using the given file text.
func documentSymbolForDecl(decl ast.Decl, text string) *protocol.DocumentSymbol {
	name := decl.DeclName().Value
	if name == "" {
		return nil
	}

	nameToken := decl.DeclName().Token
	if nameToken.Source == nil {
		return nil
	}

	kwToken := decl.TokenLiteral()
	kwOffset := nameToken.Source.Offset
	if kwToken.Source != nil && kwToken.Source.File == nameToken.Source.File {
		kwOffset = kwToken.Source.Offset
	}

	nameOffset := nameToken.Source.Offset
	nameEnd := nameOffset + len(nameToken.Literal)

	selRange := rangeForOffsets(text, nameOffset, nameEnd)
	fullRange := rangeForOffsets(text, kwOffset, nameEnd)

	kind := symbolKindForDecl(decl)
	ds := &protocol.DocumentSymbol{
		Name:           name,
		Kind:           kind,
		Range:          fullRange,
		SelectionRange: selRange,
	}

	// Add children for composite types.
	ds.Children = childSymbols(decl, text)
	return ds
}

// childSymbols returns DocumentSymbol children for declarations that have members.
func childSymbols(decl ast.Decl, text string) []protocol.DocumentSymbol {
	var children []protocol.DocumentSymbol

	switch d := decl.(type) {
	case ast.DeclData:
		for _, f := range d.Fields {
			if c := documentSymbolForDecl(f, text); c != nil {
				children = append(children, *c)
			}
		}
	case *ast.DeclData:
		for _, f := range d.Fields {
			if c := documentSymbolForDecl(f, text); c != nil {
				children = append(children, *c)
			}
		}
	case ast.DeclAttr:
		for _, f := range d.Fields {
			if c := documentSymbolForDecl(f, text); c != nil {
				children = append(children, *c)
			}
		}
	case *ast.DeclAttr:
		for _, f := range d.Fields {
			if c := documentSymbolForDecl(f, text); c != nil {
				children = append(children, *c)
			}
		}
	case ast.DeclUnion:
		for _, m := range d.Members {
			if c := documentSymbolForDecl(m, text); c != nil {
				children = append(children, *c)
			}
		}
	case *ast.DeclUnion:
		for _, m := range d.Members {
			if c := documentSymbolForDecl(m, text); c != nil {
				children = append(children, *c)
			}
		}
	}
	return children
}

// symbolKindForDecl maps a Zirric declaration to an LSP SymbolKind.
func symbolKindForDecl(decl ast.Decl) protocol.SymbolKind {
	switch decl.(type) {
	case *ast.DeclFunc, ast.DeclFunc, *ast.DeclExternFunc, ast.DeclExternFunc:
		return protocol.SymbolKindFunction
	case *ast.DeclVariable, ast.DeclVariable, *ast.DeclConstant, ast.DeclConstant, *ast.DeclExternValue, ast.DeclExternValue:
		return protocol.SymbolKindVariable
	case *ast.DeclData, ast.DeclData:
		return protocol.SymbolKindStruct
	case *ast.DeclExternType, ast.DeclExternType:
		return protocol.SymbolKindClass
	case *ast.DeclUnion, ast.DeclUnion:
		return protocol.SymbolKindEnum
	case *ast.DeclAttr, ast.DeclAttr:
		return protocol.SymbolKindInterface
	case *ast.DeclField, ast.DeclField:
		return protocol.SymbolKindField
	case *ast.DeclParameter, ast.DeclParameter:
		return protocol.SymbolKindTypeParameter
	case *ast.DeclUnionMember, ast.DeclUnionMember:
		return protocol.SymbolKindEnumMember
	case *ast.DeclModule, ast.DeclModule, *ast.DeclImport, ast.DeclImport:
		return protocol.SymbolKindModule
	default:
		return protocol.SymbolKindVariable
	}
}
