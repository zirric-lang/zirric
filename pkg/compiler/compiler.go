package compiler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/op"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/resolver"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

const (
	// A temporary address that acts placeholder.
	// Should be replaced by the actual address once known.
	placeholderJumpAddress = math.MinInt
)

func (c *Compiler) Compile(node ast.Node) error {
	// Every instruction emitted while this node is being compiled is recorded against it, which is what lets a crash be traced back to source.
	if node != nil {
		if source := node.TokenLiteral().Source; source != nil {
			previous := c.position
			c.position = source
			defer func() { c.position = previous }()
		}
	}
	switch node := node.(type) {
	case *ast.ContextModule:
		// Recorded before analysis, which already asks whether the module being worked on is the program itself.
		if c.entryModule == nil {
			c.entryModule = node
		}
		if err := c.ensureAnalyzed(node, true); err != nil {
			return err
		}

		moduleId := c.reserveGlobalModule(node.Name)
		if err := c.compileContextModule(node, moduleId); err != nil {
			return err
		}
		// A module of this package that failed to compile is reported here rather than leaving the program to run with part of its own package missing.
		return errors.Join(c.mainPackageErrs...)
	case *ast.SourceFile:
		if err := c.ensureAnalyzed(node.Decls.Module(), false); err != nil {
			return err
		}
		if c.scopeIdx == 0 {
			for _, sym := range node.Symbols.Symbols {
				if sym.Decl == nil {
					continue
				}
				if _, ok := sym.Decl.(*ast.DeclModule); ok {
					return errInvariant(sym.Decl, "a module declaration can only be compiled as part of a context module")
				}
			}
		}

		c.enterScope(node.Symbols)

		for _, sym := range node.Symbols.Symbols {
			if sym.Decl == nil {
				return errUndeclaredSymbol(sym)
			}
		}

		fileSymbols := c.sourceFileSymbols(node)
		for _, sym := range fileSymbols {
			if sym.Decl == nil {
				return errUndeclaredSymbol(sym)
			}
			err := c.reserveSymbol(sym)
			if err != nil {
				return err
			}
		}

		for _, sym := range fileSymbols {
			if sym.Decl == nil {
				return errUndeclaredSymbol(sym)
			}
			if !isFileScopeDecl(sym) {
				continue
			}
			err := c.compileSymbol(sym)
			if err != nil {
				return err
			}
		}

		for _, stmt := range node.Statements {
			err := c.Compile(stmt)
			if err != nil {
				return err
			}
		}

		scope := c.leaveScope()

		// at its core this is fine, but shouldn't this be at the module level?
		c.scopes[c.scopeIdx].Instructions = append(
			c.scopes[c.scopeIdx].Instructions,
			scope.Instructions...,
		)

		c.carryLocalsUp(scope)

		return nil

	case *ast.DeclVariable, *ast.DeclConstant, *ast.DeclFunc:
		symbols := c.currentSymbols()
		if symbols == nil {
			return errInvariant(node, "%T is being compiled with no symbols in scope", node)
		}
		sym := symbols.LookupIdentifier(node.(ast.Decl).DeclName())
		if sym == nil || sym.Decl == nil {
			return errInvariant(node, "%s was never registered by the analyzer, which records every declaration before compilation", node.(ast.Decl).DeclName().Value)
		}
		return c.compileSymbol(sym)

	case *ast.StmtExpr:
		err := c.Compile(node.Expr)
		if err != nil {
			return err
		}
		c.emit(op.Pop)
		return nil
	case *ast.StmtAssign:
		return c.compileStmtAssign(node)
	case ast.StmtIf:
		return c.compileStmtIf(node)
	case ast.StmtSwitch:
		return c.compileStmtSwitch(node)
	case ast.StmtFor:
		return c.compileStmtFor(node)
	case ast.StmtBreak:
		return c.compileStmtBreak(node)
	case ast.StmtContinue:
		return c.compileStmtContinue(node)

	case ast.ExprIf:
		return c.compileExprIf(node)
	case ast.ExprIs:
		return c.compileExprIs(node)
	case *ast.ExprSwitch:
		return c.compileExprSwitch(node)
	case ast.ExprFor:
		return c.compileExprFor(node)
	case *ast.ExprOperatorUnary:
		return c.compileExprOperatorUnary(node)
	case *ast.ExprOperatorBinary:
		return c.compileExprOperatorBinary(node)
	case *ast.ExprBool:
		if node.Literal {
			c.emit(op.ConstTrue)
		} else {
			c.emit(op.ConstFalse)
		}
		return nil
	case *ast.ExprVoid:
		c.emit(op.ConstVoid)
		return nil
	case *ast.ExprInt:
		val := c.plugins.Prelude().Int(node.Literal)
		idx := c.addConstant(val)
		c.emit(op.Const, idx)
		return nil
	case *ast.ExprFloat:
		val := c.plugins.Prelude().Float(node.Literal)
		idx := c.addConstant(val)
		c.emit(op.Const, idx)
		return nil
	case *ast.ExprString:
		val := c.plugins.Prelude().String(node.Literal)
		idx := c.addConstant(val)
		c.emit(op.Const, idx)
		return nil
	case *ast.ExprChar:
		val := c.plugins.Prelude().Char(node.Literal)
		idx := c.addConstant(val)
		c.emit(op.Const, idx)
		return nil

	case *ast.ExprFunc:
		return c.compileExprFunc(node)

	case *ast.ExprArray:
		for _, el := range node.Elements {
			err := c.Compile(el)
			if err != nil {
				return err
			}
		}
		val := c.plugins.Prelude().Int(int64(len(node.Elements)))
		idx := c.addConstant(val)
		c.emit(op.Const, idx)
		c.emit(op.Array)
		return nil

	case *ast.ExprDict:
		for _, entry := range node.Entries {
			err := c.Compile(entry.Key)
			if err != nil {
				return err
			}
			err = c.Compile(entry.Value)
			if err != nil {
				return err
			}
		}
		val := c.plugins.Prelude().Int(int64(len(node.Entries)))
		idx := c.addConstant(val)
		c.emit(op.Const, idx)
		c.emit(op.Dict)
		return nil
	case *ast.ExprIdentifier:
		symbol := node.Symbol
		if symbol == nil {
			symbols := c.currentSymbols()
			if symbols == nil {
				return errUndefinedIdentifier(node)
			}
			symbol = symbols.LookupIdentifier(node.Name)
		}
		if symbol == nil || symbol.Decl == nil {
			return errUndefinedIdentifier(node)
		}
		switch symbol.Decl.(type) {
		case *ast.DeclFunc, *ast.DeclData, *ast.DeclUnion, *ast.DeclExternFunc, *ast.DeclExternType, *ast.DeclExternValue, *ast.DeclAttr:
			sym := symbol.Original()
			// A local DeclFunc with captures gets a LocalId pointing to
			// its runtime Closure object. Check the direct symbol first
			// (FreeScope copies receive LocalId from DeclFunc compilation)
			// before the Original (which may be at module level).
			if symbol.LocalId != nil {
				c.emit(op.GetLocal, *symbol.LocalId)
				return nil
			}
			if sym.LocalId != nil {
				if symbol.Scope == ast.FreeScope {
					return c.compileFreeIdentifier(symbol)
				}
				c.emit(op.GetLocal, *sym.LocalId)
				return nil
			}
			if sym.ConstantId == nil {
				return errInvariant(node, "%s was never given a constant slot, which the analyzer assigns before compilation", node.Name)
			}
			c.emit(op.Const, *sym.ConstantId)
			return nil

		case *ast.DeclVariable, *ast.DeclConstant, *ast.DeclForBinding:
			// FreeScope symbols without a LocalId are actual cross-function
			// captures (the variable lives in a different frame). FreeScope
			// symbols WITH a LocalId are promoted locals that still live in
			// the current frame.
			if symbol.Scope == ast.FreeScope && symbol.LocalId == nil {
				return c.compileFreeIdentifier(symbol)
			}

			// Use the symbol's own LocalId first (covers both LocalScope
			// locals and FreeScope promoted locals).
			if symbol.LocalId != nil {
				if _, isVar := symbol.Decl.(*ast.DeclVariable); isVar && symbol.IsCaptured {
					c.emit(op.GetLocalCell, *symbol.LocalId)
				} else {
					c.emit(op.GetLocal, *symbol.LocalId)
				}
				return nil
			}

			sym := symbol.Original()
			if sym.LocalId != nil {
				if _, isVar := sym.Decl.(*ast.DeclVariable); isVar && sym.IsCaptured {
					c.emit(op.GetLocalCell, *sym.LocalId)
				} else {
					c.emit(op.GetLocal, *sym.LocalId)
				}
				return nil
			}
			if sym.GlobalId != nil {
				c.emit(op.GetGlobal, *sym.GlobalId)
				return nil
			}

			return errInvariant(node, "%s was never given a local or global slot, which the analyzer assigns before compilation", node.Name)

		case *ast.DeclParameter:
			if symbol.Scope == ast.FreeScope && symbol.LocalId == nil {
				return c.compileFreeIdentifier(symbol)
			}
			c.emit(op.GetLocal, *symbol.LocalId)
			return nil

		case *ast.DeclImport:
			sym := symbol.Original()
			if sym.GlobalId == nil {
				return errInvariant(node, "the module %s was never given a global slot, which the analyzer assigns before compilation", node.Name)
			}
			c.emit(op.GetGlobal, *sym.GlobalId)
			return nil
		case *ast.DeclModule:
			sym := symbol.Original()
			if sym.GlobalId == nil {
				return errInvariant(node, "the module %s was never given a global slot, which the analyzer assigns before compilation", node.Name)
			}
			c.emit(op.GetGlobal, *sym.GlobalId)
			return nil

		case ast.DeclImportMember:
			sym := symbol.Original()
			if sym.GlobalId == nil {
				return errInvariant(node, "the imported member %s was never given a global slot, which the analyzer assigns before compilation", node.Name)
			}
			c.emit(op.GetGlobal, *sym.GlobalId)
			return nil

		default:
			return errUnimplemented(node, "%s is a %T, which is not handled when compiling an identifier", node.Name, symbol.Decl)
		}

	case *ast.ExprMemberAccess:
		err := c.Compile(node.Target)
		if err != nil {
			return err
		}
		c.emit(op.GetField, c.addConstant(c.plugins.Prelude().String(node.Property.Value)))
		return nil

	case *ast.ExprIndexAccess:
		err := c.Compile(node.Target)
		if err != nil {
			return err
		}
		err = c.Compile(node.IndexExpr)
		if err != nil {
			return err
		}
		c.emit(op.GetIndex)
		return nil

	case *ast.ExprInvocation:
		for i := 0; i < len(node.Arguments); i++ {
			// compile arguments in left-to-right order
			// so they are pushed onto the stack in that order
			// and can be popped off in reverse order by the callee
			// (first argument is on the bottom of the stack)
			err := c.Compile(node.Arguments[i])
			if err != nil {
				return err
			}
		}
		err := c.Compile(node.Function)
		if err != nil {
			return err
		}

		c.emit(op.Call, len(node.Arguments))
		return nil

	case *ast.StmtReturn:
		if node.Expr == nil {
			c.emit(op.ConstVoid)
			c.emit(op.Return)
			return nil
		}

		err := c.Compile(node.Expr)
		if err != nil {
			return err
		}

		c.emit(op.Return)
		return nil

	default:
		return errUnimplemented(node, "%T is not handled when compiling", node)
	}
}

