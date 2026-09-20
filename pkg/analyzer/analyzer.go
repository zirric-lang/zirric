package analyzer

import (
	"context"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/resolver"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
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

// allocateTypeConstantId allocates a constant ID for a type declaration,
// ensuring the ID is >= runtime.NumBuiltinTypeIds. This prevents user-defined
// types from receiving an ID that collides with a hardcoded builtin TypeId
// (e.g. typeIdArray = 0), which would cause IsType checks to incorrectly
// match values of the builtin type against the user-defined type.
func (a *Analyzer) allocateTypeConstantId() int {
	if a.nextConstant < runtime.NumBuiltinTypeIds {
		a.nextConstant = runtime.NumBuiltinTypeIds
	}
	return a.AllocateConstantId()
}

// AllocateGlobalId returns the next available global ID and advances the counter.
// Both the analyzer (for module-level symbols) and the compiler (for attribute
// instances and other dynamic globals) must allocate through this single counter
// to avoid collisions.
func (a *Analyzer) AllocateGlobalId() int {
	id := a.nextGlobal
	a.nextGlobal++
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
		case *ast.DeclExternType:
			if sym.ConstantId == nil {
				// Some builtin types have hard-coded IDs
				if tid, ok := runtime.BuiltinTypeIds[sym.Name]; ok {
					id := int(tid)
					sym.ConstantId = &id
					if a.nextConstant <= id {
						a.nextConstant = id + 1
					}
				} else {
					id := a.allocateTypeConstantId()
					sym.ConstantId = &id
				}
			}
		case *ast.DeclFunc, *ast.DeclData, *ast.DeclUnion, *ast.DeclExternFunc, *ast.DeclExternValue, *ast.DeclAttr:
			if sym.ConstantId == nil {
				id := a.allocateTypeConstantId()
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

// ReserveModuleGlobal exposes module global reservation so the compiler can assign slots to modules no import reaches.
// It must be used instead of AllocateGlobalId so that a module later reached by an import dedups onto the same slot.
func (a *Analyzer) ReserveModuleGlobal(name registry.LogicalURI) int {
	return a.reserveModuleGlobal(name)
}

func (a *Analyzer) reserveModuleGlobal(name registry.LogicalURI) int {
	name = a.canonicalModuleURI(name)
	if idx, ok := a.moduleGlobals[name]; ok {
		return idx
	}
	id := a.nextGlobal
	a.nextGlobal++
	a.moduleGlobals[name] = id
	return id
}

// canonicalModuleURI resolves name to the URI a module is actually named (ContextModule.Name).
// A local project module can be imported under a short, unqualified form (e.g. "tests") that differs from the canonical form its own "mod X" self-declaration is analyzed under (e.g. "zirric.tests") — without this, reserveModuleGlobal would allocate two distinct, never-reconciled global slots for the same module, one of which is never filled by the compiler.
func (a *Analyzer) canonicalModuleURI(name registry.LogicalURI) registry.LogicalURI {
	if a.resolver == nil {
		return name
	}
	resolved, err := a.resolver.ResolveModule(context.Background(), name)
	if err != nil || resolved == nil {
		return name
	}
	return resolved.Name
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
