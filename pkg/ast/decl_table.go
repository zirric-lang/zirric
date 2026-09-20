package ast

import (
	"errors"
	"fmt"
	"sync"
)

type DeclSymbol struct {
	Name       string
	Decl       Decl
	ChildTable *DeclTable
	Errs       []error
}

type DeclTable struct {
	Parent           *DeclTable
	OpenedBy         Node
	Symbols          map[string]*DeclSymbol
	Resolved         *SymbolTable
	mu               sync.RWMutex
	exportScopeLevel ExportScope
	functionCounter  int
}

func MakeModuleDeclTable(module *ContextModule) *DeclTable {
	return &DeclTable{
		Parent:           nil,
		OpenedBy:         module,
		Symbols:          map[string]*DeclSymbol{},
		exportScopeLevel: ExportScopePublic,
	}
}

func (parent *DeclTable) MakeChild(declaringNode Node) *DeclTable {
	if parent == nil {
		panic("parent decl table cannot be nil")
	}
	return &DeclTable{
		Parent:   parent,
		OpenedBy: declaringNode,
		Symbols:  map[string]*DeclSymbol{},
	}
}

func (dt *DeclTable) Name() string {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	var prefix string
	if dt.Parent != nil {
		prefix = dt.Parent.Name() + "->"
	}
	var name string
	switch n := dt.OpenedBy.(type) {
	case Decl:
		name = n.DeclName().String()
	case ExprFunc:
		name = n.Name
	case Identifier:
		name = n.Value
	case *SourceFile:
		name = n.Path
	default:
		name = fmt.Sprintf("%T", dt.OpenedBy)
	}
	return prefix + name
}

func (dt *DeclTable) Module() *ContextModule {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	if module, ok := dt.OpenedBy.(*ContextModule); ok {
		return module
	}
	if dt.Parent != nil {
		return dt.Parent.Module()
	}
	panic("invariant error: no main module defined in " + dt.Name())
}

func (dt *DeclTable) ExportScopeLevel() ExportScope {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.exportScopeLevel
}

func (dt *DeclTable) SetExportScopeLevel(scope ExportScope) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.exportScopeLevel = scope
}

func (dt *DeclTable) Insert(decl Decl) *DeclSymbol {
	return dt.insert(decl, false)
}

// InsertForBinding inserts a `for x <- …` binding, reusing an existing binding of the same name in the same scope rather than reporting a redeclaration.
// A statement-form for body shares the enclosing scope, so two sequential loops binding the same name would otherwise collide even though neither can observe the other's value.
// The caller is responsible for rejecting a loop nested inside another that binds the same name, which is a genuine shadowing conflict.
func (dt *DeclTable) InsertForBinding(decl Decl) *DeclSymbol {
	return dt.insert(decl, true)
}

func (dt *DeclTable) insert(decl Decl, reuseForBinding bool) *DeclSymbol {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	scope := decl.ExportScope()
	if dt.exportScopeLevel >= scope && dt.Parent != nil {
		if _, ok := dt.OpenedBy.(*ContextModule); !ok {
			sym := dt.Parent.insert(decl, reuseForBinding)
			usageSymbol, ok := dt.Symbols[decl.DeclName().Value]
			if !ok {
				return sym
			}
			sym.Errs = append(sym.Errs, usageSymbol.Errs...)
			return sym
		}
	}

	name := decl.DeclName().Value
	if sym, ok := dt.Symbols[name]; ok {
		if _, isForBinding := sym.Decl.(*DeclForBinding); reuseForBinding && isForBinding {
			return sym
		}
		sym.Errs = append(sym.Errs, errors.New("symbol already defined"))
		return sym
	}
	sym := &DeclSymbol{
		Name: decl.DeclName().Value,
		Decl: decl,
	}
	dt.Symbols[name] = sym
	return sym
}

func (dt *DeclTable) resolve(name string) (*DeclSymbol, bool) {
	if dt == nil {
		return nil, false
	}
	if sym, ok := dt.Symbols[name]; ok {
		return sym, true
	}
	if dt.Parent == nil {
		return nil, false
	}
	return dt.Parent.resolve(name)
}

// Resolve looks up a symbol by name, walking the parent chain.
func (dt *DeclTable) Resolve(name string) (*DeclSymbol, bool) {
	return dt.resolve(name)
}

func (dt *DeclTable) NextAnonymousFunctionName() string {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.functionCounter++
	return fmt.Sprintf("func#%d", dt.functionCounter)
}
