package analyzer

import "code.knabel.dev/zirric-lang/zirric/pkg/ast"

// assignTypeSymbols sets TypeSymbol on each module-level symbol so that
// runtime values can report their type via TypeConstantId().
//
// Rules:
//   - DeclData, DeclUnion, DeclExternType, DeclAttr → TypeSymbol = self
//   - DeclFunc, DeclExternFunc → TypeSymbol = the "Func" extern type symbol
func (a *Analyzer) assignTypeSymbols(module *ast.ContextModule) {
	funcSym := module.Symbols.Lookup("Func", nil)

	for _, sym := range module.Symbols.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		orig := sym.Original()
		if orig.Decl == nil || orig.TypeSymbol != nil {
			continue
		}
		switch orig.Decl.(type) {
		case *ast.DeclData, *ast.DeclUnion, *ast.DeclExternType, *ast.DeclAttr:
			orig.TypeSymbol = orig
		case *ast.DeclFunc, *ast.DeclExternFunc:
			if funcSym != nil {
				orig.TypeSymbol = funcSym.Original()
			}
		}
	}
}