func (c *Compiler) reserveSymbol(sym *ast.Symbol) error {
	switch decl := sym.Decl.(type) {
	case *ast.DeclFunc:
		if sym.ConstantId == nil {
			return errInvariant(sym.Decl, "the function %s was never given a constant slot, which the analyzer assigns before compilation", decl.Name.Value)
		}
		c.ensureConstantSlot(*sym.ConstantId)
		return nil

	case *ast.DeclVariable:
		switch decl.ExportScope() {
		case ast.ExportScopeInternal, ast.ExportScopePublic:
			if sym.GlobalId == nil {
				return errInvariant(sym.Decl, "the global %s was never given a global slot, which the analyzer assigns before compilation", decl.Name.Value)
			}
			c.ensureGlobalSlot(*sym.GlobalId)
			return nil

		case ast.ExportScopeLocal:
			if sym.LocalId == nil {
				return errInvariant(sym.Decl, "the local %s was never given a local slot, which the analyzer assigns before compilation", decl.Name.Value)
			}
			c.ensureLocalSlot(*sym.LocalId)
			c.scopes[c.scopeIdx].locals[*sym.LocalId] = sym
			return nil

		default:
			return errUnimplemented(sym.Decl, "variable scope %v is not handled when reserving slots", sym.Scope)
		}

	case *ast.DeclConstant:
		switch decl.ExportScope() {
		case ast.ExportScopeInternal, ast.ExportScopePublic:
			if sym.GlobalId == nil {
				return errInvariant(sym.Decl, "the global %s was never given a global slot, which the analyzer assigns before compilation", decl.Name.Value)
			}
			c.ensureGlobalSlot(*sym.GlobalId)
			return nil

		case ast.ExportScopeLocal:
			if sym.LocalId == nil {
				return errInvariant(sym.Decl, "the local %s was never given a local slot, which the analyzer assigns before compilation", decl.Name.Value)
			}
			c.ensureLocalSlot(*sym.LocalId)
			c.scopes[c.scopeIdx].locals[*sym.LocalId] = sym
			return nil

		default:
			return errUnimplemented(sym.Decl, "variable scope %v is not handled when reserving slots", sym.Scope)
		}

	case *ast.DeclParameter:
		if sym.LocalId == nil {
			return errInvariant(sym.Decl, "the parameter %s was never given a local slot, which the analyzer assigns before compilation", decl.Name.Value)
		}
		c.ensureLocalSlot(*sym.LocalId)
		c.scopes[c.scopeIdx].locals[*sym.LocalId] = sym
		return nil

	case *ast.DeclForBinding:
		if sym.LocalId == nil {
			return errInvariant(sym.Decl, "the loop binding %s was never given a local slot, which the analyzer assigns before compilation", decl.Name.Value)
		}
		c.ensureLocalSlot(*sym.LocalId)
		c.scopes[c.scopeIdx].locals[*sym.LocalId] = sym
		return nil

	case *ast.DeclData, *ast.DeclUnion, *ast.DeclExternFunc, *ast.DeclExternType, *ast.DeclExternValue, *ast.DeclAttr:
		if sym.ConstantId == nil {
			return errInvariant(sym.Decl, "the declaration %s was never given a constant slot, which the analyzer assigns before compilation", sym.Name)
		}
		c.ensureConstantSlot(*sym.ConstantId)
		return nil

	case *ast.DeclModule:
		if sym.GlobalId == nil {
			return errInvariant(sym.Decl, "the module %s was never given a global slot, which the analyzer assigns before compilation", decl.Name.Value)
		}
		c.ensureGlobalSlot(*sym.GlobalId)
		c.moduleGlobals[c.currentSymbols().Module().Name] = *sym.GlobalId
		return nil

	case *ast.DeclImport:
		if sym.GlobalId == nil {
			return errInvariant(sym.Decl, "the import of %s was never given a global slot, which the analyzer assigns before compilation", decl.ModuleName)
		}
		c.ensureGlobalSlot(*sym.GlobalId)
		c.moduleGlobals[decl.ModuleName.URI()] = *sym.GlobalId
		return nil

	case ast.DeclImportMember:
		if sym.GlobalId == nil {
			return errInvariant(sym.Decl, "the imported member %s was never given a global slot, which the analyzer assigns before compilation", decl.Name.Value)
		}
		c.ensureGlobalSlot(*sym.GlobalId)
		return nil

	default:
		return errUnimplemented(sym.Decl, "%T is not handled when reserving slots", decl)
	}
}

func (c *Compiler) ensureAnalyzed(module *ast.ContextModule, reserveModule bool) error {
	if module == nil {
		return errInvariant(nil, "a module was analyzed without being loaded first")
	}
	if _, ok := c.analyzed[module]; ok {
		return nil
	}
	c.analyzed[module] = struct{}{}
	if c.analyzer == nil {
		return nil
	}
	errs, _ := c.analyzer.Analyze(module, reserveModule)
	if c.entryModule == module && c.scopes[0].symbols == nil {
		c.scopes[0].symbols = module.Symbols
	}
	// Only diagnostics that stop a build are returned; a warning is the caller's to report, not the compiler's to fail on.
	failing := analyzer.AnalysisErrors(errs).Failing()
	if len(failing) == 0 {
		return nil
	}
	return failing
}

// isFileScopeDecl reports whether a symbol's declaration is one of the file's own, rather than a binding promoted into its table from inside a block.
// A const or var written inside a loop or an if is promoted so that its name resolves, but it is only reached where it was written: compiling it here as well would run its initializer once before the block, where the values it reads do not exist yet.
func isFileScopeDecl(sym *ast.Symbol) bool {
	switch sym.Decl.(type) {
	case *ast.DeclConstant, *ast.DeclVariable:
		return sym.Decl.ExportScope() != ast.ExportScopeLocal
	default:
		return true
	}
}

func (c *Compiler) sourceFileSymbols(file *ast.SourceFile) []*ast.Symbol {
	symbols := make([]*ast.Symbol, 0, len(file.Symbols.Symbols))
	seen := map[*ast.Symbol]struct{}{}
	add := func(sym *ast.Symbol) {
		if sym == nil || sym.Decl == nil {
			return
		}
		sym = sym.Original()
		if _, ok := seen[sym]; ok {
			return
		}
		seen[sym] = struct{}{}
		symbols = append(symbols, sym)
	}

	for _, sym := range file.Symbols.Symbols {
		add(sym)
	}

	parent := file.Symbols.Parent
	if parent == nil {
		return symbols
	}
	fileSource := file.TokenLiteral().Source
	for _, sym := range parent.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		declSource := sym.Decl.TokenLiteral().Source
		if fileSource != nil && declSource != nil && declSource.File != fileSource.File {
			continue
		}
		add(sym)
	}

	return symbols
}

func (c *Compiler) changeOperand(pos int, operand int) {
	opcode := op.Opcode(c.currentInstructions()[pos])
	patched := op.Make(opcode, operand)
	c.replaceInstruction(pos, patched)
}

func (c *Compiler) replaceInstruction(pos int, patched []byte) {
	for i := 0; i < len(patched); i++ {
		c.currentInstructions()[pos+i] = patched[i]
	}
}

func (c *Compiler) compileBlock(block ast.Block) error {
	for _, stmt := range block {
		err := c.Compile(stmt)
		if err != nil {
			return err
		}
	}
	return nil
}

// compileFuncBody compiles a function (or closure) body block, converting its tail statement into the function's return value and guaranteeing the block always ends in a Return.
// A single trailing expression statement is already converted to an explicit `return` at parse time (see parser.go/pratt.go's parseFunctionDecl/parsePrattExprFnClosure), but that conversion only covers a single-statement body whose lone statement is a bare expression.
// This covers the rest: a multi-statement body, and a trailing `if`/`switch` statement whose branches should likewise produce the return value — recursing so a nested trailing if/switch inside a branch is converted the same way.
func (c *Compiler) compileFuncBody(block ast.Block) error {
	for i, stmt := range block {
		if i == len(block)-1 {
			handled, err := c.compileTailStmt(stmt)
			if err != nil {
				return err
			}
			if handled {
				return nil
			}
		}
		if err := c.Compile(stmt); err != nil {
			return err
		}
	}
	if !c.isLastInstruction(op.Return) {
		c.emit(op.ConstVoid)
		c.emit(op.Return)
	}
	return nil
}

// compileTailStmt compiles stmt as a function body's tail position when it has an unambiguous value — a bare expression statement, or an if/switch whose every branch recursively does — emitting Return with that value and reporting handled=true.
// Anything else reports handled=false so the caller falls back to compiling it as an ordinary statement (compileFuncBody then supplies an implicit Void return, matching prior behavior).
func (c *Compiler) compileTailStmt(stmt ast.Statement) (bool, error) {
	switch stmt := stmt.(type) {
	case *ast.StmtExpr:
		if err := c.Compile(stmt.Expr); err != nil {
			return false, err
		}
		c.emit(op.Return)
		return true, nil
	case ast.StmtIf:
		return c.compileTailStmtIf(stmt)
	case ast.StmtSwitch:
		return c.compileTailStmtSwitch(stmt)
	default:
		return false, nil
	}
}

// compileTailStmtIf mirrors compileStmtIf, but routes each branch through compileFuncBody instead of compileBlock so branches produce the function's return value instead of discarding it.
// An if with no else can't produce a value on every path, so it's left unhandled (falls back to Void).
func (c *Compiler) compileTailStmtIf(node ast.StmtIf) (bool, error) {
	if node.ElseBlock == nil {
		return false, nil
	}

	if err := c.Compile(node.Condition); err != nil {
		return false, err
	}
	jumpNext := c.emit(op.JumpFalse, placeholderJumpAddress)

	if err := c.compileFuncBody(node.IfBlock); err != nil {
		return false, err
	}

	for _, elseIf := range node.ElseIf {
		c.changeOperand(jumpNext, len(c.currentInstructions()))

		if err := c.Compile(elseIf.Condition); err != nil {
			return false, err
		}
		jumpNext = c.emit(op.JumpFalse, placeholderJumpAddress)

		if err := c.compileFuncBody(elseIf.Block); err != nil {
			return false, err
		}
	}
	c.changeOperand(jumpNext, len(c.currentInstructions()))

	if err := c.compileFuncBody(node.ElseBlock); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Compiler) compileStmtIf(node ast.StmtIf) error {
	var (
		jumpNext int
		jumpEnds = make([]int, 0, 1+len(node.ElseIf))
		endPos   int
	)
	err := c.Compile(node.Condition)
	if err != nil {
		return err
	}
	jumpNext = c.emit(op.JumpFalse, placeholderJumpAddress)

	err = c.compileBlock(node.IfBlock)
	if err != nil {
		return err
	}

	jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))

	for _, elseIf := range node.ElseIf {
		c.changeOperand(jumpNext, len(c.currentInstructions()))

		err = c.Compile(elseIf.Condition)
		if err != nil {
			return err
		}
		jumpNext = c.emit(op.JumpFalse, placeholderJumpAddress)

		err = c.compileBlock(elseIf.Block)
		if err != nil {
			return err
		}
		jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))
	}

	if node.ElseBlock != nil {
		c.changeOperand(jumpNext, len(c.currentInstructions()))

		err = c.compileBlock(node.ElseBlock)
		if err != nil {
			return err
		}
	} else {
		// No else: the condition-false path also lands after the whole if-statement, alongside the if-block's own trailing Jump (emitted above to skip a would-be else).
		// Both must be patched to endPos — jumpEnds already tracks the trailing Jump, so just add jumpNext rather than overwriting it, or that Jump keeps its placeholder operand (math.MinInt) forever, jumping to a garbage address whenever the condition is true.
		jumpEnds = append(jumpEnds, jumpNext)
	}

	endPos = len(c.currentInstructions())
	for _, pos := range jumpEnds {
		c.changeOperand(pos, endPos)
	}
	return nil
}

func (c *Compiler) compileStmtFor(node ast.StmtFor) error {
	if node.CollectionExpr != nil || node.CollectionIdent != nil {
		return c.compileStmtForCollection(node)
	}

	loopStart := len(c.currentInstructions())
	jumpOut := -1

	if node.Condition != nil {
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}
		jumpOut = c.emit(op.JumpFalse, placeholderJumpAddress)
	}

	breakJumps := make([]int, 0)
	continueJumps := make([]int, 0)
	err := c.compileLoopBlock(node.Body, &continueJumps, &breakJumps)
	if err != nil {
		return err
	}

	for _, pos := range continueJumps {
		c.changeOperand(pos, loopStart)
	}
	c.emit(op.Jump, loopStart)
	endPos := len(c.currentInstructions())

	if jumpOut != -1 {
		c.changeOperand(jumpOut, endPos)
	}
	for _, pos := range breakJumps {
		c.changeOperand(pos, endPos)
	}
	return nil
}

func (c *Compiler) compileStmtForCollection(node ast.StmtFor) error {
	if node.CollectionIdent == nil || node.CollectionExpr == nil {
		return errInvariant(node, "a collection for loop was compiled without a binding or a collection")
	}
	symbols := c.currentSymbols()
	if symbols == nil {
		return errInvariant(node, "the binding %s is being compiled with no symbols in scope", node.CollectionIdent.Value)
	}
	sym := symbols.LookupIdentifier(*node.CollectionIdent)
	if sym == nil {
		return errInvariant(node, "the binding %s was never registered by the analyzer", node.CollectionIdent.Value)
	}
	bindingLocal, err := c.requireLocalId(sym)
	if err != nil {
		return err
	}

	collectionLocal := c.allocateTempLocal()
	err = c.Compile(node.CollectionExpr)
	if err != nil {
		return err
	}
	c.emit(op.SetLocal, collectionLocal)

	arrayConstId, err := c.resolveBuiltinTypeConstantId(node, "Array", symbols)
	if err != nil {
		return err
	}
	c.emit(op.GetLocal, collectionLocal)
	c.emit(op.IsType, arrayConstId)
	genericJump := c.emit(op.JumpFalse, placeholderJumpAddress)

	if err := c.compileArrayForLoop(node.Body, collectionLocal, bindingLocal); err != nil {
		return err
	}
	endJump := c.emit(op.Jump, placeholderJumpAddress)

	c.changeOperand(genericJump, len(c.currentInstructions()))
	if err := c.compileIterableForLoop(node, node.Body, collectionLocal, bindingLocal, symbols); err != nil {
		return err
	}

	c.changeOperand(endJump, len(c.currentInstructions()))
	return nil
}

