package ast

import (
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Decl = DeclParameter{}
var _ Overviewable = DeclParameter{}

type DeclParameter struct {
	Name       Identifier
	Attributes AttributeChain
	TypeHint   TypeExpr

	Docs *Docs
}

// formatParamList formats a slice of DeclParameter for display, including type hints where present. E.g. "name: String, age: Int" or "a, b".
func formatParamList(params []DeclParameter) string {
	parts := make([]string, len(params))
	for i, p := range params {
		switch {
		case p.Name.Value == "" && p.TypeHint != nil:
			// Bare, unnamed parameter, e.g. fn(@Cmd) -> Void.
			parts[i] = p.TypeHint.TypeExpression()
		case p.TypeHint != nil:
			parts[i] = string(p.Name.Value) + ": " + p.TypeHint.TypeExpression()
		default:
			parts[i] = string(p.Name.Value)
		}
	}
	return strings.Join(parts, ", ")
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

func (e DeclParameter) DeclOverview() string {
	s := e.Name.Value
	if e.TypeHint != nil {
		s += ": " + e.TypeHint.TypeExpression()
	}
	return s
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
	if n.TypeHint != nil {
		action(n.TypeHint)
		n.TypeHint.EnumerateChildNodes(action)
	}
}
