package vm_test

import "testing"

// Preamble declares all builtin types in the exact iota order from runtime/prelude.go so the analyzer assigns constant IDs that match the hardcoded typeId values.
const compositeTypePreamble = `
extern type Array {}
extern type Bool {}
extern type Char {}
extern type Dict {}
extern type Float {}
extern type Func {}
extern type Int {}
extern type Module {}
extern type String {}
extern type Void {}
`

func TestIsArrayType(t *testing.T) {
	tests := []vmTestCase{
		{
			label:    "array matches [Array]",
			input:    compositeTypePreamble + `[1, 2, 3] is [Array]`,
			expected: true,
		},
		{
			label:    "non-array does not match [Array]",
			input:    compositeTypePreamble + `42 is [Array]`,
			expected: false,
		},
		{
			label:    "empty array matches [Array]",
			input:    compositeTypePreamble + `[] is [Array]`,
			expected: true,
		},
		{
			label:    "string does not match [Array]",
			input:    compositeTypePreamble + `"hello" is [Array]`,
			expected: false,
		},
	}
	runVmTests(t, tests)
}

func TestIsDictType(t *testing.T) {
	tests := []vmTestCase{
		{
			label:    "dict matches [Dict: Dict]",
			input:    compositeTypePreamble + `[1: "a"] is [Dict: Dict]`,
			expected: true,
		},
		{
			label:    "non-dict does not match [Dict: Dict]",
			input:    compositeTypePreamble + `42 is [Dict: Dict]`,
			expected: false,
		},
		{
			label:    "empty dict matches [Dict: Dict]",
			input:    compositeTypePreamble + `[:] is [Dict: Dict]`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestSwitchExprCompositeTypes(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "switch on array type",
			input: compositeTypePreamble + `
			const val = [1, 2, 3]
			const result = switch val {
				case is [Array]: "array"
				case _: "other"
			}
			result
			`,
			expected: "array",
		},
		{
			label: "switch on dict type",
			input: compositeTypePreamble + `
			const val = [1: "a"]
			const result = switch val {
				case is [Dict: Dict]: "dict"
				case _: "other"
			}
			result
			`,
			expected: "dict",
		},
		{
			label: "switch mixed composite and named types",
			input: compositeTypePreamble + `
			data Foo
			const val = [1, 2]
			const result = switch val {
				case is Foo: "foo"
				case is [Array]: "array"
				case is [Dict: Dict]: "dict"
				case _: "other"
			}
			result
			`,
			expected: "array",
		},
		{
			label: "switch int does not match array",
			input: compositeTypePreamble + `
			const val = 42
			const result = switch val {
				case is [Array]: "array"
				case _: "other"
			}
			result
			`,
			expected: "other",
		},
	}
	runVmTests(t, tests)
}

func TestSwitchStmtCompositeTypes(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "statement switch with array type",
			input: compositeTypePreamble + `
			fn classify(val) {
				switch val {
					case is [Array]:
						return "array"
					case is [Dict: Dict]:
						return "dict"
					case _:
						return "other"
				}
			}
			classify([1, 2, 3])
			`,
			expected: "array",
		},
		{
			label: "statement switch dict branch",
			input: compositeTypePreamble + `
			fn classify(val) {
				switch val {
					case is [Array]:
						return "array"
					case is [Dict: Dict]:
						return "dict"
					case _:
						return "other"
				}
			}
			classify([1: "a"])
			`,
			expected: "dict",
		},
	}
	runVmTests(t, tests)
}

func TestSwitchExprWithAttributeUsingIs(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "case is @Attr matches",
			input: `
			attr Marker
			@Marker()
			data Foo
			const result = switch Foo() {
				case is @Marker: "marked"
				case _: "unmarked"
			}
			result
			`,
			expected: "marked",
		},
		{
			label: "case is @Attr does not match",
			input: `
			attr Marker
			data Foo
			const result = switch Foo() {
				case is @Marker: "marked"
				case _: "unmarked"
			}
			result
			`,
			expected: "unmarked",
		},
		{
			label: "case is @A @B multi-attr matches",
			input: `
			attr A
			attr B
			@A() @B()
			data Foo
			const result = switch Foo() {
				case is @A @B: "both"
				case _: "other"
			}
			result
			`,
			expected: "both",
		},
		{
			label: "case is @A @B partial match falls through",
			input: `
			attr A
			attr B
			@A()
			data Foo
			const result = switch Foo() {
				case is @A @B: "both"
				case is @A: "just_a"
				case _: "other"
			}
			result
			`,
			expected: "just_a",
		},
	}
	runVmTests(t, tests)
}