func (c *Compiler) compileArrayForLoop(body ast.Block, collectionLocal, bindingLocal int) error {
	indexLocal := c.allocateTempLocal()
	zeroConst := c.addConstant(c.plugins.Prelude().Int(0))
	oneConst := c.addConstant(c.plugins.Prelude().Int(1))

	c.emit(op.Const, zeroConst)
	c.emit(op.SetLocal, indexLocal)

	loopStart := len(c.currentInstructions())
	c.emit(op.GetLocal, indexLocal)
	c.emit(op.GetLocal, collectionLocal)
	c.emit(op.Len)
	c.emit(op.LessThan)
	jumpOut := c.emit(op.JumpFalse, placeholderJumpAddress)

	c.emit(op.GetLocal, collectionLocal)
	c.emit(op.GetLocal, indexLocal)
	c.emit(op.GetIndex)
	c.emit(op.SetLocal, bindingLocal)

	breakJumps := make([]int, 0)
	continueJumps := make([]int, 0)
	err := c.compileLoopBlock(body, &continueJumps, &breakJumps)
	if err != nil {
		return err
	}

	continueTarget := len(c.currentInstructions())
	for _, pos := range continueJumps {
		c.changeOperand(pos, continueTarget)
	}
	c.emit(op.GetLocal, indexLocal)
	c.emit(op.Const, oneConst)
	c.emit(op.Add)
	c.emit(op.SetLocal, indexLocal)
	c.emit(op.Jump, loopStart)

	endPos := len(c.currentInstructions())
	c.changeOperand(jumpOut, endPos)
	for _, pos := range breakJumps {
		c.changeOperand(pos, endPos)
	}
	return nil
}

// node is the loop being compiled, carried only so that anything reported here can say where it came from.
func (c *Compiler) compileIterableForLoop(node ast.Node, body ast.Block, collectionLocal, bindingLocal int, symbols *ast.SymbolTable) error {
	iterableConstId, err := c.resolveBuiltinTypeConstantId(node, "Iterable", symbols)
	if err != nil {
		return err
	}

	c.emit(op.GetLocal, collectionLocal)
	c.emit(op.IsType, iterableConstId)
	okJump := c.emit(op.JumpTrue, placeholderJumpAddress)

	panicConstId, err := c.resolveBuiltinTypeConstantId(node, "panic", symbols)
	if err != nil {
		return err
	}
	msgConst := c.addConstant(c.plugins.Prelude().String("for <- requires a value with @Iterable"))
	c.emit(op.Const, msgConst)
	c.emit(op.Const, panicConstId)
	c.emit(op.Call, 1)

	c.changeOperand(okJump, len(c.currentInstructions()))

	// The body below is only entered via the yield callable's ip-jump, never by falling through.
	skipBodyJump := c.emit(op.Jump, placeholderJumpAddress)

	bodyStartIp := len(c.currentInstructions())
	breakJumps := make([]int, 0)
	continueJumps := make([]int, 0)
	if err := c.compileLoopBlock(body, &continueJumps, &breakJumps); err != nil {
		return err
	}

	continueTarget := len(c.currentInstructions())
	for _, pos := range continueJumps {
		c.changeOperand(pos, continueTarget)
	}
	c.emit(op.ConstTrue)
	jumpToEnd := c.emit(op.Jump, placeholderJumpAddress)

	breakTarget := len(c.currentInstructions())
	for _, pos := range breakJumps {
		c.changeOperand(pos, breakTarget)
	}
	c.emit(op.ConstFalse)

	bodyEndIp := len(c.currentInstructions())
	c.changeOperand(jumpToEnd, bodyEndIp)

	c.changeOperand(skipBodyJump, len(c.currentInstructions()))

	c.emit(op.GetLocal, collectionLocal)
	c.emit(op.MakeIterYield, bindingLocal, bodyStartIp, bodyEndIp)

	c.emit(op.GetLocal, collectionLocal)
	c.emit(op.Const, iterableConstId)
	c.emit(op.Call, 1)
	iterateFieldConst := c.addConstant(c.plugins.Prelude().String("iterate"))
	c.emit(op.GetField, iterateFieldConst)

	c.emit(op.CallIterate, 2)
	return nil
}

func (c *Compiler) compileLoopBlock(block ast.Block, continueJumps *[]int, breakJumps *[]int) error {
	for _, stmt := range block {
		switch stmt := stmt.(type) {
		case ast.StmtBreak:
			pos := c.emit(op.Jump, placeholderJumpAddress)
			*breakJumps = append(*breakJumps, pos)
		case ast.StmtContinue:
			pos := c.emit(op.Jump, placeholderJumpAddress)
			*continueJumps = append(*continueJumps, pos)
		case ast.StmtIf:
			err := c.compileStmtIfInLoop(stmt, continueJumps, breakJumps)
			if err != nil {
				return err
			}
		case ast.StmtSwitch:
			err := c.compileStmtSwitchInLoop(stmt, continueJumps, breakJumps)
			if err != nil {
				return err
			}
		default:
			err := c.Compile(stmt)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Compiler) compileStmtIfInLoop(node ast.StmtIf, continueJumps *[]int, breakJumps *[]int) error {
	var (
		jumpNext int
		jumpEnds = make([]int, 0, 1+len(node.ElseIf))
		endPos   int
	)
	err := c.Compile(node.Condition)
	if err != nil {
		return err
	}
	jumpNext = c.emit(op.JumpFalse, placeholderJumpAddress)

	err = c.compileLoopBlock(node.IfBlock, continueJumps, breakJumps)
	if err != nil {
		return err
	}

	jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))

	for _, elseIf := range node.ElseIf {
		c.changeOperand(jumpNext, len(c.currentInstructions()))

		err = c.Compile(elseIf.Condition)
		if err != nil {
			return err
		}
		jumpNext = c.emit(op.JumpFalse, placeholderJumpAddress)

		err = c.compileLoopBlock(elseIf.Block, continueJumps, breakJumps)
		if err != nil {
			return err
		}
		jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))
	}

	if node.ElseBlock != nil {
		c.changeOperand(jumpNext, len(c.currentInstructions()))

		err = c.compileLoopBlock(node.ElseBlock, continueJumps, breakJumps)
		if err != nil {
			return err
		}
	} else {
		// No else: the condition-false path also lands after the whole if-statement, alongside the if-block's own trailing Jump (emitted above to skip a would-be else).
		// Both must be patched to endPos — jumpEnds already tracks the trailing Jump, so just add jumpNext rather than overwriting it, or that Jump keeps its placeholder operand (math.MinInt) forever, jumping to a garbage address whenever the condition is true.
		// Mirrors the same fix in compileStmtIf (pre-existing bug, not specific to loop bodies).
		jumpEnds = append(jumpEnds, jumpNext)
	}

	endPos = len(c.currentInstructions())
	for _, pos := range jumpEnds {
		c.changeOperand(pos, endPos)
	}
	return nil
}

func (c *Compiler) compileExprFor(node ast.ExprFor) error {
	prevSymbols := c.scopes[c.scopeIdx].symbols
	var bodySymbols *ast.SymbolTable
	if node.Body.Symbols != nil {
		bodySymbols = node.Body.Symbols
	} else if node.Body.DeclsTable != nil && node.Body.DeclsTable.Resolved != nil {
		bodySymbols = node.Body.DeclsTable.Resolved
	}
	if bodySymbols != nil {
		c.scopes[c.scopeIdx].symbols = bodySymbols
		c.ensureLocalSlotsForTable(bodySymbols)
	}
	defer func() {
		c.scopes[c.scopeIdx].symbols = prevSymbols
	}()

	arrayLocal := c.allocateTempLocal()
	zeroConst := c.addConstant(c.plugins.Prelude().Int(0))

	c.emit(op.Const, zeroConst)
	c.emit(op.Array)
	c.emit(op.SetLocal, arrayLocal)

	if node.CollectionExpr != nil || node.CollectionIdent != nil {
		return c.compileExprForCollection(node, arrayLocal)
	}

	loopStart := len(c.currentInstructions())
	jumpOut := -1

	if node.Condition != nil {
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}
		jumpOut = c.emit(op.JumpFalse, placeholderJumpAddress)
	}

	breakJumps := make([]int, 0)
	continueJumps := make([]int, 0)
	err := c.compileExprForBlock(node.Body, arrayLocal, &continueJumps, &breakJumps)
	if err != nil {
		return err
	}

	for _, pos := range continueJumps {
		c.changeOperand(pos, loopStart)
	}
	c.emit(op.Jump, loopStart)
	endPos := len(c.currentInstructions())

	if jumpOut != -1 {
		c.changeOperand(jumpOut, endPos)
	}
	for _, pos := range breakJumps {
		c.changeOperand(pos, endPos)
	}
	c.emit(op.GetLocal, arrayLocal)
	return nil
}

func (c *Compiler) compileExprForCollection(node ast.ExprFor, arrayLocal int) error {
	if node.CollectionIdent == nil || node.CollectionExpr == nil {
		return errInvariant(node, "a collection for expression was compiled without a binding or a collection")
	}
	symbols := c.currentSymbols()
	if symbols == nil {
		return errInvariant(node, "the binding %s is being compiled with no symbols in scope", node.CollectionIdent.Value)
	}
	sym := symbols.LookupIdentifier(*node.CollectionIdent)
	if sym == nil {
		return errInvariant(node, "the binding %s was never registered by the analyzer", node.CollectionIdent.Value)
	}
	bindingLocal, err := c.requireLocalId(sym)
	if err != nil {
		return err
	}

	collectionLocal := c.allocateTempLocal()
	err = c.Compile(node.CollectionExpr)
	if err != nil {
		return err
	}
	c.emit(op.SetLocal, collectionLocal)

	arrayConstId, err := c.resolveBuiltinTypeConstantId(node, "Array", symbols)
	if err != nil {
		return err
	}
	c.emit(op.GetLocal, collectionLocal)
	c.emit(op.IsType, arrayConstId)
	genericJump := c.emit(op.JumpFalse, placeholderJumpAddress)

	if err := c.compileArrayForExprLoop(node.Body, collectionLocal, bindingLocal, arrayLocal); err != nil {
		return err
	}
	endJump := c.emit(op.Jump, placeholderJumpAddress)

	c.changeOperand(genericJump, len(c.currentInstructions()))
	if err := c.compileGenericIterableForExprLoop(node, node.Body, collectionLocal, bindingLocal, arrayLocal, symbols); err != nil {
		return err
	}

	c.changeOperand(endJump, len(c.currentInstructions()))
	c.emit(op.GetLocal, arrayLocal)
	return nil
}

func (c *Compiler) compileArrayForExprLoop(body ast.ExprForBody, collectionLocal, bindingLocal, arrayLocal int) error {
	indexLocal := c.allocateTempLocal()
	zeroConst := c.addConstant(c.plugins.Prelude().Int(0))
	oneConst := c.addConstant(c.plugins.Prelude().Int(1))

	c.emit(op.Const, zeroConst)
	c.emit(op.SetLocal, indexLocal)

	loopStart := len(c.currentInstructions())
	c.emit(op.GetLocal, indexLocal)
	c.emit(op.GetLocal, collectionLocal)
	c.emit(op.Len)
	c.emit(op.LessThan)
	jumpOut := c.emit(op.JumpFalse, placeholderJumpAddress)

	c.emit(op.GetLocal, collectionLocal)
	c.emit(op.GetLocal, indexLocal)
	c.emit(op.GetIndex)
	c.emit(op.SetLocal, bindingLocal)

	breakJumps := make([]int, 0)
	continueJumps := make([]int, 0)
	err := c.compileExprForBlock(body, arrayLocal, &continueJumps, &breakJumps)
	if err != nil {
		return err
	}

	continueTarget := len(c.currentInstructions())
	for _, pos := range continueJumps {
		c.changeOperand(pos, continueTarget)
	}
	c.emit(op.GetLocal, indexLocal)
	c.emit(op.Const, oneConst)
	c.emit(op.Add)
	c.emit(op.SetLocal, indexLocal)
	c.emit(op.Jump, loopStart)

	endPos := len(c.currentInstructions())
	c.changeOperand(jumpOut, endPos)
	for _, pos := range breakJumps {
		c.changeOperand(pos, endPos)
	}
	return nil
}

