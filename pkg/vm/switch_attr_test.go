package vm_test

import "testing"

func TestIsExprWithAttribute(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "data with attribute matches",
			input: `
			attr Marker
			@Marker()
			data Foo
			Foo() is @Marker
			`,
			expected: true,
		},
		{
			label: "data without attribute does not match",
			input: `
			attr Marker
			data Foo
			Foo() is @Marker
			`,
			expected: false,
		},
		{
			label: "different attribute does not match",
			input: `
			attr MarkerA
			attr MarkerB
			@MarkerA()
			data Foo
			Foo() is @MarkerB
			`,
			expected: false,
		},
		{
			label: "data with multiple attributes matches each",
			input: `
			attr Alpha
			attr Beta
			@Alpha()
			@Beta()
			data Foo
			Foo() is @Alpha
			`,
			expected: true,
		},
		{
			label: "data with multiple attributes matches second",
			input: `
			attr Alpha
			attr Beta
			@Alpha()
			@Beta()
			data Foo
			Foo() is @Beta
			`,
			expected: true,
		},
		{
			label: "is attribute in condition",
			input: `
			attr Marker
			@Marker()
			data Foo
			const val = Foo()
			if val is @Marker { 1 } else { 0 }
			`,
			expected: 1,
		},
	}

	runVmTests(t, tests)
}

func TestExprSwitchWithAttributeCase(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "match attribute case",
			input: `
			attr Marker
			@Marker()
			data Foo
			data Bar
			const val = Foo()
			const result = switch val {
				case is @Marker: 1
				case _: 0
			}
			result
			`,
			expected: 1,
		},
		{
			label: "no match falls to default",
			input: `
			attr Marker
			data Foo
			const val = Foo()
			const result = switch val {
				case is @Marker: 1
				case _: 0
			}
			result
			`,
			expected: 0,
		},
		{
			label: "attribute case before type case",
			input: `
			attr Marker
			@Marker()
			data Foo
			data Bar
			const val = Foo()
			const result = switch val {
				case is @Marker: 1
				case is Bar: 2
				case _: 0
			}
			result
			`,
			expected: 1,
		},
		{
			label: "type case before attribute case",
			input: `
			attr Marker
			@Marker()
			data Foo
			const val = Foo()
			const result = switch val {
				case is Foo: 1
				case is @Marker: 2
				case _: 0
			}
			result
			`,
			expected: 1,
		},
	}

	runVmTests(t, tests)
}

func TestStmtSwitchWithAttributeCase(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "statement switch with attribute case",
			input: `
			attr Marker
			@Marker()
			data Foo
			data Bar
			fn classify(val) {
				switch val {
					case is @Marker:
						return "marked"
					case is Bar:
						return "bar"
					case _:
						return "other"
				}
			}
			classify(Foo())
			`,
			expected: "marked",
		},
		{
			label: "statement switch attribute no match",
			input: `
			attr Marker
			data Foo
			fn classify(val) {
				switch val {
					case is @Marker:
						return "marked"
					case _:
						return "other"
				}
			}
			classify(Foo())
			`,
			expected: "other",
		},
		{
			label: "attribute on union does not propagate to member",
			input: `
			attr Marker
			@Marker()
			union Shape {
				data Circle { radius }
				data Rect { width height }
			}
			const val = Circle(5)
			const result = switch val {
				case is @Marker: val.radius
				case _: 0
			}
			result
			`,
			expected: 0,
		},
	}

	runVmTests(t, tests)
}

func TestIsExprMultipleAttributes(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "multiple attrs all present",
			input: `
			attr A
			attr B
			@A() @B()
			data Foo
			Foo() is @A @B
			`,
			expected: true,
		},
		{
			label: "multiple attrs one missing",
			input: `
			attr A
			attr B
			@A()
			data Foo
			Foo() is @A @B
			`,
			expected: false,
		},
		{
			label: "multiple attrs none present",
			input: `
			attr A
			attr B
			data Foo
			Foo() is @A @B
			`,
			expected: false,
		},
		{
			label: "three attrs all present",
			input: `
			attr A
			attr B
			attr C
			@A() @B() @C()
			data Foo
			Foo() is @A @B @C
			`,
			expected: true,
		},
		{
			label: "three attrs middle missing",
			input: `
			attr A
			attr B
			attr C
			@A() @C()
			data Foo
			Foo() is @A @B @C
			`,
			expected: false,
		},
	}

	runVmTests(t, tests)
}

func TestSwitchCaseMultipleAttributes(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "switch case multi-attr matches",
			input: `
			attr A
			attr B
			@A() @B()
			data Foo
			data Bar
			const result = switch Foo() {
				case is @A @B: 1
				case _: 0
			}
			result
			`,
			expected: 1,
		},
		{
			label: "switch case multi-attr partial match falls through",
			input: `
			attr A
			attr B
			@A()
			data Foo
			const result = switch Foo() {
				case is @A @B: 1
				case _: 0
			}
			result
			`,
			expected: 0,
		},
	}

	runVmTests(t, tests)
}

func TestSwitchValueCompiledOnce(t *testing.T) {
	// This test verifies the switch value is evaluated only once, not re-evaluated per case. We use a function with a side effect (counter).
	tests := []vmTestCase{
		{
			label: "switch expr evaluates value once",
			input: `
			var counter = 0
			data Foo
			fn getValue() {
				counter = counter + 1
				return Foo()
			}
			const result = switch getValue() {
				case is Foo: counter
				case _: -1
			}
			result
			`,
			expected: 1,
		},
		{
			label: "switch stmt evaluates value once with multiple cases",
			input: `
			var counter = 0
			data Foo
			data Bar
			data Baz
			fn getValue() {
				counter = counter + 1
				return Foo()
			}
			fn test() {
				switch getValue() {
					case is Bar:
						return -1
					case is Baz:
						return -2
					case is Foo:
						return counter
					case _:
						return -3
				}
			}
			test()
			`,
			expected: 1,
		},
	}

	runVmTests(t, tests)
}
