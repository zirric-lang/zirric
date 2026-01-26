package ast

type ExportScope int

const (
	ExportScopePublic ExportScope = iota
	ExportScopeInternal
	ExportScopeLocal
)

type Decl interface {
	Declaration
	DeclName() Identifier
	ExportScope() ExportScope
}

type DeclWithSymbols interface {
	Decl
	Symbols() *SymbolTable
}
