package compiler

import (
	"fmt"
	"math"

	"code.knabel.dev/zirric-lang/zirric/ast"
	"code.knabel.dev/zirric-lang/zirric/op"
	"code.knabel.dev/zirric-lang/zirric/runtime"
	"code.knabel.dev/zirric-lang/zirric/token"
)

const (
	// A temporary address that acts placeholder.
	// Should be replaced by the actual address once known.
	placeholderJumpAddress = math.MinInt
)

func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	case *ast.ContextModule:
		c.enterScope(node.Symbols)

		for _, src := range node.Files {
			err := c.Compile(src)
			if err != nil {
				return err
			}
		}

		c.leaveScope()

		return nil
	case *ast.SourceFile:
		c.enterScope(node.Symbols)

		for _, sym := range node.Symbols.Symbols {
			err := c.reserveSymbol(sym)
			if err != nil {
				return err
			}
		}

		for _, sym := range node.Symbols.Symbols {
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

		return nil

	case *ast.DeclVariable, *ast.DeclFunc:
		sym := c.scopes[c.scopeIdx].symbols.Insert(node.(ast.Decl))
		return c.compileSymbol(sym)

	case *ast.StmtExpr:
		err := c.Compile(node.Expr)
		if err != nil {
			return err
		}
		c.emit(op.Pop)
		return nil
	case ast.StmtIf:
		return c.compileStmtIf(node)

	case ast.ExprIf:
		return c.compileExprIf(node)
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
	case *ast.ExprNull:
		c.emit(op.ConstNull)
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
		symbol := c.scopes[c.scopeIdx].symbols.LookupIdentifier(node.Name)
		if symbol == nil {
			return fmt.Errorf("undefined identifier %q", node.Name)
		}
		switch symbol.Decl.(type) {
		case *ast.DeclFunc, *ast.DeclData, *ast.DeclEnum, *ast.DeclExternFunc, *ast.DeclAnnotation:
			sym := symbol.Original()
			if sym.ConstantId == nil {
				return fmt.Errorf("identifier %q has no constant id", node.Name)
			}
			c.emit(op.Const, *sym.ConstantId)
			return nil

		case *ast.DeclVariable:
			sym := symbol.Original()

			if sym.LocalId != nil {
				c.emit(op.GetLocal, *sym.LocalId)
				return nil
			}
			if sym.GlobalId != nil {
				c.emit(op.GetGlobal, *sym.GlobalId)
				return nil
			}

			return fmt.Errorf("variable %q has no local or global id", node.Name)

		case *ast.DeclParameter:
			c.emit(op.GetLocal, *symbol.LocalId)
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
			c.emit(op.ConstNull)
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
		id := len(c.constants)
		c.constants = append(c.constants, nil)
		sym.ConstantId = &id
		return nil

	case *ast.DeclVariable:
		switch decl.ExportScope() {
		case ast.ExportScopeInternal, ast.ExportScopePublic:
			id := c.addGlobal(nil)
			sym.GlobalId = &id
			return nil

		case ast.ExportScopeLocal:
			id := len(c.scopes[c.scopeIdx].locals)
			c.scopes[c.scopeIdx].locals = append(c.scopes[c.scopeIdx].locals, sym)
			sym.LocalId = &id
			return nil

		default:
			return fmt.Errorf("unknown variable scope %v", sym.Scope)
		}

	case *ast.DeclParameter:
		if sym.LocalId != nil {
			panic("parameter already has a local id")
		}
		id := len(c.scopes[c.scopeIdx].locals)
		c.scopes[c.scopeIdx].locals = append(c.scopes[c.scopeIdx].locals, sym)
		sym.LocalId = &id
		return nil

	case *ast.DeclData, *ast.DeclEnum, *ast.DeclExternFunc, *ast.DeclAnnotation:
		id := len(c.constants)
		c.constants = append(c.constants, nil)
		sym.ConstantId = &id
		return nil

	default:
		return fmt.Errorf("unknown declaration %T", decl)
	}
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

		c.constants[*sym.ConstantId] = dt

		return nil

	case *ast.DeclFunc:
		c.enterScope(decl.Impl.Symbols)

		for _, child := range decl.Impl.Symbols.Symbols {
			if child.Decl == nil {
				continue
			}
			err := c.reserveSymbol(child)
			if err != nil {
				return err
			}
		}
		err := c.compileBlock(decl.Impl.Impl)
		if err != nil {
			return err
		}
		scope := c.leaveScope()

		c.constants[*sym.ConstantId] = runtime.MakeCompiledFunction(
			scope.Instructions,
			len(decl.Impl.Parameters),
			sym,
		)

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
			err := c.Compile(decl.Value)
			if err != nil {
				return err
			}

			c.scopes[c.scopeIdx].locals[*sym.LocalId] = sym

			c.emit(op.SetLocal, *sym.LocalId)

			return nil

		default:
			return fmt.Errorf("unknown variable scope %v", sym.Scope)
		}

	default:
		return fmt.Errorf("unknown declaration %T", decl)
	}
}
