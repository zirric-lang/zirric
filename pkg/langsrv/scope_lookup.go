package langsrv

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
)

// findLocalDecl searches for a local declaration (parameter, local var/const, for-binding) matching the given name at the cursor position. It walks the AST to find the enclosing function scope, mirroring the approach used by completion's collectLocalsBeforeCursor. Returns nil if no local is found.
func findLocalDecl(module *ast.ContextModule, sourceURI string, cursorOffset int, name string) ast.Decl {
	locals := collectLocalsBeforeCursor(module, sourceURI, cursorOffset)
	// Walk in reverse so the innermost (most recently declared) match wins.
	for i := len(locals) - 1; i >= 0; i-- {
		if locals[i].DeclName().Value == name {
			return locals[i]
		}
	}
	return nil
}

// resolveWordDecl resolves a word at a cursor position to a declaration, checking local scope first (parameters, local vars/consts, for-bindings), then module and file scope (walks parent chain for prelude).
func (ls *zirricLangserver) resolveWordDecl(
	module *ast.ContextModule,
	currentSF *ast.SourceFile,
	path string,
	cursorOffset int,
	word string,
) *ast.DeclSymbol {
	sourceURI := string(registry.JoinModuleURI("", path))

	// Check local scope first (parameters, local vars, for-bindings).
	if local := findLocalDecl(module, sourceURI, cursorOffset, word); local != nil {
		return &ast.DeclSymbol{Name: word, Decl: local}
	}

	// Try module-level symbols (walks parent chain, so prelude symbols are found).
	if sym, ok := module.Decls.Resolve(word); ok && sym.Decl != nil {
		return sym
	}
	// Also check current file's local scope for import-scoped symbols.
	if currentSF != nil {
		if sym, ok := currentSF.Decls.Resolve(word); ok && sym.Decl != nil {
			return sym
		}
	}
	return nil
}
