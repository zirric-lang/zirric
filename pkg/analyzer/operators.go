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
