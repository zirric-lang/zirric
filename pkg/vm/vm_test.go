package vm_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/compiler"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/op"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
	"code.knabel.dev/zirric-lang/zirric/pkg/vm"
)

type vmTestCase struct {
	label    string
	input    string
	expected any
	err      string
}

func TestBasicOperations(t *testing.T) {
	tests := []vmTestCase{
		{input: "1", expected: 1},
		{input: "1+2", expected: 3},
		{input: "true", expected: true},
		{input: "false", expected: false},
		{input: "!true", expected: false},
		{input: "!false", expected: true},
		{input: "true && true", expected: true},
		{input: "true && 3", err: `a value of a different type was expected here, got Int 3`},
		{input: "(if true { 2 } else { 3 })", expected: 2},
		{input: "(if 1 == 1 { 2*3 } else { 3 })", expected: 6},
		{input: "(if 1 == 0 { 2*3 } else { 3 })", expected: 3},
		{input: "(if 1 != 0 { 2*3 } else { 3 })", expected: 6},
		{input: "(if true { 2*3 } else { 3 })", expected: 6},
		{input: "(if true || false { 2*3 } else { 3 })", expected: 6},
		{input: "if true || false { 2*3 } else { 3 }", expected: 6},
		{input: `"abc"`, expected: "abc"},
		{input: "'a'", expected: 'a'},
		{input: "'\\n'", expected: '\n'},
		{input: "'\\''", expected: '\''},
		{input: "'\\\\'", expected: '\\'},
		{input: "[]", expected: []any{}},
		{input: "[1, 2, 3]", expected: []any{1, 2, 3}},
		{input: "[1, 2, 3][0]", expected: 1},
		{input: "[1, 2, 3][3]", err: "array index 3 out of bounds, length is 3"},
		{input: "[:]", expected: map[any]any{}},
		{input: `["hello": "world", 1: 2]`, expected: map[any]any{"hello": "world", 1: 2}},
		{input: `["1": 3, 1: 2]`, expected: map[any]any{"1": 3, 1: 2}},
		{input: `["hello": "world"]["hello"]`, expected: "world"},
		{input: `["hello": "world"]["missing"]`, expected: runtime.Void{}},
	}

	runVmTests(t, tests)
}

func TestBasicFunctions(t *testing.T) {
	tests := []vmTestCase{
		{input: "fn example() { return 42 }\nexample()", expected: 42},
		{input: "fn example() { return }\nexample()", expected: nil},
		{input: `
		fn example() {
			const x = 1
			return x + x
		}
		example()
		`, expected: 2},
		{
			label: "function with parameter",
			input: `
		fn twice(n) {
			return n+n
		}
		twice(2)
		`, expected: 4},
	}

	runVmTests(t, tests)
}

func TestModuleCall(t *testing.T) {
	moduleA := prepareContextModuleParsing(t, "foo.a", `
		mod a
		fn answer() { return 42 }
	`)
	mainModule, program := prepareSourceFileParsing(t, `
		import a = foo.a
		a.answer()
	`)
	resolver := newTestModuleResolverWithModules(mainModule, map[registry.LogicalURI]*ast.ContextModule{
		moduleA.Name: moduleA,
	})

	comp := compiler.New(resolver)
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}

	vmInstance := vm.New(comp.Bytecode())
	if err := vmInstance.Run(); err != nil {
		t.Fatalf("vm error: %s", err)
	}

	testExpectedValue(t, 42, vmInstance.LastPoppedStackElem())
}

func TestData(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "empty data",
			input: `
			data Example
			Example()
			`,
			expected: data{typeId: runtime.TypeId(runtime.NumBuiltinTypeIds), values: []any{}},
		},
		{
			label: "data with values",
			input: `
			data Person {
				name
				age
			}
			Person("Max", 42)
			`,
			expected: data{typeId: runtime.TypeId(runtime.NumBuiltinTypeIds), values: []any{
				"Max", 42,
			}},
		},
		{
			label: "data with values and member access",
			input: `
			data Person {
				name
				age
			}
			Person("Max", 42).name
			`,
			expected: "Max",
		},
		{
			label: "data with values and member access",
			input: `
			data Person {
				name
				age
			}
			Person("Max", 42).age
			`,
			expected: 42,
		},
	}

	runVmTests(t, tests)
}

func TestDataAttributes(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "attribute type call",
			input: `
			attr Job {
				jobName
			}
			@Job("Singer")
			data Person {
				name
			}
			Job(Person("Valentin")).jobName
			`,
			expected: "Singer",
		},
		{
			label: "missing attribute returns void",
			input: `
			attr Job {
				jobName
			}
			data Person {
				name
			}
			Job(Person("Valentin"))
			`,
			expected: runtime.Void{},
		},
	}

	runVmTests(t, tests)
}

func TestFunctionAttributes(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "function attribute lookup",
			input: `
			attr Job {
				jobName
			}
			@Job("Singer")
			fn greet() {}
			Job(greet).jobName
			`,
			expected: "Singer",
		},
		{
			label: "missing function attribute returns void",
			input: `
			attr Job {
				jobName
			}
			fn greet() {}
			Job(greet)
			`,
			expected: runtime.Void{},
		},
	}

	runVmTests(t, tests)
}

// testExternPlugin provides bindings for extern declarations used in tests.
type testExternPlugin struct{}

func (p *testExternPlugin) Module() string { return "" }

func (p *testExternPlugin) Bind(ctx runtime.BindContext, module *ast.SymbolTable, decl *ast.Symbol) runtime.RuntimeValue {
	switch decl.Name {
	case "greet":
		return runtime.MakeExternFunc(decl, func(_ runtime.VMCaller, args []runtime.RuntimeValue) (runtime.RuntimeValue, error) {
			return runtime.String("hello"), nil
		})
	case "Void":
		return runtime.MakeBuiltinSimpleType(decl, runtime.BuiltinTypeIds["Void"])
	case "void":
		return runtime.Void{}
	}
	return nil
}

func TestExternAttributes(t *testing.T) {
	t.Run("extern fn attribute lookup", func(t *testing.T) {
		module, program := prepareSourceFileParsing(t, `
			attr Job { jobName }
			@Job("Singer")
			extern fn greet(name)
			Job(greet).jobName
		`)
		resolver := newTestModuleResolver(module)

		comp := compiler.New(resolver)
		comp.RegisterPlugin(&testExternPlugin{})
		err := comp.Compile(program)
		if err != nil {
			t.Fatalf("compiler error: %s", err)
		}

		machine := vm.New(comp.Bytecode())
		err = machine.Run()
		if err != nil {
			t.Fatalf("vm error: %s", err)
		}

		testExpectedValue(t, "Singer", machine.LastPoppedStackElem())
	})

	t.Run("extern type attribute lookup", func(t *testing.T) {
		runVmTests(t, []vmTestCase{
			{
				label: "extern type attribute lookup",
				input: `
				attr Job { jobName }
				@Job("Actor")
				extern type Person {}
				Job(Person).jobName
				`,
				expected: "Actor",
			},
		})
	})
}

func TestExternValueImport(t *testing.T) {
	// Verify that importing a module containing extern const declarations
	// compiles and runs without the "unknown declaration *ast.DeclExternValue" error.
	moduleA := prepareContextModuleParsing(t, "foo.a", `
		mod a
		extern type Void {}
		extern const void
		fn answer() { return 42 }
	`)
	mainModule, program := prepareSourceFileParsing(t, `
		import a = foo.a
		a.answer()
	`)
	resolver := newTestModuleResolverWithModules(mainModule, map[registry.LogicalURI]*ast.ContextModule{
		moduleA.Name: moduleA,
	})

	comp := compiler.New(resolver)
	comp.RegisterPlugin(&testExternPlugin{})
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}

	vmInstance := vm.New(comp.Bytecode())
	if err := vmInstance.Run(); err != nil {
		t.Fatalf("vm error: %s", err)
	}

	testExpectedValue(t, 42, vmInstance.LastPoppedStackElem())
}

