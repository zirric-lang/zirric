package compiler

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/op"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
)

// compileExprFunc compiles an anonymous function literal (ExprFunc) into
// either a plain Const (no captures) or a MakeClosure instruction.
func (c *Compiler) compileExprFunc(fn *ast.ExprFunc) error {
	symbols := fn.Symbols
	if symbols == nil {
		symbols = c.currentSymbols()
	}

	// Build free mapping: which FreeSymbols actually need runtime captures.
	// Globals and module-level constants are accessible directly and don't
	// need entries in the Free array.
	freeMapping := map[int]int{}
	freeCount := 0
	if symbols != nil {
		for i, parentSym := range symbols.FreeSymbols {
			if needsCapture(parentSym) {
				freeMapping[i] = freeCount
				freeCount++
			}
		}
	}

	// Compile function body in a new scope.
	c.enterScope(symbols)
	c.scopes[c.scopeIdx].freeMapping = freeMapping

	if symbols != nil {
		for _, child := range symbols.Symbols {
			if child.Decl == nil || child.Scope == ast.FreeScope {
				continue
			}
			if err := c.reserveSymbol(child); err != nil {
				return err
			}
		}
	}

	if err := c.compileBlock(fn.Impl); err != nil {
		return err
	}
	if !c.isLastInstruction(op.Return) {
		c.emit(op.ConstVoid)
		c.emit(op.Return)
	}

	bodyScope := c.leaveScope()

	function := runtime.MakeCompiledFunction(
		bodyScope.Instructions,
		len(fn.Parameters),
		len(bodyScope.locals),
		nil, // anonymous function - no declaration symbol
	)
	constId := c.addConstant(function)

	if freeCount == 0 {
		c.emit(op.Const, constId)
		return nil
	}

	// Push captures onto the stack in FreeSymbols order (skipping globals/constants).
	for i, parentSym := range symbols.FreeSymbols {
		if _, ok := freeMapping[i]; !ok {
			continue
		}
		if err := c.emitPushCapture(parentSym); err != nil {
			return err
		}
	}

	c.emit(op.MakeClosure, constId, freeCount)
	return nil
}

// needsCapture reports whether a parent symbol referenced by a FreeScope
// symbol requires a runtime capture slot in the closure's Free array.
// Module-level globals and constants with ConstantId (types, named functions)
// are accessible via GetGlobal/Const and don't need capturing.
func needsCapture(parentSym *ast.Symbol) bool {
	orig := parentSym.Original()
	if orig.GlobalId != nil {
		return false
	}
	if orig.ConstantId != nil && orig.LocalId == nil {
		return false
	}
	return true
}

// emitPushCapture emits instructions to push a captured value onto the stack
// when building a closure. Called from the enclosing scope after leaveScope.
func (c *Compiler) emitPushCapture(parentSym *ast.Symbol) error {
	switch parentSym.Scope {
	case ast.LocalScope, "":
		// Default empty scope means the symbol is local to its owning
		// SymbolTable (same frame). Push its value/cell via GetLocal.
		if parentSym.LocalId == nil {
			return fmt.Errorf("captured symbol %q has no local id", parentSym.Name)
		}
		// GetLocal pushes the raw value for const/param captures,
		// or the *UpvalueCell pointer for var captures (already wrapped).
		c.emit(op.GetLocal, *parentSym.LocalId)
		return nil
	case ast.FreeScope:
		// Promoted locals (FreeScope with LocalId) live in the current frame.
		if parentSym.LocalId != nil {
			c.emit(op.GetLocal, *parentSym.LocalId)
			return nil
		}
		// Transitive capture: forward the raw value/cell from our own Free array.
		currentScope := c.scopes[c.scopeIdx]
		freeIdx, ok := currentScope.freeMapping[parentSym.Index]
		if !ok {
			return fmt.Errorf("captured symbol %q (free index %d) not in current scope's free mapping", parentSym.Name, parentSym.Index)
		}
		c.emit(op.GetFree, freeIdx)
		return nil
	default:
		return fmt.Errorf("unexpected scope %s for captured symbol %q", parentSym.Scope, parentSym.Name)
	}
}

// compileFreeIdentifier compiles a read of a FreeScope identifier. If the
// original symbol is a module-level global or constant, the corresponding
// GetGlobal/Const is emitted. Otherwise the value is read from the closure's
// Free array, dereferencing an UpvalueCell for mutable var bindings.
func (c *Compiler) compileFreeIdentifier(symbol *ast.Symbol) error {
	orig := symbol.Original()

	// Module-level globals don't need captures.
	if orig.GlobalId != nil {
		c.emit(op.GetGlobal, *orig.GlobalId)
		return nil
	}
	// Module-level types/functions accessible as constants.
	if orig.ConstantId != nil && orig.LocalId == nil {
		c.emit(op.Const, *orig.ConstantId)
		return nil
	}

	currentScope := c.scopes[c.scopeIdx]
	freeIdx, ok := currentScope.freeMapping[symbol.Index]
	if !ok {
		return fmt.Errorf("free variable %q (index %d) not found in free mapping", symbol.Name, symbol.Index)
	}

	// Mutable var bindings are wrapped in UpvalueCells; dereference them.
	if _, isVar := symbol.Decl.(*ast.DeclVariable); isVar {
		c.emit(op.GetFreeCell, freeIdx)
	} else {
		c.emit(op.GetFree, freeIdx)
	}
	return nil
}

// compileFreeAssign compiles an assignment to a FreeScope mutable variable.
// The value to store must already be on top of the stack.
func (c *Compiler) compileFreeAssign(symbol *ast.Symbol) error {
	currentScope := c.scopes[c.scopeIdx]
	freeIdx, ok := currentScope.freeMapping[symbol.Index]
	if !ok {
		return fmt.Errorf("free variable %q (index %d) not found in free mapping for assignment", symbol.Name, symbol.Index)
	}
	c.emit(op.SetFreeCell, freeIdx)
	return nil
}