// node is the loop being compiled, carried only so that anything reported here can say where it came from.
func (c *Compiler) compileGenericIterableForExprLoop(node ast.Node, body ast.ExprForBody, collectionLocal, bindingLocal, arrayLocal int, symbols *ast.SymbolTable) error {
	iterableConstId, err := c.resolveBuiltinTypeConstantId(node, "Iterable", symbols)
	if err != nil {
		return err
	}

	c.emit(op.GetLocal, collectionLocal)
	c.emit(op.IsType, iterableConstId)
	okJump := c.emit(op.JumpTrue, placeholderJumpAddress)

	panicConstId, err := c.resolveBuiltinTypeConstantId(node, "panic", symbols)
	if err != nil {
		return err
	}
	msgConst := c.addConstant(c.plugins.Prelude().String("for <- requires a value with @Iterable"))
	c.emit(op.Const, msgConst)
	c.emit(op.Const, panicConstId)
	c.emit(op.Call, 1)

	c.changeOperand(okJump, len(c.currentInstructions()))

	skipBodyJump := c.emit(op.Jump, placeholderJumpAddress)

	bodyStartIp := len(c.currentInstructions())
	breakJumps := make([]int, 0)
	continueJumps := make([]int, 0)
	if err := c.compileExprForBlock(body, arrayLocal, &continueJumps, &breakJumps); err != nil {
		return err
	}

	continueTarget := len(c.currentInstructions())
	for _, pos := range continueJumps {
		c.changeOperand(pos, continueTarget)
	}
	c.emit(op.ConstTrue)
	jumpToEnd := c.emit(op.Jump, placeholderJumpAddress)

	breakTarget := len(c.currentInstructions())
	for _, pos := range breakJumps {
		c.changeOperand(pos, breakTarget)
	}
	c.emit(op.ConstFalse)

	bodyEndIp := len(c.currentInstructions())
	c.changeOperand(jumpToEnd, bodyEndIp)

	c.changeOperand(skipBodyJump, len(c.currentInstructions()))

	c.emit(op.GetLocal, collectionLocal)
	c.emit(op.MakeIterYield, bindingLocal, bodyStartIp, bodyEndIp)

	c.emit(op.GetLocal, collectionLocal)
	c.emit(op.Const, iterableConstId)
	c.emit(op.Call, 1)
	iterateFieldConst := c.addConstant(c.plugins.Prelude().String("iterate"))
	c.emit(op.GetField, iterateFieldConst)

	c.emit(op.CallIterate, 2)
	return nil
}

func (c *Compiler) compileExprForBlock(body ast.ExprForBody, arrayLocal int, continueJumps *[]int, breakJumps *[]int) error {
	symbols := c.currentSymbols()
	if symbols == nil {
		return errInvariant(nil, "a for expression body is being compiled with no symbols in scope")
	}
	for _, decl := range body.Decls {
		name := decl.DeclName()
		sym := symbols.LookupIdentifier(name)
		if sym == nil {
			return errInvariant(decl, "the declaration %s was never registered by the analyzer", name.Value)
		}
		local, err := c.requireLocalId(sym)
		if err != nil {
			return err
		}
		var value ast.Expr
		switch d := decl.(type) {
		case *ast.DeclVariable:
			value = d.Value
		case *ast.DeclConstant:
			value = d.Value
		}
		err = c.Compile(value)
		if err != nil {
			return err
		}
		c.emit(op.SetLocal, local)
	}
	for _, stmt := range body.Stmts {
		err := c.compileExprForStatement(stmt, arrayLocal, continueJumps, breakJumps)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Compiler) compileExprForStatement(stmt ast.Statement, arrayLocal int, continueJumps *[]int, breakJumps *[]int) error {
	switch stmt := stmt.(type) {
	case *ast.StmtExpr:
		c.emit(op.GetLocal, arrayLocal)
		err := c.Compile(stmt.Expr)
		if err != nil {
			return err
		}
		c.emit(op.ArrayAppend)
		c.emit(op.SetLocal, arrayLocal)
		return nil
	case ast.StmtBreak:
		pos := c.emit(op.Jump, placeholderJumpAddress)
		*breakJumps = append(*breakJumps, pos)
		return nil
	case ast.StmtContinue:
		pos := c.emit(op.Jump, placeholderJumpAddress)
		*continueJumps = append(*continueJumps, pos)
		return nil
	case ast.StmtIf:
		return c.compileExprForIfInLoop(stmt, arrayLocal, continueJumps, breakJumps)
	case ast.StmtSwitch:
		return c.compileExprForSwitchInLoop(stmt, arrayLocal, continueJumps, breakJumps)
	default:
		return errAt(stmt, "not allowed in a for expression", "only an expression, if, switch, break or continue may appear here")
	}
}

func (c *Compiler) compileExprForStatementBlock(block ast.Block, arrayLocal int, continueJumps *[]int, breakJumps *[]int) error {
	for _, stmt := range block {
		err := c.compileExprForStatement(stmt, arrayLocal, continueJumps, breakJumps)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Compiler) compileExprForIfInLoop(node ast.StmtIf, arrayLocal int, continueJumps *[]int, breakJumps *[]int) error {
	var (
		jumpNext int
		jumpEnds = make([]int, 0, 1+len(node.ElseIf))
		endPos   int
	)
	err := c.Compile(node.Condition)
	if err != nil {
		return err
	}
	jumpNext = c.emit(op.JumpFalse, placeholderJumpAddress)

	err = c.compileExprForStatementBlock(node.IfBlock, arrayLocal, continueJumps, breakJumps)
	if err != nil {
		return err
	}

	jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))

	for _, elseIf := range node.ElseIf {
		c.changeOperand(jumpNext, len(c.currentInstructions()))

		err = c.Compile(elseIf.Condition)
		if err != nil {
			return err
		}
		jumpNext = c.emit(op.JumpFalse, placeholderJumpAddress)

		err = c.compileExprForStatementBlock(elseIf.Block, arrayLocal, continueJumps, breakJumps)
		if err != nil {
			return err
		}
		jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))
	}

	if node.ElseBlock != nil {
		c.changeOperand(jumpNext, len(c.currentInstructions()))

		err = c.compileExprForStatementBlock(node.ElseBlock, arrayLocal, continueJumps, breakJumps)
		if err != nil {
			return err
		}
	} else {
		// See the identical fix (and comment) in compileStmtIfInLoop: keep jumpNext alongside the tracked trailing Jump instead of overwriting it, or that Jump's placeholder operand is never patched.
		jumpEnds = append(jumpEnds, jumpNext)
	}

	endPos = len(c.currentInstructions())
	for _, pos := range jumpEnds {
		c.changeOperand(pos, endPos)
	}
	return nil
}

func (c *Compiler) compileStmtBreak(node ast.Node) error {
	return errAt(node, "break outside a loop", "there is nothing here to break out of")
}

func (c *Compiler) compileStmtContinue(node ast.Node) error {
	return errAt(node, "continue outside a loop", "there is nothing here to continue")
}

func (c *Compiler) compileExprIf(node ast.ExprIf) error {
	var (
		jumpNext int
		jumpEnds = make([]int, 0, 1+len(node.ElseIf))
		endPos   int
	)
	err := c.Compile(node.Condition)
	if err != nil {
		return err
	}
	jumpNext = c.emit(op.JumpFalse, placeholderJumpAddress)

	err = c.Compile(node.ThenExpr)
	if err != nil {
		return err
	}

	jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))

	for _, elseIf := range node.ElseIf {
		c.changeOperand(jumpNext, len(c.currentInstructions()))

		err = c.Compile(elseIf.Condition)
		if err != nil {
			return err
		}
		jumpNext = c.emit(op.JumpFalse, placeholderJumpAddress)

		err = c.Compile(elseIf.Then)
		if err != nil {
			return err
		}
		jumpEnds = append(jumpEnds, c.emit(op.Jump, placeholderJumpAddress))
	}
	c.changeOperand(jumpNext, len(c.currentInstructions()))

	err = c.Compile(node.ElseExpr)
	if err != nil {
		return err
	}

	endPos = len(c.currentInstructions())
	for _, pos := range jumpEnds {
		c.changeOperand(pos, endPos)
	}
	return nil
}

func (c *Compiler) compileExprOperatorUnary(node *ast.ExprOperatorUnary) error {
	err := c.Compile(node.Expr)
	if err != nil {
		return err
	}
	switch node.Operator.Type {
	case token.PLUS:
		// all numbers are positive by default
		// technically we would need to check the type of the expr
		return nil
	case token.BANG:
		c.emit(op.Invert)
		return nil
	case token.MINUS:
		c.emit(op.Negate)
		return nil
	default:
		return errAt(node, "unknown prefix operator", "%s", node.Operator.Literal)
	}
}
func (c *Compiler) compileExprOperatorBinary(node *ast.ExprOperatorBinary) error {
	err := c.Compile(node.Left)
	if err != nil {
		return err
	}

	switch node.Operator.Type {
	case token.AND:
		jumpQuick := c.emit(op.JumpFalse, placeholderJumpAddress)
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}
		c.emit(op.AssertType, int(c.plugins.Prelude().Bool(true).TypeConstantId()))
		jumpEnd := c.emit(op.Jump, placeholderJumpAddress)
		pos := c.emit(op.ConstFalse)
		c.changeOperand(jumpQuick, pos)
		c.changeOperand(jumpEnd, len(c.currentInstructions()))
		return nil

	case token.OR:
		jumpQuick := c.emit(op.JumpTrue, placeholderJumpAddress)
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}
		c.emit(op.AssertType, int(c.plugins.Prelude().Bool(true).TypeConstantId()))
		jumpEnd := c.emit(op.Jump, placeholderJumpAddress)
		pos := c.emit(op.ConstTrue)
		c.changeOperand(jumpQuick, pos)
		c.changeOperand(jumpEnd, len(c.currentInstructions()))
		return nil

	case token.PLUS:
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}
		c.emit(op.Add)
		return nil
	case token.MINUS:
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}
		c.emit(op.Sub)
		return nil
	case token.ASTERISK:
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}
		c.emit(op.Mul)
		return nil
	case token.SLASH:
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}
		c.emit(op.Div)
		return nil
	case token.PERCENT:
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}
		c.emit(op.Mod)
		return nil
	case token.EQ:
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}
		c.emit(op.Equal)
		return nil
	case token.NEQ:
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}
		c.emit(op.NotEqual)
		return nil
	case token.GT:
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}
		c.emit(op.GreaterThan)
		return nil
	case token.GTE:
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}
		c.emit(op.GreaterThanOrEqual)
		return nil
	case token.LT:
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}
		c.emit(op.LessThan)
		return nil
	case token.LTE:
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}
		c.emit(op.LessThanOrEqual)
		return nil
	default:
		return errAt(node, "unknown infix operator", "%s", node.Operator.Literal)
	}
}

// recordAttributes files a declaration's compiled attributes under the symbol they were written on, so that the module can report them for members whose value carries none.
func (c *Compiler) recordAttributes(sym *ast.Symbol, attributes map[runtime.TypeId]int) {
	if sym == nil || len(attributes) == 0 {
		return
	}
	c.symbolAttributes[sym.Original()] = attributes
}

