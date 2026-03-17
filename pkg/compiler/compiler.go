package compiler

import (
	"context"
	"fmt"
	"math"
	"sort"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/op"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

const (
	// A temporary address that acts placeholder.
	// Should be replaced by the actual address once known.
	placeholderJumpAddress = math.MinInt
)

func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	case *ast.ContextModule:
		if err := c.ensureAnalyzed(node, true); err != nil {
			return err
		}

		moduleId := c.reserveGlobalModule(node.Name)
		return c.compileContextModule(node, moduleId)
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
					return fmt.Errorf("module declaration requires context module")
				}
			}
		}

		c.enterScope(node.Symbols)

		for _, sym := range node.Symbols.Symbols {
			if sym.Decl == nil {
				return fmt.Errorf("undeclared symbol %q at %s:%d", sym.Name, sym.Usages[0].Node.TokenLiteral().Source.File, sym.Usages[0].Node.TokenLiteral().Source.Offset)
			}
		}

		fileSymbols := c.sourceFileSymbols(node)
		for _, sym := range fileSymbols {
			if sym.Decl == nil {
				return fmt.Errorf("undeclared symbol %q at %s:%d", sym.Name, sym.Usages[0].Node.TokenLiteral().Source.File, sym.Usages[0].Node.TokenLiteral().Source.Offset)
			}
			err := c.reserveSymbol(sym)
			if err != nil {
				return err
			}
		}

		for _, sym := range fileSymbols {
			if sym.Decl == nil {
				return fmt.Errorf("undeclared symbol %q at %s:%d", sym.Name, sym.Usages[0].Node.TokenLiteral().Source.File, sym.Usages[0].Node.TokenLiteral().Source.Offset)
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

		// Carry any temp locals allocated in the inlined scope up to the parent,
		// so that the parent's LocalsCount() covers the SetLocal/GetLocal indices.
		for len(c.scopes[c.scopeIdx].locals) < len(scope.locals) {
			c.scopes[c.scopeIdx].locals = append(c.scopes[c.scopeIdx].locals, nil)
		}

		return nil

	case *ast.DeclVariable, *ast.DeclConstant, *ast.DeclFunc:
		symbols := c.currentSymbols()
		if symbols == nil {
			return fmt.Errorf("missing symbols for declaration %T", node)
		}
		sym := symbols.LookupIdentifier(node.(ast.Decl).DeclName())
		if sym == nil || sym.Decl == nil {
			return fmt.Errorf("declaration %q not registered by analyzer", node.(ast.Decl).DeclName().Value)
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
	case ast.StmtFor:
		return c.compileStmtFor(node)
	case ast.StmtBreak:
		return c.compileStmtBreak()
	case ast.StmtContinue:
		return c.compileStmtContinue()

	case ast.ExprIf:
		return c.compileExprIf(node)
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
				return fmt.Errorf("undefined identifier %q", node.Name)
			}
			symbol = symbols.LookupIdentifier(node.Name)
		}
		if symbol == nil || symbol.Decl == nil {
			return fmt.Errorf("undefined identifier %q", node.Name)
		}
		switch symbol.Decl.(type) {
		case *ast.DeclFunc, *ast.DeclData, *ast.DeclUnion, *ast.DeclExternFunc, *ast.DeclExternType, *ast.DeclAttr:
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
				return fmt.Errorf("identifier %q has no constant id", node.Name)
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

			return fmt.Errorf("variable %q has no local or global id", node.Name)

		case *ast.DeclParameter:
			if symbol.Scope == ast.FreeScope && symbol.LocalId == nil {
				return c.compileFreeIdentifier(symbol)
			}
			c.emit(op.GetLocal, *symbol.LocalId)
			return nil

		case *ast.DeclImport:
			sym := symbol.Original()
			if sym.GlobalId == nil {
				return fmt.Errorf("module %q has no global id", node.Name)
			}
			c.emit(op.GetGlobal, *sym.GlobalId)
			return nil
		case *ast.DeclModule:
			sym := symbol.Original()
			if sym.GlobalId == nil {
				return fmt.Errorf("module %q has no global id", node.Name)
			}
			c.emit(op.GetGlobal, *sym.GlobalId)
			return nil

		default:
			return fmt.Errorf("identifier %q has unknown declaration type %T", node.Name, symbol.Decl)
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
		return fmt.Errorf("unknown ast node %T", node)
	}
}