func TestAttributeTypeAttribute(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "attribute type attribute lookup",
			input: `
			attr Meta { label }
			@Meta("Primary")
			attr Job { jobName }
			Meta(Job).label
			`,
			expected: "Primary",
		},
	}

	runVmTests(t, tests)
}

func TestAttributeValueAttribute(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "attribute value attribute lookup",
			input: `
			attr Meta { label }
			@Meta("Primary")
			attr Job { jobName }
			@Job("Singer")
			data Person {
				name
			}
			Meta(Job(Person("Luke"))).label
			`,
			expected: "Primary",
		},
	}

	runVmTests(t, tests)
}

func TestFastBenchmark(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
	fn fib(n) {
		return if n < 2 {
			n
		} else {
			fib(n-1) + fib(n-2)
		}
	}

	fib(10)
	`,
			expected: 55,
		},
	}
	runVmTests(t, tests)
}

func BenchmarkFib10(t *testing.B) {
	runBench(t, `
	fn fib(n) {
		return if n < 2 {
			n
		} else {
			fib(n-1) + fib(n-2)				
		}
	}

	fib(10)
	`)
}

func BenchmarkFib28(t *testing.B) {
	runBench(t, `
	fn fib(n) {
		return if n < 2 {
			n
		} else {
			fib(n-1) + fib(n-2)				
		}
	}

	fib(28)
	`)
}

func BenchmarkFib30(t *testing.B) {
	runBench(t, `
	fn fib(n) {
		return if n < 2 {
			n
		} else {
			fib(n-1) + fib(n-2)				
		}
	}

	fib(30)
	`)
}

func BenchmarkFib32(t *testing.B) {
	runBench(t, `
	fn fib(n) {
		return if n < 2 {
			n
		} else {
			fib(n-1) + fib(n-2)				
		}
	}

	fib(32)
	`)
}

func TestBasicVariables(t *testing.T) {
	tests := []vmTestCase{
		{input: "const a = 42\na", expected: 42},
		{input: "var b = 10\nb", expected: 10},
		{input: "const x = 1\nconst y = 2\nx + y", expected: 3},
		{input: "var x = 1\nvar y = 2\nx + y", expected: 3},
		{input: "const x = 5\nvar y = x\ny", expected: 5},
	}

	runVmTests(t, tests)
}

func TestConstAndVarLocalBindings(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
fn f() {
	const x = 10
	return x
}
f()`,
			expected: 10,
		},
		{
			input: `
fn f() {
	var x = 20
	return x
}
f()`,
			expected: 20,
		},
		{
			input: `
fn add() {
	const a = 3
	var b = 4
	return a + b
}
add()`,
			expected: 7,
		},
	}

	runVmTests(t, tests)
}

func TestAssignment(t *testing.T) {
	tests := []vmTestCase{
		// Local var rebind
		{
			label: "local var rebind",
			input: `
fn f() {
	var x = 5
	x = 10
	return x
}
f()`,
			expected: 10,
		},
		{
			label: "local var rebind multiple times",
			input: `
fn f() {
	var x = 1
	x = 2
	x = 3
	return x
}
f()`,
			expected: 3,
		},

		// Global var rebind
		{
			label: "global var rebind",
			input: `
var x = 5
x = 10
x`,
			expected: 10,
		},
		{
			label: "global var rebind then read in function",
			input: `
var counter = 0
fn inc() {
	counter = counter + 1
}
inc()
inc()
counter`,
			expected: 2,
		},

		// Const rebind errors
		{
			label: "const local rebind is a compile error",
			input: `
fn f() {
	const x = 5
	x = 10
}`,
			err: `testing:///test/test.zirr:4:2: cannot assign to a constant: x was declared with const`,
		},
		{
			label: "const global rebind is a compile error",
			input: `
const x = 5
x = 10`,
			err: `testing:///test/test.zirr:3:1: cannot assign to a constant: x was declared with const`,
		},
		{
			label: "parameter rebind is a compile error",
			input: `
fn f(x) {
	x = 10
}`,
			err: `testing:///test/test.zirr:3:2: cannot assign to a parameter: x`,
		},

		// Compound assignment operators
		{
			label: "local var += operator",
			input: `
fn f() {
	var x = 5
	x += 3
	return x
}
f()`,
			expected: 8,
		},
		{
			label: "local var -= operator",
			input: `
fn f() {
	var x = 10
	x -= 4
	return x
}
f()`,
			expected: 6,
		},
		{
			label: "local var *= operator",
			input: `
fn f() {
	var x = 3
	x *= 4
	return x
}
f()`,
			expected: 12,
		},
		{
			label: "local var /= operator",
			input: `
fn f() {
	var x = 12
	x /= 3
	return x
}
f()`,
			expected: 4,
		},
		{
			label: "local var %= operator",
			input: `
fn f() {
	var x = 10
	x %= 3
	return x
}
f()`,
			expected: 1,
		},

		// Member (field) assignment
		{
			label: "member assignment on var data instance",
			input: `
data Person {
	name
	age
}
var p = Person("Max", 42)
p.name = "Min"
p.name`,
			expected: "Min",
		},
		{
			label: "member assignment updates only the named field",
			input: `
data Person {
	name
	age
}
var p = Person("Max", 42)
p.age = 99
p.age`,
			expected: 99,
		},
		{
			label: "member assignment on const root (allowed, mutates object)",
			input: `
data Point {
	x
	y
}
const p = Point(1, 2)
p.x = 10
p.x`,
			expected: 10,
		},
		{
			label: "member compound assignment +=",
			input: `
data Counter {
	val
}
var c = Counter(10)
c.val += 5
c.val`,
			expected: 15,
		},

		// Index assignment
		{
			label: "array index assignment",
			input: `
var a = [10, 20, 30]
a[1] = 99
a[1]`,
			expected: 99,
		},
		{
			label: "array index assignment preserves other elements",
			input: `
fn f() {
	var a = [1, 2, 3]
	a[0] = 99
	return a[2]
}
f()`,
			expected: 3,
		},
		{
			label: "dict index assignment new key",
			input: `
var d = [1: "a"]
d[2] = "b"
d[2]`,
			expected: "b",
		},
		{
			label: "dict index assignment overwrite key",
			input: `
var d = [1: "a", 2: "b"]
d[1] = "z"
d[1]`,
			expected: "z",
		},
		{
			label: "array index compound assignment +=",
			input: `
fn f() {
	var a = [10, 20, 30]
	a[0] += 5
	return a[0]
}
f()`,
			expected: 15,
		},
	}

	runVmTests(t, tests)
}

