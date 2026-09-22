package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Decl = DeclFunc{}
var _ Overviewable = DeclFunc{}

type DeclFunc struct {
	Token      token.Token
	Name       Identifier
	Impl       *ExprFunc
	Attributes AttributeChain
	ReturnType TypeExpr

	Docs *Docs
}

// TokenLiteral implements Decl.
func (d DeclFunc) TokenLiteral() token.Token {
	return d.Token
}

// declarationNode implements Decl.
func (DeclFunc) declarationNode() {}

// statementNode implements Statement.
func (DeclFunc) statementNode() {}

func (e DeclFunc) DeclName() Identifier {
	return e.Name
}

func (e DeclFunc) DeclOverview() string {
	result := fmt.Sprintf("fn %s(%s)", e.Name, formatParamList(e.Impl.Parameters))
	if e.ReturnType != nil {
		result += " -> " + e.ReturnType.TypeExpression()
	}
	return result
}

func (e DeclFunc) ExportScope() ExportScope {
	if e.Name.Value[0] == '_' {
		return ExportScopeInternal
	}
	return ExportScopePublic
}

func MakeDeclFunc(tok token.Token, name Identifier, impl *ExprFunc) *DeclFunc {
	return &DeclFunc{
		Token: tok,
		Name:  name,
		Impl:  impl,
	}
}

func (decl DeclFunc) ProvidedDocs() *Docs {
	return decl.Docs
}

// SetDocs implements Documentable.
func (decl *DeclFunc) SetDocs(docs *Docs) {
	decl.Docs = docs
}

// EnumerateChildNodes implements Decl.
func (n DeclFunc) EnumerateChildNodes(action func(child Node)) {
	if len(n.Attributes) > 0 {
		action(n.Attributes)
		n.Attributes.EnumerateChildNodes(action)
	}
	action(n.Name)
	if n.ReturnType != nil {
		action(n.ReturnType)
		n.ReturnType.EnumerateChildNodes(action)
	}
	action(n.Impl)
	n.Impl.EnumerateChildNodes(action)
}