func (c *Compiler) reserveSymbol(sym *ast.Symbol) error {
	switch decl := sym.Decl.(type) {
	case *ast.DeclFunc:
		if sym.ConstantId == nil {
			return fmt.Errorf("function %q has no constant id", decl.Name.Value)
		}
		c.ensureConstantSlot(*sym.ConstantId)
		return nil

	case *ast.DeclVariable:
		switch decl.ExportScope() {
		case ast.ExportScopeInternal, ast.ExportScopePublic:
			if sym.GlobalId == nil {
				return fmt.Errorf("global %q has no global id", decl.Name.Value)
			}
			c.ensureGlobalSlot(*sym.GlobalId)
			return nil

		case ast.ExportScopeLocal:
			if sym.LocalId == nil {
				return fmt.Errorf("local %q has no local id", decl.Name.Value)
			}
			c.ensureLocalSlot(*sym.LocalId)
			c.scopes[c.scopeIdx].locals[*sym.LocalId] = sym
			return nil

		default:
			return fmt.Errorf("unknown variable scope %v", sym.Scope)
		}

	case *ast.DeclConstant:
		switch decl.ExportScope() {
		case ast.ExportScopeInternal, ast.ExportScopePublic:
			if sym.GlobalId == nil {
				return fmt.Errorf("global %q has no global id", decl.Name.Value)
			}
			c.ensureGlobalSlot(*sym.GlobalId)
			return nil

		case ast.ExportScopeLocal:
			if sym.LocalId == nil {
				return fmt.Errorf("local %q has no local id", decl.Name.Value)
			}
			c.ensureLocalSlot(*sym.LocalId)
			c.scopes[c.scopeIdx].locals[*sym.LocalId] = sym
			return nil

		default:
			return fmt.Errorf("unknown variable scope %v", sym.Scope)
		}

	case *ast.DeclParameter:
		if sym.LocalId == nil {
			return fmt.Errorf("parameter %q has no local id", decl.Name.Value)
		}
		c.ensureLocalSlot(*sym.LocalId)
		c.scopes[c.scopeIdx].locals[*sym.LocalId] = sym
		return nil

	case *ast.DeclForBinding:
		if sym.LocalId == nil {
			return fmt.Errorf("binding %q has no local id", decl.Name.Value)
		}
		c.ensureLocalSlot(*sym.LocalId)
		c.scopes[c.scopeIdx].locals[*sym.LocalId] = sym
		return nil

	case *ast.DeclData, *ast.DeclUnion, *ast.DeclExternFunc, *ast.DeclExternType, *ast.DeclAttr:
		if sym.ConstantId == nil {
			return fmt.Errorf("declaration %q has no constant id", sym.Name)
		}
		c.ensureConstantSlot(*sym.ConstantId)
		return nil

	case *ast.DeclModule:
		if sym.GlobalId == nil {
			return fmt.Errorf("module %q has no global id", decl.Name.Value)
		}
		c.ensureGlobalSlot(*sym.GlobalId)
		c.moduleGlobals[c.currentSymbols().Module().Name] = *sym.GlobalId
		return nil

	case *ast.DeclImport:
		if sym.GlobalId == nil {
			return fmt.Errorf("import %q has no global id", decl.ModuleName)
		}
		c.ensureGlobalSlot(*sym.GlobalId)
		return nil

	default:
		return fmt.Errorf("unknown declaration %T", decl)
	}
}

