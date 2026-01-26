package ast

import "code.knabel.dev/zirric-lang/zirric/pkg/token"

var _ Decl = DeclForBinding{}

type DeclForBinding struct {
	Name  Identifier
	Token token.Token
}

func MakeDeclForBinding(tok token.Token, name Identifier) *DeclForBinding {
	return &DeclForBinding{Token: tok, Name: name}
}

func (d DeclForBinding) TokenLiteral() token.Token {
	return d.Token
}

func (DeclForBinding) declarationNode() {}

func (e DeclForBinding) DeclName() Identifier {
	return e.Name
}

func (DeclForBinding) ExportScope() ExportScope {
	return ExportScopeLocal
}

func (n DeclForBinding) EnumerateChildNodes(action func(child Node)) {
	action(n.Name)
}
