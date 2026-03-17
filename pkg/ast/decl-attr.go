package ast

import (
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Decl = &DeclAttr{}
var _ Overviewable = &DeclData{}

type DeclAttr struct {
	Token      token.Token
	Name       Identifier
	Fields     []DeclField
	Attributes AttributeChain

	Docs *Docs
}

// TokenLiteral implements Node
func (d DeclAttr) TokenLiteral() token.Token {
	return d.Token
}

// statementNode implements Statement
func (DeclAttr) statementNode() {}

// declarationNode implements Statement
func (DeclAttr) declarationNode() {}

func (e DeclAttr) DeclName() Identifier {
	return e.Name
}

func (e DeclAttr) DeclOverview() string {
	if len(e.Fields) == 0 {
		return fmt.Sprintf("data %s", e.Name)
	}
	fieldLines := make([]string, 0)
	for _, field := range e.Fields {
		fieldLines = append(fieldLines, "    "+field.DeclOverview())
	}
	return fmt.Sprintf("attr %s {\n%s\n}", e.Name, strings.Join(fieldLines, "\n"))
}

func (e DeclAttr) ExportScope() ExportScope {
	if e.Name.Value[0] == '_' {
		return ExportScopeInternal
	}
	return ExportScopePublic
}

func MakeDeclAttr(tok token.Token, name Identifier) *DeclAttr {
	return &DeclAttr{
		Token:  tok,
		Name:   name,
		Fields: []DeclField{},
		Docs:   MakeDocs([]string{}),
	}
}

func (e *DeclAttr) AddField(field DeclField) {
	e.Fields = append(e.Fields, field)
}

func (decl DeclAttr) ProvidedDocs() *Docs {
	return decl.Docs
}

// EnumerateChildNodes implements Decl.
func (d DeclAttr) EnumerateChildNodes(action func(child Node)) {
	action(d.Name)
	if len(d.Attributes) > 0 {
		action(d.Attributes)
	}
	for _, node := range d.Fields {
		action(node)
		node.EnumerateChildNodes(action)
	}
}
