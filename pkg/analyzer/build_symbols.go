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
		parent = table.Parent.Resolved
	}
	buildSymbolTableFromDeclTable(table, parent)
}

func buildSymbolTableFromDeclTable(dt *ast.DeclTable, parent *ast.SymbolTable) *ast.SymbolTable {
	if dt == nil {
		return nil
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
			sym.ChildTable = buildSymbolTableFromDeclTable(declSym.ChildTable, st)
		}
		st.Symbols[name] = sym
	}

	return st
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
