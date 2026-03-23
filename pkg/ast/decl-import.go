package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Decl = DeclImport{}
var _ Overviewable = DeclImport{}

type DeclImport struct {
	Token      token.Token
	Alias      Identifier
	ModuleName ModuleName
	Members    []DeclImportMember
}

// TokenLiteral implements Node
func (d DeclImport) TokenLiteral() token.Token {
	return d.Token
}

// statementNode implements Statement
func (d DeclImport) statementNode() {}

// declarationNode implements Declaration
func (d DeclImport) declarationNode() {}

func (e DeclImport) DeclName() Identifier {
	return e.Alias
}

func (e DeclImport) DeclOverview() string {
	if e.Alias.Value != "" {
		return fmt.Sprintf("import %s = %s", e.Alias.Value, e.ModuleName)
	} else {
		return fmt.Sprintf("import %s", e.ModuleName)
	}
}

func (e DeclImport) ExportScope() ExportScope {
	return ExportScopeLocal
}

func (e *DeclImport) AddMember(member DeclImportMember) {
	e.Members = append(e.Members, member)
}

func MakeDeclImport(tok token.Token, name StaticReference) *DeclImport {
	alias := Identifier(name[len(name)-1])
	return &DeclImport{
		Token:      tok,
		Alias:      alias,
		ModuleName: ModuleName(name),
		Members:    make([]DeclImportMember, 0),
	}
}

func MakeDeclAliasImport(tok token.Token, alias Identifier, name StaticReference) *DeclImport {
	return &DeclImport{
		Token:      tok,
		Alias:      alias,
		ModuleName: ModuleName(name),
		Members:    make([]DeclImportMember, 0),
	}
}

func (n DeclImport) EnumerateChildNodes(action func(child Node)) {
	action(n.Alias)
	for _, node := range n.Members {
		action(node)
		node.EnumerateChildNodes(action)
	}
}