func (c *Compiler) ensureAnalyzed(module *ast.ContextModule, reserveModule bool) error {
	if module == nil {
		return fmt.Errorf("analysis error: module is nil")
	}
	if _, ok := c.analyzed[module]; ok {
		return nil
	}
	c.analyzed[module] = struct{}{}
	if c.analyzer == nil {
		return nil
	}
	errs, _ := c.analyzer.Analyze(module, reserveModule)
	if c.resolver != nil && c.resolver.MainModule() == module && c.scopes[0].symbols == nil {
		c.scopes[0].symbols = module.Symbols
	}
	if len(errs) == 0 {
		return nil
	}
	err := errs[0]
	return fmt.Errorf("%s", err.Error())
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
		lastIndex := len(jumpEnds) - 1

		if c.isLastInstruction(op.Pop) {
			c.removeLastInstruction()
		}

		jumpEnds[lastIndex] = jumpNext
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
		return fmt.Errorf("collection for loops require binding and collection")
	}
	symbols := c.currentSymbols()
	if symbols == nil {
		return fmt.Errorf("collection binding %q missing symbols", node.CollectionIdent.Value)
	}
	sym := symbols.LookupIdentifier(*node.CollectionIdent)
	if sym == nil {
		return fmt.Errorf("collection binding %q missing symbol", node.CollectionIdent.Value)
	}
	bindingLocal, err := c.requireLocalId(sym)
	if err != nil {
		return err
	}

	collectionLocal := c.allocateTempLocal()
	indexLocal := c.allocateTempLocal()
	zeroConst := c.addConstant(c.plugins.Prelude().Int(0))
	oneConst := c.addConstant(c.plugins.Prelude().Int(1))

	err = c.Compile(node.CollectionExpr)
	if err != nil {
		return err
	}
	c.emit(op.SetLocal, collectionLocal)
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
	err = c.compileLoopBlock(node.Body, &continueJumps, &breakJumps)
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
		lastIndex := len(jumpEnds) - 1

		if c.isLastInstruction(op.Pop) {
			c.removeLastInstruction()
		}

		jumpEnds[lastIndex] = jumpNext
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
		return fmt.Errorf("collection for expressions require binding and collection")
	}
	symbols := c.currentSymbols()
	if symbols == nil {
		return fmt.Errorf("collection binding %q missing symbols", node.CollectionIdent.Value)
	}
	sym := symbols.LookupIdentifier(*node.CollectionIdent)
	if sym == nil {
		return fmt.Errorf("collection binding %q missing symbol", node.CollectionIdent.Value)
	}
	bindingLocal, err := c.requireLocalId(sym)
	if err != nil {
		return err
	}

	collectionLocal := c.allocateTempLocal()
	indexLocal := c.allocateTempLocal()
	zeroConst := c.addConstant(c.plugins.Prelude().Int(0))
	oneConst := c.addConstant(c.plugins.Prelude().Int(1))

	err = c.Compile(node.CollectionExpr)
	if err != nil {
		return err
	}
	c.emit(op.SetLocal, collectionLocal)
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
	err = c.compileExprForBlock(node.Body, arrayLocal, &continueJumps, &breakJumps)
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
	c.emit(op.GetLocal, arrayLocal)
	return nil
}

