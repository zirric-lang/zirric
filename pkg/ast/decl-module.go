package ast

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Decl = DeclModule{}
var _ Overviewable = DeclModule{}

type DeclModule struct {
	Token token.Token
	// Name is the alias of `mod local = a.b.c`, otherwise the last segment of Path.
	Name Identifier
	// Path is the fully qualified module path.
	Path ModuleName
	// HasAlias records that the source wrote `mod local = a.b.c`, so it can be shown back that way.
	HasAlias   bool
	Attributes AttributeChain

	Docs *Docs
}

// TokenLiteral implements Node
func (d DeclModule) TokenLiteral() token.Token {
	return d.Token
}

// statementNode implements Statement
func (d DeclModule) statementNode() {}

// declarationNode implements Declaration
func (d DeclModule) declarationNode() {}

func (e DeclModule) DeclName() Identifier {
	return e.Name
}

func (e DeclModule) DeclOverview() string {
	if e.HasAlias {
		return fmt.Sprintf("mod %s = %s", e.Name.Value, e.Path)
	}
	return fmt.Sprintf("mod %s", e.Path)
}

func (e DeclModule) ExportScope() ExportScope {
	return ExportScopeLocal
}

// MakeDeclModule binds the path's last segment locally.
func MakeDeclModule(tok token.Token, path StaticReference) *DeclModule {
	return &DeclModule{Token: tok, Name: path[len(path)-1], Path: ModuleName(path)}
}

// MakeDeclAliasModule binds the path to a local name of its own.
func MakeDeclAliasModule(tok token.Token, alias Identifier, path StaticReference) *DeclModule {
	return &DeclModule{Token: tok, Name: alias, Path: ModuleName(path), HasAlias: true}
}

func (decl DeclModule) ProvidedDocs() *Docs {
	return decl.Docs
}

// SetDocs implements Documentable.
func (decl *DeclModule) SetDocs(docs *Docs) {
	decl.Docs = docs
}

// EnumerateChildNodes implements Decl.
func (n DeclModule) EnumerateChildNodes(action func(child Node)) {
	action(n.Name)
	if len(n.Attributes) > 0 {
		action(n.Attributes)
	}
}
