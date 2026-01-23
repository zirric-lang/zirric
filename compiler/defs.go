package compiler

import (
	"code.knabel.dev/zirric-lang/zirric/ast"
	"code.knabel.dev/zirric-lang/zirric/op"
	"code.knabel.dev/zirric-lang/zirric/runtime"
)

type emittedInstruction struct {
	Opcode   op.Opcode
	Position int
}

type CompilationScope struct {
	Instructions op.Instructions
	symbols      *ast.SymbolTable
	locals       []*ast.Symbol

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
}

type Compiler struct {
	constants []runtime.RuntimeValue
	globals   []*CompilationScope
	plugins   *runtime.ExternPluginRegistry

	scopes   []*CompilationScope
	scopeIdx int
}

func New() *Compiler {
	mainScope := &CompilationScope{
		Instructions: op.Instructions{},
		symbols:      ast.MakeSymbolTable(nil, nil),
	}
	return &Compiler{
		constants: []runtime.RuntimeValue{},
		plugins:   &runtime.ExternPluginRegistry{},
		scopes:    []*CompilationScope{mainScope},
		scopeIdx:  0,
	}
}

func (c *Compiler) currentInstructions() op.Instructions {
	return c.scopes[c.scopeIdx].Instructions
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

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.currentInstructions(),
		Constants:    c.constants,
		Globals:      c.globals,
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
	c.constants = append(c.constants, v)
	// TODO: addConstant, what about types?
	return len(c.constants) - 1
}

func (c *Compiler) addGlobal(scope *CompilationScope) int {
	c.globals = append(c.globals, scope)
	return len(c.globals) - 1
}

func (c *Compiler) allocateTempLocal() int {
	idx := len(c.scopes[c.scopeIdx].locals)
	c.scopes[c.scopeIdx].locals = append(c.scopes[c.scopeIdx].locals, nil)
	return idx
}

func (c *Compiler) ensureLocalSymbol(sym *ast.Symbol) int {
	if sym.LocalId != nil {
		idx := *sym.LocalId
		if idx < len(c.scopes[c.scopeIdx].locals) && c.scopes[c.scopeIdx].locals[idx] == sym {
			return idx
		}
	}
	idx := c.allocateTempLocal()
	sym.LocalId = &idx
	c.scopes[c.scopeIdx].locals[idx] = sym
	return idx
}

func (c *Compiler) enterScope(syms *ast.SymbolTable) {
	c.scopes = append(c.scopes, &CompilationScope{
		Instructions: op.Instructions{},
		symbols:      syms,
	})
	c.scopeIdx++
}

func (c *Compiler) leaveScope() *CompilationScope {
	scope := c.scopes[c.scopeIdx]
	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIdx--
	return scope
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

func (c *Compiler) removeLastInstruction() emittedInstruction {
	last := c.scopes[c.scopeIdx].lastInstruction
	previous := c.scopes[c.scopeIdx].previousInstruction

	old := c.currentInstructions()
	new := old[:last.Position]

	c.scopes[c.scopeIdx].Instructions = new
	c.scopes[c.scopeIdx].lastInstruction = previous

	return last
}
