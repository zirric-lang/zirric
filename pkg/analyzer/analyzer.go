package analyzer

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/resolver"
)

type Analyzer struct {
	resolver resolver.ModuleResolver
	analyzed map[*ast.ContextModule]struct{}

	moduleGlobals map[registry.LogicalURI]int
	nextGlobal    int
	nextConstant  int
}

func New(resolver resolver.ModuleResolver) *Analyzer {
	return &Analyzer{
		resolver:      resolver,
		analyzed:      map[*ast.ContextModule]struct{}{},
		moduleGlobals: map[registry.LogicalURI]int{},
		nextGlobal:    ModuleGlobalIDMain,
	}
}

func (a *Analyzer) Analyze(module *ast.ContextModule, reserveModule bool) ([]AnalysisError, *ast.ContextModule) {
	if module == nil {
		return []AnalysisError{{
			Summary: "analysis error",
			Details: "module is nil",
		}}, nil
	}
	if _, ok := a.analyzed[module]; ok {
		return collectSymbolErrors(module.Symbols), module
	}
	a.analyzed[module] = struct{}{}
	a.populateFieldDecls(module)
	a.populateFunctionParams(module)
	a.buildSymbolTables(module)
	a.resolveIdentifiers(module)
	a.assignLocalIDs(module)
	a.assignModuleIDs(module, reserveModule)
	errs := a.validateStaticRefs(module)
	errs = append(errs, collectSymbolErrors(module.Symbols)...)
	return errs, module
}

func (a *Analyzer) assignModuleIDs(module *ast.ContextModule, reserveModule bool) {
	moduleGlobal := -1
	if reserveModule {
		moduleGlobal = a.reserveModuleGlobal(module.Name)
	}

	for _, sym := range module.Symbols.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		sym = sym.Original()
		if sym.Decl == nil {
			continue
		}
		switch decl := sym.Decl.(type) {
		case *ast.DeclFunc, *ast.DeclData, *ast.DeclUnion, *ast.DeclExternFunc, *ast.DeclExternType, *ast.DeclAnnotation:
			if sym.ConstantId == nil {
				id := a.nextConstant
				a.nextConstant++
				sym.ConstantId = &id
			}
		case *ast.DeclVariable:
			switch decl.ExportScope() {
			case ast.ExportScopeInternal, ast.ExportScopePublic:
				if sym.GlobalId == nil {
					id := a.nextGlobal
					a.nextGlobal++
					sym.GlobalId = &id
				}
			}
		case *ast.DeclModule:
			if sym.GlobalId == nil && moduleGlobal >= 0 {
				id := moduleGlobal
				sym.GlobalId = &id
			}
		case *ast.DeclImport:
			if sym.GlobalId == nil {
				uri := decl.ModuleName.URI()
				id := a.reserveModuleGlobal(uri)
				sym.GlobalId = &id
			}
		}
	}

	for _, file := range module.Files {
		if file == nil || file.Symbols == nil {
			continue
		}
		for _, sym := range file.Symbols.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}
			sym = sym.Original()
			if sym.Decl == nil {
				continue
			}
			switch decl := sym.Decl.(type) {
			case *ast.DeclImport:
				if sym.GlobalId == nil {
					uri := decl.ModuleName.URI()
					id := a.reserveModuleGlobal(uri)
					sym.GlobalId = &id
				}
			case *ast.DeclModule:
				if sym.GlobalId == nil && moduleGlobal >= 0 {
					id := moduleGlobal
					sym.GlobalId = &id
				}
			}
		}
	}
}

func (a *Analyzer) reserveModuleGlobal(name registry.LogicalURI) int {
	if idx, ok := a.moduleGlobals[name]; ok {
		return idx
	}
	id := a.nextGlobal
	a.nextGlobal++
	a.moduleGlobals[name] = id
	return id
}
