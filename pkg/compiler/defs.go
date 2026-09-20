package compiler

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/op"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/resolver"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
)

type emittedInstruction struct {
	Opcode   op.Opcode
	Position int
}

type CompilationScope struct {
	Instructions op.Instructions
	symbols      *ast.SymbolTable
	locals       []*ast.Symbol
	// freeMapping maps a FreeScope symbol's Index (position in
	// SymbolTable.FreeSymbols) to its actual position in the Closure.Free
	// array. Globals and constants are excluded from the Free array and
	// therefore have no entry.
	freeMapping map[int]int

	lastInstruction     emittedInstruction
	previousInstruction emittedInstruction
}

func (s *CompilationScope) LocalsCount() int {
	return len(s.locals)
}

type Bytecode struct {
	Instructions op.Instructions
	Constants    []runtime.RuntimeValue
	Globals      []*CompilationScope
	// MainLocals is the number of local slots required by the top-level script frame,
	// including any temporaries allocated by the compiler.
	MainLocals int
	// ModuleGlobals maps each compiled module's URI to the global slot holding its ModuleValue, so the VM can reach a module's exports by name at runtime.
	ModuleGlobals map[registry.LogicalURI]int
}

// mainPackageModules caches the project package's name and the global slot of each of its modules.
type mainPackageModules struct {
	name    string
	globals map[string]int
}

type Compiler struct {
	constants       []runtime.RuntimeValue
	globals         []*CompilationScope
	moduleGlobals   map[registry.LogicalURI]int
	compiledModules map[*ast.ContextModule]int
	mainPackage     *mainPackageModules
	// entryModule is the module passed to Compile, i.e. the program being run, which reflect.packages must not offer as a loadable package member.
	entryModule *ast.ContextModule
	plugins     *runtime.ExternPluginRegistry
	resolver    resolver.ModuleResolver
	analyzer    *analyzer.Analyzer
	analyzed    map[*ast.ContextModule]struct{}

	scopes   []*CompilationScope
	scopeIdx int
}

func New(moduleResolver resolver.ModuleResolver) *Compiler {
	return NewWithAnalyzer(moduleResolver, analyzer.New(moduleResolver))
}

func NewWithAnalyzer(moduleResolver resolver.ModuleResolver, analysis *analyzer.Analyzer) *Compiler {
	var symbols *ast.SymbolTable
	if moduleResolver != nil && moduleResolver.MainModule() != nil {
		symbols = moduleResolver.MainModule().Symbols
	}
	mainScope := &CompilationScope{
		Instructions: op.Instructions{},
		symbols:      symbols,
	}
	return &Compiler{
		constants:       []runtime.RuntimeValue{},
		globals:         []*CompilationScope{},
		moduleGlobals:   map[registry.LogicalURI]int{},
		compiledModules: map[*ast.ContextModule]int{},
		plugins:         runtime.NewExternPluginRegistry(&runtime.Prelude{}, &runtime.OSPlugin{}, &runtime.FmtPlugin{}, &runtime.BytesPlugin{}, &runtime.StringsPlugin{}, &runtime.ReflectPlugin{}, &runtime.ReflectPackagesPlugin{}, &runtime.MathPlugin{}),
		resolver:        moduleResolver,
		analyzer:        analysis,
		analyzed:        map[*ast.ContextModule]struct{}{},
		scopes:          []*CompilationScope{mainScope},
		scopeIdx:        0,
	}
}

func (c *Compiler) currentInstructions() op.Instructions {
	return c.scopes[c.scopeIdx].Instructions
}

func (c *Compiler) RegisterPlugin(plugin runtime.ExternPlugin) {
	c.plugins.Register(plugin)
}

func (c *Compiler) currentSymbols() *ast.SymbolTable {
	if c.scopes[c.scopeIdx].symbols != nil {
		return c.scopes[c.scopeIdx].symbols
	}
	if c.scopeIdx > 0 {
		return c.scopes[c.scopeIdx-1].symbols
	}
	return nil
}

// lookupTypeSymbol traverses the symbol table chain without side effects
// to find a declared type symbol by name. Returns nil if not found.
func (c *Compiler) lookupTypeSymbol(name string) *ast.Symbol {
	for st := c.currentSymbols(); st != nil; st = st.Parent {
		if sym, ok := st.Symbols[name]; ok && sym.Decl != nil {
			return sym.Original()
		}
	}
	return nil
}

