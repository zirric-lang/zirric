package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Decl = DeclExternFunc{}
var _ Overviewable = DeclExternFunc{}

type DeclExternFunc struct {
	Token      token.Token
	Name       Identifier
	Parameters []DeclParameter
	Attributes AttributeChain

	Docs *Docs
}

// TokenLiteral implements Node
func (d DeclExternFunc) TokenLiteral() token.Token {
	return d.Token
}

// statementNode implements Statement
func (d DeclExternFunc) statementNode() {}

// declarationNode implements Declaration
func (d DeclExternFunc) declarationNode() {}

func (e DeclExternFunc) DeclName() Identifier {
	return e.Name
}

func (e DeclExternFunc) ExportScope() ExportScope {
	if e.Name.Value[0] == '_' {
		return ExportScopeInternal
	}
	return ExportScopePublic
}

func (e DeclExternFunc) DeclOverview() string {
	return fmt.Sprintf("extern fn %s(%s)", e.Name, formatParamList(e.Parameters))
}

func MakeDeclExternFunc(tok token.Token, name Identifier) *DeclExternFunc {
	return &DeclExternFunc{
		Token: tok,
		Name:  name,
	}
}

func (ef *DeclExternFunc) SetParams(params []DeclParameter) {
	ef.Parameters = params
}

func (decl DeclExternFunc) ProvidedDocs() *Docs {
	return decl.Docs
}

// EnumerateChildNodes implements Decl.
func (n DeclExternFunc) EnumerateChildNodes(action func(child Node)) {
	action(n.Name)
	if len(n.Attributes) > 0 {
		action(n.Attributes)
	}
	for _, node := range n.Parameters {
		action(node)
		node.EnumerateChildNodes(action)
	}
}
