package parser

import (
	"code.knabel.dev/zirric-lang/zirric/ast"
	"code.knabel.dev/zirric-lang/zirric/lexer"
	"code.knabel.dev/zirric-lang/zirric/registry"
)

type ModuleParser struct {
	module        registry.ResolvedModule
	contextModule *ast.ContextModule

	srcp []*Parser
}

func NewModuleParse(module registry.ResolvedModule) *ModuleParser {
	return &ModuleParser{
		module:        module,
		contextModule: ast.MakeContextModule(module.URI()),
		srcp:          []*Parser{},
	}
}

func (mp *ModuleParser) Parse(module registry.ResolvedModule) (*ast.ContextModule, error) {
	sources, err := module.Sources()
	if err != nil {
		return nil, err
	}

	for _, src := range sources {
		lex, err := lexer.New(src)
		if err != nil {
			return nil, err
		}
		prs := NewSourceParser(lex, mp.contextModule.Symbols, string(src.URI()))
		mp.srcp = append(mp.srcp, prs)
	}

	for _, prs := range mp.srcp {
		tree := prs.ParseSourceFile()
		mp.contextModule.AddSourceFile(tree)
	}
	return mp.contextModule, nil
}

func (mp *ModuleParser) Errors() []ParseError {
	var errs []ParseError
	for _, prs := range mp.srcp {
		errs = append(errs, prs.Errors()...)
	}
	return errs
}

func (mp *ModuleParser) Symbols() *ast.SymbolTable {
	return mp.contextModule.Symbols
}
