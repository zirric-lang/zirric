package analyzer

import "code.knabel.dev/zirric-lang/zirric/pkg/ast"

func (a *Analyzer) resolveIdentifiers(module *ast.ContextModule) {
	if module == nil || module.Symbols == nil || module.Decls == nil {
		return
	}
	for _, sym := range module.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		resolveNode(sym.Decl, symbolsForPromotedDecl(module, sym.Decl))
	}

	for _, file := range module.Files {
		if file == nil || file.Symbols == nil || file.Decls == nil {
			continue
		}
		for _, sym := range file.Decls.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}
			resolveNode(sym.Decl, file.Symbols)
		}
		for _, stmt := range file.Statements {
			resolveNode(stmt, file.Symbols)
		}
	}
}

// symbolsForPromotedDecl returns the symbol table to use when resolving a module.Decls.Symbols entry's own value/body.
// DeclFunc handles this itself (resolveNode uses n.Impl.Symbols, its own properly file-parented table), but DeclConstant/DeclVariable have no such per-declaration table — when exported (ast.DeclTable.Insert promotes them into the module-level table instead of leaving them in their file's own table), module.Symbols is their only entry point here, and it's the *flattened* module-wide view, which doesn't include the declaring file's own imports (imports stay file-local, never promoted).
// So a top-level `const x = someImport.member` would fail to resolve someImport with "undefined identifier".
// Find the declaring file's own symbol table instead, by matching source file paths (mirrors compiler.sourceFileSymbols' same file-matching pattern).
func symbolsForPromotedDecl(module *ast.ContextModule, decl ast.Decl) *ast.SymbolTable {
	switch decl.(type) {
	case *ast.DeclConstant, *ast.DeclVariable:
	default:
		return module.Symbols
	}
	declSource := decl.TokenLiteral().Source
	if declSource == nil {
		return module.Symbols
	}
	for _, file := range module.Files {
		if file == nil || file.Symbols == nil {
			continue
		}
		fileSource := file.TokenLiteral().Source
		if fileSource != nil && fileSource.File == declSource.File {
			return file.Symbols
		}
	}
	return module.Symbols
}

func resolveNode(node ast.Node, symbols *ast.SymbolTable) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *ast.ExprIdentifier:
		if n.Symbol != nil || symbols == nil {
			return
		}
		n.Symbol = symbols.LookupIdentifier(n.Name)
		return
	case *ast.DeclAttrInstance:
		if symbols != nil {
			symbols.LookupRef(n.Reference, ast.RequireAttribute(n))
		}
		for _, arg := range n.Arguments {
			resolveNode(arg, symbols)
		}
		return
	case *ast.ExprFunc:
		funcSymbols := n.Symbols
		if funcSymbols == nil {
			funcSymbols = symbols
		}
		for _, param := range n.Parameters {
			resolveNode(param, funcSymbols)
		}
		for _, stmt := range n.Impl {
			resolveNode(stmt, funcSymbols)
		}
		return
	case *ast.DeclFunc:
		if len(n.Attributes) > 0 {
			resolveNode(n.Attributes, symbols)
		}
		if n.Impl != nil {
			resolveNode(n.Impl, n.Impl.Symbols)
		}
		return
	case ast.ExprFor:
		resolveExprFor(&n, symbols)
		return
	case *ast.ExprFor:
		resolveExprFor(n, symbols)
		return
	}

	node.EnumerateChildNodes(func(child ast.Node) {
		resolveNode(child, symbols)
	})
}

func resolveExprFor(node *ast.ExprFor, symbols *ast.SymbolTable) {
	if node == nil {
		return
	}
	if node.Condition != nil {
		resolveNode(node.Condition, symbols)
	}
	if node.CollectionExpr != nil {
		resolveNode(node.CollectionExpr, symbols)
	}
	bodySymbols := node.Body.Symbols
	if bodySymbols == nil && node.Body.DeclsTable != nil {
		bodySymbols = node.Body.DeclsTable.Resolved
	}
	for _, decl := range node.Body.Decls {
		resolveNode(decl, bodySymbols)
	}
	for _, stmt := range node.Body.Stmts {
		resolveNode(stmt, bodySymbols)
	}
}