func (c *Compiler) compileExprForBlock(body ast.ExprForBody, arrayLocal int, continueJumps *[]int, breakJumps *[]int) error {
	symbols := c.currentSymbols()
	if symbols == nil {
		return fmt.Errorf("expr-for body missing symbols")
	}
	for _, decl := range body.Decls {
		name := decl.DeclName()
		sym := symbols.LookupIdentifier(name)
		if sym == nil {
			return fmt.Errorf("expr-for declaration %q missing symbol", name.Value)
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
	default:
		return fmt.Errorf("expr-for allows only expression, if, break, or continue statements")
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
		lastIndex := len(jumpEnds) - 1

		if c.isLastInstruction(op.Pop) {
			c.removeLastInstruction()
		}

		jumpEnds[lastIndex] = jumpNext
	}

	endPos = len(c.currentInstructions())
	for _, pos := range jumpEnds {
		c.changeOperand(pos, endPos)
	}
	return nil
}

func (c *Compiler) compileStmtBreak() error {
	return fmt.Errorf("break used outside of loop")
}

func (c *Compiler) compileStmtContinue() error {
	return fmt.Errorf("continue used outside of loop")
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
		return fmt.Errorf("unknown prefix operator %q", node.Operator.Literal)
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
		return fmt.Errorf("unknown infix operator %q", node.Operator.Literal)
	}
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

		c.constants[*sym.ConstantId] = dt

		return nil

	case *ast.DeclUnion:
		attributes, err := c.compileAttributeChain(decl.Attributes, c.currentSymbols())
		if err != nil {
			return err
		}
		ut := runtime.MakeUnionType(sym)
		_ = attributes // unions may carry attributes in future
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
		c.constants[*sym.ConstantId] = at
		return nil

	case *ast.DeclExternType:
		attributes, err := c.compileAttributeChain(decl.Attributes, c.currentSymbols())
		if err != nil {
			return err
		}
		c.constants[*sym.ConstantId] = runtime.SimpleType{Decl: sym, Attributes: attributes}
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
		extern, err := runtime.MakeExternFunc(sym, nil)
		if err != nil {
			return err
		}
		extern.Attributes = attributes
		extern.ParamAttributes = paramAttributes
		c.constants[*sym.ConstantId] = extern
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
		err = c.compileBlock(decl.Impl.Impl)
		if err != nil {
			return err
		}
		if !c.isLastInstruction(op.Return) {
			c.emit(op.ConstVoid)
			c.emit(op.Return)
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
		// Use Original() to access ConstantId: FreeScope copies created by
		// resolve_identifiers (before assignModuleIDs) have nil ConstantId.
		origSym := sym.Original()
		if origSym.ConstantId == nil {
			return fmt.Errorf("internal: DeclFunc %q has no ConstantId", sym.Name)
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
				return fmt.Errorf("local %q has no local id", decl.Name.Value)
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
			return fmt.Errorf("unknown variable scope %v", sym.Scope)
		}

	case *ast.DeclConstant:
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
				return fmt.Errorf("local %q has no local id", decl.Name.Value)
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
			return fmt.Errorf("unknown variable scope %v", sym.Scope)
		}

	case *ast.DeclParameter:
		return nil

	case *ast.DeclModule:
		return nil

	case *ast.DeclImport:
		if c.resolver == nil {
			return fmt.Errorf("module resolver is required to compile imports")
		}

		uri := decl.ModuleName.URI()
		if sym.GlobalId == nil {
			return fmt.Errorf("import %q has no global id", decl.ModuleName)
		}
		c.ensureGlobalSlot(*sym.GlobalId)
		c.moduleGlobals[uri] = *sym.GlobalId

		return c.compileModuleIfNeeded(uri, *sym.GlobalId)

	default:
		return fmt.Errorf("unknown declaration %T", decl)
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

	for _, sym := range module.Symbols.Symbols {
		if sym.Decl == nil {
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

	var allStatements []ast.Statement
	for _, src := range module.Files {
		allStatements = append(allStatements, src.Statements...)
	}
	if len(allStatements) > 0 {
		if err := c.compileInitFunction(allStatements, module.Symbols, ModuleSymbol(module)); err != nil {
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
	if c.resolver != nil && c.resolver.MainModule() == module {
		c.scopes[c.scopeIdx].Instructions = append(c.scopes[c.scopeIdx].Instructions, scope.Instructions...)
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
			return fmt.Errorf("undeclared symbol %q at %s:%d", sym.Name, sym.Usages[0].Node.TokenLiteral().Source.File, sym.Usages[0].Node.TokenLiteral().Source.Offset)
		}
	}

	fileSymbols := c.sourceFileSymbols(node)
	for _, sym := range fileSymbols {
		if sym.Decl == nil {
			return fmt.Errorf("undeclared symbol %q at %s:%d", sym.Name, sym.Usages[0].Node.TokenLiteral().Source.File, sym.Usages[0].Node.TokenLiteral().Source.Offset)
		}
		if err := c.reserveSymbol(sym); err != nil {
			c.leaveScope()
			return err
		}
	}

	for _, sym := range fileSymbols {
		if err := c.compileSymbol(sym); err != nil {
			c.leaveScope()
			return err
		}
	}

	scope := c.leaveScope()
	c.scopes[c.scopeIdx].Instructions = append(c.scopes[c.scopeIdx].Instructions, scope.Instructions...)
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
		return -1, fmt.Errorf("source file has no symbol table: ensure it is analyzed before compiling")
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
			return fail(fmt.Errorf("undeclared symbol %q", sym.Name))
		}
	}

	fileSymbols := c.sourceFileSymbols(node)
	for _, sym := range fileSymbols {
		if sym.Decl == nil {
			c.leaveScope()
			return fail(fmt.Errorf("undeclared symbol %q", sym.Name))
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
	moduleNameId := c.addConstant(c.plugins.Prelude().String(string(module.Name)))
	c.emit(op.Module, moduleNameId)
	return nil
}

func (c *Compiler) emitModuleExport(sym *ast.Symbol) error {
	switch sym.Decl.(type) {
	case *ast.DeclFunc, *ast.DeclData, *ast.DeclUnion, *ast.DeclExternFunc, *ast.DeclExternType, *ast.DeclAttr:
		if sym.ConstantId == nil {
			return fmt.Errorf("identifier %q has no constant id", sym.Name)
		}
		c.emit(op.Const, *sym.ConstantId)
		return nil

	case *ast.DeclVariable, *ast.DeclConstant:
		if sym.GlobalId == nil {
			return fmt.Errorf("variable %q has no global id", sym.Name)
		}
		c.emit(op.GetGlobal, *sym.GlobalId)
		return nil

	default:
		return fmt.Errorf("unsupported module export %q (%T)", sym.Name, sym.Decl)
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
			return nil, fmt.Errorf("attribute %q has no constant id", annoSym.Name)
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

func (c *Compiler) compileAttributeInstance(inst *ast.DeclAttrInstance, sym *ast.Symbol, symbols *ast.SymbolTable) (int, error) {
	if inst == nil {
		return 0, fmt.Errorf("attribute instance is nil")
	}
	if sym == nil {
		return 0, fmt.Errorf("attribute symbol is nil")
	}
	if sym.ConstantId == nil {
		return 0, fmt.Errorf("attribute %q has no constant id", sym.Name)
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
	id := len(c.globals)
	c.globals = append(c.globals, scope)
	return id
}

func (c *Compiler) resolveAttributeReference(ref ast.StaticReference, symbols *ast.SymbolTable) (*ast.Symbol, error) {
	if len(ref) == 0 {
		return nil, fmt.Errorf("attribute reference is empty")
	}
	if symbols == nil {
		return nil, fmt.Errorf("missing symbols for attribute reference %q", ref.String())
	}
	if len(ref) == 1 {
		sym := symbols.LookupIdentifier(ref[0])
		if sym == nil || sym.Decl == nil {
			return nil, fmt.Errorf("unknown attribute %q", ref.String())
		}
		return requireAttributeSymbol(sym, ref.String())
	}

	head := ref[0]
	if sym := symbols.LookupIdentifier(head); sym != nil && sym.Decl != nil {
		if decl, ok := sym.Decl.(*ast.DeclImport); ok {
			return c.resolveAttributeFromImport(decl, ref[1:], ref.String())
		}
		if sym.ChildTable != nil {
			found, err := resolveStaticRefInTable(sym.ChildTable, ref[1:], false)
			if err != nil {
				return nil, err
			}
			return requireAttributeSymbol(found, ref.String())
		}
	}

	if moduleName := c.findImportedModuleByPrefix(symbols.Module(), ref); moduleName != nil {
		return c.resolveAttributeFromModuleName(moduleName, ref[len(moduleName):], ref.String())
	}

	return nil, fmt.Errorf("unknown attribute %q", ref.String())
}

func (c *Compiler) resolveAttributeFromImport(decl *ast.DeclImport, tail ast.StaticReference, refName string) (*ast.Symbol, error) {
	if decl == nil {
		return nil, fmt.Errorf("unknown attribute %q", refName)
	}
	return c.resolveAttributeFromModuleName(decl.ModuleName, tail, refName)
}

func (c *Compiler) resolveAttributeFromModuleName(moduleName ast.ModuleName, tail ast.StaticReference, refName string) (*ast.Symbol, error) {
	if c.resolver == nil {
		return nil, fmt.Errorf("module resolver is required for attribute %q", refName)
	}
	resolved, err := c.resolver.ResolveModule(context.Background(), moduleName.URI())
	if err != nil || resolved == nil {
		return nil, fmt.Errorf("unknown attribute %q", refName)
	}
	if err := c.ensureAnalyzed(resolved, true); err != nil {
		return nil, err
	}
	if _, ok := c.moduleGlobals[moduleName.URI()]; !ok {
		return nil, fmt.Errorf("module %q is not reserved", moduleName.URI())
	}
	if err := c.compileModuleIfNeeded(moduleName.URI(), c.moduleGlobals[moduleName.URI()]); err != nil {
		return nil, err
	}
	found, err := resolveStaticRefInTable(resolved.Symbols, tail, true)
	if err != nil {
		return nil, err
	}
	return requireAttributeSymbol(found, refName)
}

func requireAttributeSymbol(sym *ast.Symbol, refName string) (*ast.Symbol, error) {
	if sym == nil || sym.Decl == nil {
		return nil, fmt.Errorf("unknown attribute %q", refName)
	}
	sym = sym.Original()
	if _, ok := sym.Decl.(*ast.DeclAttr); !ok {
		return nil, fmt.Errorf("attribute %q does not refer to attribute type", refName)
	}
	return sym, nil
}

func resolveStaticRefInTable(table *ast.SymbolTable, ref ast.StaticReference, requireExport bool) (*ast.Symbol, error) {
	if table == nil || len(ref) == 0 {
		return nil, fmt.Errorf("invalid reference %q", ref.String())
	}
	cur := table
	for i := range ref {
		part := ref[i]
		sym := cur.Symbols[part.Value]
		if sym == nil || sym.Decl == nil {
			return nil, fmt.Errorf("unknown reference %q", ref.String())
		}
		if requireExport && sym.Decl.ExportScope() != ast.ExportScopePublic {
			return nil, fmt.Errorf("unknown reference %q", ref.String())
		}
		if i+1 < len(ref) {
			if sym.ChildTable == nil {
				return nil, fmt.Errorf("unknown reference %q", ref.String())
			}
			cur = sym.ChildTable
		}
		if i+1 == len(ref) {
			return sym.Original(), nil
		}
	}
	return nil, fmt.Errorf("unknown reference %q", ref.String())
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
		return fmt.Errorf("unsupported lvalue type %T", node.Target)
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
		return fmt.Errorf("undefined identifier %q", target.Name)
	}

	switch symbol.Decl.(type) {
	case *ast.DeclConstant:
		return fmt.Errorf("cannot assign to const %q", target.Name)
	case *ast.DeclParameter:
		return fmt.Errorf("cannot assign to parameter %q", target.Name)
	}

	if augOp != "" {
		// Read current value, compile rhs, apply op, then write.
		if err := c.Compile(target); err != nil {
			return err
		}
		if err := c.Compile(value); err != nil {
			return err
		}
		if err := c.emitBinaryOp(augOp); err != nil {
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
		return fmt.Errorf("variable %q has no local or global id", target.Name)
	case *ast.DeclImport, *ast.DeclModule:
		return fmt.Errorf("cannot assign to import/module %q", target.Name)
	default:
		return fmt.Errorf("cannot assign to %T %q", symbol.Decl, target.Name)
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
		if err := c.emitBinaryOp(augOp); err != nil {
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
		if err := c.emitBinaryOp(augOp); err != nil {
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
func (c *Compiler) emitBinaryOp(op_ token.TokenType) error {
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
		return fmt.Errorf("unsupported augmented assignment operator %q", op_)
	}
	return nil
}
