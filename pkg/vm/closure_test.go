package vm_test

import "testing"

func TestFnClosureExecution(t *testing.T) {
	tests := []vmTestCase{
		{
			label:    "basic fn closure",
			input:    `const f = fn() { return 42 } f()`,
			expected: 42,
		},
		{
			label:    "fn closure with param",
			input:    `const f = fn(x) { return x + 1 } f(5)`,
			expected: 6,
		},
		{
			label:    "fn closure with two params",
			input:    `const f = fn(a, b) { return a + b } f(3, 4)`,
			expected: 7,
		},
		{
			label:    "fn closure inline call",
			input:    `fn(x) { return x * 2 }(21)`,
			expected: 42,
		},
		{
			label: "fn closure as argument",
			input: `
			const apply = fn(f, x) { return f(x) }
			apply(fn(x) { return x + 10 }, 32)`,
			expected: 42,
		},
		{
			label:    "fn closure capturing variable",
			input:    `const offset = 100 const f = fn(x) { return x + offset } f(5)`,
			expected: 105,
		},
		{
			label:    "fn closure with return",
			input:    `const f = fn(x) { return x * 3 } f(14)`,
			expected: 42,
		},
		{
			label:    "nested fn closures",
			input:    `const f = fn(x) { return fn(y) { return x + y } } f(10)(32)`,
			expected: 42,
		},
	}
	runVmTests(t, tests)
}

func TestFnClosureWithTypedParams(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "typed param closure",
			input: compositeTypePreamble + `
			const f = fn(x: Int) { return x + 1 }
			f(5)
			`,
			expected: 6,
		},
		{
			label: "multi typed param closure",
			input: compositeTypePreamble + `
			const f = fn(a: Int, b: Int) { return a * b }
			f(6, 7)
			`,
			expected: 42,
		},
		{
			label: "closure with return type hint",
			input: compositeTypePreamble + `
			const f = fn(x: Int) -> Int { return x + 1 }
			f(41)
			`,
			expected: 42,
		},
	}
	runVmTests(t, tests)
}

func TestIsFuncType(t *testing.T) {
	tests := []vmTestCase{
		{
			label:    "closure matches Func type",
			input:    compositeTypePreamble + `const f = fn(x) { x } f is Func`,
			expected: true,
		},
		{
			label:    "fn closure matches Func type",
			input:    compositeTypePreamble + `const f = fn() { 42 } f is Func`,
			expected: true,
		},
		{
			label:    "int does not match Func type",
			input:    compositeTypePreamble + `42 is Func`,
			expected: false,
		},
		{
			label:    "fn type expr matches Func",
			input:    compositeTypePreamble + `const f = fn(x) { x } f is fn(a)`,
			expected: true,
		},
		{
			label:    "int does not match fn type expr",
			input:    compositeTypePreamble + `42 is fn(a)`,
			expected: false,
		},
	}
	runVmTests(t, tests)
}

func TestSwitchWithFuncType(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "switch case is fn(a) matches closure",
			input: compositeTypePreamble + `
			const val = fn(x) { x }
			const result = switch val {
				case is fn(a): "func"
				case _: "other"
			}
			result
			`,
			expected: "func",
		},
		{
			label: "switch case is fn(a) does not match int",
			input: compositeTypePreamble + `
			const val = 42
			const result = switch val {
				case is fn(a): "func"
				case _: "other"
			}
			result
			`,
			expected: "other",
		},
	}
	runVmTests(t, tests)
}

func TestNestedCompositeTypeExecution(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "nested array type in is",
			input: compositeTypePreamble + `
			const val = [[1, 2], [3, 4]]
			val is [Array]
			`,
			expected: true,
		},
		{
			label: "switch with array and dict types",
			input: compositeTypePreamble + `
			fn classify(val) {
				switch val {
					case is [Array]:
						return "array"
					case is [Dict: Dict]:
						return "dict"
					case is fn(a):
						return "func"
					case _:
						return "other"
				}
			}
			classify(fn(x) { x })
			`,
			expected: "func",
		},
	}
	runVmTests(t, tests)
}