func (c *Compiler) compileSymbol(sym *ast.Symbol) error {
	switch decl := sym.Decl.(type) {
	case *ast.DeclData:
		dt, err := runtime.MakeDataType(sym)
		if err != nil {
			return err
		}

		attributes, err := c.compileAttributeChain(decl.Attributes, c.currentSymbols())
		if err != nil {
			return err
		}
		dt.Attributes = attributes
		c.recordAttributes(sym, attributes)

		fieldAttributes, err := c.compileFieldAttributes(decl.Fields, c.currentSymbols())
		if err != nil {
			return err
		}
		dt.FieldAttributes = fieldAttributes
		dt.FieldTypes = c.compileFieldTypes(decl.Fields)
		dt.FieldDocs = fieldDocs(decl.Fields)

		c.constants[*sym.ConstantId] = dt

		return nil

	case *ast.DeclUnion:
		attributes, err := c.compileAttributeChain(decl.Attributes, c.currentSymbols())
		if err != nil {
			return err
		}
		memberTypeIds, err := c.resolveUnionMemberTypeIds(decl)
		if err != nil {
			return err
		}
		ut := runtime.MakeUnionType(sym, memberTypeIds)
		ut.Attributes = attributes
		c.recordAttributes(sym, attributes)
		c.constants[*sym.ConstantId] = ut
		return nil

	case *ast.DeclAttr:
		at, err := runtime.MakeAttributeType(sym)
		if err != nil {
			return err
		}
		attributes, err := c.compileAttributeChain(decl.Attributes, c.currentSymbols())
		if err != nil {
			return err
		}
		at.Attributes = attributes
		c.recordAttributes(sym, attributes)

		fieldAttributes, err := c.compileFieldAttributes(decl.Fields, c.currentSymbols())
		if err != nil {
			return err
		}
		at.FieldAttributes = fieldAttributes
		at.FieldTypes = c.compileFieldTypes(decl.Fields)
		at.FieldDocs = fieldDocs(decl.Fields)

		c.constants[*sym.ConstantId] = at
		return nil

	case *ast.DeclExternType:
		attributes, err := c.compileAttributeChain(decl.Attributes, c.currentSymbols())
		if err != nil {
			return err
		}

		if err := c.validateFieldAttributes(decl.Fields, c.currentSymbols()); err != nil {
			return err
		}

		c.recordAttributes(sym, attributes)

		if tid, ok := runtime.BuiltinTypeIds[sym.Name]; ok {
			st := runtime.MakeBuiltinSimpleType(sym, tid)
			st.Attributes = attributes
			c.constants[*sym.ConstantId] = st
		} else {
			c.constants[*sym.ConstantId] = runtime.SimpleType{Decl: sym, Attributes: attributes}
		}
		return nil

	case *ast.DeclExternValue:
		attributes, err := c.compileAttributeChain(decl.Attributes, c.currentSymbols())
		if err != nil {
			return err
		}
		c.recordAttributes(sym, attributes)

		val := c.plugins.Bind(c, c.currentSymbols(), sym)
		if val == nil {
			return errAt(sym.Decl, "extern value has no binding", "nothing in the runtime provides %s", sym.Name)
		}
		c.constants[*sym.ConstantId] = val
		return nil

	case *ast.DeclExternFunc:
		attributes, err := c.compileAttributeChain(decl.Attributes, c.currentSymbols())
		if err != nil {
			return err
		}

		paramSymbols := sym.ChildTable
		if paramSymbols == nil {
			paramSymbols = c.currentSymbols()
		}

		paramAttributes, err := c.compileParamAttributes(decl.Parameters, paramSymbols)
		if err != nil {
			return err
		}

		fn := c.plugins.Bind(c, c.currentSymbols(), sym)
		if fn == nil {
			return errAt(sym.Decl, "extern fn has no binding", "nothing in the runtime provides %s", sym.Name)
		}

		extfn, ok := fn.(*runtime.ExternFunc)
		if !ok {
			return errInvariant(sym.Decl, "%s is declared as an extern fn, but the runtime binds it to a %T rather than to a function", sym.Name, fn)
		}

		extfn.Attributes = attributes
		extfn.ParamAttributes = paramAttributes
		c.recordAttributes(sym, attributes)

		c.constants[*sym.ConstantId] = fn
		return nil

	case *ast.DeclFunc:
		// Skip promoted nested functions at file/module scope.
		// DeclTable promotion causes inner DeclFuncs to appear at module
		// level, but they must be compiled in their enclosing function's
		// scope so that free variable captures work correctly.
		if sym.Scope != ast.FreeScope && decl.Impl != nil && decl.Impl.Symbols != nil {
			parentST := decl.Impl.Symbols.Parent
			if parentST != nil {
				if _, isFunc := parentST.OpenedBy.(*ast.ExprFunc); isFunc {
					origSym := sym.Original()
					if origSym.ConstantId != nil {
						c.ensureConstantSlot(*origSym.ConstantId)
					}
					return nil
				}
			}
		}

		functionAttributes, err := c.compileAttributeChain(decl.Attributes, c.currentSymbols())
		if err != nil {
			return err
		}
		paramSymbols := decl.Impl.Symbols
		if paramSymbols == nil {
			paramSymbols = c.currentSymbols()
		}
		paramAttributes, err := c.compileParamAttributes(decl.Impl.Parameters, paramSymbols)
		if err != nil {
			return err
		}

		// Build free mapping for the function body.
		symbols := decl.Impl.Symbols
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

		for _, child := range decl.Impl.Symbols.Symbols {
			if child.Decl == nil || child.Scope == ast.FreeScope {
				continue
			}
			err = c.reserveSymbol(child)
			if err != nil {
				return err
			}
		}
		err = c.compileFuncBody(decl.Impl.Impl)
		if err != nil {
			return err
		}
		scope := c.leaveScope()

		function := runtime.MakeCompiledFunction(
			scope.Instructions,
			len(decl.Impl.Parameters),
			len(scope.locals),
			sym,
		)
		function.Attributes = functionAttributes
		function.ParamAttributes = paramAttributes
		c.recordAttributes(sym, functionAttributes)
		// Use Original() to access ConstantId: FreeScope copies created by
		// resolve_identifiers (before assignModuleIDs) have nil ConstantId.
		origSym := sym.Original()
		if origSym.ConstantId == nil {
			return errInvariant(sym.Decl, "the function %s was never given a constant slot, which the analyzer assigns before compilation", sym.Name)
		}
		c.ensureConstantSlot(*origSym.ConstantId)
		c.constants[*origSym.ConstantId] = function

		// If the function captures local variables, create a closure at
		// runtime and store it in a temp local so references use GetLocal
		// instead of Const.
		if freeCount > 0 {
			for i, parentSym := range symbols.FreeSymbols {
				if _, ok := freeMapping[i]; !ok {
					continue
				}
				if err := c.emitPushCapture(parentSym); err != nil {
					return err
				}
			}
			localId := c.allocateTempLocal()
			sym.LocalId = &localId
			c.emit(op.MakeClosure, *origSym.ConstantId, freeCount)
			c.emit(op.SetLocal, localId)
		}

		return nil

	case *ast.DeclVariable:
		variableAttributes, err := c.compileAttributeChain(decl.Attributes, c.currentSymbols())
		if err != nil {
			return err
		}
		c.recordAttributes(sym, variableAttributes)

		switch decl.ExportScope() {
		case ast.ExportScopeInternal, ast.ExportScopePublic:
			c.enterScope(sym.ChildTable)

			err := c.Compile(decl.Value)
			if err != nil {
				return err
			}

			scope := c.leaveScope()

			c.globals[*sym.GlobalId] = scope

			return nil

		case ast.ExportScopeLocal:
			if sym.LocalId == nil {
				return errInvariant(sym.Decl, "the local %s was never given a local slot, which the analyzer assigns before compilation", decl.Name.Value)
			}
			err := c.Compile(decl.Value)
			if err != nil {
				return err
			}

			c.ensureLocalSlot(*sym.LocalId)
			c.scopes[c.scopeIdx].locals[*sym.LocalId] = sym

			c.emit(op.SetLocal, *sym.LocalId)

			// If captured by a closure, wrap in an UpvalueCell so
			// inner scopes share the same mutable slot.
			if sym.IsCaptured {
				c.emit(op.WrapLocal, *sym.LocalId)
			}

			return nil

		default:
			return errUnimplemented(sym.Decl, "variable scope %v is not handled when compiling a variable", sym.Scope)
		}

	case *ast.DeclConstant:
		constantAttributes, err := c.compileAttributeChain(decl.Attributes, c.currentSymbols())
		if err != nil {
			return err
		}
		c.recordAttributes(sym, constantAttributes)

		switch decl.ExportScope() {
		case ast.ExportScopeInternal, ast.ExportScopePublic:
			c.enterScope(sym.ChildTable)

			err := c.Compile(decl.Value)
			if err != nil {
				return err
			}

			scope := c.leaveScope()

			c.globals[*sym.GlobalId] = scope

			return nil

		case ast.ExportScopeLocal:
			if sym.LocalId == nil {
				return errInvariant(sym.Decl, "the local %s was never given a local slot, which the analyzer assigns before compilation", decl.Name.Value)
			}
			err := c.Compile(decl.Value)
			if err != nil {
				return err
			}

			c.ensureLocalSlot(*sym.LocalId)
			c.scopes[c.scopeIdx].locals[*sym.LocalId] = sym

			c.emit(op.SetLocal, *sym.LocalId)

			return nil

		default:
			return errUnimplemented(sym.Decl, "variable scope %v is not handled when compiling a constant", sym.Scope)
		}

	case *ast.DeclParameter:
		return nil

	case *ast.DeclForBinding:
		return nil

	case *ast.DeclModule:
		return nil

	case *ast.DeclImport:
		if c.resolver == nil {
			return errInvariant(sym.Decl, "imports cannot be compiled without a module resolver, which the compiler is always built with")
		}

		uri := decl.ModuleName.URI()
		if sym.GlobalId == nil {
			return errInvariant(sym.Decl, "the import of %s was never given a global slot, which the analyzer assigns before compilation", decl.ModuleName)
		}
		c.ensureGlobalSlot(*sym.GlobalId)

		return c.compileModuleIfNeeded(uri, *sym.GlobalId)

	case ast.DeclImportMember:
		moduleURI := decl.ModuleName.URI()
		moduleGlobalId, ok := c.moduleGlobals[moduleURI]
		if !ok {
			return errInvariant(sym.Decl, "%s is imported from %s, which was never given a global slot, so its members cannot be reached", decl.Name.Value, moduleURI)
		}
		if sym.GlobalId == nil {
			return errInvariant(sym.Decl, "the imported member %s was never given a global slot, which the analyzer assigns before compilation", decl.Name.Value)
		}

		// Create an init scope that loads the module and gets the member.
		// At runtime, GetGlobal lazily executes this scope on first access.
		c.enterScope(nil)
		c.emit(op.GetGlobal, moduleGlobalId)
		nameId := c.addConstant(c.plugins.Prelude().String(decl.Name.Value))
		c.emit(op.GetField, nameId)
		scope := c.leaveScope()
		c.globals[*sym.GlobalId] = scope
		return nil

	default:
		return errUnimplemented(sym.Decl, "%T is not handled when compiling a declaration", decl)
	}
}

func (c *Compiler) compileModuleIfNeeded(moduleName registry.LogicalURI, id int) error {
	if scope := c.globals[id]; scope != nil {
		return nil
	}
	module, err := c.resolver.ResolveModule(context.Background(), moduleName)
	if err != nil {
		return err
	}
	if err := c.ensureAnalyzed(module, true); err != nil {
		return err
	}
	return c.compileContextModule(module, id)
}

func (c *Compiler) compileContextModule(module *ast.ContextModule, id int) error {
	if scope := c.globals[id]; scope != nil {
		return nil
	}
	// Prevent double compilation when the same module is imported under
	// different global IDs (e.g. prelude imported by both fmt and main).
	if firstId, ok := c.compiledModules[module]; ok {
		c.globals[id] = c.globals[firstId]
		return nil
	}
	c.compiledModules[module] = id
	c.enterScope(module.Symbols)

	for _, sym := range module.Symbols.Symbols {
		if sym.Decl == nil {
			continue
		}
		if err := c.reserveSymbol(sym); err != nil {
			c.leaveScope()
			return err
		}
	}

	// Reserve file-level imports before compiling module-level symbols.
	// Promoted declarations (e.g. data) may reference file-local imports
	// in their attributes (e.g. @cave.Package), so the import's
	// module global must be registered before attribute resolution.
	for _, src := range module.Files {
		if src.Symbols == nil {
			continue
		}
		for _, sym := range src.Symbols.Symbols {
			if sym.Decl == nil {
				continue
			}
			// Skip FreeScope symbols — these are captures from parent scopes
			// (e.g. prelude imports resolved during identifier resolution),
			// not actual file-level declarations.
			if sym.Scope == ast.FreeScope {
				continue
			}
			switch sym.Decl.(type) {
			case *ast.DeclImport, ast.DeclImportMember:
				if err := c.reserveSymbol(sym); err != nil {
					c.leaveScope()
					return err
				}
			}
		}
	}

	for _, sym := range module.Symbols.Symbols {
		if sym.Decl == nil || !isFileScopeDecl(sym) {
			continue
		}
		if err := c.compileSymbol(sym); err != nil {
			c.leaveScope()
			return err
		}
	}

	for _, src := range module.Files {
		if err := c.compileSourceFileDecls(src); err != nil {
			c.leaveScope()
			return err
		}
	}

	for _, src := range module.Files {
		if len(src.Statements) == 0 {
			continue
		}
		if err := c.compileInitFunction(src.Statements, src.Symbols, ModuleSymbol(module)); err != nil {
			c.leaveScope()
			return err
		}
	}

	if err := c.compileModuleValue(module); err != nil {
		c.leaveScope()
		return err
	}

	scope := c.leaveScope()
	c.globals[id] = scope
	// The program being run is the module the compiler was handed, whatever its name: a script in a subdirectory belongs to that directory's module rather than to the project root, and its top-level code still has to run.
	if c.entryModule == module {
		c.scopes[c.scopeIdx].Instructions = append(c.scopes[c.scopeIdx].Instructions, scope.Instructions...)
		c.carryLocalsUp(scope)
	}
	return nil
}

// compileSourceFileDecls compiles only the declarations of a source file (not statements).
// The resulting instructions are merged into the parent scope.
func (c *Compiler) compileSourceFileDecls(node *ast.SourceFile) error {
	if err := c.ensureAnalyzed(node.Decls.Module(), false); err != nil {
		return err
	}

	c.enterScope(node.Symbols)

	for _, sym := range node.Symbols.Symbols {
		if sym.Decl == nil {
			return errUndeclaredSymbol(sym)
		}
	}

	fileSymbols := c.sourceFileSymbols(node)
	for _, sym := range fileSymbols {
		if sym.Decl == nil {
			return errUndeclaredSymbol(sym)
		}
		if err := c.reserveSymbol(sym); err != nil {
			c.leaveScope()
			return err
		}
	}

	for _, sym := range fileSymbols {
		if !isFileScopeDecl(sym) {
			continue
		}
		if err := c.compileSymbol(sym); err != nil {
			c.leaveScope()
			return err
		}
	}

	scope := c.leaveScope()
	c.scopes[c.scopeIdx].Instructions = append(c.scopes[c.scopeIdx].Instructions, scope.Instructions...)
	c.carryLocalsUp(scope)
	return nil
}

