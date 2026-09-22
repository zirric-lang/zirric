package compiler

import (
	"context"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/op"
)

func (c *Compiler) compileExprIs(node ast.ExprIs) error {
	err := c.Compile(node.Value)
	if err != nil {
		return err
	}
	return c.compileTypeExprCheck(node.TypeRef)
}

// compileTypeExprCheck emits code to check the value on top of the stack
// against a type expression. Pushes Bool result.
// For multi-attribute expressions, uses a temp local for short-circuit AND.
func (c *Compiler) compileTypeExprCheck(typeExpr ast.TypeExpr) error {
	// Multi-attribute type expression: @A @B @C → short-circuit AND
	if attrs, ok := typeExpr.(ast.TypeExprAttrs); ok {
		return c.compileMultiAttrCheck(attrs)
	}

	typeConstId, err := c.resolveTypeConstantId(typeExpr)
	if err != nil {
		return err
	}
	c.emit(op.IsType, typeConstId)
	return nil
}

// compileMultiAttrCheck compiles a multi-attribute check with short-circuit AND.
// Value is already on the stack. For N attributes:
//
//	temp = pop; GetLocal(temp) → IsType(A) → JumpFalse(fail) → ... → IsType(last) → Jump(done) → fail: ConstFalse → done:
func (c *Compiler) compileMultiAttrCheck(attrs ast.TypeExprAttrs) error {
	if len(attrs.Attrs) == 1 {
		attrConstId, err := c.resolveTypeConstantId(attrs.Attrs[0])
		if err != nil {
			return err
		}
		c.emit(op.IsType, attrConstId)
		return nil
	}

	// Store value in temp local for multi-check
	tempLocal := c.allocateTempLocal()
	c.emit(op.SetLocal, tempLocal)

	var failJumps []int
	for _, attr := range attrs.Attrs {
		c.emit(op.GetLocal, tempLocal)
		attrConstId, err := c.resolveTypeConstantId(attr)
		if err != nil {
			return err
		}
		c.emit(op.IsType, attrConstId)
		failJumps = append(failJumps, c.emit(op.JumpFalse, placeholderJumpAddress))
	}
	c.emit(op.ConstTrue)
	jumpDone := c.emit(op.Jump, placeholderJumpAddress)

	failPos := len(c.currentInstructions())
	for _, fj := range failJumps {
		c.changeOperand(fj, failPos)
	}
	c.emit(op.ConstFalse)

	c.changeOperand(jumpDone, len(c.currentInstructions()))
	return nil
}

// compileSwitchIsTypeCheck emits type expression checks for a switch case.
// Loads value from tempLocal, checks each attribute (for multi-attr) or single type.
// Returns the JumpFalse addresses that need patching to the next case.
// A type that does not resolve is reported rather than skipped: emitting no check at all would leave the case matching everything, so a misspelled type would silently become the branch that always runs.
func (c *Compiler) compileSwitchIsTypeCheck(tempLocal int, typeExpr ast.TypeExpr) ([]int, error) {
	// Multi-attribute type expression: need per-attr checks
	if attrs, ok := typeExpr.(ast.TypeExprAttrs); ok {
		var jumpNexts []int
		for _, attr := range attrs.Attrs {
			c.emit(op.GetLocal, tempLocal)
			attrConstId, err := c.resolveTypeConstantId(attr)
			if err != nil {
				return nil, err
			}
			c.emit(op.IsType, attrConstId)
			jumpNexts = append(jumpNexts, c.emit(op.JumpFalse, placeholderJumpAddress))
		}
		return jumpNexts, nil
	}

	// Single type expression
	c.emit(op.GetLocal, tempLocal)
	typeConstId, err := c.resolveTypeConstantId(typeExpr)
	if err != nil {
		return nil, err
	}
	c.emit(op.IsType, typeConstId)
	return []int{c.emit(op.JumpFalse, placeholderJumpAddress)}, nil
}