func TestAssignmentAdditional(t *testing.T) {
	tests := []vmTestCase{
		// Global compound assignment
		{
			label: "global var compound +=",
			input: `
var x = 5
x += 3
x`,
			expected: 8,
		},
		{
			label: "global var compound -= then read in fn",
			input: `
var score = 100
fn penalty() {
	score -= 10
}
penalty()
penalty()
score`,
			expected: 80,
		},

		// Index compound with all operators
		{
			label: "array index compound -=",
			input: `
fn f() {
	var a = [100]
	a[0] -= 30
	return a[0]
}
f()`,
			expected: 70,
		},
		{
			label: "array index compound *=",
			input: `
fn f() {
	var a = [6]
	a[0] *= 7
	return a[0]
}
f()`,
			expected: 42,
		},
		{
			label: "array index compound /=",
			input: `
fn f() {
	var a = [20]
	a[0] /= 4
	return a[0]
}
f()`,
			expected: 5,
		},
		{
			label: "array index compound %=",
			input: `
fn f() {
	var a = [17]
	a[0] %= 5
	return a[0]
}
f()`,
			expected: 2,
		},

		// Dict compound assignment
		{
			label: "dict index compound +=",
			input: `
fn f() {
	var d = ["count": 10]
	d["count"] += 1
	return d["count"]
}
f()`,
			expected: 11,
		},

		// Member compound with remaining operators
		{
			label: "member compound -=",
			input: `
data Val { n }
var v = Val(100)
v.n -= 25
v.n`,
			expected: 75,
		},
		{
			label: "member compound *=",
			input: `
data Val { n }
var v = Val(3)
v.n *= 7
v.n`,
			expected: 21,
		},
		{
			label: "member compound /=",
			input: `
data Val { n }
var v = Val(20)
v.n /= 4
v.n`,
			expected: 5,
		},
		{
			label: "member compound %=",
			input: `
data Val { n }
var v = Val(17)
v.n %= 5
v.n`,
			expected: 2,
		},

		// Chained lvalue: x[i].f = v and x.f[i] = v
		{
			label: "chained x[i].field assignment",
			input: `
data Item { name }
fn f() {
	var items = [Item("a"), Item("b")]
	items[0].name = "z"
	return items[0].name
}
f()`,
			expected: "z",
		},
		{
			label: "chained x.field[i] assignment",
			input: `
data Box { values }
fn f() {
	var b = Box([1, 2, 3])
	b.values[1] = 99
	return b.values[1]
}
f()`,
			expected: 99,
		},

		// Plain % expression (regression: op.Mod was missing from definitions)
		{
			label:    "plain 10 % 3",
			input:    "10 % 3",
			expected: 1,
		},
		{
			label:    "plain modulo in expression",
			input:    "7 % 3",
			expected: 1,
		},
		{
			// Regression: compound member assignment must evaluate the object
			// expression exactly once, not twice. A counter incremented by a
			// helper fn is used to detect double evaluation.
			label: "compound member assignment evaluates object exactly once",
			input: `
data Box { val }
var calls = 0
var b = Box(10)
fn getBox() {
	calls += 1
	return b
}
getBox().val += 5
calls`,
			expected: 1,
		},
		{
			// Regression: compound index assignment must evaluate target exactly once.
			label: "compound index assignment evaluates target exactly once",
			input: `
var arr = [10, 20, 30]
var tCalls = 0
fn getArr() {
	tCalls += 1
	return arr
}
fn getIdx() { return 1 }
getArr()[getIdx()] += 5
tCalls`,
			expected: 1,
		},
	}

	runVmTests(t, tests)
}

// iterablePreamble adds the @Iterable attr and a stub panic fn that for-loops now reference.
const iterablePreamble = compositeTypePreamble + `
attr Iterable {
	iterate(value, yield)
}
fn panic(message) { }
`

func TestForStatements(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "infinite loop break",
			input: `
			fn example() {
				for { break }
				return 1
			}
			example()
			`,
			expected: 1,
		},
		{
			label: "conditional loop false",
			input: `
			fn example() {
				for false { return 2 }
				return 3
			}
			example()
			`,
			expected: 3,
		},
		{
			label: "conditional loop break",
			input: `
			fn example() {
				for true { break }
				return 4
			}
			example()
			`,
			expected: 4,
		},
		{
			label: "nested continue break",
			input: `
			fn example() {
				for {
					if false {
						continue
					} else {
						break
					}
				}
				return 5
			}
			example()
			`,
			expected: 5,
		},
		{
			label: "return inside loop",
			input: `
			fn example() {
				for { return 6 }
			}
			example()
			`,
			expected: 6,
		},
		{
			label: "statement after loop",
			input: `
			fn example() {
				for false { return 7 }
				const x = 8
				return x
			}
			example()
			`,
			expected: 8,
		},
		{
			label: "array collection loop literal",
			input: iterablePreamble + `
			fn example() {
				for item <- [1, 2] { return item }
				return 9
			}
			example()
			`,
			expected: 1,
		},
		{
			label: "array collection loop empty",
			input: iterablePreamble + `
			fn example() {
				for item <- [] { return 1 }
				return 2
			}
			example()
			`,
			expected: 2,
		},
		{
			label: "array for expression",
			input: iterablePreamble + `
			const result = for item <- [1, 2, 3] { item }
			result
			`,
			expected: []any{1, 2, 3},
		},
		{
			label: "array for expression decls",
			input: iterablePreamble + `
			const result = for item <- [1, 2, 3] {
				const doubled = item * 2
				doubled
			}
			result
			`,
			expected: []any{2, 4, 6},
		},
		{
			label: "array for expression continue",
			input: iterablePreamble + `
			const result = for item <- [1, 2, 3] {
				if item == 2 {
					continue
				}
				item
			}
			result
			`,
			expected: []any{1, 3},
		},
		{
			label: "for expression empty",
			input: `
			const result = for false { 1 }
			result
			`,
			expected: []any{},
		},
	}

	runVmTests(t, tests)
}

// TestForGenericIterable exercises `for x <- value` dispatch through @Iterable.iterate for a custom type.
func TestForGenericIterable(t *testing.T) {
	const counterPreamble = iterablePreamble + `
	@Iterable(_counterIterate)
	data Counter {
		limit
	}

	fn _counterIterate(v, yield) {
		var i = 0
		for {
			if i >= v.limit {
				return
			}
			if !yield(i) {
				break
			}
			i = i + 1
		}
	}
	`

	tests := []vmTestCase{
		{
			label: "sums all yielded values",
			input: counterPreamble + `
			fn example() {
				var sum = 0
				for x <- Counter(5) {
					sum = sum + x
				}
				return sum
			}
			example()
			`,
			expected: 0 + 1 + 2 + 3 + 4,
		},
		{
			label: "break stops iteration early",
			input: counterPreamble + `
			fn example() {
				var sum = 0
				for x <- Counter(5) {
					if x == 3 {
						break
					}
					sum = sum + x
				}
				return sum
			}
			example()
			`,
			expected: 0 + 1 + 2,
		},
		{
			label: "continue skips an element",
			input: counterPreamble + `
			fn example() {
				var sum = 0
				for x <- Counter(5) {
					if x == 2 {
						continue
					}
					sum = sum + x
				}
				return sum
			}
			example()
			`,
			expected: 0 + 1 + 3 + 4,
		},
		{
			label: "return unwinds through the reentrant iterate() call",
			input: counterPreamble + `
			fn example() {
				for x <- Counter(5) {
					if x == 3 {
						return x
					}
				}
				return -1
			}
			example()
			`,
			expected: 3,
		},
		{
			label: "return inside nested if/else unwinds correctly",
			input: counterPreamble + `
			fn example() {
				for x <- Counter(5) {
					if x == 2 {
						if true {
							return x * 10
						}
					} else {
						const noop = 0
					}
				}
				return -1
			}
			example()
			`,
			expected: 20,
		},
		{
			label: "nested generic for loops with a deep return",
			input: counterPreamble + `
			fn example() {
				for x <- Counter(3) {
					for y <- Counter(3) {
						if x == 1 && y == 1 {
							return x * 100 + y
						}
					}
				}
				return -1
			}
			example()
			`,
			expected: 101,
		},
		{
			label: "expression-form for over a custom iterable",
			input: counterPreamble + `
			const result = for x <- Counter(4) { x * 2 }
			result
			`,
			expected: []any{0, 2, 4, 6},
		},
		{
			label: "break in expression-form over a custom iterable",
			input: counterPreamble + `
			const result = for x <- Counter(10) {
				if x == 3 {
					break
				}
				x
			}
			result
			`,
			expected: []any{0, 1, 2},
		},
		{
			// Inner break/continue must stay scoped to the inner loop, not the outer one.
			label: "inner break/continue does not leak into the outer loop",
			input: counterPreamble + `
			fn example() {
				var visits = 0
				var innerSum = 0
				for x <- Counter(3) {
					visits = visits + 1
					for y <- Counter(5) {
						if y == 1 {
							continue
						}
						if y == 3 {
							break
						}
						innerSum = innerSum + 1
					}
				}
				return visits * 1000 + innerSum
			}
			example()
			`,
			// 3 outer visits; each inner loop adds y=0,2 (y=1 skipped, y=3 breaks) -> innerSum=6.
			expected: 3*1000 + 6,
		},
		{
			// The same for-loop site must dispatch fast-path vs generic at runtime, not compile time.
			label: "the same for-loop site dispatches polymorphically at runtime",
			input: counterPreamble + `
			fn sumOf(collection) {
				var sum = 0
				for x <- collection {
					sum = sum + x
				}
				return sum
			}
			sumOf([1, 2, 3]) + sumOf(Counter(4))
			`,
			// sumOf([1,2,3]) = 6 (Array fast path), sumOf(Counter(4)) = 0+1+2+3 = 6 (generic path)
			expected: 6 + 6,
		},
		{
			// Storing `yield` and calling it after iterate() returns must error clearly, not corrupt state.
			label: "stale yield called after iterate() already returned errors clearly",
			input: iterablePreamble + `
			var savedYield = void

			@Iterable(_misbehavingIterate)
			data Sneaky { value }

			fn _misbehavingIterate(v, yield) {
				savedYield = yield
			}

			fn example() {
				for x <- Sneaky(1) {}
				return savedYield(99)
			}
			example()
			`,
			err: "testing:///test/test.zirr:2:1: error calling extern function: yield called after its for-loop's iterate() call already returned",
		},
		{
			// panic is faked here (Bind only wires the real one inside a "prelude"-named module).
			label: "value without @Iterable produces a clear error",
			input: compositeTypePreamble + `
			attr Iterable {
				iterate(value, yield)
			}
			fn panic(message) {
				return 1 / 0
			}
			data NotIterable { value }
			for x <- NotIterable(1) { x }
			`,
			err: "testing:///test/test.zirr:17:12: division by zero",
		},
	}

	runVmTests(t, tests)
}

