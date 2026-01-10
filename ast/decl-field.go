package ast

import (
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/token"
)

var _ Decl = DeclField{}
var _ Overviewable = DeclField{}

type DeclField struct {
	Name       Identifier
	Parameters []DeclParameter

	Annotations AnnotationChain

	Docs *Docs
}

// TokenLiteral implements Decl.
func (d DeclField) TokenLiteral() token.Token {
	return d.Name.Token
}

// declarationNode implements Decl.
func (DeclField) declarationNode() {}

// IsConstantMember implements DeclMember.
func (DeclField) IsConstantMember() bool {
	return false
}

func (e DeclField) DeclName() Identifier {
	return e.Name
}

func (e DeclField) DeclOverview() string {
	if len(e.Parameters) == 0 {
		return string(e.Name.Value)
	}
	paramNames := make([]string, len(e.Parameters))
	for i, param := range e.Parameters {
		paramNames[i] = string(param.Name.Value)
	}
	return fmt.Sprintf("%s %s", e.Name, strings.Join(paramNames, ", "))
}

func (e DeclField) ExportScope() ExportScope {
	if e.Name.Value[0] == '_' {
		return ExportScopePublic
	}
	return ExportScopeInternal
}

func MakeDeclField(name Identifier, params []DeclParameter, annotations AnnotationChain) *DeclField {
	return &DeclField{
		Name:        name,
		Parameters:  params,
		Annotations: annotations,
		Docs:        MakeDocs([]string{}),
	}
}

func (decl DeclField) ProvidedDocs() *Docs {
	return decl.Docs
}

// EnumerateChildNodes implements Decl.
func (f DeclField) EnumerateChildNodes(action func(child Node)) {
	action(f.Name)
	for _, node := range f.Parameters {
		action(node)
		node.EnumerateChildNodes(action)
	}
}
