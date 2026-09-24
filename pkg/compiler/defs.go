package compiler

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/debuginfo"
	"code.knabel.dev/zirric-lang/zirric/pkg/op"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/resolver"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

type emittedInstruction struct {
	Opcode   op.Opcode
	Position int
}

type CompilationScope struct {
	Instructions op.Instructions
	// positions records where each emitted instruction came from, kept here while the stream is still growing and handed to the debug table once it is finished.
	positions []debuginfo.Entry
	symbols   *ast.SymbolTable
	locals    []*ast.Symbol
	// freeMapping maps a FreeScope symbol's Index (position in
	// SymbolTable.FreeSymbols) to its actual position in the Closure.Free
	// array. Globals and constants are excluded from the Free array and
	// therefore have no entry.
	freeMapping map[int]int
	// freeTable is the SymbolTable freeMapping's indices belong to, kept because `symbols` is swapped for a `for` expression body.
	freeTable *ast.SymbolTable

	lastInstruction     emittedInstruction
	previousInstruction emittedInstruction
}

func (s *CompilationScope) LocalsCount() int {
	return len(s.locals)
}

// ownsFree reports whether sym is one of this scope's own captures rather than an inner table's free symbol numbered the same.
// A `for` body's free symbols start at 0 just as this frame's captures do, so the index alone cannot tell them apart.
func (s *CompilationScope) ownsFree(sym *ast.Symbol) bool {
	if s.freeTable == nil {
		return true
	}
	if sym.Index < 0 || sym.Index >= len(s.freeTable.FreeSymbols) {
		return false
	}
	// FreeSymbols holds the captured symbol, while a lookup hands back the FreeScope symbol pointing at it, so identity is one hop up.
	return s.freeTable.FreeSymbols[sym.Index] == sym.Parent
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
	// Debug maps instructions back to the source they came from, for explaining a crash. Nothing reads it while a program is working.
	Debug *debuginfo.Table
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
	// position is where the node being compiled came from, recorded against each instruction it emits.
	position *token.Source
	// debug collects those positions, so a crash can be traced back to source without anything carrying a position at runtime.
	debug *debuginfo.Table
	// mainPackageErrs holds the failures met while compiling the project's own modules, so they can be reported instead of silently dropping a module.
	mainPackageErrs []error
	// entryModule is the module passed to Compile, i.e. the program being run, which reflect.packages must not offer as a loadable package member.
	entryModule *ast.ContextModule
	// symbolAttributes remembers the attributes compiled for each declaration, so that a module can report them for a member whose runtime value cannot carry any, such as a const holding an Int.
	symbolAttributes map[*ast.Symbol]map[runtime.TypeId]int
	plugins          *runtime.ExternPluginRegistry
	resolver         resolver.ModuleResolver
	analyzer         *analyzer.Analyzer
	analyzed         map[*ast.ContextModule]struct{}

	scopes   []*CompilationScope
	scopeIdx int
	// funcDepth counts the function bodies currently being compiled, so `!.` can be rejected where returning from one is not possible.
	funcDepth int
	// optionJumps collects the short-circuits of the optional chain being compiled, all of which land at its end.
	optionJumps []pendingJump
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
		constants:        []runtime.RuntimeValue{},
		globals:          []*CompilationScope{},
		moduleGlobals:    map[registry.LogicalURI]int{},
		compiledModules:  map[*ast.ContextModule]int{},
		symbolAttributes: map[*ast.Symbol]map[runtime.TypeId]int{},
		plugins:          runtime.NewExternPluginRegistry(&runtime.Prelude{}, &runtime.OSPlugin{}, &runtime.IOPlugin{}, &runtime.FmtPlugin{}, &runtime.BytesPlugin{}, &runtime.StringsPlugin{}, &runtime.ReflectPlugin{}, &runtime.ReflectPackagesPlugin{}, &runtime.MathPlugin{}, &runtime.PathsPlugin{}, &runtime.FSPlugin{}, &runtime.RandomPlugin{}, &runtime.TimePlugin{}, &runtime.CoPlugin{}, &runtime.JSONPlugin{}, &runtime.YAMLPlugin{}),
		resolver:         moduleResolver,
		analyzer:         analysis,
		analyzed:         map[*ast.ContextModule]struct{}{},
		scopes:           []*CompilationScope{mainScope},
		scopeIdx:         0,
		debug:            debuginfo.NewTable(),
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
	c.attachPositions(c.scopes[0], "main")
	return &Bytecode{
		Instructions:  c.currentInstructions(),
		Debug:         c.debug,
		Constants:     c.constants,
		Globals:       c.globals,
		MainLocals:    c.scopes[0].LocalsCount(),
		ModuleGlobals: moduleGlobals,
	}
}

func (c *Compiler) emit(opcode op.Opcode, operands ...int) int {
	ins := op.Make(opcode, operands...)
	pos := c.addInstruction(ins)
	c.recordPosition(pos)

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

// recordPosition notes where the instruction at this offset came from, so that a crash can be traced back to it.
func (c *Compiler) recordPosition(offset int) {
	// A source with no line cannot point anywhere, and recording it would put a file name where a position belongs. A module's own token is like this, since a module is not written at any one place.
	if c.position == nil || c.position.Line <= 0 {
		return
	}
	scope := c.scopes[c.scopeIdx]
	scope.positions = append(scope.positions, debuginfo.Entry{Offset: offset, Source: c.position})
}

// attachPositions hands a finished stream to the debug table, keyed by where its bytes ended up.
func (c *Compiler) attachPositions(scope *CompilationScope, name string) {
	if c.debug == nil || scope == nil {
		return
	}
	c.debug.Attach(scope.Instructions, name, scope.positions)
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

// carryLocalsUp widens the current scope to cover the local slots of a scope whose instructions were just inlined into it.
// The inlined instructions keep their own SetLocal/GetLocal indices, so the frame that ends up running them must have at least as many slots, or the index reaches past the frame's locals at runtime.
func (c *Compiler) carryLocalsUp(inlined *CompilationScope) {
	for len(c.scopes[c.scopeIdx].locals) < len(inlined.locals) {
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
		return 0, errInvariant(nil, "a local was compiled without a resolved symbol")
	}
	sym = sym.Original()
	if sym.LocalId == nil {
		return 0, errInvariant(sym.Decl, "the local %s was never given a local slot, which the analyzer assigns before compilation", sym.Name)
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
		freeTable:    syms,
		locals:       make([]*ast.Symbol, maxLocalId(syms)+1),
	})
	c.scopeIdx++
}

func (c *Compiler) leaveScope() *CompilationScope {
	scope := c.scopes[c.scopeIdx]
	// The stream is finished here, which is the only moment its address is the one a frame will run.
	c.attachPositions(scope, scopeName(scope))
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

// scopeName is what to call a frame running a scope's instructions, taken from the function it was opened by.
func scopeName(scope *CompilationScope) string {
	if scope == nil || scope.symbols == nil {
		return ""
	}
	switch opened := scope.symbols.OpenedBy.(type) {
	case *ast.ExprFunc:
		return opened.Name
	case *ast.DeclFunc:
		return opened.Name.Value
	}
	return ""
}