func runVmTests(t *testing.T, tests []vmTestCase) {
	t.Helper()

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d. %s", i, tt.label), func(t *testing.T) {
			module, program := prepareSourceFileParsing(t, tt.input)
			resolver := newTestModuleResolver(module)

			comp := compiler.New(resolver)
			err := comp.Compile(program)
			if err != nil {
				if tt.err != "" {
					if err.Error() != tt.err {
						t.Fatalf("expected error %q, got %q", tt.err, err)
					}
					return
				}
				t.Fatalf("compiler error: %s", err)
			}

			vm := vm.New(comp.Bytecode())
			err = vm.Run()
			if err != nil && tt.err == "" {
				t.Fatalf("vm error: %s", err)
			}

			if tt.err != "" {
				if err == nil || err.Error() != tt.err {
					t.Errorf("expected error %q, got %q", tt.err, err)
				}
			}
			if tt.expected != nil {
				stackElem := vm.LastPoppedStackElem()

				testExpectedValue(t, tt.expected, stackElem)
			}
		})
	}

}

func TestVMExtend(t *testing.T) {
	t.Run("ExtendConstants makes new constants accessible", func(t *testing.T) {
		module, program := prepareSourceFileParsing(t, "1")
		resolver := newTestModuleResolver(module)
		comp := compiler.New(resolver)
		if err := comp.Compile(program); err != nil {
			t.Fatal(err)
		}
		machine := vm.New(comp.Bytecode())
		if err := machine.Run(); err != nil {
			t.Fatal(err)
		}

		machine.ExtendConstants([]runtime.RuntimeValue{runtime.Int(42)})
		// Extending doesn't break anything: the VM is still usable.
	})

	t.Run("ExtendGlobals makes new globals accessible", func(t *testing.T) {
		module, program := prepareSourceFileParsing(t, "fn foo() {}")
		resolver := newTestModuleResolver(module)
		analysis := analyzer.New(resolver)
		if errs, _ := analysis.Analyze(module, false); len(errs) > 0 {
			t.Fatalf("analyze: %s", errs[0].Error())
		}
		comp := compiler.NewWithAnalyzer(resolver, analysis)
		if err := comp.Compile(program); err != nil {
			t.Fatal(err)
		}
		bytecode := comp.Bytecode()
		machine := vm.New(bytecode)
		if err := machine.Run(); err != nil {
			t.Fatal(err)
		}

		prevLen := len(bytecode.Globals)

		// Parse a new variable and compile it incrementally.
		l, err := lexer.New(staticmodule.NewSourceString("testing:///test/line2.zirr", "const x = 99"))
		if err != nil {
			t.Fatal(err)
		}
		src2 := parser.NewSourceParser(l, module.Decls, "line2.zirr").ParseSourceFile()
		module.AddSourceFile(src2)
		if errs := analysis.AnalyzeSourceFile(module, src2); len(errs) > 0 {
			t.Fatalf("analyze src2: %s", errs[0].Error())
		}
		_, err = comp.CompileSourceFileIncremental(src2)
		if err != nil {
			t.Fatal(err)
		}

		newBytecode := comp.Bytecode()
		machine.ExtendGlobals(newBytecode.Globals[prevLen:])
		machine.ExtendConstants(newBytecode.Constants)
	})

	t.Run("CallFunction executes a zero-arg compiled function", func(t *testing.T) {
		module, program := prepareSourceFileParsing(t, "fn addOne() { return 1 + 2 }")
		resolver := newTestModuleResolver(module)
		analysis := analyzer.New(resolver)
		if errs, _ := analysis.Analyze(module, false); len(errs) > 0 {
			t.Fatalf("analyze: %s", errs[0].Error())
		}
		comp := compiler.NewWithAnalyzer(resolver, analysis)
		if err := comp.Compile(program); err != nil {
			t.Fatal(err)
		}
		machine := vm.New(comp.Bytecode())
		if err := machine.Run(); err != nil {
			t.Fatal(err)
		}

		// Create a simple statement that returns 5.
		l, err := lexer.New(staticmodule.NewSourceString("testing:///test/init.zirr", "5"))
		if err != nil {
			t.Fatal(err)
		}
		src := parser.NewSourceParser(l, module.Decls, "init.zirr").ParseSourceFile()
		module.AddSourceFile(src)
		if errs := analysis.AnalyzeSourceFile(module, src); len(errs) > 0 {
			t.Fatalf("analyze src: %s", errs[0].Error())
		}

		initId, err := comp.CompileSourceFileIncremental(src)
		if err != nil {
			t.Fatal(err)
		}
		if initId < 0 {
			t.Fatal("expected __init__ function")
		}

		newBytecode := comp.Bytecode()
		machine.ExtendConstants(newBytecode.Constants)

		initFn := newBytecode.Constants[initId]
		result, err := machine.CallFunction(initFn)
		if err != nil {
			t.Fatalf("CallFunction: %s", err)
		}
		// __init__ returns void (implicit return after StmtExpr Pop)
		if result != nil {
			t.Logf("result: %v (type %T)", result, result)
		}
	})
}

func runBench(t *testing.B, input string) {
	module, program := prepareSourceFileParsing(t, input)
	resolver := newTestModuleResolver(module)

	comp := compiler.New(resolver)
	err := comp.Compile(program)
	if err != nil {
		t.Fatalf("compiler error: %s", err)
	}

	vm := vm.New(comp.Bytecode())
	err = vm.Run()

	if err != nil {
		t.Error(err)
	}
}

func prepareSourceFileParsing(t testing.TB, input string) (*ast.ContextModule, *ast.SourceFile) {
	l, err := lexer.New(staticmodule.NewSourceString("testing:///test/test.zirr", input))
	if err != nil {
		t.Fatal(err)
	}

	module := ast.MakeContextModule(registry.LogicalURI("test"))
	p := parser.NewSourceParser(l, module.Decls, "test.zirr")
	srcFile := p.ParseSourceFile()
	module.AddSourceFile(srcFile)
	checkParserErrors(t, p, input)
	return module, srcFile
}

