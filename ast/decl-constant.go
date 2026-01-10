package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/token"
)

var _ Decl = DeclVariable{}
var _ Overviewable = DeclVariable{}

type DeclVariable struct {
	Name        Identifier
	Value       Expr
	Token       token.Token
	Annotations AnnotationChain
	IsGlobal    bool

	Docs *Docs
}

// TokenLiteral implements Node
func (d DeclVariable) TokenLiteral() token.Token {
	return d.Token
}

// statementNode implements Statement
func (DeclVariable) statementNode() {}

// declarationNode implements Declaration
func (DeclVariable) declarationNode() {}

func (e DeclVariable) DeclName() Identifier {
	return e.Name
}

func (e DeclVariable) DeclOverview() string {
	return fmt.Sprintf("let %s", e.Name)
}

func (e DeclVariable) ExportScope() ExportScope {
	if !e.IsGlobal {
		return ExportScopeLocal
	}
	if e.Name.Value[0] == '_' {
		return ExportScopePublic
	}
	return ExportScopeInternal
}

func MakeDeclVariable(tok token.Token, name Identifier, value Expr) *DeclVariable {
	return &DeclVariable{
		Token: tok,
		Name:  name,
		Value: value,
	}
}

func (e DeclVariable) ProvidedDocs() *Docs {
	return e.Docs
}

// EnumerateChildNodes implements Decl.
func (n DeclVariable) EnumerateChildNodes(action func(child Node)) {
	action(n.Name)
	action(n.Value)
}
