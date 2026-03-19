package parser

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
)

type ModuleParser struct {
	module        registry.ResolvedModule
	contextModule *ast.ContextModule

	srcp         []*Parser
	sourceErrors []ParseError
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
		return mp.contextModule, err
	}

	for _, src := range sources {
		lex, err := lexer.New(src)
		if err != nil {
			mp.sourceErrors = append(mp.sourceErrors, ParseError{
				Summary: "failed to read source",
				Details: err.Error(),
			})
			continue
		}
		prs := NewSourceParser(lex, mp.contextModule.Decls, string(src.URI()))
		mp.srcp = append(mp.srcp, prs)
	}

	for _, prs := range mp.srcp {
		tree := prs.ParseSourceFile()
		mp.contextModule.AddSourceFile(tree)
	}
	return mp.contextModule, nil
}

func (mp *ModuleParser) Errors() []ParseError {
	errs := append([]ParseError{}, mp.sourceErrors...)
	for _, prs := range mp.srcp {
		errs = append(errs, prs.Errors()...)
	}
	return errs
}

func (mp *ModuleParser) Decls() *ast.DeclTable {
	return mp.contextModule.Decls
}
