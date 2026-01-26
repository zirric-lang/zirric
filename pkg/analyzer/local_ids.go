package analyzer

import "code.knabel.dev/zirric-lang/zirric/pkg/ast"

func (a *Analyzer) assignLocalIDs(module *ast.ContextModule) {
	if module == nil {
		return
	}
	seen := map[*ast.ExprFunc]struct{}{}

	for _, file := range module.Files {
		if file == nil || file.Symbols == nil {
			continue
		}
		counter := 0
		assignLocalIDsInBlock(file.Symbols, &counter, file.Statements, seen)
		for _, sym := range file.Symbols.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}
			assignLocalIDsInDecl(file.Symbols, &counter, sym.Decl, seen)
		}
	}

	if module.Symbols != nil {
		counter := 0
		for _, sym := range module.Symbols.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}
			assignLocalIDsInDecl(module.Symbols, &counter, sym.Decl, seen)
		}
	}

	walkNode(module, func(node ast.Node) {
		fn, ok := node.(*ast.ExprFunc)
		if !ok {
			return
		}
		if _, ok := seen[fn]; ok {
			return
		}
		seen[fn] = struct{}{}
		assignFunctionLocals(fn, seen)
	})
}

func assignFunctionLocals(fn *ast.ExprFunc, seen map[*ast.ExprFunc]struct{}) {
	if fn == nil || fn.Symbols == nil {
		return
	}
	counter := 0
	for _, param := range fn.Parameters {
		assignLocalSymbol(fn.Symbols, param.Name.Value, &counter)
	}
	assignLocalIDsInBlock(fn.Symbols, &counter, fn.Impl, seen)
}

func assignLocalIDsInDecl(symbols *ast.SymbolTable, counter *int, decl ast.Decl, seen map[*ast.ExprFunc]struct{}) {
	switch d := decl.(type) {
	case *ast.DeclVariable:
		assignLocalIDsInExpr(symbols, counter, d.Value, seen)
	case *ast.DeclFunc:
		if d.Impl == nil {
			return
		}
		if _, ok := seen[d.Impl]; ok {
			return
		}
		seen[d.Impl] = struct{}{}
		assignFunctionLocals(d.Impl, seen)
	}
}

func assignLocalIDsInBlock(symbols *ast.SymbolTable, counter *int, block ast.Block, seen map[*ast.ExprFunc]struct{}) {
	for _, stmt := range block {
		assignLocalIDsInStmt(symbols, counter, stmt, seen)
	}
}

func assignLocalIDsInStmt(symbols *ast.SymbolTable, counter *int, stmt ast.Statement, seen map[*ast.ExprFunc]struct{}) {
	switch s := stmt.(type) {
	case *ast.DeclVariable:
		if s.ExportScope() == ast.ExportScopeLocal {
			assignLocalSymbol(symbols, s.Name.Value, counter)
		}
		assignLocalIDsInExpr(symbols, counter, s.Value, seen)
	case *ast.StmtExpr:
		assignLocalIDsInExpr(symbols, counter, s.Expr, seen)
	case ast.StmtIf:
		assignLocalIDsInExpr(symbols, counter, s.Condition, seen)
		assignLocalIDsInBlock(symbols, counter, s.IfBlock, seen)
		for _, elif := range s.ElseIf {
			assignLocalIDsInExpr(symbols, counter, elif.Condition, seen)
			assignLocalIDsInBlock(symbols, counter, elif.Block, seen)
		}
		assignLocalIDsInBlock(symbols, counter, s.ElseBlock, seen)
	case ast.StmtFor:
		if s.CollectionIdent != nil {
			assignLocalSymbol(symbols, s.CollectionIdent.Value, counter)
		}
		if s.Condition != nil {
			assignLocalIDsInExpr(symbols, counter, s.Condition, seen)
		}
		if s.CollectionExpr != nil {
			assignLocalIDsInExpr(symbols, counter, s.CollectionExpr, seen)
		}
		assignLocalIDsInBlock(symbols, counter, s.Body, seen)
	case *ast.StmtReturn:
		if s.Expr != nil {
			assignLocalIDsInExpr(symbols, counter, s.Expr, seen)
		}
	}
}

func assignLocalIDsInExpr(symbols *ast.SymbolTable, counter *int, expr ast.Expr, seen map[*ast.ExprFunc]struct{}) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *ast.ExprFunc:
		if _, ok := seen[e]; ok {
			return
		}
		seen[e] = struct{}{}
		assignFunctionLocals(e, seen)
		return
	case ast.ExprFor:
		assignLocalIDsInExprFor(&e, symbols, counter, seen)
		return
	case *ast.ExprFor:
		assignLocalIDsInExprFor(e, symbols, counter, seen)
		return
	}
	expr.EnumerateChildNodes(func(child ast.Node) {
		if subExpr, ok := child.(ast.Expr); ok {
			assignLocalIDsInExpr(symbols, counter, subExpr, seen)
			return
		}
		if stmt, ok := child.(ast.Statement); ok {
			assignLocalIDsInStmt(symbols, counter, stmt, seen)
			return
		}
		if decl, ok := child.(ast.Decl); ok {
			if fnDecl, ok := decl.(*ast.DeclFunc); ok {
				if fnDecl.Impl == nil {
					return
				}
				if _, ok := seen[fnDecl.Impl]; ok {
					return
				}
				seen[fnDecl.Impl] = struct{}{}
				assignFunctionLocals(fnDecl.Impl, seen)
			}
		}
	})
}

func assignLocalIDsInExprFor(node *ast.ExprFor, symbols *ast.SymbolTable, counter *int, seen map[*ast.ExprFunc]struct{}) {
	if node == nil {
		return
	}
	if node.CollectionExpr != nil {
		assignLocalIDsInExpr(symbols, counter, node.CollectionExpr, seen)
	}
	if node.Condition != nil {
		assignLocalIDsInExpr(symbols, counter, node.Condition, seen)
	}
	bodySymbols := node.Body.Symbols
	if bodySymbols == nil && node.Body.DeclsTable != nil {
		bodySymbols = node.Body.DeclsTable.Resolved
	}
	if bodySymbols == nil {
		bodySymbols = symbols
	}
	if node.CollectionIdent != nil {
		assignLocalSymbol(bodySymbols, node.CollectionIdent.Value, counter)
	}
	for _, decl := range node.Body.Decls {
		if decl.ExportScope() == ast.ExportScopeLocal {
			assignLocalSymbol(bodySymbols, decl.Name.Value, counter)
		}
		assignLocalIDsInExpr(bodySymbols, counter, decl.Value, seen)
	}
	assignLocalIDsInBlock(bodySymbols, counter, node.Body.Stmts, seen)
}

func assignLocalSymbol(symbols *ast.SymbolTable, name string, counter *int) {
	if symbols == nil || counter == nil {
		return
	}
	sym := symbols.Symbols[name]
	if sym == nil || sym.Decl == nil {
		return
	}
	if sym.LocalId != nil {
		return
	}
	id := *counter
	*counter = *counter + 1
	sym.LocalId = &id
}
