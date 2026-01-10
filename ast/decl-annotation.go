package ast

import (
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/token"
)

var _ Decl = &DeclAnnotation{}
var _ Overviewable = &DeclData{}

type DeclAnnotation struct {
	Token       token.Token
	Name        Identifier
	Fields      []DeclField
	Annotations AnnotationChain

	Docs *Docs
}

// TokenLiteral implements Node
func (d DeclAnnotation) TokenLiteral() token.Token {
	return d.Token
}

// statementNode implements Statement
func (DeclAnnotation) statementNode() {}

// declarationNode implements Statement
func (DeclAnnotation) declarationNode() {}

func (e DeclAnnotation) DeclName() Identifier {
	return e.Name
}

func (e DeclAnnotation) DeclOverview() string {
	if len(e.Fields) == 0 {
		return fmt.Sprintf("data %s", e.Name)
	}
	fieldLines := make([]string, 0)
	for _, field := range e.Fields {
		fieldLines = append(fieldLines, "    "+field.DeclOverview())
	}
	return fmt.Sprintf("annotation %s {\n%s\n}", e.Name, strings.Join(fieldLines, "\n"))
}

func (e DeclAnnotation) ExportScope() ExportScope {
	if e.Name.Value[0] == '_' {
		return ExportScopePublic
	}
	return ExportScopeInternal
}

func MakeDeclAnnotation(tok token.Token, name Identifier) *DeclAnnotation {
	return &DeclAnnotation{
		Token:  tok,
		Name:   name,
		Fields: []DeclField{},
		Docs:   MakeDocs([]string{}),
	}
}

func (e *DeclAnnotation) AddField(field DeclField) {
	e.Fields = append(e.Fields, field)
}

func (decl DeclAnnotation) ProvidedDocs() *Docs {
	return decl.Docs
}

// EnumerateChildNodes implements Decl.
func (d DeclAnnotation) EnumerateChildNodes(action func(child Node)) {
	action(d.Name)
	for _, node := range d.Fields {
		action(node)
		node.EnumerateChildNodes(action)
	}
}
