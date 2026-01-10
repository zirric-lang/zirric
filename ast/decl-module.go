package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/token"
)

var _ Decl = DeclModule{}
var _ Overviewable = DeclModule{}

type DeclModule struct {
	Token       token.Token
	Name        Identifier
	Annotations AnnotationChain

	Docs *Docs
}

// TokenLiteral implements Node
func (d DeclModule) TokenLiteral() token.Token {
	return d.Token
}

// statementNode implements Statement
func (d DeclModule) statementNode() {}

// declarationNode implements Declaration
func (d DeclModule) declarationNode() {}

func (e DeclModule) DeclName() Identifier {
	return e.Name
}

func (e DeclModule) DeclOverview() string {
	return fmt.Sprintf("module %s", e.Name)
}

func (e DeclModule) ExportScope() ExportScope {
	return ExportScopeLocal
}

func MakeDeclModule(tok token.Token, internalName Identifier) *DeclModule {
	return &DeclModule{Token: tok, Name: internalName}
}

func (decl DeclModule) ProvidedDocs() *Docs {
	return decl.Docs
}

// EnumerateChildNodes implements Decl.
func (n DeclModule) EnumerateChildNodes(action func(child Node)) {
	action(n.Name)
}
