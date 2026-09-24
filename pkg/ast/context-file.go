package ast

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Node = &SourceFile{}

type SourceFile struct {
	Token      token.Token
	Path       string
	Statements []Statement
	Decls      *DeclTable
	Symbols    *SymbolTable

	// Module is the file's own `mod` declaration, if it wrote one. A module is declared once per file, so its documentation is only whole once every file of it has been read.
	Module *DeclModule
}

func MakeSourceFile(parent *DeclTable, path string, token token.Token) *SourceFile {
	sf := &SourceFile{
		Token:      token,
		Path:       path,
		Statements: make([]Statement, 0),
	}
	sf.Decls = parent.MakeChild(sf)
	sf.Decls.SetExportScopeLevel(ExportScopeInternal)
	return sf
}

func (sf *SourceFile) Add(globalStmt Statement) {
	if globalStmt == nil {
		panic("compiler-bug: nil statement")
	}
	if decl, ok := globalStmt.(Decl); ok {
		if sym, ok := sf.Decls.resolve(decl.DeclName().Value); !ok || sym.Decl == nil {
			sf.Decls.Insert(decl)
		}
		// Also register import members as individual declarations so they are visible in the file's symbol table (e.g. for attribute resolution).
		if importDecl, ok := decl.(*DeclImport); ok {
			for _, member := range importDecl.Members {
				if _, exists := sf.Decls.resolve(member.DeclName().Value); !exists {
					sf.Decls.Insert(member)
				}
			}
		}
		return
	}
	sf.Statements = append(sf.Statements, globalStmt)
}

func (sf SourceFile) EnumerateChildNodes(action func(child Node)) {
	for _, sym := range sf.Decls.Symbols {
		if sym.Decl == nil {
			continue
		}
		action(sym.Decl)
		sym.Decl.EnumerateChildNodes(action)
	}
	if sf.Decls != nil && sf.Decls.Parent != nil {
		for _, sym := range sf.Decls.Parent.Symbols {
			if sym.Decl == nil {
				continue
			}
			action(sym.Decl)
			sym.Decl.EnumerateChildNodes(action)
		}
	}

	for _, node := range sf.Statements {
		if node == nil {
			panic("compiler bug: missing stmt")
		}
		action(node)
		node.EnumerateChildNodes(action)
	}
}

// TokenLiteral implements Node.
func (sf *SourceFile) TokenLiteral() token.Token {
	return sf.Token
}
