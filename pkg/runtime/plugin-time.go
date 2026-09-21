package runtime

import (
	"fmt"
	gotime "time"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ ExternPlugin = &TimePlugin{}

// TimePlugin provides runtime bindings for the time module's extern declarations.
// Nothing here reads a clock: every function is a pure conversion between spans, points and text, which is what keeps the module testable.
type TimePlugin struct{}

func (*TimePlugin) Module() string { return "time" }

// Bind implements ExternPlugin.
func (*TimePlugin) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "nanoseconds":
		return makeDurationOf(decl, 1)
	case "microseconds":
		return makeDurationOf(decl, int64(gotime.Microsecond))
	case "milliseconds":
		return makeDurationOf(decl, int64(gotime.Millisecond))
	case "seconds":
		return makeDurationOf(decl, int64(gotime.Second))
	case "minutes":
		return makeDurationOf(decl, int64(gotime.Minute))
	case "hours":
		return makeDurationOf(decl, int64(gotime.Hour))
	case "zero":
		return MakeExternFunc(decl, func(_ VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			return Duration(0), nil
		})
	case "asNanoseconds":
		return makeDurationTo(decl, func(d Duration) RuntimeValue { return Int(d) })
	case "asMilliseconds":
		return makeDurationTo(decl, func(d Duration) RuntimeValue { return Int(int64(d) / int64(gotime.Millisecond)) })
	case "asSeconds":
		return makeDurationTo(decl, func(d Duration) RuntimeValue { return Float(float64(d) / float64(gotime.Second)) })
	case "asMinutes":
		return makeDurationTo(decl, func(d Duration) RuntimeValue { return Float(float64(d) / float64(gotime.Minute)) })
	case "asHours":
		return makeDurationTo(decl, func(d Duration) RuntimeValue { return Float(float64(d) / float64(gotime.Hour)) })
	case "absolute":
		return makeDurationTo(decl, func(d Duration) RuntimeValue {
			if d < 0 {
				return -d
			}
			return d
		})
	case "formatDuration":
		return makeDurationTo(decl, func(d Duration) RuntimeValue { return String(d.Inspect()) })

	case "parseDuration":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			text, ok := args[0].(String)
			if !ok {
				return nil, fmt.Errorf("parseDuration expects a String, got %s", TypeName(args[0]))
			}
			parsed, err := gotime.ParseDuration(string(text))
			if err != nil {
				return ResultErr(caller, String(fmt.Sprintf("not a duration: %q", string(text))))
			}
			return ResultOk(caller, Duration(parsed))
		})
	case "origin":
		return MakeExternFunc(decl, func(_ VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			return Instant(0), nil
		})
	case "since":
		return makeSpanBetween(decl, func(a, b RuntimeValue) (RuntimeValue, error) {
			earlier, ok := a.(Instant)
			later, ok2 := b.(Instant)
			if !ok || !ok2 {
				return nil, fmt.Errorf("since expects two Instant arguments, got %s and %s", TypeName(a), TypeName(b))
			}
			return Duration(int64(later) - int64(earlier)), nil
		})
	case "between":
		return makeSpanBetween(decl, func(a, b RuntimeValue) (RuntimeValue, error) {
			earlier, ok := a.(Timestamp)
			later, ok2 := b.(Timestamp)
			if !ok || !ok2 {
				return nil, fmt.Errorf("between expects two Timestamp arguments, got %s and %s", TypeName(a), TypeName(b))
			}
			return Duration(int64(later) - int64(earlier)), nil
		})
	case "fromEpoch":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			nanos, ok := args[0].(Int)
			if !ok {
				return nil, fmt.Errorf("fromEpoch expects an Int, got %s", TypeName(args[0]))
			}
			return Timestamp(nanos), nil
		})
	case "toEpoch":
		return makeTimestampTo(decl, func(t Timestamp) RuntimeValue { return Int(t) })
	case "format":
		return makeTimestampTo(decl, func(t Timestamp) RuntimeValue { return String(t.Inspect()) })
	case "parse":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			text, ok := args[0].(String)
			if !ok {
				return nil, fmt.Errorf("parse expects a String, got %s", TypeName(args[0]))
			}
			parsed, err := gotime.Parse(gotime.RFC3339Nano, string(text))
			if err != nil {
				return ResultErr(caller, String(fmt.Sprintf("not an RFC 3339 timestamp: %q", string(text))))
			}
			return ResultOk(caller, Timestamp(parsed.UnixNano()))
		})
	case "year":
		return makeTimestampTo(decl, func(t Timestamp) RuntimeValue { return Int(t.Time().UTC().Year()) })
	case "month":
		return makeTimestampTo(decl, func(t Timestamp) RuntimeValue { return Int(int(t.Time().UTC().Month())) })
	case "day":
		return makeTimestampTo(decl, func(t Timestamp) RuntimeValue { return Int(t.Time().UTC().Day()) })
	case "hour":
		return makeTimestampTo(decl, func(t Timestamp) RuntimeValue { return Int(t.Time().UTC().Hour()) })
	case "minute":
		return makeTimestampTo(decl, func(t Timestamp) RuntimeValue { return Int(t.Time().UTC().Minute()) })
	case "second":
		return makeTimestampTo(decl, func(t Timestamp) RuntimeValue { return Int(t.Time().UTC().Second()) })
	}
	return nil
}

func makeDurationOf(decl *ast.Symbol, unit int64) RuntimeValue {
	return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
		count, ok := args[0].(Int)
		if !ok {
			return nil, fmt.Errorf("%s expects an Int count, got %s", decl.Name, TypeName(args[0]))
		}
		return Duration(int64(count) * unit), nil
	})
}

func makeDurationTo(decl *ast.Symbol, apply func(Duration) RuntimeValue) RuntimeValue {
	return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
		d, ok := args[0].(Duration)
		if !ok {
			return nil, fmt.Errorf("%s expects a Duration, got %s", decl.Name, TypeName(args[0]))
		}
		return apply(d), nil
	})
}

func makeTimestampTo(decl *ast.Symbol, apply func(Timestamp) RuntimeValue) RuntimeValue {
	return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
		t, ok := args[0].(Timestamp)
		if !ok {
			return nil, fmt.Errorf("%s expects a Timestamp, got %s", decl.Name, TypeName(args[0]))
		}
		return apply(t), nil
	})
}

func makeSpanBetween(decl *ast.Symbol, apply func(a, b RuntimeValue) (RuntimeValue, error)) RuntimeValue {
	return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
		return apply(args[0], args[1])
	})
}
