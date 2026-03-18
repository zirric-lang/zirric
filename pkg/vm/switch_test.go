package vm_test

import "testing"

func TestExprIs(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "data value is data type",
			input: `
data Foo
Foo() is Foo
`,
			expected: true,
		},
		{
			label: "data value is wrong type",
			input: `
data Foo
data Bar
Foo() is Bar
`,
			expected: false,
		},
		{
			label: "value is union member",
			input: `
data A
data B
union AB { A B }
A() is AB
`,
			expected: true,
		},
		{
			label: "value is not union member",
			input: `
data A
data B
data C
union AB { A B }
C() is AB
`,
			expected: false,
		},
		{
			label: "second union member is member",
			input: `
data A
data B
union AB { A B }
B() is AB
`,
			expected: true,
		},
		{
			label: "is expression in condition",
			input: `
data A
data B
const val = A()
if val is A { 1 } else { 0 }
`,
			expected: 1,
		},
	}

	runVmTests(t, tests)
}

func TestExprSwitchIsType(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "match first case",
			input: `
data A
data B
const val = A()
const result = switch val {
case is A: 1
case is B: 2
case _: 0
}
result
`,
			expected: 1,
		},
		{
			label: "match second case",
			input: `
data A
data B
const val = B()
const result = switch val {
case is A: 1
case is B: 2
case _: 0
}
result
`,
			expected: 2,
		},
		{
			label: "match default case",
			input: `
data A
data B
const result = switch 42 {
case is A: 1
case is B: 2
case _: 0
}
result
`,
			expected: 0,
		},
		{
			label: "match union member",
			input: `
data A
data B
union AB { A B }
const val = A()
const result = switch val {
case is AB: 1
case _: 0
}
result
`,
			expected: 1,
		},
		{
			label: "only default case",
			input: `
const result = switch 42 {
case _: 99
}
result
`,
			expected: 99,
		},
	}

	runVmTests(t, tests)
}

func TestExprSwitchValue(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "match int literal",
			input: `
const val = 42
const result = switch val {
case 42: 1
case 99: 2
case _: 0
}
result
`,
			expected: 1,
		},
		{
			label: "match string literal",
			input: `
const val = "hello"
const result = switch val {
case "hello": 1
case "world": 2
case _: 0
}
result
`,
			expected: 1,
		},
		{
			label: "match second literal",
			input: `
const val = "world"
const result = switch val {
case "hello": 1
case "world": 2
case _: 0
}
result
`,
			expected: 2,
		},
		{
			label: "match default",
			input: `
const val = "other"
const result = switch val {
case "hello": 1
case "world": 2
case _: 0
}
result
`,
			expected: 0,
		},
		{
			label: "match bool",
			input: `
const val = true
const result = switch val {
case true: 1
case false: 2
case _: 0
}
result
`,
			expected: 1,
		},
	}

	runVmTests(t, tests)
}

func TestStmtSwitch(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "switch statement with return in first case",
			input: `
data A
data B
fn classify(val) {
switch val {
case is A:
return 1
case is B:
return 2
case _:
return 0
}
}
classify(A())
`,
			expected: 1,
		},
		{
			label: "switch statement with return in second case",
			input: `
data A
data B
fn classify(val) {
switch val {
case is A:
return 1
case is B:
return 2
case _:
return 0
}
}
classify(B())
`,
			expected: 2,
		},
		{
			label: "switch statement default case",
			input: `
data A
data B
fn classify(val) {
switch val {
case is A:
return 1
case is B:
return 2
case _:
return 0
}
}
classify(42)
`,
			expected: 0,
		},
		{
			label: "switch statement with value matching",
			input: `
fn describe(n) {
switch n {
case 1:
return "one"
case 2:
return "two"
case _:
return "other"
}
}
describe(2)
`,
			expected: "two",
		},
		{
			label: "switch statement with local vars",
			input: `
data A { x }
data B { y }
fn process(val) {
switch val {
case is A:
const r = val.x + 10
return r
case is B:
return val.y
case _:
return 0
}
}
process(A(5))
`,
			expected: 15,
		},
		{
			label: "switch statement without default",
			input: `
data A
data B
fn classify(val) {
switch val {
case is A:
return 1
case is B:
return 2
}
return -1
}
classify(42)
`,
			expected: -1,
		},
	}

	runVmTests(t, tests)
}

func TestSwitchWithUnionTypes(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "switch on union member types - circle",
			input: `
union Shape {
data Circle { radius }
data Rect { width height }
}
fn area(shape) {
switch shape {
case is Circle:
return shape.radius * shape.radius
case is Rect:
return shape.width * shape.height
case _:
return 0
}
}
area(Circle(5))
`,
			expected: 25,
		},
		{
			label: "switch on union member types - rect",
			input: `
union Shape {
data Circle { radius }
data Rect { width height }
}
fn area(shape) {
switch shape {
case is Circle:
return shape.radius * shape.radius
case is Rect:
return shape.width * shape.height
case _:
return 0
}
}
area(Rect(3, 4))
`,
			expected: 12,
		},
		{
			label: "expr switch with union",
			input: `
union Shape {
data Circle { radius }
data Rect { width height }
}
const shape = Circle(7)
const result = switch shape {
case is Circle: shape.radius
case is Rect: shape.width
case _: 0
}
result
`,
			expected: 7,
		},
		{
			label: "is with union type",
			input: `
union Shape {
data Circle { radius }
data Rect { width height }
}
Circle(5) is Shape
`,
			expected: true,
		},
		{
			label: "non-member is not union",
			input: `
data Other
union Shape {
data Circle { radius }
data Rect { width height }
}
Other() is Shape
`,
			expected: false,
		},
	}

	runVmTests(t, tests)
}

func TestSwitchMixedPatterns(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "mix type and value patterns",
			input: `
data Special
fn check(val) {
switch val {
case is Special:
return "special"
case 42:
return "forty-two"
case _:
return "other"
}
}
check(42)
`,
			expected: "forty-two",
		},
		{
			label: "mix type and value - type wins first",
			input: `
data Special
fn check(val) {
switch val {
case is Special:
return "special"
case 42:
return "forty-two"
case _:
return "other"
}
}
check(Special())
`,
			expected: "special",
		},
	}

	runVmTests(t, tests)
}
