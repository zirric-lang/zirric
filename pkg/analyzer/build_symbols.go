package analyzer

import "code.knabel.dev/zirric-lang/zirric/pkg/ast"

func (a *Analyzer) buildSymbolTables(module *ast.ContextModule) {
	if module == nil || module.Decls == nil {
		return
	}
	root := buildSymbolTableFromDeclTable(module.Decls, nil)
	for _, file := range module.Files {
		if file == nil || file.Decls == nil {
			continue
		}
		buildSymbolTableFromDeclTable(file.Decls, root)
	}
	buildExprForSymbols(module)
	buildExprFuncSymbols(module)
}

func buildExprForSymbols(module *ast.ContextModule) {
	for _, file := range module.Files {
		if file == nil {
			continue
		}
		file.EnumerateChildNodes(func(child ast.Node) {
			switch expr := child.(type) {
			case ast.ExprFor:
				attachExprForSymbols(expr.Body.DeclsTable)
			case *ast.ExprFor:
				attachExprForSymbols(expr.Body.DeclsTable)
			}
		})
	}
}

func attachExprForSymbols(table *ast.DeclTable) {
	if table == nil || table.Resolved != nil {
		return
	}
	var parent *ast.SymbolTable
	if table.Parent != nil {
		// buildExprForSymbols runs before buildExprFuncSymbols, so an expr-for
		// nested inside a closure literal has an unresolved parent (the
		// closure's own DeclTable) at this point — build it eagerly instead
		// of leaving this table parentless, mirroring buildExprFuncSymbols's
		// own defensive handling of the same ordering gap.
		if table.Parent.Resolved == nil {
			buildSymbolTableFromDeclTable(table.Parent, resolveParentSymbolTable(table.Parent.Parent))
		}
		parent = table.Parent.Resolved
	}
	buildSymbolTableFromDeclTable(table, parent)
}

func buildSymbolTableFromDeclTable(dt *ast.DeclTable, parent *ast.SymbolTable) *ast.SymbolTable {
	if dt == nil {
		return nil
	}
	// Avoid re-building if already resolved.
	if dt.Resolved != nil {
		return dt.Resolved
	}

	st := &ast.SymbolTable{
		Parent:   parent,
		OpenedBy: dt.OpenedBy,
		Symbols:  map[string]*ast.Symbol{},
	}
	st.SetExportScopeLevel(dt.ExportScopeLevel())
	dt.Resolved = st

	attachSymbols(dt.OpenedBy, st)

	for name, declSym := range dt.Symbols {
		if declSym == nil || declSym.Decl == nil {
			continue
		}
		sym := &ast.Symbol{
			Name: name,
			Decl: declSym.Decl,
			Errs: declSym.Errs,
		}
		if declSym.ChildTable != nil {
			// Use the ChildTable's DeclTable.Parent to determine the
			// correct parent SymbolTable. DeclTable promotion flattens
			// nesting, but the DeclTable.Parent chain preserves the
			// original lexical hierarchy required for closures.
			childParent := st
			if declSym.ChildTable.Parent != nil {
				// Eagerly resolve the parent DeclTable if needed.
				parentDT := declSym.ChildTable.Parent
				if parentDT.Resolved == nil {
					buildSymbolTableFromDeclTable(parentDT, resolveParentSymbolTable(parentDT.Parent))
				}
				if parentDT.Resolved != nil {
					childParent = parentDT.Resolved
				}
			}
			sym.ChildTable = buildSymbolTableFromDeclTable(declSym.ChildTable, childParent)
		}
		st.Symbols[name] = sym
	}

	return st
}

// resolveParentSymbolTable walks up the DeclTable.Parent chain to find a
// resolved SymbolTable. Returns nil if none is found.
func resolveParentSymbolTable(dt *ast.DeclTable) *ast.SymbolTable {
	for dt != nil {
		if dt.Resolved != nil {
			return dt.Resolved
		}
		dt = dt.Parent
	}
	return nil
}

func attachSymbols(node ast.Node, st *ast.SymbolTable) {
	switch n := node.(type) {
	case *ast.ContextModule:
		n.Symbols = st
	case *ast.SourceFile:
		n.Symbols = st
	case *ast.ExprFunc:
		n.Symbols = st
	}
}

// buildExprFuncSymbols walks the AST and builds symbol tables for ExprFunc
// nodes whose DeclTable was not already processed (i.e. standalone lambdas
// used as expression values, not DeclFunc.Impl which is handled via ChildTable).
func buildExprFuncSymbols(module *ast.ContextModule) {
	walkNode(module, func(node ast.Node) {
		fn, ok := node.(*ast.ExprFunc)
		if !ok || fn.Symbols != nil || fn.Decls == nil {
			return
		}
		if fn.Decls.Resolved != nil {
			// Already built but not attached; just attach.
			fn.Symbols = fn.Decls.Resolved
			return
		}
		var parent *ast.SymbolTable
		if fn.Decls.Parent != nil {
			parentDT := fn.Decls.Parent
			if parentDT.Resolved == nil {
				buildSymbolTableFromDeclTable(parentDT, resolveParentSymbolTable(parentDT.Parent))
			}
			parent = parentDT.Resolved
		}
		buildSymbolTableFromDeclTable(fn.Decls, parent)
	})
}
