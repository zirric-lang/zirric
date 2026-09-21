package vm

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/op"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
)

// errWrongType reports a value an instruction cannot work with.
//
// The value is taken rather than a type name, so a caller has to hand over what it actually popped. Writing the message by hand invited the opposite: after a failed type assertion the asserted variable holds the zero value of the type that was wanted, so `!5` reported a Bool of "false" — naming a value that was never there.
func errWrongType(what string, want string, got runtime.RuntimeValue) error {
	if got == nil {
		return fmt.Errorf("%s is only defined on %s", what, want)
	}
	return fmt.Errorf("%s is only defined on %s, got %s %s", what, want, typeNameOf(got), got.Inspect())
}

// errWrongArgCount reports a call the callee cannot accept.
func errWrongArgCount(what string, want int, got int) error {
	if what == "" {
		return fmt.Errorf("wrong number of arguments: want=%d, got=%d", want, got)
	}
	return fmt.Errorf("%s: wrong number of arguments: want=%d, got=%d", what, want, got)
}

// errIndexOutOfBounds reports an index outside what a value holds.
func errIndexOutOfBounds(what string, index int, length int) error {
	return fmt.Errorf("%s index %d out of bounds, length is %d", what, index, length)
}

// errNotIndexable reports indexing a value that cannot be indexed.
func errNotIndexable(target runtime.RuntimeValue) error {
	return fmt.Errorf("index operator not supported on %s", typeNameOf(target))
}

// errDivisionByZero reports a division whose divisor is zero.
func errDivisionByZero() error {
	return fmt.Errorf("division by zero")
}

// typeNameOf names a value's type the way the language spells it.
func typeNameOf(value runtime.RuntimeValue) string {
	return runtime.TypeName(value)
}

// operatorSymbol names an operator the way it is written, rather than by its opcode.
// A message saying "operator 16" asks a reader to know the bytecode; one saying "operator -" asks them to look at their own line.
func operatorSymbol(operator op.Opcode) string {
	switch operator {
	case op.Add:
		return "+"
	case op.Sub:
		return "-"
	case op.Mul:
		return "*"
	case op.Div:
		return "/"
	case op.Mod:
		return "%"
	case op.LessThan:
		return "<"
	case op.LessThanOrEqual:
		return "<="
	case op.GreaterThan:
		return ">"
	case op.GreaterThanOrEqual:
		return ">="
	}
	if def, err := op.LookupDefinition(byte(operator)); err == nil {
		return def.Name
	}
	return "?"
}

// errUnsupportedOperands reports an operator that has no meaning for the values it was given.
func errUnsupportedOperands(operator op.Opcode, lhs runtime.RuntimeValue, rhs runtime.RuntimeValue) error {
	return fmt.Errorf("operator %s is not defined for %s and %s", operatorSymbol(operator), typeNameOf(lhs), typeNameOf(rhs))
}
