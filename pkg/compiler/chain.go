package compiler

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/op"
)

// pendingJump is a JumpIsType whose address is not known yet, kept together with the type it tests so the patched instruction can be rebuilt whole.
type pendingJump struct {
	pos    int
	typeId int
}

// compilePostfixChain compiles a member or index access together with everything it reads through.
//
// A `?.` anywhere in the chain short-circuits the whole of it, so the chain is compiled as one unit rather than link by link: this is the outermost link, which is where the None lands and where the result is turned back into an Option.
// A chain without any `?.` compiles exactly as before, one instruction per link.
func (c *Compiler) compilePostfixChain(expr ast.Expr) error {
	if !chainShortCircuits(expr) {
		return c.compileChainLink(expr)
	}

	// A nested chain (an index expression, say) has its own end to jump to, so the enclosing one's jumps are set aside rather than patched to the wrong place.
	enclosing := c.optionJumps
	c.optionJumps = nil
	defer func() { c.optionJumps = enclosing }()

	if err := c.compileChainLink(expr); err != nil {
		return err
	}

	symbols := c.currentSymbols()
	optionId, err := c.resolveBuiltinTypeConstantId(expr, "Option", symbols)
	if err != nil {
		return err
	}
	someId, err := c.resolveBuiltinTypeConstantId(expr, "Some", symbols)
	if err != nil {
		return err
	}
	c.emit(op.WrapOption, optionId, someId)

	end := len(c.currentInstructions())
	for _, jump := range c.optionJumps {
		c.changeOperand(jump.pos, end, jump.typeId)
	}
	return nil
}

// compileChainLink compiles one link of a postfix chain, recursing into its target so the whole chain stays in the one unit compilePostfixChain opened.
func (c *Compiler) compileChainLink(expr ast.Expr) error {
	switch node := expr.(type) {
	case *ast.ExprMemberAccess:
		if err := c.compileChainTarget(node.Target); err != nil {
			return err
		}
		if err := c.emitAccessGuard(node); err != nil {
			return err
		}
		c.emit(op.GetField, c.addConstant(c.plugins.Prelude().String(node.Property.Value)))
		return nil

	case *ast.ExprIndexAccess:
		if err := c.compileChainTarget(node.Target); err != nil {
			return err
		}
		// The index is pushed after the target, so a short-circuit above jumps before it and leaves nothing behind on the stack.
		if err := c.Compile(node.IndexExpr); err != nil {
			return err
		}
		c.emit(op.GetIndex)
		return nil
	}
	return c.Compile(expr)
}

func (c *Compiler) compileChainTarget(expr ast.Expr) error {
	switch expr.(type) {
	case *ast.ExprMemberAccess, *ast.ExprIndexAccess:
		return c.compileChainLink(expr)
	}
	return c.Compile(expr)
}

// emitAccessGuard emits what has to happen before a guarded field read: nothing for `.`, a short-circuit to the end of the chain for `?.`, and an early return for `!.`.
//
// Both guarded forms read the target as the prelude Option or Result first, so that a union of one's own carrying @AnyOption or @AnyResult is read exactly as the prelude's own is. What is left on the stack afterwards is the wrapper, which the caller unwraps before reading the field — that is the `.value` nobody has to write.
func (c *Compiler) emitAccessGuard(node *ast.ExprMemberAccess) error {
	access := node.Access()
	if access == ast.MemberAccessPlain {
		return nil
	}
	if access == ast.MemberAccessResult && c.funcDepth == 0 {
		return errAt(node, "!. must be inside a function", "it propagates an error by returning from the enclosing function, and a top-level statement has none")
	}
	if err := c.emitNormalize(node, access); err != nil {
		return err
	}

	symbols := c.currentSymbols()
	if access == ast.MemberAccessResult {
		errId, err := c.resolveBuiltinTypeConstantId(node, "Err", symbols)
		if err != nil {
			return err
		}
		c.emit(op.ReturnIsType, errId)
		return c.emitUnwrap(node)
	}

	noneId, err := c.resolveBuiltinTypeConstantId(node, "None", symbols)
	if err != nil {
		return err
	}
	c.optionJumps = append(c.optionJumps, pendingJump{
		pos:    c.emit(op.JumpIsType, placeholderJumpAddress, noneId),
		typeId: noneId,
	})
	return c.emitUnwrap(node)
}

// emitNormalize replaces the value on top of the stack with the prelude Option or Result standing for it, which is what lets @AnyOption and @AnyResult decide how a shape of one's own is read.
func (c *Compiler) emitNormalize(node ast.Node, access ast.MemberAccess) error {
	union, attribute, opcode := "Option", "AnyOption", op.AsOption
	if access == ast.MemberAccessResult {
		union, attribute, opcode = "Result", "AnyResult", op.AsResult
	}
	symbols := c.currentSymbols()
	unionId, err := c.resolveBuiltinTypeConstantId(node, union, symbols)
	if err != nil {
		return err
	}
	attributeId, err := c.resolveBuiltinTypeConstantId(node, attribute, symbols)
	if err != nil {
		return err
	}
	c.emit(opcode, unionId, attributeId)
	return nil
}

// emitUnwrap reads the value out of the Some or Ok on top of the stack, which is the field access a guarded read performs on the caller's behalf.
func (c *Compiler) emitUnwrap(node ast.Node) error {
	c.emit(op.GetField, c.addConstant(c.plugins.Prelude().String("value")))
	return nil
}

// chainShortCircuits reports whether reading this chain can end early with None.
// Only the spine counts: an index expression or a call's arguments are separate expressions, and a chain of their own if they hold one.
func chainShortCircuits(expr ast.Expr) bool {
	for {
		switch node := expr.(type) {
		case *ast.ExprMemberAccess:
			if node.Access() == ast.MemberAccessOption {
				return true
			}
			expr = node.Target
		case *ast.ExprIndexAccess:
			expr = node.Target
		default:
			return false
		}
	}
}

// compileFallbackOperator compiles `a ?? b` and `a !! b`, which differ only in which prelude union the left side is read as and which of its members counts as missing.
//
// The left side is read as that union first, so a value of one's own carrying @AnyOption or @AnyResult is handled as the prelude's own is, and the value is then taken out of the wrapper — the `.value` nobody has to write. The right side is compiled behind the jump, so it is never evaluated unless it is needed.
//
//	<left>
//	AsOption                         ; or AsResult
//	JumpIsType fallback, <missing>   ; None for ??, Err for !!
//	GetField "value"
//	Jump done
//	fallback: Pop; <right>
//	done:
func (c *Compiler) compileFallbackOperator(node *ast.ExprOperatorBinary, access ast.MemberAccess, missing string) error {
	missingId, err := c.resolveBuiltinTypeConstantId(node, missing, c.currentSymbols())
	if err != nil {
		return err
	}

	if err := c.Compile(node.Left); err != nil {
		return err
	}
	if err := c.emitNormalize(node, access); err != nil {
		return err
	}
	toFallback := c.emit(op.JumpIsType, placeholderJumpAddress, missingId)
	if err := c.emitUnwrap(node); err != nil {
		return err
	}
	unwrapped := c.emit(op.Jump, placeholderJumpAddress)

	c.changeOperand(toFallback, len(c.currentInstructions()), missingId)
	c.emit(op.Pop)
	if err := c.Compile(node.Right); err != nil {
		return err
	}

	c.changeOperand(unwrapped, len(c.currentInstructions()))
	return nil
}
