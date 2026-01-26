package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Decl = DeclImportMember{}
var _ Overviewable = DeclImportMember{}

type DeclImportMember struct {
	Token      token.Token
	Name       Identifier
	ModuleName ModuleName
}

// TokenLiteral implements Decl.
func (d DeclImportMember) TokenLiteral() token.Token {
	return d.Token
}

// declarationNode implements Decl.
func (DeclImportMember) declarationNode() {}

func (e DeclImportMember) DeclName() Identifier {
	return e.Name
}

func (e DeclImportMember) DeclOverview() string {
	return fmt.Sprintf("import %s { %s }", e.ModuleName, e.Name)
}

func (e DeclImportMember) ExportScope() ExportScope {
	return ExportScopeLocal
}

func MakeDeclImportMember(tok token.Token, moduleName ModuleName, name Identifier) DeclImportMember {
	return DeclImportMember{
		Token:      tok,
		Name:       name,
		ModuleName: moduleName,
	}
}

// EnumerateChildNodes implements Decl.
func (n DeclImportMember) EnumerateChildNodes(action func(child Node)) {
	action(n.Name)
}
