package ast

import (
	"errors"
	"fmt"
	"sync"
)

var (
	errSymbolAlreadyDefinedInSameScope = errors.New("symbol already defined")
)

type SymbolScope string

const (
	GlobalScope   SymbolScope = "Global"
	LocalScope    SymbolScope = "Local"
	FreeScope     SymbolScope = "Free"
	FunctionScope SymbolScope = "Function"
)

type Symbol struct {
	Name   string
	Scope  SymbolScope
	Index  int
	Decl   Decl
	Parent *Symbol

	Usages     []SymbolUsage
	ChildTable *SymbolTable
	Errs       []error

	// Filled by later phases

	ConstantId *int
	GlobalId   *int
	LocalId    *int
	TypeSymbol *Symbol
}

func (sym *Symbol) Original() *Symbol {
	if sym.Parent != nil {
		return sym.Parent.Original()
	}
	return sym
}

type SymbolUsage struct {
	Node             Node
	typeRequirements []SymbolRequirement
	Errs             []error
}

type SymbolRequirement interface{}
type RequireAnnotation *DeclAnnotationInstance
type RequireStaticRef struct {
	StaticReference
	ResolveRequirements SymbolRequirement
}

type SymbolTable struct {
	Parent      *SymbolTable
	OpenedBy    Node
	Symbols     map[string]*Symbol
	FreeSymbols []*Symbol

	symbolCounter    int
	functionCounter  int
	exportScopeLevel ExportScope
	mu               sync.RWMutex
}

func MakeModuleSymbolTable(module *ContextModule) *SymbolTable {
	return &SymbolTable{
		Parent:           nil,
		OpenedBy:         module,
		Symbols:          map[string]*Symbol{},
		exportScopeLevel: ExportScopePublic,
	}
}

func (parent *SymbolTable) MakeChild(declaringNode Node) *SymbolTable {
	if parent == nil {
		panic("parent symbol table cannot be nil")
	}
	return &SymbolTable{
		Parent:   parent,
		OpenedBy: declaringNode,
		Symbols:  map[string]*Symbol{},
	}
}

func (st *SymbolTable) Name() string {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var prefix string
	if st.Parent != nil {
		prefix = st.Parent.Name() + "->"
	}
	var name string
	switch n := st.OpenedBy.(type) {
	case Decl:
		name = n.DeclName().String()
	case ExprFunc:
		name = n.Name
	case Identifier:
		name = n.Value
	case *SourceFile:
		name = n.Path
	default:
		name = fmt.Sprintf("%T", st.OpenedBy)
	}
	return prefix + name
}

func (st *SymbolTable) Module() *ContextModule {
	st.mu.RLock()
	defer st.mu.RUnlock()

	if module, ok := st.OpenedBy.(*ContextModule); ok {
		return module
	}

	if st.Parent != nil {
		return st.Parent.Module()
	}

	panic("invariant error: no main module defined in " + st.Name())
}

func (st *SymbolTable) ExportScopeLevel() ExportScope {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.exportScopeLevel
}

func (st *SymbolTable) SetExportScopeLevel(scope ExportScope) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.exportScopeLevel = scope
}

func (st *SymbolTable) Insert(decl Decl) *Symbol {
	st.mu.Lock()
	defer st.mu.Unlock()

	scope := decl.ExportScope()
	if st.exportScopeLevel >= scope && st.Parent != nil {
		sym := st.Parent.Insert(decl)
		usageSymbol, ok := st.Symbols[decl.DeclName().Value]
		if !ok {
			return sym
		}
		sym.Errs = append(sym.Errs, usageSymbol.Errs...)
		sym.Usages = append(sym.Usages, usageSymbol.Usages...)
		return sym
	}
	name := decl.DeclName().Value
	if sym, ok := st.Symbols[name]; ok {
		sym.Errs = append(sym.Errs, errSymbolAlreadyDefinedInSameScope)
		sym.Usages = append(sym.Usages, SymbolUsage{
			Node:             decl,
			typeRequirements: nil,
			Errs:             []error{errSymbolAlreadyDefinedInSameScope},
		})
		return sym
	}
	sym := &Symbol{
		Name:  decl.DeclName().Value,
		Decl:  decl,
		Index: st.symbolCounter,
	}
	st.Symbols[name] = sym
	st.symbolCounter++
	return sym
}

