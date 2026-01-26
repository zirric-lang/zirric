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
		resolveNode(sym.Decl, module.Symbols)
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
	case *ast.DeclAnnotationInstance:
		if symbols != nil {
			symbols.LookupRef(n.Reference, ast.RequireAnnotation(n))
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
		if len(n.Annotations) > 0 {
			resolveNode(n.Annotations, symbols)
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
