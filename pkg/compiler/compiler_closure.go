package compiler

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/op"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
)

// compileExprFunc compiles an anonymous function literal (ExprFunc) into either a plain Const (no captures) or a MakeClosure instruction.
func (c *Compiler) compileExprFunc(fn *ast.ExprFunc) error {
	symbols := fn.Symbols
	if symbols == nil {
		symbols = c.currentSymbols()
	}

	// Build free mapping: which FreeSymbols actually need runtime captures.
	// Globals and module-level constants are accessible directly and don't need entries in the Free array.
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

	if err := c.compileFuncBody(fn.Impl); err != nil {
		return err
	}

	bodyScope := c.leaveScope()

	// Create a symbol for the anonymous closure so TypeConstantId works.
	anonSym := &ast.Symbol{
		Name: fn.Name,
		Decl: &ast.DeclFunc{
			Name: ast.Identifier{Value: fn.Name},
		},
	}
	if funcTypeSym := c.lookupTypeSymbol("Func"); funcTypeSym != nil {
		anonSym.TypeSymbol = funcTypeSym
	}

	function := runtime.MakeCompiledFunction(
		bodyScope.Instructions,
		len(fn.Parameters),
		len(bodyScope.locals),
		anonSym,
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

// needsCapture reports whether a parent symbol referenced by a FreeScope symbol requires a runtime capture slot in the closure's Free array.
// Module-level globals and constants with ConstantId (types, named functions) are accessible via GetGlobal/Const and don't need capturing.
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

// emitPushCapture emits instructions to push a captured value onto the stack when building a closure. Called from the enclosing scope after leaveScope.
func (c *Compiler) emitPushCapture(parentSym *ast.Symbol) error {
	switch parentSym.Scope {
	case ast.LocalScope, "":
		// Default empty scope means the symbol is local to its owning SymbolTable (same frame). Push its value/cell via GetLocal.
		if parentSym.LocalId == nil {
			return errInvariant(parentSym.Decl, "the captured %s was never given a local slot, which the analyzer assigns before compilation", parentSym.Name)
		}
		// GetLocal pushes the raw value for const/param captures, or the *UpvalueCell pointer for var captures (already wrapped).
		c.emit(op.GetLocal, *parentSym.LocalId)
		return nil
	case ast.FreeScope:
		// Promoted locals (FreeScope with LocalId) live in the current frame.
		if parentSym.LocalId != nil {
			c.emit(op.GetLocal, *parentSym.LocalId)
			return nil
		}
		// Transitive capture: forward from our own Free array, or read it from this frame.
		// The hops are walked as a read of the symbol walks them, because a `for` body has its own SymbolTable while running in this frame.
		access, ok := c.resolveFreeAccess(parentSym)
		if !ok {
			return errInvariant(parentSym.Decl, "the captured %s is recorded as free variable %d, which this scope does not have a mapping for", parentSym.Name, parentSym.Index)
		}
		if access.isFree {
			c.emit(op.GetFree, access.index)
		} else {
			c.emit(op.GetLocal, access.index)
		}
		return nil
	default:
		return errUnimplemented(parentSym.Decl, "scope %s is not handled when capturing %s", parentSym.Scope, parentSym.Name)
	}
}

// compileFreeIdentifier compiles a read of a FreeScope identifier. If the original symbol is a module-level global or constant, the corresponding GetGlobal/Const is emitted. Otherwise it walks one .Parent hop at a time (see resolveFreeAccess) to find either a real closure capture or a same-frame local, dereferencing an UpvalueCell for mutable var bindings.
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

	access, ok := c.resolveFreeAccess(symbol)
	if !ok {
		return errInvariant(symbol.Decl, "%s is recorded as free variable %d, which this scope does not have a mapping for", symbol.Name, symbol.Index)
	}
	if access.isFree {
		// Mutable var bindings are wrapped in UpvalueCells; dereference them.
		if _, isVar := access.symbol.Decl.(*ast.DeclVariable); isVar {
			c.emit(op.GetFreeCell, access.index)
		} else {
			c.emit(op.GetFree, access.index)
		}
		return nil
	}
	if _, isVar := access.symbol.Decl.(*ast.DeclVariable); isVar && access.symbol.IsCaptured {
		c.emit(op.GetLocalCell, access.index)
	} else {
		c.emit(op.GetLocal, access.index)
	}
	return nil
}

// compileFreeAssign compiles an assignment to a FreeScope mutable variable.
// The value to store must already be on top of the stack.
func (c *Compiler) compileFreeAssign(symbol *ast.Symbol) error {
	access, ok := c.resolveFreeAccess(symbol)
	if !ok {
		return errInvariant(symbol.Decl, "%s is recorded as free variable %d, which this scope does not have a mapping for, so it cannot be assigned to", symbol.Name, symbol.Index)
	}
	if access.isFree {
		c.emit(op.SetFreeCell, access.index)
		return nil
	}
	if _, isVar := access.symbol.Decl.(*ast.DeclVariable); isVar && access.symbol.IsCaptured {
		c.emit(op.SetLocalCell, access.index)
	} else {
		c.emit(op.SetLocal, access.index)
	}
	return nil
}

// freeAccess describes how to reach a FreeScope symbol's value from the current compilation scope.
type freeAccess struct {
	symbol *ast.Symbol // the hop at which resolution stopped — a real capture (isFree) or an in-frame local
	index  int         // free-array index (isFree) or local slot (!isFree)
	isFree bool
}

// resolveFreeAccess walks symbol.Parent one hop at a time toward its Original(), stopping at the first hop that's either resolvable through the current scope's freeMapping (a real capture set up by an enclosing compileExprFunc) or already a local in the current frame.
//
// Jumping straight to symbol.Original() (as read/assign used to) is only correct when every hop in between crossed no real closure boundary: an expr-for body's own SymbolTable has a Parent link to its enclosing function purely for lexical (same-frame) scoping, and the analyzer's generic SymbolTable.resolve() promotes any cross-table lookup to a FreeScope symbol via defineFree — even for this same-frame case, since it has no way to tell "nested block scope" from "real closure boundary" apart. Original() would then reach past a genuine intermediate closure's own capture (silently reading the wrong frame's local slot) whenever an expr-for inside a real closure refers to a variable from further out. Stopping at the first resolvable hop — rather than the last one — avoids that.
func (c *Compiler) resolveFreeAccess(symbol *ast.Symbol) (freeAccess, bool) {
	currentScope := c.scopes[c.scopeIdx]
	for s := symbol; s != nil; s = s.Parent {
		if freeIdx, ok := currentScope.freeMapping[s.Index]; ok && currentScope.ownsFree(s) {
			return freeAccess{symbol: s, index: freeIdx, isFree: true}, true
		}
		if s.LocalId != nil {
			return freeAccess{symbol: s, index: *s.LocalId, isFree: false}, true
		}
		if s.Scope != ast.FreeScope {
			break
		}
	}
	return freeAccess{}, false
}
