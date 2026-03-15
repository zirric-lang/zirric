package ast

import (
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

var _ Decl = DeclFunc{}
var _ Overviewable = DeclFunc{}

type DeclFunc struct {
	Token      token.Token
	Name       Identifier
	Impl       *ExprFunc
	Attributes AttributeChain

	Docs *Docs
}

// TokenLiteral implements Decl.
func (d DeclFunc) TokenLiteral() token.Token {
	return d.Token
}

// declarationNode implements Decl.
func (DeclFunc) declarationNode() {}

// statementNode implements Statement.
func (DeclFunc) statementNode() {}

func (e DeclFunc) DeclName() Identifier {
	return e.Name
}

func (e DeclFunc) DeclOverview() string {
	if len(e.Impl.Parameters) == 0 {
		return fmt.Sprintf("fn %s { -> }", e.Name)
	}
	paramNames := make([]string, len(e.Impl.Parameters))
	for i, param := range e.Impl.Parameters {
		paramNames[i] = string(param.Name.Value)
	}
	return fmt.Sprintf("fn %s { %s -> }", e.Name, strings.Join(paramNames, ", "))
}

func (e DeclFunc) ExportScope() ExportScope {
	if e.Name.Value[0] == '_' {
		return ExportScopeInternal
	}
	return ExportScopePublic
}

func MakeDeclFunc(tok token.Token, name Identifier, impl *ExprFunc) *DeclFunc {
	return &DeclFunc{
		Token: tok,
		Name:  name,
		Impl:  impl,
	}
}

func (decl DeclFunc) ProvidedDocs() *Docs {
	return decl.Docs
}

// EnumerateChildNodes implements Decl.
func (n DeclFunc) EnumerateChildNodes(action func(child Node)) {
	if len(n.Attributes) > 0 {
		action(n.Attributes)
		n.Attributes.EnumerateChildNodes(action)
	}
	action(n.Name)
	action(n.Impl)
	n.Impl.EnumerateChildNodes(action)
}