// ModuleSymbol returns the symbol for the module's own declaration (the `module X` statement),
// searching through the module's source files. Returns nil if no module declaration is found.
func ModuleSymbol(module *ast.ContextModule) *ast.Symbol {
	for _, file := range module.Files {
		if file.Symbols == nil {
			continue
		}
		for _, sym := range file.Symbols.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}
			if _, ok := sym.Decl.(*ast.DeclModule); ok {
				return sym
			}
		}
	}
	return nil
}

// compileInitFunction compiles the given statements into a synthetic __init__ CompiledFunction
// and emits Const <id>; Call 0; Pop in the current scope.
// sym should be the module's own symbol (from ModuleSymbol) so the function is identifiable.
func (c *Compiler) compileInitFunction(statements []ast.Statement, symbols *ast.SymbolTable, sym *ast.Symbol) error {
	c.enterScope(symbols)
	for _, stmt := range statements {
		if err := c.Compile(stmt); err != nil {
			c.leaveScope()
			return err
		}
	}
	if !c.isLastInstruction(op.Return) {
		c.emit(op.ConstVoid)
		c.emit(op.Return)
	}
	scope := c.leaveScope()

	initFn := runtime.MakeCompiledFunction(scope.Instructions, 0, scope.LocalsCount(), sym)
	initConstantId := c.addConstant(initFn)
	c.emit(op.Const, initConstantId)
	c.emit(op.Call, 0)
	c.emit(op.Pop)
	return nil
}

// CompileSourceFileIncremental compiles a single source file incrementally for the REPL.
// It adds new declarations to the compiler's globals/constants and wraps any statements
// into a synthetic __init__ function stored as a constant.
// Returns the constant ID of the __init__ function, or -1 if there are no statements.
// The caller is responsible for ensuring the source file is analyzed before calling this.
func (c *Compiler) CompileSourceFileIncremental(node *ast.SourceFile) (int, error) {
	if node.Symbols == nil {
		return -1, errInvariant(nil, "a source file is being compiled before it was analyzed, so it carries no symbol table")
	}

	// Snapshot slice lengths so we can roll back on any compile error.
	prevGlobalsLen := len(c.globals)
	prevConstantsLen := len(c.constants)
	fail := func(err error) (int, error) {
		c.globals = c.globals[:prevGlobalsLen]
		c.constants = c.constants[:prevConstantsLen]
		return -1, err
	}

	c.enterScope(node.Symbols)

	for _, sym := range node.Symbols.Symbols {
		if sym.Decl == nil {
			c.leaveScope()
			return fail(errUndeclaredSymbol(sym))
		}
	}

	fileSymbols := c.sourceFileSymbols(node)
	for _, sym := range fileSymbols {
		if sym.Decl == nil {
			c.leaveScope()
			return fail(errUndeclaredSymbol(sym))
		}
		if err := c.reserveSymbol(sym); err != nil {
			c.leaveScope()
			return fail(err)
		}
	}
	for _, sym := range fileSymbols {
		if err := c.compileSymbol(sym); err != nil {
			c.leaveScope()
			return fail(err)
		}
	}

	// Discard outer scope instructions (declarations don't emit to parent scope)
	c.leaveScope()

	if len(node.Statements) == 0 {
		return -1, nil
	}

	c.enterScope(node.Symbols)
	stmts := node.Statements
	for i, stmt := range stmts {
		isLast := i == len(stmts)-1
		if isLast {
			if exprStmt, ok := stmt.(*ast.StmtExpr); ok {
				// Return the last expression's value so the REPL can display it.
				if err := c.Compile(exprStmt.Expr); err != nil {
					c.leaveScope()
					return fail(err)
				}
				c.emit(op.Return)
				break
			}
		}
		if err := c.Compile(stmt); err != nil {
			c.leaveScope()
			return fail(err)
		}
	}
	if !c.isLastInstruction(op.Return) {
		c.emit(op.ConstVoid)
		c.emit(op.Return)
	}
	initScope := c.leaveScope()

	initFn := runtime.MakeCompiledFunction(initScope.Instructions, 0, initScope.LocalsCount(), ModuleSymbol(node.Decls.Module()))
	// Left unnamed on purpose: a mod declaration is optional and its name need not match the module or the other files in it, so a frame running top-level statements is named after the file it is in instead.
	c.attachPositions(initScope, "")
	initConstantId := c.addConstant(initFn)
	return initConstantId, nil
}

func (c *Compiler) compileModuleValue(module *ast.ContextModule) error {
	exports := make([]*ast.Symbol, 0, len(module.Symbols.Symbols))
	for _, sym := range module.Symbols.Symbols {
		if sym.Decl == nil {
			continue
		}
		if sym.Decl.ExportScope() != ast.ExportScopePublic {
			continue
		}
		exports = append(exports, sym.Original())
	}

	sort.Slice(exports, func(i, j int) bool {
		return exports[i].Name < exports[j].Name
	})

	for _, sym := range exports {
		nameId := c.addConstant(c.plugins.Prelude().String(sym.Name))
		c.emit(op.Const, nameId)
		if err := c.emitModuleExport(sym); err != nil {
			return err
		}
	}

	countId := c.addConstant(c.plugins.Prelude().Int(int64(len(exports))))
	c.emit(op.Const, countId)

	files := moduleFilesInNameOrder(module)
	info := &runtime.ModuleInfo{
		Name:    string(module.Name),
		Docs:    moduleDocs(files),
		Imports: moduleImports(module),
		Sources: moduleSources(module),
	}
	attributes, err := c.compileModuleAttributes(files)
	if err != nil {
		return err
	}
	info.Attributes = attributes
	info.Declarations = make([]runtime.ModuleDeclaration, 0, len(exports))
	for _, sym := range exports {
		info.Declarations = append(info.Declarations, runtime.ModuleDeclaration{
			Name:       sym.Name,
			Decl:       sym.Decl,
			Attributes: c.symbolAttributes[sym],
		})
	}
	c.emit(op.Module, c.addConstant(info))
	return nil
}

