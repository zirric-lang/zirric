package ast

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Decl = DeclParameter{}

type DeclParameter struct {
	Name       Identifier
	Attributes AttributeChain
	TypeHint   TypeExpr

	Docs *Docs
}

// TokenLiteral implements Decl.
func (d DeclParameter) TokenLiteral() token.Token {
	return d.Name.Token
}

// declarationNode implements Decl.
func (DeclParameter) declarationNode() {}

func (e DeclParameter) DeclName() Identifier {
	return e.Name
}

func (e DeclParameter) ExportScope() ExportScope {
	return ExportScopeLocal
}

func MakeDeclParameter(name Identifier, attributes AttributeChain, typeHint TypeExpr) *DeclParameter {
	return &DeclParameter{
		Name:       name,
		Attributes: attributes,
		TypeHint:   typeHint,
	}
}

func (decl DeclParameter) ProvidedDocs() *Docs {
	return decl.Docs
}

// EnumerateChildNodes implements Decl.
func (n DeclParameter) EnumerateChildNodes(action func(child Node)) {
	if n.Attributes != nil {
		n.Attributes.EnumerateChildNodes(action)
	}
	action(n.Name)
}