func prepareContextModuleParsing(t testing.TB, uri string, input string) *ast.ContextModule {
	t.Helper()

	moduleURI := registry.LogicalURI(uri)
	src := staticmodule.NewSourceString(moduleURI.Join("main.zirr"), input)
	mod := staticmodule.NewModule(moduleURI, []registry.Source{src})
	mp := parser.NewModuleParse(mod)
	ctxMod, err := mp.Parse(mod)
	if err != nil {
		t.Fatal(err)
	}
	checkModuleParserErrors(t, mp, input)
	return ctxMod
}

func checkParserErrors(t testing.TB, p *parser.Parser, contents string) {
	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			src := err.Token.Source
			contentsBeforeOffset := contents[:src.Offset]
			loc := strings.Count(contentsBeforeOffset, "\n")
			lastLineIndex := strings.LastIndex(contentsBeforeOffset, "\n")
			col := src.Offset - lastLineIndex
			relevantLine, _, _ := strings.Cut(contents[lastLineIndex+1:], "\n")

			t.Errorf("%s:%d:%d: %s\n\n  %s\n  %s^\n  %s", err.Token.Source.File, loc, col, err.Summary, relevantLine, strings.Repeat(" ", col-1), err.Details)
		}
		t.FailNow()
	}
}

func checkModuleParserErrors(t testing.TB, mp *parser.ModuleParser, contents string) {
	t.Helper()

	if len(mp.Errors()) > 0 {
		for _, err := range mp.Errors() {
			src := err.Token.Source
			if src == nil {
				t.Errorf("<no source>: %q\n  %s", err.Token.Literal, err.Details)
				continue
			}
			contentsBeforeOffset := contents[:src.Offset]
			loc := strings.Count(contentsBeforeOffset, "\n")
			lastLineIndex := strings.LastIndex(contentsBeforeOffset, "\n")
			col := src.Offset - lastLineIndex
			relevantLine, _, _ := strings.Cut(contents[lastLineIndex+1:], "\n")

			t.Errorf("%s:%d:%d: %s\n\n  %s\n  %s^\n  %s", err.Token.Source.File, loc, col, err.Summary, relevantLine, strings.Repeat(" ", col-1), err.Details)
		}
		t.FailNow()
	}
}

type testModuleResolver struct {
	main    *ast.ContextModule
	modules map[registry.LogicalURI]*ast.ContextModule
}

func newTestModuleResolver(main *ast.ContextModule) testModuleResolver {
	modules := map[registry.LogicalURI]*ast.ContextModule{
		main.Name: main,
	}
	return testModuleResolver{main: main, modules: modules}
}

func newTestModuleResolverWithModules(main *ast.ContextModule, modules map[registry.LogicalURI]*ast.ContextModule) testModuleResolver {
	if modules == nil {
		modules = map[registry.LogicalURI]*ast.ContextModule{}
	}
	modules[main.Name] = main
	return testModuleResolver{main: main, modules: modules}
}

func (r testModuleResolver) MainModule() *ast.ContextModule {
	return r.main
}

func (r testModuleResolver) ResolveModule(ctx context.Context, name registry.LogicalURI) (*ast.ContextModule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	module, ok := r.modules[name]
	if !ok {
		return nil, fmt.Errorf("module %q not found", name)
	}
	return module, nil
}

func testExpectedValue(t *testing.T, expected interface{}, actual runtime.RuntimeValue) {
	t.Helper()
	err := testValue(expected, actual)
	if err != nil {
		t.Error(err)
	}
}

func testValue(expected interface{}, actual runtime.RuntimeValue) error {
	switch expected := expected.(type) {
	case runtime.Void:
		return testVoid(actual)
	case int:
		return testInt(int64(expected), actual)
	case bool:
		return testBool(bool(expected), actual)
	case rune:
		return testChar(expected, actual)
	case string:
		return testString(expected, actual)
	case []any:
		return testArray([]any(expected), actual)
	case map[any]any:
		return testDict(map[any]any(expected), actual)
	case data:
		return testData(expected, actual)
	case attribute:
		return testAttribute(expected, actual)
	default:
		return fmt.Errorf("unhandled type %T", expected)
	}
}

func testVoid(actual runtime.RuntimeValue) error {
	_, ok := actual.(runtime.Void)
	if !ok {
		return fmt.Errorf("object is not Void. got=%T (%+v)", actual, actual)
	}
	return nil
}

func testInt(expected int64, actual runtime.RuntimeValue) error {
	result, ok := actual.(runtime.Int)
	if !ok {
		return fmt.Errorf("object is not Integer. got=%T (%+v)", actual, actual)
	}

	if int64(result) != expected {
		return fmt.Errorf("object has wrong value. got=%d, want=%d",
			result, expected)
	}

	return nil
}

func testBool(expected bool, actual runtime.RuntimeValue) error {
	result, ok := actual.(runtime.Bool)
	if !ok {
		return fmt.Errorf("object is not Bool. got=%T (%+v)", actual, actual)
	}

	if bool(result) != expected {
		return fmt.Errorf("object has wrong value. got=%t, want=%t",
			result, expected)
	}

	return nil
}

func testChar(expected rune, actual runtime.RuntimeValue) error {
	result, ok := actual.(runtime.Char)
	if !ok {
		return fmt.Errorf("object is not Char. got=%T (%+v)", actual, actual)
	}

	if rune(result) != expected {
		return fmt.Errorf("object has wrong value. got=%q, want=%q", result, expected)
	}

	return nil
}

func testString(expected string, actual runtime.RuntimeValue) error {
	result, ok := actual.(runtime.String)
	if !ok {
		return fmt.Errorf("object is not String. got=%T (%+v)", actual, actual)
	}

	if string(result) != expected {
		return fmt.Errorf("object has wrong value. got=%q, want=%q", result, expected)
	}

	return nil
}

func testArray(expected []any, actual runtime.RuntimeValue) error {
	result, ok := actual.(runtime.Array)
	if !ok {
		return fmt.Errorf("object is not Array. got=%T (%+v)", actual, actual)
	}

	if len(expected) != len(result) {
		return fmt.Errorf("length does not match. got=%d, want=%d", len(result), len(expected))
	}
	for i, el := range result {
		err := testValue(expected[i], el)
		if err != nil {
			return fmt.Errorf("at index %d: %w", i, err)
		}
	}
	return nil
}

func testDict(expected map[any]any, actual runtime.RuntimeValue) error {
	result, ok := actual.(runtime.Dict)
	if !ok {
		return fmt.Errorf("object is not Dict. got=%T (%+v)", actual, actual)
	}

	if len(expected) != len(result) {
		return fmt.Errorf("length does not match. got=%d, want=%d", len(result), len(expected))
	}

	for key, el := range result {
		nkey, err := native(key)
		if err != nil {
			return fmt.Errorf("at index %q: %w", key, err)
		}
		err = testValue(expected[nkey], el)
		if err != nil {
			return fmt.Errorf("at index %q: %w", key, err)
		}
	}
	return nil
}

type data struct {
	typeId runtime.TypeId
	values []any
}

func testData(expected data, actual runtime.RuntimeValue) error {
	result, ok := actual.(*runtime.DataValue)
	if !ok {
		return fmt.Errorf("object is not Data. got=%T (%+v)", actual, actual)
	}

	if result.TypeConstantId() != expected.typeId {
		return fmt.Errorf("data type does not match. got=%q, want=%q", result.TypeConstantId(), expected.typeId)
	}

	if len(expected.values) != len(result.Values) {
		return fmt.Errorf("length does not match. got=%d, want=%d", len(result.Values), len(expected.values))
	}

	for i, el := range result.Values {
		err := testValue(expected.values[i], el)
		if err != nil {
			return fmt.Errorf("at index %d: %w", i, err)
		}
	}

	return nil
}

type attribute struct {
	typeId runtime.TypeId
	values []any
}

