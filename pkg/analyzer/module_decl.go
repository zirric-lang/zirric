package analyzer

import (
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/resolver"
)

// checkModuleDeclarations holds a module's files to the name their location implies: every file names the module, they all name the same one, and the name is the package base joined with the file's directory.
//
// Only modules of the package the Cavefile describes are checked, since its `mod` fixes the base. A loose script, the REPL or a dependency has no such base to be held to.
func (a *Analyzer) checkModuleDeclarations(module *ast.ContextModule) []AnalysisError {
	base := a.declaredPackageBase()
	if base == "" {
		return nil
	}
	if module.Name != base && !strings.HasPrefix(string(module.Name), string(base)+".") {
		return nil
	}

	var errs []AnalysisError
	var declared *ast.DeclModule
	for _, file := range module.Files {
		if file == nil {
			continue
		}
		decl := fileModuleDecl(file)
		if decl == nil {
			errs = append(errs, AnalysisError{
				Token:   file.TokenLiteral(),
				Summary: "missing module declaration",
				Details: fmt.Sprintf("every source file must name the module it belongs to, so declare 'mod %s'", module.Name),
			})
			continue
		}
		path := registry.LogicalURI(decl.Path.String())
		if declared != nil && path != registry.LogicalURI(declared.Path.String()) {
			errs = append(errs, AnalysisError{
				Token:   ast.StaticReference(decl.Path).TokenLiteral(),
				Summary: "conflicting module declaration",
				Details: fmt.Sprintf("the files of a module must agree on its name, and another file of this module declares %s", declared.Path),
			})
			continue
		}
		if declared == nil {
			declared = decl
		}
		if path != module.Name {
			errs = append(errs, AnalysisError{
				Token:   ast.StaticReference(decl.Path).TokenLiteral(),
				Summary: "wrong module name",
				Details: fmt.Sprintf("this file belongs to %s, so declare 'mod %s' rather than %s", module.Name, module.Name, decl.Path),
			})
		}
	}
	return errs
}

func fileModuleDecl(file *ast.SourceFile) *ast.DeclModule {
	if file.Decls == nil {
		return nil
	}
	for _, sym := range file.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		if decl, ok := sym.Decl.(*ast.DeclModule); ok {
			return decl
		}
	}
	return nil
}

func (a *Analyzer) declaredPackageBase() registry.LogicalURI {
	if a.resolver == nil {
		return ""
	}
	declared, ok := a.resolver.(resolver.DeclaredPackageBase)
	if !ok {
		return ""
	}
	return declared.DeclaredPackageBase()
}
