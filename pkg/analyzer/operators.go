package analyzer

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// isArithmeticOperator reports whether an operator is one the VM resolves by the types of its operands.
// Equality works on any two values and the logical operators are their own thing, so neither can fail on types alone.
func isArithmeticOperator(operator ast.OperatorBinary) bool {
	switch token.Token(operator).Type {
	case token.PLUS, token.MINUS, token.ASTERISK, token.SLASH, token.PERCENT,
		token.LT, token.GT, token.LTE, token.GTE:
		return true
	}
	return false
}

func operatorText(operator ast.OperatorBinary) string {
	return string(token.Token(operator).Type)
}

// arithmeticApplies mirrors what the VM does with an operator, so that the checker rejects only what running the program would.
//
// The rules are the VM's own: a String concatenates with anything that has an unambiguous textual form, but only with `+`; Int and Float mix freely; `%` is Int alone; and everything else is a failure at the moment it runs.
func arithmeticApplies(operator ast.OperatorBinary, lhs, rhs checked) bool {
	// Anything that might turn out to be one of several types could be the one that works.
	if mightBeAnything(lhs) || mightBeAnything(rhs) {
		return true
	}

	kind := token.Token(operator).Type
	if kind == token.PERCENT {
		return isBuiltin(lhs, "Int") && isBuiltin(rhs, "Int")
	}

	// The time types carry a unit and have an algebra of their own, which is involved enough that the checker leaves it to the VM.
	if isTimeType(lhs) || isTimeType(rhs) {
		return true
	}

	if isBuiltin(lhs, "String") || isBuiltin(rhs, "String") {
		if kind != token.PLUS {
			return false
		}
		other := rhs
		if isBuiltin(rhs, "String") {
			other = lhs
		}
		return isBuiltin(other, "String") || hasTextualForm(other)
	}

	return isNumeric(lhs) && isNumeric(rhs)
}

// mightBeAnything reports whether a type leaves room for the operator to work after all.
func mightBeAnything(c checked) bool {
	switch c.kind {
	case kindUnknown, kindUnion, kindAttrs:
		return true
	}
	return false
}

func isBuiltin(c checked, name string) bool {
	return c.kind == kindBuiltin && c.name == name
}

func isNumeric(c checked) bool {
	return isBuiltin(c, "Int") || isBuiltin(c, "Float")
}

func isTimeType(c checked) bool {
	return isBuiltin(c, "Duration") || isBuiltin(c, "Instant") || isBuiltin(c, "Timestamp")
}

// hasTextualForm lists the types a String concatenates with, which is exactly what runtime.TrivialString accepts.
func hasTextualForm(c checked) bool {
	if c.kind != kindBuiltin {
		return false
	}
	switch c.name {
	case "String", "Int", "Float", "Char", "Byte", "Binary", "Bool", "Void", "Duration", "Instant", "Timestamp":
		return true
	}
	return false
}

// inferBinaryOperator describes what a binary expression yields, following the VM the way arithmeticApplies follows it for whether the operator applies at all.
//
// An operator whose result depends on operands the checker cannot see answers unknown, which is what keeps a hint written over one from being reported on a guess.
func (a *Analyzer) inferBinaryOperator(expr *ast.ExprOperatorBinary, symbols *ast.SymbolTable) checked {
	switch token.Token(expr.Operator).Type {
	// Equality is defined for any two values, the logical operators take and give Bool, and a comparison that applies at all compares into one. A comparison that does not apply is reported as an operator misuse rather than described here.
	case token.EQ, token.NEQ, token.AND, token.OR, token.LT, token.LTE, token.GT, token.GTE:
		return a.builtinNamed("Bool", symbols)
	case token.QUESTION_QUESTION:
		return a.inferFallback(expr, "Option", symbols)
	case token.BANG_BANG:
		return a.inferFallback(expr, "Result", symbols)
	case token.PLUS, token.MINUS, token.ASTERISK, token.SLASH, token.PERCENT:
		return a.inferArithmetic(expr, symbols)
	}
	return unknownType()
}

// inferArithmetic mirrors runtime.numericBinaryOperation: a String concatenates into a String, Int and Float promote to the wider of the two, and `%` is Int alone.
func (a *Analyzer) inferArithmetic(expr *ast.ExprOperatorBinary, symbols *ast.SymbolTable) checked {
	lhs := a.infer(expr.Left, symbols)
	rhs := a.infer(expr.Right, symbols)
	// A type that leaves room for several answers leaves room for several results.
	if mightBeAnything(lhs) || mightBeAnything(rhs) {
		return unknownType()
	}

	kind := token.Token(expr.Operator).Type
	if kind == token.PERCENT {
		if isBuiltin(lhs, "Int") && isBuiltin(rhs, "Int") {
			return a.builtinNamed("Int", symbols)
		}
		return unknownType()
	}

	// The time types have an algebra of their own — a Duration times an Int is a Duration, an Instant minus an Instant a Duration — which arithmeticApplies already leaves to the VM, and so does this.
	if isTimeType(lhs) || isTimeType(rhs) {
		return unknownType()
	}

	if isBuiltin(lhs, "String") || isBuiltin(rhs, "String") {
		if kind == token.PLUS {
			return a.builtinNamed("String", symbols)
		}
		return unknownType()
	}

	if !isNumeric(lhs) || !isNumeric(rhs) {
		return unknownType()
	}
	// Mixing the two promotes to Float, which is what the VM does by converting the Int operand before it operates.
	if isBuiltin(lhs, "Float") || isBuiltin(rhs, "Float") {
		return a.builtinNamed("Float", symbols)
	}
	return a.builtinNamed("Int", symbols)
}

// inferFallback describes `a ?? b` and `a !! b`, which answer with either what the left side holds or the right side outright, and so name one type only when those two agree on it.
//
// The left side is answerable only when it was written as a `T?` or `T!`, since that is the one place the element is recorded. A value can be a wrapper — or, for `!!`, an error — without saying what is inside it.
func (a *Analyzer) inferFallback(expr *ast.ExprOperatorBinary, union string, symbols *ast.SymbolTable) checked {
	wrapper := a.resolveNamedHint(ast.StaticReference{{Value: union}}, symbols)
	left := a.infer(expr.Left, symbols)
	if !wrapper.isKnown() || !sameType(left, wrapper) || left.elem == nil {
		return unknownType()
	}
	held := *left.elem
	fallback := a.infer(expr.Right, symbols)
	if !held.isKnown() || !fallback.isKnown() || !sameType(held, fallback) {
		return unknownType()
	}
	return held
}
