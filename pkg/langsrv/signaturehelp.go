package langsrv

import (
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

	module, _, _, err := ls.parseModuleFilesForPath(path)
	if err != nil {
		return nil, nil
	}

	offset := offsetForPosition(text, params.Position)
	funcName, qualChain, activeParam, found := callChainContext(text, offset)
	if !found {
		return nil, nil
	}

	var (
		sourceURI = string(registry.JoinModuleURI("", path))
		currentSF = findSourceFile(module, sourceURI)

		sigLabel  string
		sigParams []protocol.ParameterInformation
		resolved  bool
	)

	if len(qualChain) > 0 {
		// Single qualifier: try import alias or mod declaration first.
		if len(qualChain) == 1 {
			alias := qualChain[0]
			if imp, ok := findImportDecl(currentSF, alias); ok {
				if importedMod, _, ok := ls.loadImportedModule(imp); ok {
					sigLabel, sigParams, resolved = callableParamsFromModule(importedMod, funcName)
				}
			}
			if !resolved && isModuleDecl(module, alias) {
				sigLabel, sigParams, resolved = callableParamsFromModule(module, funcName)
			}
		}

		// Multi-segment or non-module: resolve through type chain.
		if !resolved {
			result := ls.resolveDotChain(module, currentSF, path, offset, qualChain)
			if result != nil {
				if result.isModuleScope {
					sigLabel, sigParams, resolved = callableParamsFromModule(result.module, funcName)
				} else if len(result.fields) > 0 {
					sigLabel, sigParams, resolved = callableParamsFromFields(result.fields, funcName)
				}
			}
		}
	} else {
		sigLabel, sigParams, resolved = callableParamsFromModule(module, funcName)
	}

	if !resolved {
		return nil, nil
	}

	var (
		zero   = protocol.UInteger(0)
		active = protocol.UInteger(activeParam)
	)

	sig := protocol.SignatureInformation{
		Label:      sigLabel,
		Parameters: sigParams,
	}

	return &protocol.SignatureHelp{
		Signatures:      []protocol.SignatureInformation{sig},
		ActiveSignature: &zero,
		ActiveParameter: &active,
	}, nil
}

// callChainContext scans backward from offset to find the enclosing function call.
// It returns the function name, the full dot-chain qualifier (if any),
// active parameter index, and whether found.
//
// For "person.name.toggle(": funcName="toggle", qualChain=["person", "name"], activeParam=0
// For "alias.func(a, ": funcName="func", qualChain=["alias"], activeParam=1
// For "func(": funcName="func", qualChain=nil, activeParam=0
func callChainContext(text string, offset int) (funcName string, qualChain []string, activeParam int, found bool) {
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

				// Scan backward collecting "ident." chain segments.
				cursor := start
				for cursor > 1 && text[cursor-1] == '.' {
					segEnd := cursor - 1
					segStart := segEnd
					for segStart > 0 && isSimpleIdentByte(text[segStart-1]) {
						segStart--
					}
					if segStart == segEnd {
						break
					}
					qualChain = append(qualChain, text[segStart:segEnd])
					cursor = segStart
				}
				// Reverse chain (collected backwards).
				for i, j := 0, len(qualChain)-1; i < j; i, j = i+1, j-1 {
					qualChain[i], qualChain[j] = qualChain[j], qualChain[i]
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

// callableParamsFromFields looks up a field by name and returns its parameters
// for signature help. Works for method-like fields with parameters.
func callableParamsFromFields(fields []ast.DeclField, name string) (label string, params []protocol.ParameterInformation, resolved bool) {
	for _, f := range fields {
		if f.Name.Value != name {
			continue
		}
		if len(f.Parameters) > 0 {
			paramNames := declFieldParamNames(f.Parameters)
			return buildCallableSignature("", name, paramNames)
		}
		return name + "()", nil, true
	}
	return "", nil, false
}

// declFieldParamNames formats field method parameters for display.
func declFieldParamNames(params []ast.DeclParameter) []string {
	names := make([]string, len(params))
	for i, p := range params {
		if p.TypeHint != nil {
			names[i] = p.Name.Value + ": " + p.TypeHint.TypeExpression()
		} else {
			names[i] = p.Name.Value
		}
	}
	return names
}

// callableParamsFromModule looks up a declaration by name and returns its parameters
// for signature help. Works for functions, data constructors, and attribute calls.
func callableParamsFromModule(mod *ast.ContextModule, name string) (label string, params []protocol.ParameterInformation, resolved bool) {
	sym, _ := mod.Decls.Resolve(name)
	if sym == nil || sym.Decl == nil {
		return "", nil, false
	}

	switch d := sym.Decl.(type) {
	case ast.DeclFunc:
		if d.Impl != nil {
			return buildCallableSignature("fn", name, declParamNames(d.Impl.Parameters))
		}
		return "fn " + name + "()", nil, true

	case *ast.DeclFunc:
		if d.Impl != nil {
			return buildCallableSignature("fn", name, declParamNames(d.Impl.Parameters))
		}
		return "fn " + name + "()", nil, true

	case ast.DeclExternFunc:
		return buildCallableSignature("fn", name, declParamNames(d.Parameters))

	case *ast.DeclExternFunc:
		return buildCallableSignature("fn", name, declParamNames(d.Parameters))

	case ast.DeclData:
		return buildCallableSignature("data", name, declFieldNames(d.Fields))

	case *ast.DeclData:
		return buildCallableSignature("data", name, declFieldNames(d.Fields))

	case ast.DeclAttr:
		return buildCallableSignature("attr", name, declFieldNames(d.Fields))

	case *ast.DeclAttr:
		return buildCallableSignature("attr", name, declFieldNames(d.Fields))
	}
	return "", nil, false
}

func declParamNames(params []ast.DeclParameter) []string {
	names := make([]string, len(params))
	for i, p := range params {
		names[i] = p.Name.Value
	}
	return names
}

func declFieldNames(fields []ast.DeclField) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		if f.TypeHint != nil {
			names[i] = f.Name.Value + ": " + f.TypeHint.TypeExpression()
		} else {
			names[i] = f.Name.Value
		}
	}
	return names
}

func buildCallableSignature(keyword, name string, paramNames []string) (string, []protocol.ParameterInformation, bool) {
	label := keyword + " " + name + "(" + strings.Join(paramNames, ", ") + ")"
	paramInfos := make([]protocol.ParameterInformation, len(paramNames))
	for i, pname := range paramNames {
		paramInfos[i] = protocol.ParameterInformation{Label: pname}
	}
	return label, paramInfos, true
}