func (c *Compiler) compileExprSwitch(node *ast.ExprSwitch) error {
	// Compile the switch value once and store in a temporary local.
	err := c.Compile(node.Value)
	if err != nil {
		return err
	}
	tempLocal := c.allocateTempLocal()
	c.emit(op.SetLocal, tempLocal)

	jumpEnds := make([]int, 0, len(node.Cases))

	for i, cs := range node.Cases {
		switch cs.Kind {
		case ast.SwitchCaseDefault:
			err := c.Compile(cs.Body)
			if err != nil {
				return err
			}

		case ast.SwitchCaseIsType:
			jumpNexts, err := c.compileSwitchIsTypeCheck(tempLocal, cs.TypeRef)
			if err != nil {
				return err
			}

			err = c.Compile(cs.Body)
			if err != nil {
				return err
			}
			if i < len(node.Cases)-1 {
				jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))
			}
			nextPos := len(c.currentInstructions())
			for _, jn := range jumpNexts {
				c.changeOperand(jn, nextPos)
			}

		case ast.SwitchCaseValue:
			c.emit(op.GetLocal, tempLocal)
			err := c.Compile(cs.Pattern)
			if err != nil {
				return err
			}
			c.emit(op.Equal)

			jumpNext := c.emit(op.JumpFalse, placeholderJumpAddress)

			err = c.Compile(cs.Body)
			if err != nil {
				return err
			}
			if i < len(node.Cases)-1 {
				jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))
			}
			c.changeOperand(jumpNext, len(c.currentInstructions()))
		}
	}

	endPos := len(c.currentInstructions())
	for _, pos := range jumpEnds {
		c.changeOperand(pos, endPos)
	}
	return nil
}

// compileTailStmtSwitch mirrors compileStmtSwitch, but routes each case body through compileFuncBody instead of compileBlock so cases produce the function's return value instead of discarding it.
// Since every case body is guaranteed to end in a Return (compileFuncBody's own Void fallback included), no jump-to-end bookkeeping is needed between cases.
func (c *Compiler) compileTailStmtSwitch(node ast.StmtSwitch) (bool, error) {
	err := c.Compile(node.Value)
	if err != nil {
		return false, err
	}
	tempLocal := c.allocateTempLocal()
	c.emit(op.SetLocal, tempLocal)

	for _, cs := range node.Cases {
		switch cs.Kind {
		case ast.SwitchCaseDefault:
			if err := c.compileFuncBody(cs.Body); err != nil {
				return false, err
			}

		case ast.SwitchCaseIsType:
			jumpNexts, err := c.compileSwitchIsTypeCheck(tempLocal, cs.TypeRef)
			if err != nil {
				return false, err
			}

			if err := c.compileFuncBody(cs.Body); err != nil {
				return false, err
			}
			nextPos := len(c.currentInstructions())
			for _, jn := range jumpNexts {
				c.changeOperand(jn, nextPos)
			}

		case ast.SwitchCaseValue:
			c.emit(op.GetLocal, tempLocal)
			if err := c.Compile(cs.Pattern); err != nil {
				return false, err
			}
			c.emit(op.Equal)

			jumpNext := c.emit(op.JumpFalse, placeholderJumpAddress)

			if err := c.compileFuncBody(cs.Body); err != nil {
				return false, err
			}
			c.changeOperand(jumpNext, len(c.currentInstructions()))
		}
	}
	// If no case matches (no exhaustive default/catch-all present), a non-matching case's JumpFalse lands exactly here regardless of what was last emitted (a matched case's own Return is only reached when that case's check actually passes) — so unconditionally guarantee a Void return for this fallthrough path.
	// Without it, the frame would run off the end of its instructions, which silently halts the whole program rather than just this function.
	c.emit(op.ConstVoid)
	c.emit(op.Return)
	return true, nil
}