func (c *Compiler) Bytecode() *Bytecode {
	moduleGlobals := make(map[registry.LogicalURI]int, len(c.moduleGlobals))
	for uri, id := range c.moduleGlobals {
		moduleGlobals[uri] = id
	}
	return &Bytecode{
		Instructions:  c.currentInstructions(),
		Constants:     c.constants,
		Globals:       c.globals,
		MainLocals:    c.scopes[0].LocalsCount(),
		ModuleGlobals: moduleGlobals,
	}
}

func (c *Compiler) emit(opcode op.Opcode, operands ...int) int {
	ins := op.Make(opcode, operands...)
	pos := c.addInstruction(ins)

	c.scopes[c.scopeIdx].previousInstruction = c.scopes[c.scopeIdx].lastInstruction
	c.scopes[c.scopeIdx].lastInstruction = emittedInstruction{
		Opcode:   opcode,
		Position: pos,
	}
	return pos
}

func (c *Compiler) addInstruction(ins []byte) int {
	newPos := len(c.currentInstructions())
	c.scopes[c.scopeIdx].Instructions = append(c.scopes[c.scopeIdx].Instructions, ins...)
	return newPos
}

func (c *Compiler) addConstant(v runtime.RuntimeValue) int {
	id := c.analyzer.AllocateConstantId()
	c.ensureConstantSlot(id)
	c.constants[id] = v
	return id
}

func (c *Compiler) ensureConstantSlot(id int) {
	for len(c.constants) <= id {
		c.constants = append(c.constants, nil)
	}
}

func (c *Compiler) ensureGlobalSlot(id int) {
	for len(c.globals) <= id {
		c.globals = append(c.globals, nil)
	}
}

func (c *Compiler) ensureLocalSlot(id int) {
	for len(c.scopes[c.scopeIdx].locals) <= id {
		c.scopes[c.scopeIdx].locals = append(c.scopes[c.scopeIdx].locals, nil)
	}
}

func (c *Compiler) ensureLocalSlotsForTable(syms *ast.SymbolTable) {
	if syms == nil {
		return
	}
	for _, sym := range syms.Symbols {
		if sym == nil || sym.Decl == nil || sym.LocalId == nil {
			continue
		}
		c.ensureLocalSlot(*sym.LocalId)
	}
}

func (c *Compiler) reserveGlobalModule(mod registry.LogicalURI) int {
	if idx, ok := c.moduleGlobals[mod]; ok {
		return idx
	}
	c.moduleGlobals[mod] = len(c.globals)
	c.globals = append(c.globals, nil) // TODO: or a placeholder?
	return len(c.globals) - 1
}

func (c *Compiler) allocateTempLocal() int {
	idx := len(c.scopes[c.scopeIdx].locals)
	c.scopes[c.scopeIdx].locals = append(c.scopes[c.scopeIdx].locals, nil)
	return idx
}

func (c *Compiler) requireLocalId(sym *ast.Symbol) (int, error) {
	if sym == nil {
		return 0, fmt.Errorf("missing symbol for local")
	}
	sym = sym.Original()
	if sym.LocalId == nil {
		return 0, fmt.Errorf("symbol %q has no local id", sym.Name)
	}
	idx := *sym.LocalId
	c.ensureLocalSlot(idx)
	c.scopes[c.scopeIdx].locals[idx] = sym
	return idx, nil
}

func (c *Compiler) enterScope(syms *ast.SymbolTable) {
	c.scopes = append(c.scopes, &CompilationScope{
		Instructions: op.Instructions{},
		symbols:      syms,
		locals:       make([]*ast.Symbol, maxLocalId(syms)+1),
	})
	c.scopeIdx++
}

func (c *Compiler) leaveScope() *CompilationScope {
	scope := c.scopes[c.scopeIdx]
	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIdx--
	return scope
}

func maxLocalId(syms *ast.SymbolTable) int {
	if syms == nil {
		return -1
	}
	maxID := -1
	for _, sym := range syms.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		if sym.LocalId != nil && *sym.LocalId > maxID {
			maxID = *sym.LocalId
		}
		if sym.ChildTable == nil {
			continue
		}
		if _, ok := sym.ChildTable.OpenedBy.(*ast.ExprFunc); ok {
			continue
		}
		if childMax := maxLocalId(sym.ChildTable); childMax > maxID {
			maxID = childMax
		}
	}
	return maxID
}

func (c *Compiler) isLastInstruction(opcodes ...op.Opcode) bool {
	if len(c.currentInstructions()) == 0 {
		return false
	}

	ins := c.scopes[c.scopeIdx].lastInstruction
	for _, opcode := range opcodes {
		if ins.Opcode == opcode {
			return true
		}
	}

	return false
}
