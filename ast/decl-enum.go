package ast

import (
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/token"
)

var _ Decl = DeclEnum{}
var _ Overviewable = DeclEnum{}

type DeclEnum struct {
	Token       token.Token
	Name        Identifier
	Cases       []*DeclEnumCase
	Annotations AnnotationChain

	Docs *Docs
}

// TokenLiteral implements Node
func (d DeclEnum) TokenLiteral() token.Token {
	return d.Token
}

// statementNode implements Statement
func (d DeclEnum) statementNode() {}

// declarationNode implements Declaration
func (d DeclEnum) declarationNode() {}

func (e DeclEnum) DeclName() Identifier {
	return e.Name
}

func (e DeclEnum) DeclOverview() string {
	if len(e.Cases) == 0 {
		return fmt.Sprintf("enum %s", e.Name)
	}
	caseLines := make([]string, 0)
	for _, cs := range e.Cases {
		caseLines = append(caseLines, "    "+cs.Case.String())
	}
	return fmt.Sprintf("enum %s {\n%s\n}", e.Name, strings.Join(caseLines, "\n"))
}

func (e DeclEnum) ExportScope() ExportScope {
	if e.Name.Value[0] == '_' {
		return ExportScopePublic
	}
	return ExportScopeInternal
}

func MakeDeclEnum(tok token.Token, name Identifier) *DeclEnum {
	return &DeclEnum{
		Token: tok,
		Name:  name,
		Cases: []*DeclEnumCase{},
		Docs:  MakeDocs([]string{}),
	}
}

func (e *DeclEnum) AddCase(case_ *DeclEnumCase) {
	e.Cases = append(e.Cases, case_)
}

func (e DeclEnum) String() string {
	declarationClause := fmt.Sprintf("enum %s", e.Name)
	if len(e.Cases) == 0 {
		return declarationClause
	}
	declarationClause += " { "
	for _, caseDecl := range e.Cases {
		declarationClause += caseDecl.Case.String() + "; "
	}
	return declarationClause + "}"
}

func (decl DeclEnum) ProvidedDocs() *Docs {
	return decl.Docs
}

// EnumerateChildNodes implements Decl.
func (n DeclEnum) EnumerateChildNodes(action func(child Node)) {
	if len(n.Annotations) > 0 {
		action(n.Annotations)
		n.Annotations.EnumerateChildNodes(action)
	}
	action(n.Name)
	for _, node := range n.Cases {
		action(node)
	}
}
