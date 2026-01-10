package ast

import (
	"code.knabel.dev/zirric-lang/zirric/token"
)

var _ Node = &SourceFile{}

type SourceFile struct {
	Token      token.Token
	Path       string
	Statements []Statement
	Symbols    *SymbolTable
}

func MakeSourceFile(parent *SymbolTable, path string, token token.Token) *SourceFile {
	sf := &SourceFile{
		Token:      token,
		Path:       path,
		Statements: make([]Statement, 0),
	}
	sf.Symbols = MakeSymbolTable(parent, sf)
	sf.Symbols.exportScopeLevel = ExportScopeInternal
	return sf
}

func (sf *SourceFile) Add(globalStmt Statement) {
	if globalStmt == nil {
		panic("compiler-bug: nil statement")
	}
	if decl, ok := globalStmt.(Decl); ok {
		if sym, ok := sf.Symbols.resolve(decl.DeclName().Value); !ok || sym.Decl == nil {
			sf.Symbols.Insert(decl)
		}
		return
	}
	sf.Statements = append(sf.Statements, globalStmt)
}

func (sf SourceFile) EnumerateChildNodes(action func(child Node)) {
	for _, sym := range sf.Symbols.Symbols {
		if sym.Decl == nil {
			continue
		}
		action(sym.Decl)
		sym.Decl.EnumerateChildNodes(action)
	}
	for _, sym := range sf.Symbols.Parent.Symbols {
		if sym.Decl == nil {
			continue
		}
		action(sym.Decl)
		sym.Decl.EnumerateChildNodes(action)
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
