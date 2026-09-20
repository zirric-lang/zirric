package runtime

import (
	"fmt"
	gomath "math"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ ExternPlugin = &MathPlugin{}

// MathPlugin provides runtime bindings for the math module's extern declarations.
// Arithmetic on Int and Float is built into the VM, but everything beyond it needs Go's math package.
type MathPlugin struct{}

func (*MathPlugin) Module() string { return "math" }

// Bind implements ExternPlugin.
func (*MathPlugin) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "pi":
		return Float(gomath.Pi)
	case "e":
		return Float(gomath.E)
	case "toFloat":
		return makeMathUnary(decl, func(v float64) RuntimeValue { return Float(v) })
	case "toInt":
		return makeMathUnary(decl, func(v float64) RuntimeValue { return Int(gomath.Trunc(v)) })
	case "floor":
		return makeMathUnary(decl, func(v float64) RuntimeValue { return Float(gomath.Floor(v)) })
	case "ceil":
		return makeMathUnary(decl, func(v float64) RuntimeValue { return Float(gomath.Ceil(v)) })
	case "round":
		return makeMathUnary(decl, func(v float64) RuntimeValue { return Float(gomath.Round(v)) })
	case "trunc":
		return makeMathUnary(decl, func(v float64) RuntimeValue { return Float(gomath.Trunc(v)) })
	case "sqrt":
		return makeMathUnary(decl, func(v float64) RuntimeValue { return Float(gomath.Sqrt(v)) })
	case "isNaN":
		return makeMathUnary(decl, func(v float64) RuntimeValue { return Bool(gomath.IsNaN(v)) })
	case "isInfinite":
		return makeMathUnary(decl, func(v float64) RuntimeValue { return Bool(gomath.IsInf(v, 0)) })
	case "sign":
		return makeMathUnary(decl, func(v float64) RuntimeValue {
			switch {
			case v > 0:
				return Int(1)
			case v < 0:
				return Int(-1)
			default:
				return Int(0)
			}
		})
	case "abs":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			switch v := args[0].(type) {
			case Int:
				if v < 0 {
					return -v, nil
				}
				return v, nil
			case Float:
				return Float(gomath.Abs(float64(v))), nil
			}
			return nil, fmt.Errorf("abs expects an Int or Float argument, got %T", args[0])
		})
	case "min":
		return makeMathChoice(decl, "min", func(a, b float64) bool { return a <= b })
	case "max":
		return makeMathChoice(decl, "max", func(a, b float64) bool { return a >= b })
	case "pow":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			base, err := mathFloat("pow", args[0])
			if err != nil {
				return nil, err
			}
			exponent, err := mathFloat("pow", args[1])
			if err != nil {
				return nil, err
			}
			return Float(gomath.Pow(base, exponent)), nil
		})
	}
	return nil
}

func makeMathUnary(decl *ast.Symbol, apply func(float64) RuntimeValue) RuntimeValue {
	return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
		v, err := mathFloat(decl.Name, args[0])
		if err != nil {
			return nil, err
		}
		return apply(v), nil
	})
}

// makeMathChoice builds min or max, returning whichever argument prefers reports as preferred.
// The argument itself is returned rather than a converted copy, so comparing two Ints yields an Int.
func makeMathChoice(decl *ast.Symbol, fnName string, prefers func(a, b float64) bool) RuntimeValue {
	return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
		a, err := mathFloat(fnName, args[0])
		if err != nil {
			return nil, err
		}
		b, err := mathFloat(fnName, args[1])
		if err != nil {
			return nil, err
		}
		if prefers(a, b) {
			return args[0], nil
		}
		return args[1], nil
	})
}

func mathFloat(fnName string, v RuntimeValue) (float64, error) {
	switch v := v.(type) {
	case Int:
		return float64(v), nil
	case Float:
		return float64(v), nil
	}
	return 0, fmt.Errorf("%s expects an Int or Float argument, got %T", fnName, v)
}