func testAttribute(expected attribute, actual runtime.RuntimeValue) error {
	result, ok := actual.(*runtime.AttributeValue)
	if !ok {
		return fmt.Errorf("object is not Attribute. got=%T (%+v)", actual, actual)
	}

	if result.TypeConstantId() != expected.typeId {
		return fmt.Errorf("attribute type does not match. got=%q, want=%q", result.TypeConstantId(), expected.typeId)
	}

	if len(expected.values) != len(result.Values) {
		return fmt.Errorf("length does not match. got=%d, want=%d", len(result.Values), len(expected.values))
	}

	for i, el := range result.Values {
		err := testValue(expected.values[i], el)
		if err != nil {
			return fmt.Errorf("at index %d: %w", i, err)
		}
	}

	return nil
}

func native(val runtime.RuntimeValue) (any, error) {
	switch val := val.(type) {
	case runtime.Bool:
		return bool(val), nil
	case runtime.Int:
		return int(val), nil
	case runtime.String:
		return string(val), nil
	default:
		return nil, fmt.Errorf("cannot convert %T into native Go type, got=%q", val, val.Inspect())
	}
}

func TestReplRollbackAndReuse(t *testing.T) {
	// Simulates the REPL rollback scenario:
	// 1. Failed `const x = undeclaredVar` — compile error, rollback symbol tables and compiler state.
	// 2. Successful `const x = 42`        — same name now works.
	// 3. `x`                            — must return 42 (not panic with index OOB).

	module := prepareContextModuleParsing(t, "test", "mod repl")
	resolver := newTestModuleResolver(module)
	analysis := analyzer.New(resolver)
	if errs, _ := analysis.Analyze(module, true); len(errs) > 0 {
		t.Fatalf("initial analyze: %s", errs[0].Error())
	}
	comp := compiler.NewWithAnalyzer(resolver, analysis)
	if err := comp.Compile(module); err != nil {
		t.Fatal(err)
	}
	machine := vm.New(comp.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatal(err)
	}

	evalLine := func(line, uri string) (runtime.RuntimeValue, error) {
		src := staticmodule.NewSourceString(registry.LogicalURI(uri), line)
		lex, err := lexer.New(src)
		if err != nil {
			return nil, err
		}

		declsBefore := make(map[string]struct{}, len(module.Decls.Symbols))
		for k := range module.Decls.Symbols {
			declsBefore[k] = struct{}{}
		}

		prs := parser.NewSourceParser(lex, module.Decls, uri)
		file := prs.ParseSourceFile()
		if len(prs.Errors()) > 0 {
			for k := range module.Decls.Symbols {
				if _, ok := declsBefore[k]; !ok {
					delete(module.Decls.Symbols, k)
				}
			}
			return nil, prs.Errors()[0]
		}
		module.AddSourceFile(file)

		symsBefore := make(map[string]struct{}, len(module.Symbols.Symbols))
		for k := range module.Symbols.Symbols {
			symsBefore[k] = struct{}{}
		}
		rollback := func() {
			module.Files = module.Files[:len(module.Files)-1]
			for k := range module.Decls.Symbols {
				if _, ok := declsBefore[k]; !ok {
					delete(module.Decls.Symbols, k)
				}
			}
			for k := range module.Symbols.Symbols {
				if _, ok := symsBefore[k]; !ok {
					delete(module.Symbols.Symbols, k)
				}
			}
		}

		if errs := analysis.AnalyzeSourceFile(module, file); len(errs) > 0 {
			rollback()
			return nil, fmt.Errorf("%s", errs[0])
		}

		prevG := len(comp.Bytecode().Globals)
		prevC := len(comp.Bytecode().Constants)
		id, err := comp.CompileSourceFileIncremental(file)
		if err != nil {
			// CompileSourceFileIncremental rolls back c.globals/c.constants internally.
			rollback()
			return nil, err
		}
		bc := comp.Bytecode()
		newG := bc.Globals[prevG:]
		newC := bc.Constants[prevC:]
		machine.ExtendGlobals(newG)
		machine.ExtendConstants(newC)

		if id < 0 {
			return nil, nil
		}
		return machine.CallFunction(bc.Constants[id])
	}

	// Step 1: fail — undeclaredVar doesn't exist; the compiler catches it.
	_, err := evalLine("const x = undeclaredVar", "testing:///test/line1.zirr")
	if err == nil {
		t.Fatal("expected compile error for undeclaredVar, got nil")
	}

	// Step 2: same name x must work now (rollback cleared the zombie).
	_, err = evalLine("const x = 42", "testing:///test/line2.zirr")
	if err != nil {
		t.Fatalf("expected success for const x = 42: %v", err)
	}

	// Step 3: reading x must return 42, not panic.
	val, err := evalLine("x", "testing:///test/line3.zirr")
	if err != nil {
		t.Fatalf("expected success reading x: %v", err)
	}
	if val == nil {
		t.Fatal("expected 42, got nil")
	}
	if val.Inspect() != "42" {
		t.Fatalf("expected 42, got %q", val.Inspect())
	}
}

func TestClosures(t *testing.T) {
	tests := []vmTestCase{
		// Basic lambda invocation
		{
			label: "lambda identity",
			input: `
			const id = fn(x) { return x }
			id(42)
			`,
			expected: 42,
		},
		// Const capture (by value)
		{
			label: "const capture by value",
			input: `
			fn outer() {
				const x = 10
				const f = fn() { return x }
				return f()
			}
			outer()
			`,
			expected: 10,
		},
		// Parameter capture (by value)
		{
			label: "parameter capture",
			input: `
			fn adder(a) {
				return fn(b) { return a + b }
			}
			const add5 = adder(5)
			add5(3)
			`,
			expected: 8,
		},
		// Var capture (shared mutable cell)
		{
			label: "var capture read",
			input: `
			fn outer() {
				var x = 1
				const f = fn() { return x }
				return f()
			}
			outer()
			`,
			expected: 1,
		},
		{
			label: "var capture mutation",
			input: `
			fn outer() {
				var x = 0
				const inc = fn() { x = x + 1 }
				inc()
				inc()
				return x
			}
			outer()
			`,
			expected: 2,
		},
		// Multiple closures sharing the same var cell
		{
			label: "shared var cell",
			input: `
			fn make() {
				var count = 0
				const inc = fn() { count = count + 1 }
				const get = fn() { return count }
				inc()
				inc()
				inc()
				return get()
			}
			make()
			`,
			expected: 3,
		},
		// Nested closures (transitive capture)
		{
			label: "nested closure const capture",
			input: `
			fn outer() {
				const x = 99
				fn middle() {
					const f = fn() { return x }
					return f()
				}
				return middle()
			}
			outer()
			`,
			expected: 99,
		},
		// Named function with captures
		{
			label: "named fn with param capture",
			input: `
			fn make(n) {
				fn add(m) { return n + m }
				return add(10)
			}
			make(5)
			`,
			expected: 15,
		},
		// Lambda with no captures (plain function)
		{
			label: "lambda no captures",
			input: `
			const double = fn(n) { return n * 2 }
			double(7)
			`,
			expected: 14,
		},
		// Closure over a global variable (no UpvalueCell needed)
		{
			label: "closure over global var",
			input: `
			var g = 100
			fn reader() { return g }
			reader()
			`,
			expected: 100,
		},
		// Var capture with compound assignment
		{
			label: "var capture compound assign",
			input: `
			fn counter() {
				var n = 0
				const step = fn() { n += 3 }
				step()
				step()
				return n
			}
			counter()
			`,
			expected: 6,
		},
		// Closure returned and called later
		{
			label: "returned closure",
			input: `
			fn makeCounter() {
				var n = 0
				return fn() {
					n = n + 1
					return n
				}
			}
			const c = makeCounter()
			c()
			c()
			c()
			`,
			expected: 3,
		},
		// Multiple captures in a single closure (const + var)
		{
			label: "multiple captures const and var",
			input: `
			fn combine() {
				const base = 10
				var offset = 5
				const f = fn() { return base + offset }
				offset = 20
				return f()
			}
			combine()
			`,
			expected: 30,
		},
		// Lambda with parameters and captures
		{
			label: "lambda with params and captures",
			input: `
			fn outer(x) {
				return fn(y) { return x * y }
			}
			const mul3 = outer(3)
			mul3(7)
			`,
			expected: 21,
		},
		// Two independent returned closures
		{
			label: "independent returned closures",
			input: `
			fn makeCounter() {
				var n = 0
				return fn() {
					n = n + 1
					return n
				}
			}
			const a = makeCounter()
			const b = makeCounter()
			a()
			a()
			a()
			b()
			`,
			expected: 1,
		},
		// Nested var capture (transitive mutable cell through 3 levels)
		{
			label: "nested var capture transitive",
			input: `
			fn outer() {
				var x = 0
				fn middle() {
					const inc = fn() { x = x + 1 }
					inc()
					inc()
				}
				middle()
				return x
			}
			outer()
			`,
			expected: 2,
		},
		// Closure capturing a named function
		{
			label: "capture named function",
			input: `
			fn outer() {
				fn helper(n) { return n * 2 }
				const f = fn(x) { return helper(x) + 1 }
				return f(5)
			}
			outer()
			`,
			expected: 11,
		},
		// Deeply nested transitive const capture (3 levels: fn → fn → lambda)
		{
			label: "deep transitive const capture",
			input: `
			fn a() {
				const val = 42
				fn b() {
					return fn() { return val }
				}
				const f = b()
				return f()
			}
			a()
			`,
			expected: 42,
		},
		// Closure in a conditional branch
		{
			label: "closure in conditional",
			input: `
			fn pick(flag) {
				const x = 10
				if flag {
					return fn() { return x + 1 }
				} else {
					return fn() { return x - 1 }
				}
			}
			const f = pick(true)
			f()
			`,
			expected: 11,
		},
	}

	runVmTests(t, tests)
}

