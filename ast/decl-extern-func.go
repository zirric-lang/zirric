package ast

import (
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/token"
)

var _ Decl = DeclExternFunc{}
var _ Overviewable = DeclExternFunc{}

type DeclExternFunc struct {
	Token       token.Token
	Name        Identifier
	Parameters  []DeclParameter
	Annotations AnnotationChain

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
		return ExportScopePublic
	}
	return ExportScopeInternal
}

func (e DeclExternFunc) DeclOverview() string {
	if len(e.Parameters) == 0 {
		return fmt.Sprintf("extern func %s()", e.Name)
	}
	paramNames := make([]string, len(e.Parameters))
	for i, param := range e.Parameters {
		paramNames[i] = string(param.Name.Value)
	}
	return fmt.Sprintf("extern func %s(%s)", e.Name, strings.Join(paramNames, ", "))
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
	for _, node := range n.Parameters {
		action(node)
	}
}
