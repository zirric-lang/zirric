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

// AnalyzerSnapshot captures the ID counter state of an Analyzer for rollback.
// Use Snapshot before an incremental operation and Restore to undo its allocations.
type AnalyzerSnapshot struct {
	nextGlobal    int
	nextConstant  int
	moduleGlobals map[registry.LogicalURI]int
}

// NextGlobal returns the snapshotted nextGlobal counter value.
func (s AnalyzerSnapshot) NextGlobal() int { return s.nextGlobal }

// NextConstant returns the snapshotted nextConstant counter value.
func (s AnalyzerSnapshot) NextConstant() int { return s.nextConstant }

// Snapshot captures the current ID counter state.
func (a *Analyzer) Snapshot() AnalyzerSnapshot {
	snap := AnalyzerSnapshot{
		nextGlobal:    a.nextGlobal,
		nextConstant:  a.nextConstant,
		moduleGlobals: make(map[registry.LogicalURI]int, len(a.moduleGlobals)),
	}
	for k, v := range a.moduleGlobals {
		snap.moduleGlobals[k] = v
	}
	return snap
}

// Restore resets the ID counters to a previously captured snapshot,
// freeing any IDs allocated since the snapshot was taken.
func (a *Analyzer) Restore(snap AnalyzerSnapshot) {
	a.nextGlobal = snap.nextGlobal
	a.nextConstant = snap.nextConstant
	a.moduleGlobals = snap.moduleGlobals
}

// Both the analyzer (for user-declared symbols) and the compiler (for internal
// constants such as literal strings) must allocate through this single counter
// to avoid collisions.
func (a *Analyzer) AllocateConstantId() int {
	id := a.nextConstant
	a.nextConstant++
	return id
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
	a.assignTypeSymbols(module)
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
		case *ast.DeclFunc, *ast.DeclData, *ast.DeclUnion, *ast.DeclExternFunc, *ast.DeclExternType, *ast.DeclExternValue, *ast.DeclAttr:
			if sym.ConstantId == nil {
				id := a.AllocateConstantId()
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
		case *ast.DeclConstant:
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
		case ast.DeclImportMember:
			if sym.GlobalId == nil {
				id := a.nextGlobal
				a.nextGlobal++
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
			case ast.DeclImportMember:
				_ = decl
				if sym.GlobalId == nil {
					id := a.nextGlobal
					a.nextGlobal++
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

// AnalyzeSourceFile incrementally analyzes a single source file against an existing module.
// The module must already be analyzed. New declarations from the file are merged into
// module.Symbols, and new IDs are assigned continuing from the analyzer's current counters.
func (a *Analyzer) AnalyzeSourceFile(module *ast.ContextModule, file *ast.SourceFile) []AnalysisError {
	if module == nil || module.Symbols == nil {
		return []AnalysisError{{Summary: "module or symbols is nil"}}
	}

	// Populate field and parameter decl tables before building symbol tables so that
	// function parameters and data fields are present when symbol tables are constructed.
	a.populateFieldDecls(module)
	a.populateFunctionParams(module)

	// Add new symbols from module.Decls to module.Symbols for cross-file resolution.
	for name, declSym := range module.Decls.Symbols {
		if declSym == nil || declSym.Decl == nil {
			continue
		}
		if _, exists := module.Symbols.Symbols[name]; exists {
			continue
		}
		sym := &ast.Symbol{
			Name: name,
			Decl: declSym.Decl,
			Errs: declSym.Errs,
		}
		if declSym.ChildTable != nil {
			sym.ChildTable = buildSymbolTableFromDeclTable(declSym.ChildTable, module.Symbols)
		}
		module.Symbols.Symbols[name] = sym
	}

	// Build the new file's symbol table.
	if file.Decls != nil && file.Decls.Resolved == nil {
		buildSymbolTableFromDeclTable(file.Decls, module.Symbols)
	}

	// Build ExprFor symbol tables for the new file.
	if file.Symbols != nil {
		file.EnumerateChildNodes(func(child ast.Node) {
			switch expr := child.(type) {
			case ast.ExprFor:
				attachExprForSymbols(expr.Body.DeclsTable)
			case *ast.ExprFor:
				attachExprForSymbols(expr.Body.DeclsTable)
			}
		})
	}
	a.resolveIdentifiers(module)
	a.assignLocalIDs(module)
	a.assignModuleIDs(module, false)
	a.assignTypeSymbols(module)

	errs := a.validateStaticRefs(module)
	errs = append(errs, collectSymbolErrors(module.Symbols)...)
	return errs
}
