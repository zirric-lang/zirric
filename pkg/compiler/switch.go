package compiler

import (
	"fmt"

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
func (c *Compiler) compileSwitchIsTypeCheck(tempLocal int, typeExpr ast.TypeExpr) []int {
	// Multi-attribute type expression: need per-attr checks
	if attrs, ok := typeExpr.(ast.TypeExprAttrs); ok {
		var jumpNexts []int
		for _, attr := range attrs.Attrs {
			c.emit(op.GetLocal, tempLocal)
			attrConstId, err := c.resolveTypeConstantId(attr)
			if err != nil {
				continue
			}
			c.emit(op.IsType, attrConstId)
			jumpNexts = append(jumpNexts, c.emit(op.JumpFalse, placeholderJumpAddress))
		}
		return jumpNexts
	}

	// Single type expression
	c.emit(op.GetLocal, tempLocal)
	typeConstId, err := c.resolveTypeConstantId(typeExpr)
	if err != nil {
		return nil
	}
	c.emit(op.IsType, typeConstId)
	return []int{c.emit(op.JumpFalse, placeholderJumpAddress)}
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
			jumpNexts := c.compileSwitchIsTypeCheck(tempLocal, cs.TypeRef)

			err := c.Compile(cs.Body)
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
			jumpNexts := c.compileSwitchIsTypeCheck(tempLocal, cs.TypeRef)

			err := c.compileBlock(cs.Body)
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

// resolveTypeConstantId resolves a type expression to its constant slot index.
// Handles simple identifiers, dotted member access, and composite types.
func (c *Compiler) resolveTypeConstantId(typeExpr ast.TypeExpr) (int, error) {
	symbols := c.currentSymbols()
	if symbols == nil {
		return 0, fmt.Errorf("undefined type in type check")
	}

	switch e := typeExpr.(type) {
	case ast.TypeExprRef:
		if len(e.Reference) == 1 {
			return c.resolveIdentifierConstantId(e.Reference[0], symbols)
		}
		sym := symbols.LookupRef(e.Reference)
		if sym == nil || sym.Decl == nil {
			return 0, fmt.Errorf("undefined type %q in type check", e.Reference.String())
		}
		origSym := sym.Original()
		if origSym.ConstantId == nil {
			return 0, fmt.Errorf("type %q has no constant id", e.Reference.String())
		}
		return *origSym.ConstantId, nil
	case ast.TypeExprArray:
		return c.resolveBuiltinTypeConstantId("Array", symbols)
	case ast.TypeExprDict:
		return c.resolveBuiltinTypeConstantId("Dict", symbols)
	case ast.TypeExprFunc:
		return c.resolveBuiltinTypeConstantId("Func", symbols)
	default:
		return 0, fmt.Errorf("unsupported type expression %T in type check", typeExpr)
	}
}

func (c *Compiler) resolveIdentifierConstantId(name ast.Identifier, symbols *ast.SymbolTable) (int, error) {
	symbol := symbols.LookupIdentifier(name)
	if symbol == nil || symbol.Decl == nil {
		return 0, fmt.Errorf("undefined type %q in type check", name.Value)
	}
	origSym := symbol.Original()
	if origSym.ConstantId == nil {
		return 0, fmt.Errorf("type %q has no constant id", name.Value)
	}
	return *origSym.ConstantId, nil
}

func (c *Compiler) resolveBuiltinTypeConstantId(name string, symbols *ast.SymbolTable) (int, error) {
	symbol := symbols.Lookup(name, nil)
	if symbol == nil || symbol.Decl == nil {
		return 0, fmt.Errorf("undefined built-in type %q in type check", name)
	}
	origSym := symbol.Original()
	if origSym.ConstantId == nil {
		return 0, fmt.Errorf("built-in type %q has no constant id", name)
	}
	return *origSym.ConstantId, nil
}