func TestIsTypeOpcode(t *testing.T) {
	// These tests construct bytecode manually since no Zirric syntax
	// emits IsType yet (it will be used by switch/case @Type).

	makeSymbol := func(name string, constId, typeConstId int) *ast.Symbol {
		cid := constId
		tcid := typeConstId
		return &ast.Symbol{
			Name: name,
			Decl: &ast.DeclData{
				Name: ast.Identifier{Value: name},
			},
			ConstantId: &cid,
			TypeSymbol: &ast.Symbol{
				Name:       name + "Meta",
				ConstantId: &tcid,
			},
		}
	}

	t.Run("IsType with exact DataType match", func(t *testing.T) {
		// Constants:
		//   0 = DataType "A" (slot 0 in this manual bytecode; TypeId=11 via TypeSymbol)
		//   1 = DataValue with TypeId=11 (matching the DataType's TypeConstantId)
		// typeConstId=11 simulates a user-defined type that received an ID above the
		// builtin range (0-10), as assignModuleIDs now guarantees. The slot index (0)
		// intentionally differs from the TypeId (11) to verify that IsType uses
		// tv.TypeConstantId() rather than the raw slot index.
		symA := makeSymbol("A", 0, 11)
		dtA := &runtime.DataType{
			Symbol:       symA,
			FieldSymbols: nil,
		}
		dvA := &runtime.DataValue{TypeId: 11, Values: nil, Fields: nil}

		instructions := flatten(
			op.Make(op.Const, 1),  // push DataValue
			op.Make(op.IsType, 0), // IsType against constant[0] = DataType "A"
			op.Make(op.Pop),
		)

		bytecode := &compiler.Bytecode{
			Instructions: instructions,
			Constants:    []runtime.RuntimeValue{dtA, dvA},
			MainLocals:   0,
		}
		machine := vm.New(bytecode)
		if err := machine.Run(); err != nil {
			t.Fatalf("vm error: %s", err)
		}
		result := machine.LastPoppedStackElem()
		if result != runtime.Bool(true) {
			t.Errorf("expected true, got %v", result)
		}
	})

	t.Run("IsType with non-matching DataType", func(t *testing.T) {
		symA := makeSymbol("A", 0, 100)
		dtA := &runtime.DataType{
			Symbol:       symA,
			FieldSymbols: nil,
		}
		// DataValue with TypeId=20 (different from A's slot 0)
		dvB := &runtime.DataValue{TypeId: 20, Values: nil, Fields: nil}

		instructions := flatten(
			op.Make(op.Const, 1),  // push DataValue
			op.Make(op.IsType, 0), // IsType against constant[0] = DataType "A"
			op.Make(op.Pop),
		)

		bytecode := &compiler.Bytecode{
			Instructions: instructions,
			Constants:    []runtime.RuntimeValue{dtA, dvB},
			MainLocals:   0,
		}
		machine := vm.New(bytecode)
		if err := machine.Run(); err != nil {
			t.Fatalf("vm error: %s", err)
		}
		result := machine.LastPoppedStackElem()
		if result != runtime.Bool(false) {
			t.Errorf("expected false, got %v", result)
		}
	})

	t.Run("IsType with UnionType member", func(t *testing.T) {
		symU := makeSymbol("U", 30, 300)
		symU.Decl = &ast.DeclUnion{Name: ast.Identifier{Value: "U"}}
		union := runtime.MakeUnionType(symU, []runtime.TypeId{10, 20})

		dvA := &runtime.DataValue{TypeId: 10, Values: nil, Fields: nil}

		instructions := flatten(
			op.Make(op.Const, 1),  // push DataValue
			op.Make(op.IsType, 0), // IsType against UnionType "U"
			op.Make(op.Pop),
		)

		bytecode := &compiler.Bytecode{
			Instructions: instructions,
			Constants:    []runtime.RuntimeValue{union, dvA},
			MainLocals:   0,
		}
		machine := vm.New(bytecode)
		if err := machine.Run(); err != nil {
			t.Fatalf("vm error: %s", err)
		}
		result := machine.LastPoppedStackElem()
		if result != runtime.Bool(true) {
			t.Errorf("expected true (member of union), got %v", result)
		}
	})

	t.Run("IsType with UnionType non-member", func(t *testing.T) {
		symU := makeSymbol("U", 30, 300)
		symU.Decl = &ast.DeclUnion{Name: ast.Identifier{Value: "U"}}
		union := runtime.MakeUnionType(symU, []runtime.TypeId{10, 20})

		dvC := &runtime.DataValue{TypeId: 99, Values: nil, Fields: nil}

		instructions := flatten(
			op.Make(op.Const, 1),  // push DataValue
			op.Make(op.IsType, 0), // IsType against UnionType "U"
			op.Make(op.Pop),
		)

		bytecode := &compiler.Bytecode{
			Instructions: instructions,
			Constants:    []runtime.RuntimeValue{union, dvC},
			MainLocals:   0,
		}
		machine := vm.New(bytecode)
		if err := machine.Run(); err != nil {
			t.Fatalf("vm error: %s", err)
		}
		result := machine.LastPoppedStackElem()
		if result != runtime.Bool(false) {
			t.Errorf("expected false (not a member), got %v", result)
		}
	})

	t.Run("IsType with empty UnionType", func(t *testing.T) {
		symU := makeSymbol("U", 30, 300)
		symU.Decl = &ast.DeclUnion{Name: ast.Identifier{Value: "U"}}
		union := runtime.MakeUnionType(symU, []runtime.TypeId{})

		dvA := &runtime.DataValue{TypeId: 10, Values: nil, Fields: nil}

		instructions := flatten(
			op.Make(op.Const, 1),
			op.Make(op.IsType, 0),
			op.Make(op.Pop),
		)

		bytecode := &compiler.Bytecode{
			Instructions: instructions,
			Constants:    []runtime.RuntimeValue{union, dvA},
			MainLocals:   0,
		}
		machine := vm.New(bytecode)
		if err := machine.Run(); err != nil {
			t.Fatalf("vm error: %s", err)
		}
		result := machine.LastPoppedStackElem()
		if result != runtime.Bool(false) {
			t.Errorf("expected false (empty union), got %v", result)
		}
	})

	t.Run("IsType with primitive Int value", func(t *testing.T) {
		symU := makeSymbol("U", 30, 300)
		symU.Decl = &ast.DeclUnion{Name: ast.Identifier{Value: "U"}}
		intTypeId := runtime.Int(0).TypeConstantId()
		union := runtime.MakeUnionType(symU, []runtime.TypeId{intTypeId})

		instructions := flatten(
			op.Make(op.Const, 1),  // push Int(42)
			op.Make(op.IsType, 0), // IsType against union containing Int
			op.Make(op.Pop),
		)

		bytecode := &compiler.Bytecode{
			Instructions: instructions,
			Constants:    []runtime.RuntimeValue{union, runtime.Int(42)},
			MainLocals:   0,
		}
		machine := vm.New(bytecode)
		if err := machine.Run(); err != nil {
			t.Fatalf("vm error: %s", err)
		}
		result := machine.LastPoppedStackElem()
		if result != runtime.Bool(true) {
			t.Errorf("expected true (Int is member), got %v", result)
		}
	})
}