func (c *Compiler) compileStmtSwitch(node ast.StmtSwitch) error {
	// Compile the switch value once and store in a temporary local.
	err := c.Compile(node.Value)
	if err != nil {
		return err
	}
	tempLocal := c.allocateTempLocal()
	c.emit(op.SetLocal, tempLocal)

	jumpEnds := make([]int, 0, len(node.Cases))

	for i, cs := range node.Cases {
		switch cs.Kind {
		case ast.SwitchCaseDefault:
			err := c.compileBlock(cs.Body)
			if err != nil {
				return err
			}

		case ast.SwitchCaseIsType:
			jumpNexts, err := c.compileSwitchIsTypeCheck(tempLocal, cs.TypeRef)
			if err != nil {
				return err
			}

			err = c.compileBlock(cs.Body)
			if err != nil {
				return err
			}
			if i < len(node.Cases)-1 {
				jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))
			}
			nextPos := len(c.currentInstructions())
			for _, jn := range jumpNexts {
				c.changeOperand(jn, nextPos)
			}

		case ast.SwitchCaseValue:
			c.emit(op.GetLocal, tempLocal)
			err := c.Compile(cs.Pattern)
			if err != nil {
				return err
			}
			c.emit(op.Equal)

			jumpNext := c.emit(op.JumpFalse, placeholderJumpAddress)

			err = c.compileBlock(cs.Body)
			if err != nil {
				return err
			}
			if i < len(node.Cases)-1 {
				jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))
			}
			c.changeOperand(jumpNext, len(c.currentInstructions()))
		}
	}

	endPos := len(c.currentInstructions())
	for _, pos := range jumpEnds {
		c.changeOperand(pos, endPos)
	}
	return nil
}

// compileExprForSwitchInLoop compiles a switch statement nested directly in an expr-for body, mirroring compileStmtSwitch but routing each case's block through compileExprForStatementBlock instead of compileBlock, so a bare expression statement in a case still gets appended to the expr-for's result array, and break/continue inside a case still jump to the enclosing loop via continueJumps/breakJumps.
func (c *Compiler) compileExprForSwitchInLoop(node ast.StmtSwitch, arrayLocal int, continueJumps *[]int, breakJumps *[]int) error {
	err := c.Compile(node.Value)
	if err != nil {
		return err
	}
	tempLocal := c.allocateTempLocal()
	c.emit(op.SetLocal, tempLocal)

	jumpEnds := make([]int, 0, len(node.Cases))

	for i, cs := range node.Cases {
		switch cs.Kind {
		case ast.SwitchCaseDefault:
			err := c.compileExprForStatementBlock(cs.Body, arrayLocal, continueJumps, breakJumps)
			if err != nil {
				return err
			}

		case ast.SwitchCaseIsType:
			jumpNexts, err := c.compileSwitchIsTypeCheck(tempLocal, cs.TypeRef)
			if err != nil {
				return err
			}

			err = c.compileExprForStatementBlock(cs.Body, arrayLocal, continueJumps, breakJumps)
			if err != nil {
				return err
			}
			if i < len(node.Cases)-1 {
				jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))
			}
			nextPos := len(c.currentInstructions())
			for _, jn := range jumpNexts {
				c.changeOperand(jn, nextPos)
			}

		case ast.SwitchCaseValue:
			c.emit(op.GetLocal, tempLocal)
			err := c.Compile(cs.Pattern)
			if err != nil {
				return err
			}
			c.emit(op.Equal)

			jumpNext := c.emit(op.JumpFalse, placeholderJumpAddress)

			err = c.compileExprForStatementBlock(cs.Body, arrayLocal, continueJumps, breakJumps)
			if err != nil {
				return err
			}
			if i < len(node.Cases)-1 {
				jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))
			}
			c.changeOperand(jumpNext, len(c.currentInstructions()))
		}
	}

	endPos := len(c.currentInstructions())
	for _, pos := range jumpEnds {
		c.changeOperand(pos, endPos)
	}
	return nil
}

