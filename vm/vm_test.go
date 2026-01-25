package vm_test

import (
	"fmt"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/ast"
	"code.knabel.dev/zirric-lang/zirric/compiler"
	"code.knabel.dev/zirric-lang/zirric/lexer"
	"code.knabel.dev/zirric-lang/zirric/parser"
	"code.knabel.dev/zirric-lang/zirric/registry/staticmodule"
	"code.knabel.dev/zirric-lang/zirric/runtime"
	"code.knabel.dev/zirric-lang/zirric/vm"
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
		{input: "true && 3", err: `unexpected type (runtime.Int "3")`},
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
		{input: "[1, 2, 3][3]", err: "array index 3 out of bounds"},
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
		{input: "func example() { return 42 }\nexample()", expected: 42},
		{input: "func example() { return }\nexample()", expected: nil},
		{input: `
		func example() {
			let x = 1
			return x + x
		}
		example()
		`, expected: 2},
		{
			label: "function with parameter",
			input: `
		func twice(n) {
			return n+n
		}
		twice(2)
		`, expected: 4},
	}

	runVmTests(t, tests)
}

func TestData(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "empty data",
			input: `
			data Example
			Example()
			`,
			expected: data{typeId: 0, values: []any{}},
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
			expected: data{typeId: 0, values: []any{
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

func TestDataAnnotations(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "annotation type call",
			input: `
			annotation Job {
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
			label: "missing annotation returns void",
			input: `
			annotation Job {
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

func TestFunctionAnnotations(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "function annotation lookup",
			input: `
			annotation Job {
				jobName
			}
			@Job("Singer")
			func greet() {}
			Job(greet).jobName
			`,
			expected: "Singer",
		},
		{
			label: "missing function annotation returns void",
			input: `
			annotation Job {
				jobName
			}
			func greet() {}
			Job(greet)
			`,
			expected: runtime.Void{},
		},
	}

	runVmTests(t, tests)
}

func TestExternAnnotations(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "extern func annotation lookup",
			input: `
			annotation Job { jobName }
			@Job("Singer")
			extern func greet(name)
			Job(greet).jobName
			`,
			expected: "Singer",
		},
		{
			label: "extern type annotation lookup",
			input: `
			annotation Job { jobName }
			@Job("Actor")
			extern type Person {}
			Job(Person).jobName
			`,
			expected: "Actor",
		},
	}

	runVmTests(t, tests)
}

func TestAnnotationTypeAnnotations(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "annotation type annotation lookup",
			input: `
			annotation Meta { label }
			@Meta("Primary")
			annotation Job { jobName }
			Meta(Job).label
			`,
			expected: "Primary",
		},
	}

	runVmTests(t, tests)
}

func TestAnnotationValueAnnotations(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "annotation value annotation lookup",
			input: `
			annotation Meta { label }
			@Meta("Primary")
			annotation Job { jobName }
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

func BenchmarkFib10(t *testing.B) {
	runBench(t, `
	func fib(n) {
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
	func fib(n) {
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
	func fib(n) {
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
	func fib(n) {
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
		{input: "let a = 42\na", expected: 42},
	}

	runVmTests(t, tests)
}

func TestForStatements(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "infinite loop break",
			input: `
			func example() {
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
			func example() {
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
			func example() {
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
			func example() {
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
			func example() {
				for { return 6 }
			}
			example()
			`,
			expected: 6,
		},
		{
			label: "statement after loop",
			input: `
			func example() {
				for false { return 7 }
				let x = 8
				return x
			}
			example()
			`,
			expected: 8,
		},
		{
			label: "array collection loop literal",
			input: `
			func example() {
				for item <- [1, 2] { return item }
				return 9
			}
			example()
			`,
			expected: 1,
		},
		{
			label: "array collection loop empty",
			input: `
			func example() {
				for item <- [] { return 1 }
				return 2
			}
			example()
			`,
			expected: 2,
		},
		{
			label: "array for expression",
			input: `
			let result = for item <- [1, 2, 3] { item }
			result
			`,
			expected: []any{1, 2, 3},
		},
		{
			label: "array for expression decls",
			input: `
			let result = for item <- [1, 2, 3] {
				let doubled = item * 2
				doubled
			}
			result
			`,
			expected: []any{2, 4, 6},
		},
		{
			label: "array for expression continue",
			input: `
			let result = for item <- [1, 2, 3] {
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
			let result = for false { 1 }
			result
			`,
			expected: []any{},
		},
	}

	runVmTests(t, tests)
}

func runVmTests(t *testing.T, tests []vmTestCase) {
	t.Helper()

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d. %s", i, tt.label), func(t *testing.T) {
			program := prepareSourceFileParsing(t, tt.input)

			comp := compiler.New()
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

func runBench(t *testing.B, input string) {
	program := prepareSourceFileParsing(t, input)

	comp := compiler.New()
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

func prepareSourceFileParsing(t testing.TB, input string) *ast.SourceFile {
	l, err := lexer.New(staticmodule.NewSourceString("testing:///test/test.zirr", input))
	if err != nil {
		t.Fatal(err)
	}

	p := parser.NewSourceParser(l, nil, "test.zirr")
	srcFile := p.ParseSourceFile()
	checkParserErrors(t, p, input)
	return srcFile
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
	case annotation:
		return testAnnotation(expected, actual)
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

type annotation struct {
	typeId runtime.TypeId
	values []any
}

func testAnnotation(expected annotation, actual runtime.RuntimeValue) error {
	result, ok := actual.(*runtime.AnnotationValue)
	if !ok {
		return fmt.Errorf("object is not Annotation. got=%T (%+v)", actual, actual)
	}

	if result.TypeConstantId() != expected.typeId {
		return fmt.Errorf("annotation type does not match. got=%q, want=%q", result.TypeConstantId(), expected.typeId)
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
