package ast

import (
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Decl = DeclExternType{}
var _ Overviewable = DeclExternType{}

type DeclExternType struct {
	Token       token.Token
	Name        Identifier
	Fields      map[string]DeclField
	Annotations AnnotationChain

	Docs *Docs
}

// TokenLiteral implements Decl.
func (d DeclExternType) TokenLiteral() token.Token {
	return d.Token
}

// declarationNode implements Decl.
func (DeclExternType) declarationNode() {}

// statementNode implements StatementDeclaration.
func (DeclExternType) statementNode() {}

func (e DeclExternType) DeclName() Identifier {
	return e.Name
}

func (e DeclExternType) DeclOverview() string {
	if len(e.Fields) == 0 {
		return fmt.Sprintf("extern type %s", e.Name)
	}
	fieldLines := make([]string, 0)
	for _, field := range e.Fields {
		fieldLines = append(fieldLines, "    "+field.DeclOverview())
	}
	return fmt.Sprintf("extern type %s {\n%s\n}", e.Name, strings.Join(fieldLines, "\n"))
}

func (e DeclExternType) ExportScope() ExportScope {
	if e.Name.Value[0] == '_' {
		return ExportScopeInternal
	}
	return ExportScopePublic
}

func MakeDeclExternType(tok token.Token, name Identifier) *DeclExternType {
	return &DeclExternType{
		Token:  tok,
		Name:   name,
		Fields: make(map[string]DeclField),
		Docs:   nil,
	}
}

func (e *DeclExternType) AddField(decl DeclField) {
	e.Fields[decl.Name.Value] = decl
}

func (decl DeclExternType) ProvidedDocs() *Docs {
	return decl.Docs
}

// EnumerateChildNodes implements Decl.
func (n DeclExternType) EnumerateChildNodes(action func(child Node)) {
	action(n.Name)
	for _, node := range n.Fields {
		action(node)
	}
}
