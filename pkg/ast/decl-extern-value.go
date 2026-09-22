package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Decl = DeclExternValue{}
var _ Overviewable = DeclExternValue{}

type DeclExternValue struct {
	Token      token.Token
	Name       Identifier
	Attributes AttributeChain
	TypeHint   TypeExpr

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
		return ExportScopeInternal
	}
	return ExportScopePublic
}

func (e DeclExternValue) DeclOverview() string {
	if e.TypeHint != nil {
		return fmt.Sprintf("extern const %s: %s", e.Name, e.TypeHint.TypeExpression())
	}
	return fmt.Sprintf("extern const %s", e.Name)
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

// SetDocs implements Documentable.
func (decl *DeclExternValue) SetDocs(docs *Docs) {
	decl.Docs = docs
}

// EnumerateChildNodes implements Decl.
func (n DeclExternValue) EnumerateChildNodes(action func(child Node)) {
	action(n.Name)
	if len(n.Attributes) > 0 {
		action(n.Attributes)
	}
	if n.TypeHint != nil {
		action(n.TypeHint)
		n.TypeHint.EnumerateChildNodes(action)
	}
}