func (st *SymbolTable) addSymbol(symbol Symbol) *Symbol {
	if symbol.Decl != nil && st.exportScopeLevel >= symbol.Decl.ExportScope() {
		if st.Parent != nil {
			st.Parent.mu.Lock()
			defer st.Parent.mu.Unlock()
			return st.Parent.addSymbol(symbol)
		}
		// If Parent is nil, fall through to add to current table
	}
	ref := &symbol
	st.Symbols[symbol.Name] = ref
	return ref
}

func (st *SymbolTable) resolve(name string) (*Symbol, bool) {
	if st == nil {
		return nil, false
	}

	if sym, ok := st.Symbols[name]; ok {
		return sym, true
	}

	if st.Parent == nil {
		return nil, false
	}

	st.Parent.mu.Lock()
	defer st.Parent.mu.Unlock()

	if sym, ok := st.Parent.resolve(name); ok {
		return st.defineFree(sym), true
	}
	return nil, false
}

func (st *SymbolTable) defineFree(sym *Symbol) *Symbol {
	idx := len(st.FreeSymbols)
	st.FreeSymbols = append(st.FreeSymbols, sym)
	free := &Symbol{
		Name:       sym.Name,
		Scope:      FreeScope,
		Index:      idx,
		Decl:       sym.Decl,
		Usages:     nil,
		ChildTable: sym.ChildTable,
		Errs:       sym.Errs,
		ConstantId: sym.ConstantId,
		TypeSymbol: sym.TypeSymbol,
		Parent:     sym,
	}
	st.Symbols[sym.Name] = free
	return free
}

func (st *SymbolTable) Lookup(name string, fromNode Node, requirements ...SymbolRequirement) *Symbol {
	st.mu.Lock()
	defer st.mu.Unlock()

	usage := SymbolUsage{
		Node:             fromNode,
		typeRequirements: requirements,
	}
	if sym, ok := st.resolve(name); ok {
		sym.Usages = append(sym.Usages, usage)
		return sym
	}
	return st.addSymbol(
		Symbol{
			Name:   name,
			Decl:   nil,
			Usages: []SymbolUsage{usage},
		},
	)
}

func (st *SymbolTable) LookupIdentifier(name Identifier, requirements ...SymbolRequirement) *Symbol {
	st.mu.Lock()
	defer st.mu.Unlock()

	usage := SymbolUsage{
		Node:             name,
		typeRequirements: requirements,
	}
	if sym, ok := st.resolve(name.Value); ok {
		sym.Usages = append(sym.Usages, usage)
		return sym
	}
	return st.addSymbol(
		Symbol{
			Name:   name.Value,
			Decl:   nil,
			Usages: []SymbolUsage{usage},
		},
	)
}

func (st *SymbolTable) LookupRef(ref StaticReference, requirements ...SymbolRequirement) *Symbol {
	if len(ref) == 1 {
		return st.LookupIdentifier(ref[0], requirements...)
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	name := ref[0]
	usage := SymbolUsage{
		Node:             name,
		typeRequirements: append(requirements, RequireStaticRef{ref[1:], requirements}),
	}
	if sym, ok := st.resolve(name.Value); ok {
		if sym.ChildTable != nil {
			return sym.ChildTable.LookupRef(ref[1:])
		}
		if sym.Decl != nil {
			if _, ok := sym.Decl.(*DeclImport); ok {
				sym.Usages = append(sym.Usages, usage)
				return sym
			}
			usage.Errs = append(usage.Errs, fmt.Errorf("expected to have member %s", ref[1:]))
		}
		sym.Usages = append(sym.Usages, usage)
		return sym
	}
	return st.addSymbol(
		Symbol{
			Name:   name.Value,
			Decl:   nil,
			Usages: []SymbolUsage{usage},
		},
	)
}

func (st *SymbolTable) NextAnonymousFunctionName() string {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.functionCounter++
	return fmt.Sprintf("func#%d", st.functionCounter)
}
