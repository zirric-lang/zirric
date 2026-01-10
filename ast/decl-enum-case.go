package ast

import "code.knabel.dev/zirric-lang/zirric/token"

var _ Decl = DeclEnumCase{}

type DeclEnumCase struct {
	Token token.Token
	Case  StaticReference

	Docs *Docs
}

// TokenLiteral implements Node
func (d DeclEnumCase) TokenLiteral() token.Token {
	return d.Token
}

// declarationNode implements Declaration
func (d DeclEnumCase) declarationNode() {}

func (e DeclEnumCase) DeclName() Identifier {
	return e.Case.Name()
}

func (e DeclEnumCase) ExportScope() ExportScope {
	if e.Case.Name().Value[0] == '_' {
		return ExportScopePublic
	}
	return ExportScopeInternal
}

func MakeDeclEnumCase(tok token.Token, name StaticReference) *DeclEnumCase {
	return &DeclEnumCase{
		Token: tok,
		Case:  name,
	}
}

func (decl DeclEnumCase) ProvidedDocs() *Docs {
	return decl.Docs
}

func (n DeclEnumCase) EnumerateChildNodes(action func(child Node)) {
	action(n.Case)
}
