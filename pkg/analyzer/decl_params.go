package analyzer

import "code.knabel.dev/zirric-lang/zirric/pkg/ast"

func (a *Analyzer) populateFunctionParams(module *ast.ContextModule) {
	if module == nil || module.Decls == nil {
		return
	}
	populateDeclTableFuncParams(module.Decls)
	for _, file := range module.Files {
		if file == nil || file.Decls == nil {
			continue
		}
		populateDeclTableFuncParams(file.Decls)
		walkNode(file, func(child ast.Node) {
			fn, ok := child.(*ast.ExprFunc)
			if !ok || fn.Decls == nil {
				return
			}
			insertDeclParams(fn.Decls, fn.Parameters)
		})
	}
}

func populateDeclTableFuncParams(dt *ast.DeclTable) {
	if dt == nil {
		return
	}
	for _, sym := range dt.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		switch decl := sym.Decl.(type) {
		case *ast.DeclFunc:
			if decl.Impl != nil && decl.Impl.Decls != nil {
				insertDeclParams(decl.Impl.Decls, decl.Impl.Parameters)
			}
		case *ast.DeclExternFunc:
			insertDeclParams(sym.ChildTable, decl.Parameters)
		}
		if sym.ChildTable != nil {
			populateDeclTableFuncParams(sym.ChildTable)
		}
	}
}