// compileStmtSwitchInLoop compiles a switch statement nested directly in a (statement-form) for loop's body, mirroring compileStmtSwitch but routing each case's block through compileLoopBlock instead of compileBlock, so break/continue inside a case still reach the enclosing loop via continueJumps/breakJumps instead of falling through to the generic compileStmtBreak/compileStmtContinue, which always error.
func (c *Compiler) compileStmtSwitchInLoop(node ast.StmtSwitch, continueJumps *[]int, breakJumps *[]int) error {
	err := c.Compile(node.Value)
	if err != nil {
		return err
	}
	tempLocal := c.allocateTempLocal()
	c.emit(op.SetLocal, tempLocal)

	jumpEnds := make([]int, 0, len(node.Cases))

	for i, cs := range node.Cases {
		switch cs.Kind {
		case ast.SwitchCaseDefault:
			err := c.compileLoopBlock(cs.Body, continueJumps, breakJumps)
			if err != nil {
				return err
			}

		case ast.SwitchCaseIsType:
			jumpNexts, err := c.compileSwitchIsTypeCheck(tempLocal, cs.TypeRef)
			if err != nil {
				return err
			}

			err = c.compileLoopBlock(cs.Body, continueJumps, breakJumps)
			if err != nil {
				return err
			}
			if i < len(node.Cases)-1 {
				jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))
			}
			nextPos := len(c.currentInstructions())
			for _, jn := range jumpNexts {
				c.changeOperand(jn, nextPos)
			}

		case ast.SwitchCaseValue:
			c.emit(op.GetLocal, tempLocal)
			err := c.Compile(cs.Pattern)
			if err != nil {
				return err
			}
			c.emit(op.Equal)

			jumpNext := c.emit(op.JumpFalse, placeholderJumpAddress)

			err = c.compileLoopBlock(cs.Body, continueJumps, breakJumps)
			if err != nil {
				return err
			}
			if i < len(node.Cases)-1 {
				jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))
			}
			c.changeOperand(jumpNext, len(c.currentInstructions()))
		}
	}

	endPos := len(c.currentInstructions())
	for _, pos := range jumpEnds {
		c.changeOperand(pos, endPos)
	}
	return nil
}

// resolveTypeConstantId resolves a type expression to its constant slot index.
// Handles simple identifiers, dotted member access, and composite types.
func (c *Compiler) resolveTypeConstantId(typeExpr ast.TypeExpr) (int, error) {
	symbols := c.currentSymbols()
	if symbols == nil {
		return 0, errAt(typeExpr, "undefined type", "no symbols are in scope here")
	}

	switch e := typeExpr.(type) {
	case ast.TypeExprRef:
		if len(e.Reference) == 1 {
			return c.resolveIdentifierConstantId(e.Reference[0], symbols)
		}
		// SymbolTable.LookupRef only dereferences into an imported module via a symbol's ChildTable, which isn't populated for a plain `import pkg` (only for member imports) — so for a reference like `pkg.Type` it just returns the `pkg` import symbol itself, which has no ConstantId.
		// Try resolving across the module boundary first (mirroring resolveAttributeReference's handling for decorators), falling back to the plain lookup for local/nested references.
		if sym, err := c.resolveQualifiedTypeSymbol(e.Reference, symbols); err == nil {
			origSym, err := c.resolveThroughImportMember(sym.Original())
			if err != nil {
				return 0, errAt(e, "undefined type", "%s: %s", e.Reference.String(), err)
			}
			if origSym.ConstantId == nil {
				return 0, errAt(e, "not a type", "%s cannot be used in a type check", e.Reference.String())
			}
			return *origSym.ConstantId, nil
		}
		sym := symbols.LookupRef(e.Reference)
		if sym == nil || sym.Decl == nil {
			return 0, errAt(e, "undefined type", "%s", e.Reference.String())
		}
		origSym, err := c.resolveThroughImportMember(sym.Original())
		if err != nil {
			return 0, errAt(e, "undefined type", "%s: %s", e.Reference.String(), err)
		}
		if origSym.ConstantId == nil {
			return 0, errAt(e, "not a type", "%s cannot be used in a type check", e.Reference.String())
		}
		return *origSym.ConstantId, nil
	case ast.TypeExprOption:
		// Only the container is checked at runtime, exactly as TypeExprArray checks for an Array and leaves its element to the analyzer.
		return c.resolveBuiltinTypeConstantId(e, "Option", symbols)
	case ast.TypeExprResult:
		return c.resolveBuiltinTypeConstantId(e, "Result", symbols)
	case ast.TypeExprArray:
		return c.resolveBuiltinTypeConstantId(e, "Array", symbols)
	case ast.TypeExprDict:
		return c.resolveBuiltinTypeConstantId(e, "Dict", symbols)
	case ast.TypeExprFunc:
		return c.resolveBuiltinTypeConstantId(e, "Func", symbols)
	default:
		return 0, errAt(typeExpr, "unsupported type", "%s cannot be used in a type check", typeExpr.TypeExpression())
	}
}

