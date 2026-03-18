package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Decl = DeclConstant{}

// DeclConstant represents an immutable binding declared with the `const` keyword.
type DeclConstant struct {
	Name       Identifier
	Value      Expr
	Token      token.Token
	Attributes AttributeChain
	TypeHint   TypeExpr
	IsGlobal   bool

	Docs *Docs
}

// TokenLiteral implements Node
func (d DeclConstant) TokenLiteral() token.Token {
	return d.Token
}

// statementNode implements Statement
func (DeclConstant) statementNode() {}

// declarationNode implements Declaration
func (DeclConstant) declarationNode() {}

func (e DeclConstant) DeclName() Identifier {
	return e.Name
}

func (e DeclConstant) DeclOverview() string {
	return fmt.Sprintf("const %s", e.Name)
}

func (e DeclConstant) ExportScope() ExportScope {
	if !e.IsGlobal {
		return ExportScopeLocal
	}
	if e.Name.Value[0] == '_' {
		return ExportScopeInternal
	}
	return ExportScopePublic
}

func MakeDeclConstant(tok token.Token, name Identifier, value Expr) *DeclConstant {
	return &DeclConstant{
		Token: tok,
		Name:  name,
		Value: value,
	}
}

func (e DeclConstant) ProvidedDocs() *Docs {
	return e.Docs
}

// EnumerateChildNodes implements Decl.
func (n DeclConstant) EnumerateChildNodes(action func(child Node)) {
	action(n.Name)
	if len(n.Attributes) > 0 {
		action(n.Attributes)
	}
	action(n.Value)
}
