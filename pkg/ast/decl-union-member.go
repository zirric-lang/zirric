package ast

import "code.knabel.dev/zirric-lang/zirric/pkg/token"

var _ Decl = DeclUnionMember{}

type DeclUnionMember struct {
	Token  token.Token
	Member StaticReference

	Docs *Docs
}

// TokenLiteral implements Node
func (d DeclUnionMember) TokenLiteral() token.Token {
	return d.Token
}

// declarationNode implements Declaration
func (d DeclUnionMember) declarationNode() {}

func (e DeclUnionMember) DeclName() Identifier {
	return e.Member.Name()
}

func (e DeclUnionMember) ExportScope() ExportScope {
	if e.Member.Name().Value[0] == '_' {
		return ExportScopeInternal
	}
	return ExportScopePublic
}

func MakeDeclUnionMember(tok token.Token, name StaticReference) *DeclUnionMember {
	return &DeclUnionMember{
		Token:  tok,
		Member: name,
	}
}

func (decl DeclUnionMember) ProvidedDocs() *Docs {
	return decl.Docs
}

func (n DeclUnionMember) EnumerateChildNodes(action func(child Node)) {
	action(n.Member)
}
