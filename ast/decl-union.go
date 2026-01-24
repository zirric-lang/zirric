package ast

import (
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/token"
)

var _ Decl = DeclUnion{}
var _ Overviewable = DeclUnion{}

type DeclUnion struct {
	Token       token.Token
	Name        Identifier
	Members     []*DeclUnionMember
	Annotations AnnotationChain

	Docs *Docs
}

// TokenLiteral implements Node
func (d DeclUnion) TokenLiteral() token.Token {
	return d.Token
}

// statementNode implements Statement
func (d DeclUnion) statementNode() {}

// declarationNode implements Declaration
func (d DeclUnion) declarationNode() {}

func (e DeclUnion) DeclName() Identifier {
	return e.Name
}

func (e DeclUnion) DeclOverview() string {
	if len(e.Members) == 0 {
		return fmt.Sprintf("union %s", e.Name)
	}
	memberLines := make([]string, 0)
	for _, cs := range e.Members {
		memberLines = append(memberLines, "    "+cs.Member.String())
	}
	return fmt.Sprintf("union %s {\n%s\n}", e.Name, strings.Join(memberLines, "\n"))
}

func (e DeclUnion) ExportScope() ExportScope {
	if e.Name.Value[0] == '_' {
		return ExportScopePublic
	}
	return ExportScopeInternal
}

func MakeDeclUnion(tok token.Token, name Identifier) *DeclUnion {
	return &DeclUnion{
		Token:   tok,
		Name:    name,
		Members: []*DeclUnionMember{},
		Docs:    MakeDocs([]string{}),
	}
}

func (e *DeclUnion) AddMember(member *DeclUnionMember) {
	e.Members = append(e.Members, member)
}

func (e DeclUnion) String() string {
	declarationClause := fmt.Sprintf("union %s", e.Name)
	if len(e.Members) == 0 {
		return declarationClause
	}
	declarationClause += " { "
	for _, memberDecl := range e.Members {
		declarationClause += memberDecl.Member.String() + "; "
	}
	return declarationClause + "}"
}

func (decl DeclUnion) ProvidedDocs() *Docs {
	return decl.Docs
}

// EnumerateChildNodes implements Decl.
func (n DeclUnion) EnumerateChildNodes(action func(child Node)) {
	if len(n.Annotations) > 0 {
		action(n.Annotations)
		n.Annotations.EnumerateChildNodes(action)
	}
	action(n.Name)
	for _, node := range n.Members {
		action(node)
	}
}
