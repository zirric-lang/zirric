package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/token"
)

var _ Decl = DeclExternValue{}
var _ Overviewable = DeclExternValue{}

type DeclExternValue struct {
	Token       token.Token
	Name        Identifier
	Annotations AnnotationChain

	Docs *Docs
}

// TokenLiteral implements Node
func (d DeclExternValue) TokenLiteral() token.Token {
	return d.Token
}

// statementNode implements Statement
func (d DeclExternValue) statementNode() {}

// declarationNode implements Declaration
func (d DeclExternValue) declarationNode() {}

func (e DeclExternValue) DeclName() Identifier {
	return e.Name
}

func (e DeclExternValue) ExportScope() ExportScope {
	if e.Name.Value[0] == '_' {
		return ExportScopePublic
	}
	return ExportScopeInternal
}

func (e DeclExternValue) DeclOverview() string {
	return fmt.Sprintf("extern let %s", e.Name)
}

func MakeDeclExternValue(tok token.Token, name Identifier) *DeclExternValue {
	return &DeclExternValue{
		Token: tok,
		Name:  name,
	}
}

func (decl DeclExternValue) ProvidedDocs() *Docs {
	return decl.Docs
}

// EnumerateChildNodes implements Decl.
func (n DeclExternValue) EnumerateChildNodes(action func(child Node)) {
	action(n.Name)
}