// resolveQualifiedTypeSymbol resolves a multi-segment type reference (e.g. pkg.Type, or pkg.Attr wrapped in @) that crosses a module import boundary.
// Mirrors resolveAttributeReference's cross-module handling but doesn't require the result to be a *ast.DeclAttr, since it's also used for plain type checks like `case is pkg.SomeType`.
func (c *Compiler) resolveQualifiedTypeSymbol(ref ast.StaticReference, symbols *ast.SymbolTable) (*ast.Symbol, error) {
	if len(ref) < 2 {
		return nil, errAt(ref, "not a qualified reference", "%s names no module to look in", ref.String())
	}
	head := ref[0]
	if sym := c.lookupAttributeSymbol(head.Value, symbols); sym != nil {
		if decl, ok := sym.Decl.(*ast.DeclImport); ok {
			return c.resolveSymbolInModule(decl.ModuleName, ref[1:])
		}
		if sym.ChildTable != nil {
			return resolveStaticRefInTable(sym.ChildTable, ref[1:], false)
		}
	}
	if moduleName := c.findImportedModuleByPrefix(symbols.Module(), ref); moduleName != nil {
		return c.resolveSymbolInModule(moduleName, ref[len(moduleName):])
	}
	return nil, errAt(ref, "unknown reference", "%s", ref.String())
}

// resolveSymbolInModule resolves tail within moduleName's own exported symbol table, ensuring the module is analyzed first so ConstantId/GlobalId assignment has already happened.
func (c *Compiler) resolveSymbolInModule(moduleName ast.ModuleName, tail ast.StaticReference) (*ast.Symbol, error) {
	if c.resolver == nil {
		return nil, errInvariant(tail, "%s cannot be resolved without a module resolver, which the compiler is always built with", moduleName)
	}
	resolved, err := c.resolver.ResolveModule(context.Background(), moduleName.URI())
	if err != nil || resolved == nil {
		return nil, errAt(tail, "unknown module", "%s", moduleName)
	}
	if err := c.ensureAnalyzed(resolved, true); err != nil {
		return nil, err
	}
	return resolveStaticRefInTable(resolved.Symbols, tail, true)
}

// resolveThroughImportMember follows sym further if Original() left it as a DeclImportMember.
func (c *Compiler) resolveThroughImportMember(sym *ast.Symbol) (*ast.Symbol, error) {
	importMember, ok := sym.Decl.(ast.DeclImportMember)
	if !ok {
		return sym, nil
	}
	resolved, err := c.resolveTypeSymbolFromImport(importMember)
	if err != nil {
		return nil, err
	}
	return resolved, nil
}

func (c *Compiler) resolveIdentifierConstantId(name ast.Identifier, symbols *ast.SymbolTable) (int, error) {
	symbol := symbols.LookupIdentifier(name)
	if symbol == nil || symbol.Decl == nil {
		return 0, errAt(name, "undefined type", "%s", name.Value)
	}
	origSym, err := c.resolveThroughImportMember(symbol.Original())
	if err != nil {
		return 0, errAt(name, "undefined type", "%s: %s", name.Value, err)
	}
	if origSym.ConstantId == nil {
		return 0, errAt(name, "not a type", "%s cannot be used in a type check", name.Value)
	}
	return *origSym.ConstantId, nil
}

// resolveBuiltinTypeConstantId resolves one of the prelude types a composite type expression stands for, taking the expression itself so that a missing prelude still reports where it was needed.
func (c *Compiler) resolveBuiltinTypeConstantId(node ast.Node, name string, symbols *ast.SymbolTable) (int, error) {
	symbol := symbols.Lookup(name, nil)
	if symbol == nil || symbol.Decl == nil {
		return 0, errAt(node, "undefined built-in type", "%s is not in scope, which usually means prelude was not imported", name)
	}
	origSym, err := c.resolveThroughImportMember(symbol.Original())
	if err != nil {
		return 0, errAt(node, "undefined built-in type", "%s: %s", name, err)
	}
	if origSym.ConstantId == nil {
		return 0, errAt(node, "not a type", "the built-in %s cannot be used in a type check", name)
	}
	return *origSym.ConstantId, nil
}
