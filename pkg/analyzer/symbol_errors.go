package analyzer

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

func collectSymbolErrors(st *ast.SymbolTable) []AnalysisError {
	if st == nil {
		return nil
	}
	return symbolErrors(symroot(st))
}

func symroot(st *ast.SymbolTable) *ast.SymbolTable {
	if st.Parent == nil {
		return st
	}
	return symroot(st.Parent)
}

func symbolErrors(st *ast.SymbolTable) []AnalysisError {
	var errs []AnalysisError
	for _, s := range st.Symbols {
		var tok token.Token
		if s.Decl != nil {
			tok = s.Decl.TokenLiteral()
		} else {
			tok = st.OpenedBy.TokenLiteral()
		}
		for _, err := range s.Errs {
			errs = append(errs, AnalysisError{
				Token:   tok,
				Summary: "declaration error",
				Details: err.Error(),
			})
		}

		for _, usage := range s.Usages {
			for _, err := range usage.Errs {
				errs = append(errs, AnalysisError{
					Token:   usage.Node.TokenLiteral(),
					Summary: "usage error",
					Details: err.Error(),
				})
			}
		}

		if s.ChildTable != nil {
			errs = append(errs, symbolErrors(s.ChildTable)...)
		}
	}
	return errs
}
