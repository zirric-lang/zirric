package ast

import (
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/token"
)

var _ Decl = &DeclData{}
var _ Overviewable = &DeclData{}

type DeclData struct {
	Token       token.Token
	Name        Identifier
	Fields      []DeclField
	Annotations AnnotationChain
}

func MakeDeclData(tok token.Token, name Identifier) *DeclData {
	return &DeclData{
		Token:  tok,
		Name:   name,
		Fields: []DeclField{},
	}
}

// TokenLiteral implements Node
func (d DeclData) TokenLiteral() token.Token {
	return d.Token
}

// statementNode implements Statement
func (DeclData) statementNode() {}

// declarationNode implements Statement
func (DeclData) declarationNode() {}

func (e DeclData) DeclName() Identifier {
	return e.Name
}

func (e DeclData) DeclOverview() string {
	if len(e.Fields) == 0 {
		return fmt.Sprintf("data %s", e.Name)
	}
	fieldLines := make([]string, 0)
	for _, field := range e.Fields {
		fieldLines = append(fieldLines, "    "+field.DeclOverview())
	}
	return fmt.Sprintf("data %s {\n%s\n}", e.Name, strings.Join(fieldLines, "\n"))
}

func (e DeclData) ExportScope() ExportScope {
	if e.Name.Value[0] == '_' {
		return ExportScopePublic
	}
	return ExportScopeInternal
}

func (e *DeclData) AddField(field DeclField) {
	e.Fields = append(e.Fields, field)
}

// EnumerateChildNodes implements Decl.
func (d DeclData) EnumerateChildNodes(action func(child Node)) {
	if len(d.Annotations) > 0 {
		action(d.Annotations)
		d.Annotations.EnumerateChildNodes(action)
	}
	action(d.Name)
	for _, node := range d.Fields {
		action(node)
	}
}