// flatten concatenates multiple byte slices into a single Instructions slice.
func flatten(slices ...[]byte) op.Instructions {
	var out op.Instructions
	for _, s := range slices {
		out = append(out, s...)
	}
	return out
}

func TestUnionDeclaration(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "union declaration does not error",
			input: `
			data A
			data B
			union AB {
				A
				B
			}
			1
			`,
			expected: 1,
		},
		{
			label: "union with inline data does not error",
			input: `
			union Shape {
				data Circle { radius }
				data Rect { width height }
			}
			Circle(5).radius
			`,
			expected: 5,
		},
		{
			label: "inline data members accessible from outside union",
			input: `
			union Shape {
				data Circle { radius }
				data Rect { width height }
			}
			Rect(10, 20).height
			`,
			expected: 20,
		},
	}

	runVmTests(t, tests)
}

func TestStringConcatenation(t *testing.T) {
	tests := []vmTestCase{
		{
			label:    "simple string concat",
			input:    `"hello" + " world"`,
			expected: "hello world",
		},
		{
			label:    "empty string concat",
			input:    `"" + ""`,
			expected: "",
		},
		{
			label:    "multi concat",
			input:    `"a" + "b" + "c"`,
			expected: "abc",
		},
		{
			label:    "concat in function",
			input:    "fn greet(name) { return \"Hello, \" + name + \"!\" }\ngreet(\"World\")",
			expected: "Hello, World!",
		},
	}

	runVmTests(t, tests)
}

func TestDataAttributeLookup(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "attribute on data type",
			input: `
			attr Label { text }
			@Label("MyStruct")
			data Foo { x }
			Label(Foo).text
			`,
			expected: "MyStruct",
		},
		{
			label: "attribute on data instance",
			input: `
			attr Label { text }
			@Label("MyStruct")
			data Foo { x }
			Label(Foo(42)).text
			`,
			expected: "MyStruct",
		},
		{
			label: "missing attribute returns void",
			input: `
			attr Label { text }
			attr Other { value }
			@Label("MyStruct")
			data Foo { x }
			Other(Foo)
			`,
			expected: runtime.Void{},
		},
	}

	runVmTests(t, tests)
}

func TestCrossModuleExternFnWithDataTypes(t *testing.T) {
	// The io module defines data types; the "mylib" module imports them
	// and uses extern fn to return instances constructed in Go.
	ioModule := prepareContextModuleParsing(t, "test.io", `
		mod io
		data Wrapper { value }
	`)

	mylibModule := prepareContextModuleParsing(t, "test.mylib", `
		mod mylib
		import io = test.io { Wrapper }
		extern fn wrap(x) -> Wrapper
	`)

	mainModule, program := prepareSourceFileParsing(t, `
		import mylib = test.mylib
		mylib.wrap(42).value
	`)

	modules := map[registry.LogicalURI]*ast.ContextModule{
		ioModule.Name:    ioModule,
		mylibModule.Name: mylibModule,
	}
	resolver := newTestModuleResolverWithModules(mainModule, modules)

	comp := compiler.New(resolver)
	comp.RegisterPlugin(&crossModuleTestPlugin{resolver: resolver})
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}

	machine := vm.New(comp.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatalf("vm error: %s", err)
	}

	testExpectedValue(t, 42, machine.LastPoppedStackElem())
}

// crossModuleTestPlugin demonstrates cross-module symbol resolution via BindContext.
type crossModuleTestPlugin struct {
	resolver testModuleResolver
}

func (p *crossModuleTestPlugin) Module() string { return "mylib" }

func (p *crossModuleTestPlugin) Bind(ctx runtime.BindContext, module *ast.SymbolTable, decl *ast.Symbol) runtime.RuntimeValue {
	switch decl.Name {
	case "wrap":
		wrapperSym := ctx.ResolveModuleSymbol("io", "Wrapper")
		return runtime.MakeExternFunc(decl, func(_ runtime.VMCaller, args []runtime.RuntimeValue) (runtime.RuntimeValue, error) {
			if wrapperSym == nil || wrapperSym.ConstantId == nil {
				return nil, fmt.Errorf("Wrapper type not resolved")
			}
			return &runtime.DataValue{
				TypeId: runtime.TypeId(*wrapperSym.ConstantId),
				Fields: map[string]int{"value": 0},
				Values: []runtime.RuntimeValue{args[0]},
			}, nil
		})
	}
	return nil
}

// TestResolveModuleMemberYieldsAConstructibleType covers why VMCaller can reach a module's members at all: an extern plugin has no way to obtain a declared type, and a DataValue built by hand carries no attributes, so a Result made that way would silently lack @AnyResult.
// Resolving the type and going through MakeDataValue keeps the attributes, which is what lets a plugin return a value the rest of the language treats as genuine.
func TestResolveModuleMemberYieldsAConstructibleType(t *testing.T) {
	moduleA := prepareContextModuleParsing(t, "foo.a", `
		mod a
		attr Marker {}

		@Marker()
		data Thing { value }
	`)
	mainModule, program := prepareSourceFileParsing(t, `
		import a = foo.a
		a.Thing(1)
	`)
	resolver := newTestModuleResolverWithModules(mainModule, map[registry.LogicalURI]*ast.ContextModule{
		moduleA.Name: moduleA,
	})

	comp := compiler.New(resolver)
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vmInstance := vm.New(comp.Bytecode())
	if err := vmInstance.Run(); err != nil {
		t.Fatalf("vm error: %s", err)
	}

	member, err := vmInstance.ResolveModuleMember("a", "Thing")
	if err != nil {
		t.Fatalf("resolve module member: %v", err)
	}
	dataType, ok := member.(*runtime.DataType)
	if !ok {
		t.Fatalf("expected a *runtime.DataType, got %T", member)
	}

	built := runtime.MakeDataValue(dataType, []runtime.RuntimeValue{runtime.Int(7)})
	if got := len(vmInstance.AttributesOf(built)); got == 0 {
		t.Fatal("expected a value built from the resolved type to carry its type's attributes")
	}
	if built.TypeConstantId() != vmInstance.LastPoppedStackElem().TypeConstantId() {
		t.Fatal("expected the same type as the one the program itself constructed")
	}

	byHand := &runtime.DataValue{TypeId: built.TypeId, Fields: built.Fields, Values: built.Values}
	if got := len(vmInstance.AttributesOf(byHand)); got != 0 {
		t.Fatalf("expected a hand-built DataValue to carry no attributes, got %d — the hazard this API exists to avoid", got)
	}

	if _, err := vmInstance.ResolveModuleMember("a", "Missing"); err == nil {
		t.Fatal("expected an error for a member the module does not export")
	}
	if _, err := vmInstance.ResolveModuleMember("nope", "Thing"); err == nil {
		t.Fatal("expected an error for a module that is not part of the program")
	}
}