// moduleFilesInNameOrder returns the files that declare the module, ordered by name.
// A module is declared once per file, and both its documentation and its attributes are only whole once every one of them has been read.
func moduleFilesInNameOrder(module *ast.ContextModule) []*ast.SourceFile {
	files := make([]*ast.SourceFile, 0, len(module.Files))
	for _, file := range module.Files {
		if file != nil && file.Module != nil {
			files = append(files, file)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files
}

// moduleDocs joins the comment written above each `mod` declaration, in file name order, separated by a blank line.
func moduleDocs(files []*ast.SourceFile) string {
	paragraphs := make([]string, 0, len(files))
	for _, file := range files {
		if docs := file.Module.ProvidedDocs(); docs != nil && docs.Content != "" {
			paragraphs = append(paragraphs, docs.Content)
		}
	}
	return strings.Join(paragraphs, "\n\n")
}

// moduleImports names every module the files of this one import, ordered by name and free of duplicates.
// Imports are file-local, so a module's own dependencies are only whole once every file of it has been read.
func moduleImports(module *ast.ContextModule) []string {
	seen := map[string]struct{}{}
	for _, file := range module.Files {
		if file == nil || file.Decls == nil {
			continue
		}
		for _, sym := range file.Decls.Symbols {
			if sym == nil {
				continue
			}
			decl, ok := sym.Decl.(*ast.DeclImport)
			if !ok {
				continue
			}
			seen[string(decl.ModuleName.URI())] = struct{}{}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// moduleSources names every file the module was read from, ordered by name.
// Every file is named, including one that declares nothing, because a file carrying only the module's documentation is still where that documentation is edited.
func moduleSources(module *ast.ContextModule) []string {
	paths := make([]string, 0, len(module.Files))
	for _, file := range module.Files {
		if file == nil || file.Path == "" {
			continue
		}
		paths = append(paths, file.Path)
	}
	sort.Strings(paths)
	return paths
}

// compileModuleAttributes compiles the attributes written on a module's `mod` declarations.
func (c *Compiler) compileModuleAttributes(files []*ast.SourceFile) (map[runtime.TypeId]int, error) {
	chain := make(ast.AttributeChain, 0, len(files))
	for _, file := range files {
		for _, instance := range file.Module.Attributes {
			if instance == nil {
				continue
			}
			chain = append(chain, instance)
		}
	}
	return c.compileAttributeChain(chain, c.currentSymbols())
}

func (c *Compiler) emitModuleExport(sym *ast.Symbol) error {
	switch sym.Decl.(type) {
	case *ast.DeclFunc, *ast.DeclData, *ast.DeclUnion, *ast.DeclExternFunc, *ast.DeclExternType, *ast.DeclExternValue, *ast.DeclAttr:
		if sym.ConstantId == nil {
			return errInvariant(sym.Decl, "the export %s was never given a constant slot, which the analyzer assigns before compilation", sym.Name)
		}
		c.emit(op.Const, *sym.ConstantId)
		return nil

	case *ast.DeclVariable, *ast.DeclConstant:
		if sym.GlobalId == nil {
			return errInvariant(sym.Decl, "the export %s was never given a global slot, which the analyzer assigns before compilation", sym.Name)
		}
		c.emit(op.GetGlobal, *sym.GlobalId)
		return nil

	default:
		return errUnimplemented(sym.Decl, "%s is a %T, which is not handled when exporting a module member", sym.Name, sym.Decl)
	}
}

func (c *Compiler) compileAttributeChain(chain ast.AttributeChain, symbols *ast.SymbolTable) (map[runtime.TypeId]int, error) {
	if len(chain) == 0 {
		return nil, nil
	}
	result := make(map[runtime.TypeId]int, len(chain))
	for _, inst := range chain {
		if inst == nil {
			continue
		}
		annoSym, err := c.resolveAttributeReference(inst.Reference, symbols)
		if err != nil {
			return nil, err
		}
		if annoSym.ConstantId == nil {
			return nil, errInvariant(annoSym.Decl, "the attribute %s was never given a constant slot, which the analyzer assigns before compilation", annoSym.Name)
		}
		c.ensureConstantSlot(*annoSym.ConstantId)
		globalId, err := c.compileAttributeInstance(inst, annoSym, symbols)
		if err != nil {
			return nil, err
		}
		result[runtime.TypeId(*annoSym.ConstantId)] = globalId
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func (c *Compiler) compileParamAttributes(params []ast.DeclParameter, symbols *ast.SymbolTable) ([]map[runtime.TypeId]int, error) {
	if len(params) == 0 {
		return nil, nil
	}
	result := make([]map[runtime.TypeId]int, len(params))
	hasAny := false
	for i := range params {
		param := params[i]
		if len(param.Attributes) == 0 {
			continue
		}
		attributes, err := c.compileAttributeChain(param.Attributes, symbols)
		if err != nil {
			return nil, err
		}
		if attributes != nil {
			result[i] = attributes
			hasAny = true
		}
	}
	if !hasAny {
		return nil, nil
	}
	return result, nil
}

// compileFieldAttributes compiles the attributes written on each field and returns them by field position, or nil when no field carries any.
// Attributes on the parameters of a function-typed field are compiled so that mistakes in them are still reported, but nothing reads them back.
func (c *Compiler) compileFieldAttributes(fields []ast.DeclField, symbols *ast.SymbolTable) ([]map[runtime.TypeId]int, error) {
	if len(fields) == 0 {
		return nil, nil
	}
	result := make([]map[runtime.TypeId]int, len(fields))
	hasAny := false
	for i := range fields {
		field := fields[i]
		for _, param := range field.Parameters {
			if len(param.Attributes) == 0 {
				continue
			}
			if _, err := c.compileAttributeChain(param.Attributes, symbols); err != nil {
				return nil, err
			}
		}
		if len(field.Attributes) == 0 {
			continue
		}
		attributes, err := c.compileAttributeChain(field.Attributes, symbols)
		if err != nil {
			return nil, err
		}
		if attributes != nil {
			result[i] = attributes
			hasAny = true
		}
	}
	if !hasAny {
		return nil, nil
	}
	return result, nil
}

// compileFieldTypes describes the type hint written on each field, by field position, or nil when no field carries one.
// Hints are not enforced at runtime, so this only records what was written; an unresolvable name is left unresolved rather than reported, since that is the checker's business and reflection must not turn it into a compile error.
func (c *Compiler) compileFieldTypes(fields []ast.DeclField) []runtime.TypeRef {
	if len(fields) == 0 {
		return nil
	}
	result := make([]runtime.TypeRef, len(fields))
	hasAny := false
	for i := range fields {
		if fields[i].TypeHint == nil {
			continue
		}
		result[i] = c.describeTypeExpr(fields[i].TypeHint)
		hasAny = true
	}
	if !hasAny {
		return nil
	}
	return result
}

// fieldDocs records the comment written above each field, by field position, or nil when no field carries one.
func fieldDocs(fields []ast.DeclField) []string {
	result := make([]string, len(fields))
	hasAny := false
	for i := range fields {
		result[i] = ast.DocsOf(fields[i])
		if result[i] != "" {
			hasAny = true
		}
	}
	if !hasAny {
		return nil
	}
	return result
}

// describeTypeExpr converts a written type expression into the structural form reflection exposes, resolving names against the symbols currently being compiled.
// A name that resolves to nothing is left unresolved rather than reported, since an unknown type is the checker's business and reflection must not turn it into a compile error.
func (c *Compiler) describeTypeExpr(expr ast.TypeExpr) runtime.TypeRef {
	switch expr := expr.(type) {
	case ast.TypeExprRef:
		named := runtime.TypeRef{Kind: runtime.TypeRefNamed, Name: expr.Reference.String()}
		if constantId, err := c.resolveTypeConstantId(expr); err == nil {
			typeId := runtime.TypeId(constantId)
			named.Type = &typeId
		}
		return named
	case ast.TypeExprArray:
		element := c.describeTypeExpr(expr.Element)
		return runtime.TypeRef{Kind: runtime.TypeRefArray, Element: &element}
	case ast.TypeExprDict:
		key := c.describeTypeExpr(expr.Key)
		value := c.describeTypeExpr(expr.Value)
		return runtime.TypeRef{Kind: runtime.TypeRefDict, Key: &key, Value: &value}
	case ast.TypeExprFunc:
		parameters := make([]runtime.TypeRef, len(expr.Parameters))
		for i, param := range expr.Parameters {
			parameters[i] = c.describeTypeExpr(param.TypeHint)
		}
		returns := c.describeTypeExpr(expr.ReturnType)
		return runtime.TypeRef{Kind: runtime.TypeRefFunc, Parameters: parameters, Returns: &returns}
	case ast.TypeExprAttrs:
		attributes := make([]runtime.TypeRef, len(expr.Attrs))
		for i, attr := range expr.Attrs {
			attributes[i] = c.describeTypeExpr(attr)
		}
		return runtime.TypeRef{Kind: runtime.TypeRefAttrs, Attributes: attributes}
	}
	return runtime.TypeRef{}
}

// validateFieldAttributes compiles field attributes only to report errors in them, for declarations that keep no field attributes of their own.
func (c *Compiler) validateFieldAttributes(fields []ast.DeclField, symbols *ast.SymbolTable) error {
	_, err := c.compileFieldAttributes(fields, symbols)
	return err
}

func (c *Compiler) compileAttributeInstance(inst *ast.DeclAttrInstance, sym *ast.Symbol, symbols *ast.SymbolTable) (int, error) {
	if inst == nil {
		return 0, errInvariant(nil, "an attribute instance was compiled without a declaration")
	}
	if sym == nil {
		return 0, errInvariant(inst, "an attribute instance was compiled without a resolved symbol")
	}
	if sym.ConstantId == nil {
		return 0, errInvariant(inst, "the attribute %s was never given a constant slot, which the analyzer assigns before compilation", sym.Name)
	}

	c.enterScope(symbols)
	for _, arg := range inst.Arguments {
		if err := c.Compile(arg); err != nil {
			c.leaveScope()
			return 0, err
		}
	}
	c.emit(op.Const, *sym.ConstantId)
	c.emit(op.MakeAttribute, len(inst.Arguments))
	scope := c.leaveScope()

	return c.addGlobalScope(scope), nil
}

func (c *Compiler) addGlobalScope(scope *CompilationScope) int {
	if scope == nil {
		return -1
	}
	var id int
	if c.analyzer != nil {
		id = c.analyzer.AllocateGlobalId()
	} else {
		id = len(c.globals)
	}
	c.ensureGlobalSlot(id)
	c.globals[id] = scope
	return id
}

func (c *Compiler) resolveAttributeReference(ref ast.StaticReference, symbols *ast.SymbolTable) (*ast.Symbol, error) {
	if len(ref) == 0 {
		return nil, errInvariant(nil, "an attribute was compiled with an empty reference")
	}
	if symbols == nil {
		return nil, errAt(ref, "unknown attribute", "%s: no symbols are in scope here", ref.String())
	}
	if len(ref) == 1 {
		sym := c.lookupAttributeSymbol(ref[0].Value, symbols)
		if sym == nil {
			return nil, errAt(ref, "unknown attribute", "%s", ref.String())
		}
		// If the symbol is an import member, resolve the actual attribute
		// from the imported module.
		if member, ok := sym.Original().Decl.(ast.DeclImportMember); ok {
			return c.resolveAttributeFromModuleName(member.ModuleName, ast.StaticReference{ref[0]}, ref)
		}
		return requireAttributeSymbol(sym, ref)
	}

	head := ref[0]
	if sym := c.lookupAttributeSymbol(head.Value, symbols); sym != nil {
		if decl, ok := sym.Decl.(*ast.DeclImport); ok {
			return c.resolveAttributeFromImport(decl, ref[1:], ref)
		}
		if sym.ChildTable != nil {
			found, err := resolveStaticRefInTable(sym.ChildTable, ref[1:], false)
			if err != nil {
				return nil, err
			}
			return requireAttributeSymbol(found, ref)
		}
	}

	if moduleName := c.findImportedModuleByPrefix(symbols.Module(), ref); moduleName != nil {
		return c.resolveAttributeFromModuleName(moduleName, ref[len(moduleName):], ref)
	}

	return nil, errAt(ref, "unknown attribute", "%s", ref.String())
}

// lookupAttributeSymbol finds a symbol by name for attribute resolution.
// Unlike LookupIdentifier, it does not create phantom symbols.
// When at module scope, it also checks file scopes for file-local
// declarations like imports that are not promoted to module level.
func (c *Compiler) lookupAttributeSymbol(name string, symbols *ast.SymbolTable) *ast.Symbol {
	for cur := symbols; cur != nil; cur = cur.Parent {
		if sym, ok := cur.Symbols[name]; ok && sym != nil && sym.Decl != nil {
			return sym
		}
	}
	// Declarations like data/func are promoted to module scope but imports
	// stay file-local. When compiling a promoted symbol's attributes in
	// module scope, the import is only visible in the originating file's
	// SymbolTable.
	if module, ok := symbols.OpenedBy.(*ast.ContextModule); ok {
		for _, file := range module.Files {
			if file.Symbols == nil {
				continue
			}
			if sym, ok := file.Symbols.Symbols[name]; ok && sym != nil && sym.Decl != nil {
				return sym
			}
		}
	}
	return nil
}

func (c *Compiler) resolveAttributeFromImport(decl *ast.DeclImport, tail ast.StaticReference, ref ast.StaticReference) (*ast.Symbol, error) {
	if decl == nil {
		return nil, errAt(ref, "unknown attribute", "%s", ref.String())
	}
	return c.resolveAttributeFromModuleName(decl.ModuleName, tail, ref)
}

func (c *Compiler) resolveAttributeFromModuleName(moduleName ast.ModuleName, tail ast.StaticReference, ref ast.StaticReference) (*ast.Symbol, error) {
	if c.resolver == nil {
		return nil, errAt(ref, "unknown attribute", "%s cannot be looked up without a module resolver", ref.String())
	}
	resolved, err := c.resolver.ResolveModule(context.Background(), moduleName.URI())
	if err != nil || resolved == nil {
		return nil, errAt(ref, "unknown attribute", "%s names the module %s, which was not found", ref.String(), moduleName)
	}
	if err := c.ensureAnalyzed(resolved, true); err != nil {
		return nil, err
	}
	if _, ok := c.moduleGlobals[moduleName.URI()]; !ok {
		return nil, errAt(ref, "module not loaded", "%s is not part of this program", moduleName.URI())
	}
	if err := c.compileModuleIfNeeded(moduleName.URI(), c.moduleGlobals[moduleName.URI()]); err != nil {
		return nil, err
	}
	found, err := resolveStaticRefInTable(resolved.Symbols, tail, true)
	if err != nil {
		return nil, err
	}
	return requireAttributeSymbol(found, ref)
}

func requireAttributeSymbol(sym *ast.Symbol, ref ast.StaticReference) (*ast.Symbol, error) {
	if sym == nil || sym.Decl == nil {
		return nil, errAt(ref, "unknown attribute", "%s", ref.String())
	}
	sym = sym.Original()
	if _, ok := sym.Decl.(*ast.DeclAttr); !ok {
		return nil, errAt(ref, "not an attribute", "%s is a %T, which cannot be written with @", ref.String(), sym.Decl)
	}
	return sym, nil
}

func resolveStaticRefInTable(table *ast.SymbolTable, ref ast.StaticReference, requireExport bool) (*ast.Symbol, error) {
	if table == nil || len(ref) == 0 {
		return nil, errAt(ref, "invalid reference", "%s names nothing to look in", ref.String())
	}
	cur := table
	for i := range ref {
		part := ref[i]
		sym := cur.Symbols[part.Value]
		if sym == nil || sym.Decl == nil {
			return nil, errAt(ref, "unknown reference", "%s", ref.String())
		}
		if requireExport && sym.Decl.ExportScope() != ast.ExportScopePublic {
			return nil, errAt(ref, "unknown reference", "%s", ref.String())
		}
		if i+1 < len(ref) {
			if sym.ChildTable == nil {
				return nil, errAt(ref, "unknown reference", "%s", ref.String())
			}
			cur = sym.ChildTable
		}
		if i+1 == len(ref) {
			return sym.Original(), nil
		}
	}
	return nil, errAt(ref, "unknown reference", "%s", ref.String())
}

// resolveUnionMemberTypeIds resolves the member type constant IDs for a union declaration.
func (c *Compiler) resolveUnionMemberTypeIds(decl *ast.DeclUnion) ([]runtime.TypeId, error) {
	symbols := c.currentSymbols()
	if symbols == nil {
		return nil, errInvariant(decl, "the union %s is being compiled with no symbols in scope", decl.Name)
	}
	memberTypeIds := make([]runtime.TypeId, 0, len(decl.Members))
	for _, member := range decl.Members {
		memberSym := symbols.LookupRef(member.Member)
		if memberSym == nil || memberSym.Decl == nil {
			return nil, errAt(member.Member, "unknown union member", "%s", member.Member.String())
		}
		memberSym = memberSym.Original()
		// If the symbol is a DeclImportMember, resolve the actual type from the imported module.
		if importMember, ok := memberSym.Decl.(ast.DeclImportMember); ok {
			resolved, err := c.resolveTypeSymbolFromImport(importMember)
			if err != nil {
				return nil, errAt(member.Member, "unknown union member", "%s: %s", member.Member.String(), err)
			}
			memberSym = resolved
		}
		if memberSym.ConstantId == nil {
			return nil, errInvariant(member.Member, "the union member %s was never given a constant slot, which the analyzer assigns before compilation", member.Member.String())
		}
		memberTypeIds = append(memberTypeIds, runtime.TypeId(*memberSym.ConstantId))
	}
	return memberTypeIds, nil
}

// resolveTypeSymbolFromImport follows a DeclImportMember to the actual symbol in the imported
// module and returns it. The module is analyzed if not yet analyzed.
func (c *Compiler) resolveTypeSymbolFromImport(importMember ast.DeclImportMember) (*ast.Symbol, error) {
	if c.resolver == nil {
		return nil, errInvariant(importMember, "an imported type cannot be resolved without a module resolver, which the compiler is always built with")
	}
	module, err := c.resolver.ResolveModule(context.Background(), importMember.ModuleName.URI())
	if err != nil || module == nil {
		return nil, errAt(importMember, "unknown module", "%s", importMember.ModuleName)
	}
	if err := c.ensureAnalyzed(module, true); err != nil {
		return nil, err
	}
	ref := ast.StaticReference{importMember.Name}
	sym, err := resolveStaticRefInTable(module.Symbols, ref, true)
	if err != nil {
		return nil, errAt(importMember, "unknown import", "%s: %s", importMember.Name.Value, err)
	}
	return sym, nil
}

func (c *Compiler) findImportedModuleByPrefix(module *ast.ContextModule, ref ast.StaticReference) ast.ModuleName {
	if module == nil || len(ref) < 2 {
		return nil
	}
	visitImport := func(sym *ast.Symbol) ast.ModuleName {
		if sym == nil || sym.Decl == nil {
			return nil
		}
		imp, ok := sym.Decl.(*ast.DeclImport)
		if !ok {
			return nil
		}
		if hasStaticPrefix(ref, ast.StaticReference(imp.ModuleName)) {
			return imp.ModuleName
		}
		return nil
	}
	for _, sym := range module.Symbols.Symbols {
		if match := visitImport(sym); match != nil {
			return match
		}
	}
	for _, file := range module.Files {
		if file == nil || file.Symbols == nil {
			continue
		}
		for _, sym := range file.Symbols.Symbols {
			if match := visitImport(sym); match != nil {
				return match
			}
		}
	}
	return nil
}

func hasStaticPrefix(ref ast.StaticReference, prefix ast.StaticReference) bool {
	if len(prefix) > len(ref) {
		return false
	}
	for i := range prefix {
		if ref[i].Value != prefix[i].Value {
			return false
		}
	}
	return true
}

// compileStmtAssign compiles an assignment statement.
//
// Stack conventions used by the emitted instructions:
//   - SetField: expects [..., val, obj] — pops obj (top), then pops val, sets obj.field = val
//   - SetIndex: expects [..., val, target, index] — pops index (top), target, then val; sets target[index] = val
func (c *Compiler) compileStmtAssign(node *ast.StmtAssign) error {
	switch target := node.Target.(type) {
	case *ast.ExprIdentifier:
		return c.compileIdentAssign(target, node.Op, node.Value)
	case *ast.ExprMemberAccess:
		return c.compileMemberAssign(target, node.Op, node.Value)
	case *ast.ExprIndexAccess:
		return c.compileIndexAssign(target, node.Op, node.Value)
	default:
		return errAt(node, "cannot be assigned to", "%T is not something a value can be stored in", node.Target)
	}
}

// compileIdentAssign compiles assignment to a simple identifier lvalue.
// Returns an error if the identifier resolves to a const or parameter binding.
func (c *Compiler) compileIdentAssign(target *ast.ExprIdentifier, augOp token.TokenType, value ast.Expr) error {
	symbol := target.Symbol
	if symbol == nil {
		syms := c.currentSymbols()
		if syms != nil {
			symbol = syms.LookupIdentifier(target.Name)
		}
	}
	if symbol == nil || symbol.Decl == nil {
		return errUndefinedIdentifier(target)
	}

	switch symbol.Decl.(type) {
	case *ast.DeclConstant:
		return errAt(target, "cannot assign to a constant", "%s was declared with const", target.Name)
	case *ast.DeclParameter:
		return errAt(target, "cannot assign to a parameter", "%s", target.Name)
	}

	if augOp != "" {
		// Read current value, compile rhs, apply op, then write.
		if err := c.Compile(target); err != nil {
			return err
		}
		if err := c.Compile(value); err != nil {
			return err
		}
		if err := c.emitBinaryOp(value, augOp); err != nil {
			return err
		}
	} else {
		if err := c.Compile(value); err != nil {
			return err
		}
	}

	sym := symbol.Original()
	switch symbol.Decl.(type) {
	case *ast.DeclVariable, *ast.DeclForBinding:
		// FreeScope without LocalId: actual closure capture or global.
		if symbol.Scope == ast.FreeScope && symbol.LocalId == nil {
			orig := symbol.Original()
			if orig.GlobalId != nil {
				c.emit(op.SetGlobal, *orig.GlobalId)
				return nil
			}
			return c.compileFreeAssign(symbol)
		}

		// Use the symbol's own LocalId (covers promoted locals too).
		if symbol.LocalId != nil {
			if _, isVar := symbol.Decl.(*ast.DeclVariable); isVar && symbol.IsCaptured {
				c.emit(op.SetLocalCell, *symbol.LocalId)
			} else {
				c.emit(op.SetLocal, *symbol.LocalId)
			}
			return nil
		}

		if sym.LocalId != nil {
			if _, isVar := sym.Decl.(*ast.DeclVariable); isVar && sym.IsCaptured {
				c.emit(op.SetLocalCell, *sym.LocalId)
			} else {
				c.emit(op.SetLocal, *sym.LocalId)
			}
			return nil
		}
		if sym.GlobalId != nil {
			c.emit(op.SetGlobal, *sym.GlobalId)
			return nil
		}
		return errAt(target, "unresolved variable", "%s has no storage assigned to it", target.Name)
	case *ast.DeclImport, *ast.DeclModule, ast.DeclImportMember:
		return errAt(target, "cannot assign to a module", "%s names an import", target.Name)
	default:
		return errAt(target, "cannot be assigned to", "%s is a %T", target.Name, symbol.Decl)
	}
}

// compileMemberAssign compiles assignment to a member access lvalue (obj.field = val).
// Stack layout for SetField: [..., val, obj] — obj is on top, val beneath it.
func (c *Compiler) compileMemberAssign(target *ast.ExprMemberAccess, augOp token.TokenType, value ast.Expr) error {
	nameConst := c.addConstant(c.plugins.Prelude().String(target.Property.Value))

	if augOp != "" {
		// Evaluate the object once and cache it in a temp local to avoid double evaluation.
		objLocal := c.allocateTempLocal()
		if err := c.Compile(target.Target); err != nil {
			return err
		}
		c.emit(op.SetLocal, objLocal)
		// Read old value: GetLocal obj, GetField → old_val
		c.emit(op.GetLocal, objLocal)
		c.emit(op.GetField, nameConst)
		// Push rhs and apply op → result on stack
		if err := c.Compile(value); err != nil {
			return err
		}
		if err := c.emitBinaryOp(value, augOp); err != nil {
			return err
		}
		// Push cached obj for the write: [..., result, obj]
		c.emit(op.GetLocal, objLocal)
	} else {
		// Push val first, then obj: [..., val, obj]
		if err := c.Compile(value); err != nil {
			return err
		}
		if err := c.Compile(target.Target); err != nil {
			return err
		}
	}
	c.emit(op.SetField, nameConst)
	return nil
}

// compileIndexAssign compiles assignment to an index access lvalue (target[index] = val).
// Stack layout for SetIndex: [..., val, target, index] — index on top, target beneath, val at bottom.
func (c *Compiler) compileIndexAssign(target *ast.ExprIndexAccess, augOp token.TokenType, value ast.Expr) error {
	if augOp != "" {
		// Evaluate target and index once; cache in temp locals to avoid double evaluation.
		targetLocal := c.allocateTempLocal()
		indexLocal := c.allocateTempLocal()
		if err := c.Compile(target.Target); err != nil {
			return err
		}
		c.emit(op.SetLocal, targetLocal)
		if err := c.Compile(target.IndexExpr); err != nil {
			return err
		}
		c.emit(op.SetLocal, indexLocal)
		// Read old value: GetLocal target, GetLocal index, GetIndex → old_val
		c.emit(op.GetLocal, targetLocal)
		c.emit(op.GetLocal, indexLocal)
		c.emit(op.GetIndex)
		// Push rhs and apply op → result
		if err := c.Compile(value); err != nil {
			return err
		}
		if err := c.emitBinaryOp(value, augOp); err != nil {
			return err
		}
		// Push cached target and index for the write
		c.emit(op.GetLocal, targetLocal)
		c.emit(op.GetLocal, indexLocal)
	} else {
		// Stack: [..., val, target, index]
		if err := c.Compile(value); err != nil {
			return err
		}
		if err := c.Compile(target.Target); err != nil {
			return err
		}
		if err := c.Compile(target.IndexExpr); err != nil {
			return err
		}
	}
	c.emit(op.SetIndex)
	return nil
}

// emitBinaryOp emits the opcode for an arithmetic binary operator.
// node is the value being combined, carried so that an operator this cannot emit says where it was written.
func (c *Compiler) emitBinaryOp(node ast.Node, op_ token.TokenType) error {
	switch op_ {
	case token.PLUS:
		c.emit(op.Add)
	case token.MINUS:
		c.emit(op.Sub)
	case token.ASTERISK:
		c.emit(op.Mul)
	case token.SLASH:
		c.emit(op.Div)
	case token.PERCENT:
		c.emit(op.Mod)
	default:
		return errUnimplemented(node, "the augmented assignment operator %s is not handled", op_)
	}
	return nil
}

// ResolveModuleSymbol implements runtime.BindContext.
// It resolves a symbol by name from a module identified by its suffix.
func (c *Compiler) ResolveModuleSymbol(moduleName string, symbolName string) *ast.Symbol {
	if c.resolver == nil {
		return nil
	}
	// A URI matching outright is the module meant; the suffix is only for one carried under a package name, e.g. code.knabel.dev...zirric.prelude.
	// Both are searched in a fixed order, because more than one module can end in the same name — future.prelude ends in .prelude too — and every module's table carries prelude's own members as imports, so whichever answered first used to decide which Printable an extern was bound to.
	candidates := make([]registry.LogicalURI, 0, len(c.moduleGlobals))
	for uri := range c.moduleGlobals {
		uriStr := string(uri)
		if uriStr != moduleName && !strings.HasSuffix(uriStr, "."+moduleName) {
			continue
		}
		candidates = append(candidates, uri)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if (candidates[i] == registry.LogicalURI(moduleName)) != (candidates[j] == registry.LogicalURI(moduleName)) {
			return candidates[i] == registry.LogicalURI(moduleName)
		}
		return candidates[i] < candidates[j]
	})

	for _, uri := range candidates {
		mod, err := c.resolver.ResolveModule(context.Background(), uri)
		if err != nil || mod == nil || mod.Symbols == nil {
			continue
		}
		if sym, ok := mod.Symbols.Symbols[symbolName]; ok && sym != nil {
			return sym.Original()
		}
	}
	return nil
}

// MainPackageModules implements runtime.BindContext.
// The result is memoized: the project package's identity does not change during a compilation, and recomputing it would restart the whole compile pass for every extern that asks.
func (c *Compiler) MainPackageModules() (string, map[string]int) {
	if c.mainPackage != nil {
		return c.mainPackage.name, c.mainPackage.globals
	}
	lister, ok := c.resolver.(resolver.MainPackageLister)
	if !ok || c.analyzer == nil {
		return "", nil
	}

	info := &mainPackageModules{name: lister.MainPackageName(), globals: map[string]int{}}
	// Published before any module is compiled, because compiling one can reach a module that binds these same externs — a project vendoring its own copy of reflect.packages — and that must observe this pass rather than start a competing one.
	c.mainPackage = info

	uris, err := lister.MainPackageModules(context.Background())
	if err != nil {
		return info.name, info.globals
	}

	// Every slot is reserved before anything is compiled, so the map a reentrant caller sees is already complete.
	reserved := make([]registry.LogicalURI, 0, len(uris))
	for _, uri := range uris {
		module, err := c.resolver.ResolveModule(context.Background(), uri)
		if err != nil || module == nil {
			continue
		}
		canonical := module.Name
		if c.isShadowedByLoadedModule(info.name, module) || c.isEntryModule(module) {
			continue
		}
		id, ok := c.moduleGlobals[canonical]
		if !ok {
			id = c.analyzer.ReserveModuleGlobal(canonical)
			c.moduleGlobals[canonical] = id
		}
		c.ensureGlobalSlot(id)
		if _, seen := info.globals[string(canonical)]; seen {
			continue
		}
		info.globals[string(canonical)] = id
		reserved = append(reserved, canonical)
	}

	for _, canonical := range reserved {
		// A module of this package that does not compile is not offered, since it could never be loaded, and the failure is remembered so that it is reported rather than quietly leaving a hole in what the package contains.
		if err := c.compileModuleIfNeeded(canonical, info.globals[string(canonical)]); err != nil {
			delete(info.globals, string(canonical))
			c.mainPackageErrs = append(c.mainPackageErrs, fmt.Errorf("module %s of this package does not compile: %w", canonical, err))
		}
	}
	return info.name, info.globals
}

// MainPackageCavefile implements runtime.BindContext.
// A resolver that cannot supply a manifest — the stubs used in tests — reports none rather than an empty one, so that a program can tell "no Cavefile" from "a Cavefile declaring nothing".
func (c *Compiler) MainPackageCavefile() (cavefile.Cavefile, bool) {
	provider, ok := c.resolver.(resolver.CavefileProvider)
	if !ok {
		return cavefile.Cavefile{}, false
	}
	return provider.MainPackageCavefile(), true
}

// MainPackageErrors returns the failures met while compiling the modules of the project's own package.
//
// Asking what a package contains compiles every module in it, and one that does not compile used to be dropped without a word — leaving a program running against a package quietly missing part of itself. They are reported instead, and compiling fails.
func (c *Compiler) MainPackageErrors() []error {
	return c.mainPackageErrs
}

// isShadowedByLoadedModule reports whether a package module is unreachable because its unqualified name resolves to a different module, as happens when a project carries its own copy of a standard library module.
// Such a module can never be imported, and compiling it anyway is actively harmful when it redeclares core types, since the program would then hold two incompatible definitions of them.
func (c *Compiler) isShadowedByLoadedModule(packageName string, module *ast.ContextModule) bool {
	if module == nil || packageName == "" {
		return false
	}
	unqualified := strings.TrimPrefix(string(module.Name), packageName+".")
	if unqualified == string(module.Name) {
		return false
	}
	// Asking the resolver rather than the compiler's own table keeps this independent of how far the current pass has progressed.
	other, err := c.resolver.ResolveModule(context.Background(), registry.LogicalURI(unqualified))
	if err != nil || other == nil {
		return false
	}
	return other != module
}

// isEntryModule reports whether module is the program being run, either by name or because it was built from the same source file.
// Such a module must never be loaded as a package member: the running script is already executing, so resolving its global would either re-enter an initialization in progress or, when the entry file also belongs to a directory module of its own, run the whole script a second time.
func (c *Compiler) isEntryModule(module *ast.ContextModule) bool {
	if c.entryModule == nil || module == nil {
		return false
	}
	if module == c.entryModule || module.Name == c.entryModule.Name {
		return true
	}
	for _, entryFile := range c.entryModule.Files {
		if entryFile == nil || entryFile.Path == "" {
			continue
		}
		for _, file := range module.Files {
			if file != nil && file.Path == entryFile.Path {
				return true
			}
		}
	}
	return false
}
