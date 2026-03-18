package langsrv

import (
	"path/filepath"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (ls *zirricLangserver) textDocumentSignatureHelp(
	ctx *glsp.Context,
	params *protocol.SignatureHelpParams,
) (*protocol.SignatureHelp, error) {
	defer func() {
		if r := recover(); r != nil {
			ls.logMessage(ctx, "panic in textDocumentSignatureHelp: %v", r)
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

	module, _, _, err := ls.parseModuleFiles(filepath.Dir(path))
	if err != nil {
		return nil, nil
	}

	offset := offsetForPosition(text, params.Position)
	funcName, qualAlias, activeParam, found := callContext(text, offset)
	if !found {
		return nil, nil
	}

	var (
		sourceURI = string(registry.JoinModuleURI("", path))
		currentSF = findSourceFile(module, sourceURI)

		fnParams []ast.DeclParameter
		resolved bool
	)

	if qualAlias != "" {
		if imp, ok := findImportDecl(currentSF, qualAlias); ok {
			if importedMod, _, ok := ls.loadImportedModule(imp); ok {
				fnParams, resolved = funcParamsFromModule(importedMod, funcName)
			}
		}
	} else {
		fnParams, resolved = funcParamsFromModule(module, funcName)
	}

	if !resolved {
		return nil, nil
	}

	var (
		sig    = buildSignatureInfo(funcName, fnParams)
		zero   = protocol.UInteger(0)
		active = protocol.UInteger(activeParam)
	)

	return &protocol.SignatureHelp{
		Signatures:      []protocol.SignatureInformation{sig},
		ActiveSignature: &zero,
		ActiveParameter: &active,
	}, nil
}

// callContext scans backward from offset to find the enclosing function call.
// It returns the function name, optional module qualifier, active parameter index, and whether found.
func callContext(text string, offset int) (funcName, qualAlias string, activeParam int, found bool) {
	depth := 0
	commas := 0

	for i := offset - 1; i >= 0; i-- {
		switch text[i] {
		case ')':
			depth++
		case '(':
			if depth == 0 {
				end := i
				start := end
				for start > 0 && isSimpleIdentByte(text[start-1]) {
					start--
				}

				if start == end {
					return
				}

				funcName = text[start:end]

				// Check for "alias.funcName("
				if start > 1 && text[start-1] == '.' {
					aliasEnd := start - 1
					aliasStart := aliasEnd
					for aliasStart > 0 && isSimpleIdentByte(text[aliasStart-1]) {
						aliasStart--
					}

					if aliasStart < aliasEnd {
						qualAlias = text[aliasStart:aliasEnd]
					}
				}

				activeParam = commas
				found = true
				return
			}
			depth--
		case ',':
			if depth == 0 {
				commas++
			}
		}
	}
	return
}

// funcParamsFromModule looks up a function declaration by name and returns its parameters.
func funcParamsFromModule(mod *ast.ContextModule, name string) ([]ast.DeclParameter, bool) {
	sym := mod.Decls.Symbols[name]
	if sym == nil || sym.Decl == nil {
		return nil, false
	}

	switch d := sym.Decl.(type) {
	case ast.DeclFunc:
		if d.Impl != nil {
			return d.Impl.Parameters, true
		}
		return nil, true

	case *ast.DeclFunc:
		if d.Impl != nil {
			return d.Impl.Parameters, true
		}
		return nil, true

	case ast.DeclExternFunc:
		return d.Parameters, true

	case *ast.DeclExternFunc:
		return d.Parameters, true
	}
	return nil, false
}

// buildSignatureInfo constructs an LSP SignatureInformation for a Zirric function.
// The label uses Zirric syntax: "fn name(p1, p2)".
// Each parameter's label is its name string, allowing the editor to highlight it.
func buildSignatureInfo(name string, params []ast.DeclParameter) protocol.SignatureInformation {
	var (
		prefix = "fn " + name + "("
		suffix = ")"

		paramNames = make([]string, len(params))
	)
	for i, p := range params {
		paramNames[i] = p.Name.Value
	}
	label := prefix + strings.Join(paramNames, ", ") + suffix

	paramInfos := make([]protocol.ParameterInformation, len(params))
	for i, pname := range paramNames {
		paramInfos[i] = protocol.ParameterInformation{Label: pname}
	}

	return protocol.SignatureInformation{
		Label:      label,
		Parameters: paramInfos,
	}
}
