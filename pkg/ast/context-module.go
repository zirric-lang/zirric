package ast

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

type ModuleName StaticReference

func (m ModuleName) URI() registry.LogicalURI {
	return registry.LogicalURI(StaticReference(m).String())
}

type ContextModule struct {
	Name    registry.LogicalURI
	Decls   *DeclTable
	Symbols *SymbolTable

	Files []*SourceFile
}

func MakeContextModule(name registry.LogicalURI) *ContextModule {
	m := &ContextModule{
		Name:  name,
		Files: []*SourceFile{},
	}
	m.Decls = MakeModuleDeclTable(m)
	return m
}

func (m *ContextModule) AddSourceFile(sourceFile *SourceFile) {
	m.Files = append(m.Files, sourceFile)
}

func (m *ContextModule) TokenLiteral() token.Token {
	return token.Token{
		Type:    token.MODULE_DIRECTORY,
		Literal: string(m.Name),
		Source: &token.Source{
			File:   string(m.Name),
			Offset: 0,
		},
		Leading: []token.DecorativeToken{},
	}
}

func (m *ContextModule) EnumerateChildNodes(action func(child Node)) {
	for _, src := range m.Files {
		action(src)
	}
}
