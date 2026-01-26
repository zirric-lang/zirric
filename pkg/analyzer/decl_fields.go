package analyzer

import "code.knabel.dev/zirric-lang/zirric/pkg/ast"

func (a *Analyzer) populateFieldDecls(module *ast.ContextModule) {
	if module == nil || module.Decls == nil {
		return
	}
	populateDeclTableFieldDecls(module.Decls)
	for _, file := range module.Files {
		if file == nil || file.Decls == nil {
			continue
		}
		populateDeclTableFieldDecls(file.Decls)
	}
}

func populateDeclTableFieldDecls(dt *ast.DeclTable) {
	if dt == nil {
		return
	}
	for _, sym := range dt.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		child := sym.ChildTable
		if child != nil {
			switch decl := sym.Decl.(type) {
			case *ast.DeclData:
				for i := range decl.Fields {
					field := decl.Fields[i]
					if existing, ok := child.Symbols[field.Name.Value]; ok && existing.Decl != nil {
						continue
					}
					child.Insert(field)
					insertDeclParams(child, field.Parameters)
				}
			case *ast.DeclAnnotation:
				for i := range decl.Fields {
					field := decl.Fields[i]
					if existing, ok := child.Symbols[field.Name.Value]; ok && existing.Decl != nil {
						continue
					}
					child.Insert(field)
					insertDeclParams(child, field.Parameters)
				}
			case *ast.DeclExternType:
				for _, field := range decl.Fields {
					if existing, ok := child.Symbols[field.Name.Value]; ok && existing.Decl != nil {
						continue
					}
					child.Insert(field)
					insertDeclParams(child, field.Parameters)
				}
			}
			populateDeclTableFieldDecls(child)
		}
	}
}

func insertDeclParams(table *ast.DeclTable, params []ast.DeclParameter) {
	if table == nil {
		return
	}
	for i := range params {
		param := &params[i]
		if existing, ok := table.Symbols[param.Name.Value]; ok && existing.Decl != nil {
			continue
		}
		table.Insert(param)
	}
}
