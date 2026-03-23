package compiler_test

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/compiler"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	code "code.knabel.dev/zirric-lang/zirric/pkg/op"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
	"github.com/google/go-cmp/cmp"
)

type compilerTestCase struct {
	label                string
	input                string
	expectedConstants    []interface{}
	expectedGlobals      [][]code.Instructions
	expectedInstructions []code.Instructions
}

func TestUnaryOperators(t *testing.T) {
	tests := []compilerTestCase{
		{
			input:             "!true",
			expectedConstants: nil,
			expectedInstructions: []code.Instructions{
				code.Make(code.ConstTrue),
				code.Make(code.Invert),
				code.Make(code.Pop),
			},
		},
		{
			input:             "-3",
			expectedConstants: []any{3},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Negate),
				code.Make(code.Pop),
			},
		},
		{
			input:             "+42",
			expectedConstants: []any{42},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Pop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestCharLiterals(t *testing.T) {
	tests := []compilerTestCase{
		{
			label:             "simple char",
			input:             "'a'",
			expectedConstants: []any{'a'},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Pop),
			},
		},
		{
			label:             "escaped newline",
			input:             "'\\n'",
			expectedConstants: []any{'\n'},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Pop),
			},
		},
		{
			label:             "escaped quote",
			input:             "'\\''",
			expectedConstants: []any{'\''},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Pop),
			},
		},
		{
			label:             "escaped backslash",
			input:             "'\\\\'",
			expectedConstants: []any{'\\'},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Pop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestNumberLiterals(t *testing.T) {
	tests := []compilerTestCase{
		{
			label:             "decimal integer",
			input:             "42",
			expectedConstants: []any{42},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Pop),
			},
		},
		{
			label:             "hexadecimal integer",
			input:             "0xFFF",
			expectedConstants: []any{4095}, // 0xFFF = 4095
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Pop),
			},
		},
		{
			label:             "octal integer",
			input:             "0777",
			expectedConstants: []any{511}, // 0777 = 511
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Pop),
			},
		},
		{
			label:             "binary integer",
			input:             "0b101010",
			expectedConstants: []any{42}, // 0b101010 = 42
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Pop),
			},
		},
		{
			label:             "float literal",
			input:             "3.14",
			expectedConstants: []any{3.14},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Pop),
			},
		},
		{
			label:             "scientific notation float",
			input:             "2e10",
			expectedConstants: []any{2e10},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Pop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestBinaryOperators(t *testing.T) {
	tests := []compilerTestCase{
		{
			input:             "1 + 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				// 1
				code.Make(code.Const, 0),
				// 2
				code.Make(code.Const, 1),
				// +
				code.Make(code.Add),
				// drop expr
				code.Make(code.Pop),
			},
		},
		{
			input:             "1 - 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				// 1
				code.Make(code.Const, 0),
				// 2
				code.Make(code.Const, 1),
				// -
				code.Make(code.Sub),
				// drop expr
				code.Make(code.Pop),
			},
		},
		{
			input:             "1 * 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				// 1
				code.Make(code.Const, 0),
				// 2
				code.Make(code.Const, 1),
				// *
				code.Make(code.Mul),
				// drop expr
				code.Make(code.Pop),
			},
		},
		{
			input:             "1 / 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				// 1
				code.Make(code.Const, 0),
				// 2
				code.Make(code.Const, 1),
				// /
				code.Make(code.Div),
				// drop expr
				code.Make(code.Pop),
			},
		},
		{
			input:             "1 == 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				// 1
				code.Make(code.Const, 0),
				// 2
				code.Make(code.Const, 1),
				// ==
				code.Make(code.Equal),
				// drop expr
				code.Make(code.Pop),
			},
		},
		{
			input:             "1 != 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				// 1
				code.Make(code.Const, 0),
				// 2
				code.Make(code.Const, 1),
				// !=
				code.Make(code.NotEqual),
				// drop expr
				code.Make(code.Pop),
			},
		},
		{
			input:             "1 > 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				// 1
				code.Make(code.Const, 0),
				// 2
				code.Make(code.Const, 1),
				// >
				code.Make(code.GreaterThan),
				// drop expr
				code.Make(code.Pop),
			},
		},
		{
			input:             "1 < 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				// 1
				code.Make(code.Const, 0),
				// 2
				code.Make(code.Const, 1),
				// <
				code.Make(code.LessThan),
				// drop expr
				code.Make(code.Pop),
			},
		},
		{
			input:             "1 >= 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				// 1
				code.Make(code.Const, 0),
				// 2
				code.Make(code.Const, 1),
				// >=
				code.Make(code.GreaterThanOrEqual),
				// drop expr
				code.Make(code.Pop),
			},
		},
		{
			input:             "1 <= 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				// 1
				code.Make(code.Const, 0),
				// 2
				code.Make(code.Const, 1),
				// <=
				code.Make(code.LessThanOrEqual),
				// drop expr
				code.Make(code.Pop),
			},
		},
		{
			input:             "1 % 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				// 1
				code.Make(code.Const, 0),
				// 2
				code.Make(code.Const, 1),
				// %
				code.Make(code.Mod),
				// drop expr
				code.Make(code.Pop),
			},
		},
		{
			input:             "true && false",
			expectedConstants: []interface{}{},
			expectedInstructions: []code.Instructions{
				// left
				code.Make(code.ConstTrue),
				// when false do not exectue right
				code.Make(code.JumpFalse, 11),
				// right
				code.Make(code.ConstFalse),
				code.Make(code.AssertType, int(runtime.Bool(true).TypeConstantId())),
				// result is right
				code.Make(code.Jump, 12),
				// put false back up
				code.Make(code.ConstFalse),
				// drop expr
				code.Make(code.Pop),
			},
		},
		{
			input:             "true || false",
			expectedConstants: []interface{}{},
			expectedInstructions: []code.Instructions{
				// left
				code.Make(code.ConstTrue),
				// when true do not exectue right
				code.Make(code.JumpTrue, 11),
				// right
				code.Make(code.ConstFalse),
				code.Make(code.AssertType, int(runtime.Bool(true).TypeConstantId())),
				// result is right
				code.Make(code.Jump, 12),
				// put true back up
				code.Make(code.ConstTrue),
				// drop expr
				code.Make(code.Pop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func TesEQtIfStmtsArithmetic(t *testing.T) {
	tests := []compilerTestCase{
		{
			input:             "if 1 { 2 } else { 3 }",
			expectedConstants: []interface{}{1, 2, 3},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.JumpFalse, 13),
				code.Make(code.Const, 1),
				code.Make(code.Pop),
				code.Make(code.Jump, 17),
				code.Make(code.Const, 2),
				code.Make(code.Pop),
			},
		},
		{
			input:             "if 1 { 2 }",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.JumpFalse, 10),
				code.Make(code.Const, 1),
				code.Make(code.Pop),
			},
		},
		{
			input:             "if 0 { 1 } else if 2 { 3 } else { 4 }",
			expectedConstants: []interface{}{0, 1, 2, 3, 4},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.JumpFalse, 13),
				code.Make(code.Const, 1),
				code.Make(code.Pop),
				code.Make(code.Jump, 30),
				code.Make(code.Const, 2),
				code.Make(code.JumpFalse, 26),
				code.Make(code.Const, 3),
				code.Make(code.Pop),
				code.Make(code.Jump, 30),
				code.Make(code.Const, 4),
				code.Make(code.Pop),
			},
		},
		{
			input:             "if 0 { 1 } else if 2 { 3 }",
			expectedConstants: []interface{}{0, 1, 2, 3},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.JumpFalse, 13),
				code.Make(code.Const, 1),
				code.Make(code.Pop),
				code.Make(code.Jump, 23),
				code.Make(code.Const, 2),
				code.Make(code.JumpFalse, 23),
				code.Make(code.Const, 3),
				code.Make(code.Pop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestIfExpressionsArithmetic(t *testing.T) {
	tests := []compilerTestCase{
		{
			input:             "(if 1 { 2 } else { 3 })",
			expectedConstants: []interface{}{1, 2, 3},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.JumpFalse, 12),
				code.Make(code.Const, 1),
				code.Make(code.Jump, 15),
				code.Make(code.Const, 2),
				code.Make(code.Pop),
			},
		},
		{
			input:             "(if 0 { 1 } else if 2 { 3 } else { 4 })",
			expectedConstants: []interface{}{0, 1, 2, 3, 4},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.JumpFalse, 12),
				code.Make(code.Const, 1),
				code.Make(code.Jump, 27),
				code.Make(code.Const, 2),
				code.Make(code.JumpFalse, 24),
				code.Make(code.Const, 3),
				code.Make(code.Jump, 27),
				code.Make(code.Const, 4),
				code.Make(code.Pop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestArrayExpressions(t *testing.T) {
	tests := []compilerTestCase{
		{
			input:             "[]",
			expectedConstants: []any{0},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Array),
				code.Make(code.Pop),
			},
		},
		{
			input:             "[42, 1337]",
			expectedConstants: []any{42, 1337, 2},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Const, 1),
				code.Make(code.Const, 2),
				code.Make(code.Array),
				code.Make(code.Pop),
			},
		},
		{
			input:             "[42 + 1337]",
			expectedConstants: []any{42, 1337, 1},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Const, 1),
				code.Make(code.Add),
				code.Make(code.Const, 2),
				code.Make(code.Array),
				code.Make(code.Pop),
			},
		},
		{
			label:             "array index",
			input:             "[1, 2, 3][1]",
			expectedConstants: []any{1, 2, 3, 3, 1},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Const, 1),
				code.Make(code.Const, 2),
				code.Make(code.Const, 3),
				code.Make(code.Array),
				code.Make(code.Const, 4),
				code.Make(code.GetIndex),
				code.Make(code.Pop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestDictExpressions(t *testing.T) {
	tests := []compilerTestCase{
		{
			input:             "[:]",
			expectedConstants: []any{0},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Dict),
				code.Make(code.Pop),
			},
		},
		{
			label:             "dict with two key-value pairs",
			input:             "[1: 2, 3: 4]",
			expectedConstants: []any{1, 2, 3, 4, 2},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Const, 1),
				code.Make(code.Const, 2),
				code.Make(code.Const, 3),
				code.Make(code.Const, 4),
				code.Make(code.Dict),
				code.Make(code.Pop),
			},
		},
		{
			label:             "dict with expressions",
			input:             "[1 + 1: 2 * 2]",
			expectedConstants: []any{1, 1, 2, 2, 1},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Const, 1),
				code.Make(code.Add),
				code.Make(code.Const, 2),
				code.Make(code.Const, 3),
				code.Make(code.Mul),
				code.Make(code.Const, 4),
				code.Make(code.Dict),
				code.Make(code.Pop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestDeclFunction(t *testing.T) {
	tests := []compilerTestCase{
		{
			label: "function with value return",
			input: "fn example() { return 42 }",
			expectedConstants: []any{
				compiledFunction{
					name:   "example",
					params: 0,
					ins: []code.Instructions{
						code.Make(code.Const, 1),
						code.Make(code.Return),
					},
				},
				42,
			},
			expectedInstructions: []code.Instructions{},
		},
		{
			label: "function call with value return",
			input: "fn example() { return 42 }\nexample()",
			expectedConstants: []any{
				compiledFunction{
					name:   "example",
					params: 0,
					ins: []code.Instructions{
						code.Make(code.Const, 1),
						code.Make(code.Return),
					},
				},
				42,
			},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Call, 0),
				code.Make(code.Pop),
			},
		},
		{
			label: "function with blank return",
			input: "fn example() { return }",
			expectedConstants: []any{
				compiledFunction{
					name:   "example",
					params: 0,
					ins: []code.Instructions{
						code.Make(code.ConstVoid),
						code.Make(code.Return),
					},
				},
			},
			expectedInstructions: []code.Instructions{},
		},
		{
			label: "function call with blank return",
			input: "fn example() { return }\nexample()",
			expectedConstants: []any{
				compiledFunction{
					name:   "example",
					params: 0,
					ins: []code.Instructions{
						code.Make(code.ConstVoid),
						code.Make(code.Return),
					},
				},
			},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Call, 0),
				code.Make(code.Pop),
			},
		},
		{
			label: "function can access global variables",
			input: `
			const x = 42
			fn example() {
				return x
			}
			`,
			expectedConstants: []any{
				compiledFunction{
					name:   "example",
					params: 0,
					ins: []code.Instructions{
						code.Make(code.GetGlobal, 0),
						code.Make(code.Return),
					},
				},
				42,
			},
			expectedGlobals: [][]code.Instructions{
				{code.Make(code.Const, 1)},
			},
			expectedInstructions: []code.Instructions{},
		},
		{
			label: "function can access global variables declared after usage",
			input: `
			fn example() {
				return x
			}
			const x = 42
			`,
			expectedConstants: []any{
				compiledFunction{
					name:   "example",
					params: 0,
					ins: []code.Instructions{
						code.Make(code.GetGlobal, 0),
						code.Make(code.Return),
					},
				},
				42,
			},
			expectedGlobals: [][]code.Instructions{
				{code.Make(code.Const, 1)},
			},
			expectedInstructions: []code.Instructions{},
		},
		{
			label: "function with local variable",
			input: `
			fn example() {
				const x = 42
				return x+x
			}
			`,
			expectedConstants: []any{
				compiledFunction{
					name:   "example",
					params: 0,
					ins: []code.Instructions{
						code.Make(code.Const, 1),
						code.Make(code.SetLocal, 0),
						code.Make(code.GetLocal, 0),
						code.Make(code.GetLocal, 0),
						code.Make(code.Add),
						code.Make(code.Return),
					},
				},
				42,
			},
			expectedInstructions: []code.Instructions{},
		},
	}

	runCompilerTests(t, tests)
}

func TestVariables(t *testing.T) {
	tests := []compilerTestCase{
		{
			label: "const global binding",
			input: "const a = 42\na",
			expectedConstants: []any{
				42,
			},
			expectedGlobals: [][]code.Instructions{
				{code.Make(code.Const, 0)},
			},
			expectedInstructions: []code.Instructions{
				code.Make(code.GetGlobal, 0),
				code.Make(code.Pop),
			},
		},
		{
			label: "var global binding",
			input: "var a = 42\na",
			expectedConstants: []any{
				42,
			},
			expectedGlobals: [][]code.Instructions{
				{code.Make(code.Const, 0)},
			},
			expectedInstructions: []code.Instructions{
				code.Make(code.GetGlobal, 0),
				code.Make(code.Pop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestLocalConstAndVar(t *testing.T) {
	tests := []compilerTestCase{
		{
			label: "const local binding",
			input: `fn f() { const x = 7 return x }`,
			expectedConstants: []any{
				compiledFunction{
					name:   "f",
					params: 0,
					ins: []code.Instructions{
						code.Make(code.Const, 1),
						code.Make(code.SetLocal, 0),
						code.Make(code.GetLocal, 0),
						code.Make(code.Return),
					},
				},
				7,
			},
		},
		{
			label: "var local binding",
			input: `fn f() { var x = 7 return x }`,
			expectedConstants: []any{
				compiledFunction{
					name:   "f",
					params: 0,
					ins: []code.Instructions{
						code.Make(code.Const, 1),
						code.Make(code.SetLocal, 0),
						code.Make(code.GetLocal, 0),
						code.Make(code.Return),
					},
				},
				7,
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestDeclData(t *testing.T) {
	tests := []compilerTestCase{
		{
			label: "empty data declaration",
			input: `data Example`,
			expectedConstants: []any{
				compiledDataType{
					name:   "Example",
					fields: []compiledField{},
				},
			},
		},
		{
			label: "data declaration",
			input: `data Example { field }`,
			expectedConstants: []any{
				compiledDataType{
					name:   "Example",
					fields: []compiledField{{name: "field"}},
				},
			},
		},
		{
			label: "data declaration and call",
			input: `
				data Person {
					name
				}
				Person("Max").name
				`,
			expectedConstants: []any{
				compiledDataType{
					name: "Person",
					fields: []compiledField{
						{name: "name"},
					},
				},
				"Max",
				"name",
			},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 1),
				code.Make(code.Const, 0),
				code.Make(code.Call, 1),
				code.Make(code.GetField, 2),
				code.Make(code.Pop),
			},
		},
		{
			label: "data declaration and call with two fields",
			input: `
				data Person {
					name
					age
				}
				Person("Max", 42).name
				`,
			expectedConstants: []any{
				compiledDataType{
					name: "Person",
					fields: []compiledField{
						{name: "name"},
						{name: "age"},
					},
				},
				"Max",
				42,
				"name",
			},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 1),
				code.Make(code.Const, 2),
				code.Make(code.Const, 0),
				code.Make(code.Call, 2),
				code.Make(code.GetField, 3),
				code.Make(code.Pop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestModuleDecl(t *testing.T) {
	tests := []struct {
		label                string
		input                string
		expectedGlobals      [][]code.Instructions
		expectedConstants    []any
		expectedInstructions []code.Instructions
	}{
		{
			label: "module declaration provides global slot",
			input: "mod foo",
			expectedGlobals: [][]code.Instructions{
				{
					code.Make(code.Const, 0),
					code.Make(code.Module, 1),
				},
			},
			expectedConstants: []any{
				0,
				"module.test",
			},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Module, 1),
			},
		},
		{
			label: "module identifier emits GetGlobal",
			input: "mod foo\nfoo",
			expectedGlobals: [][]code.Instructions{
				{
					code.Make(code.Const, 0),
					code.Make(code.Call, 0),
					code.Make(code.Pop),
					code.Make(code.Const, 1),
					code.Make(code.Module, 2),
				},
			},
			expectedConstants: []any{
				synthCompiledFunction{
					ins: []code.Instructions{
						code.Make(code.GetGlobal, 0),
						code.Make(code.Pop),
						code.Make(code.ConstVoid),
						code.Make(code.Return),
					},
				},
				0,
				"module.test",
			},
			expectedInstructions: []code.Instructions{
				code.Make(code.Const, 0),
				code.Make(code.Call, 0),
				code.Make(code.Pop),
				code.Make(code.Const, 1),
				code.Make(code.Module, 2),
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d. %s", i, tt.label), func(t *testing.T) {
			module := prepareContextModuleParsing(t, "module.test", tt.input)
			resolver := newTestModuleResolver(module, nil)
			comp := compiler.New(resolver)
			if err := comp.Compile(module); err != nil {
				t.Fatalf("compile: %s", err)
			}

			bytecode := comp.Bytecode()

			if err := testInstructions(t, tt.expectedInstructions, bytecode.Instructions); err != nil {
				t.Fatalf("testInstructions failed: %s", err)
			}
			if err := testConstants(t, tt.expectedConstants, bytecode.Constants); err != nil {
				t.Fatalf("testConstants failed: %s", err)
			}
			if err := testGlobals(t, tt.expectedGlobals, bytecode.Globals); err != nil {
				t.Fatalf("testGlobals failed: %s", err)
			}
		})
	}
}

func TestModuleDeclRequiresContextModule(t *testing.T) {
	module, program := prepareSourceFileParsing(t, "mod foo")
	resolver := newTestModuleResolver(module, nil)
	comp := compiler.New(resolver)
	if err := comp.Compile(program); err == nil {
		t.Fatal("expected error when compiling module declaration without context module")
	}
}

func TestImports(t *testing.T) {
	tests := []struct {
		label                string
		input                string
		expectedGlobals      [][]code.Instructions
		expectedInstructions []code.Instructions
	}{
		{
			label: "import emits global lookup",
			input: "import foo.bar\nbar",
			expectedGlobals: [][]code.Instructions{
				{
					code.Make(code.Const, 0),
					code.Make(code.Module, 1),
				},
			},
			expectedInstructions: []code.Instructions{
				code.Make(code.GetGlobal, 0),
				code.Make(code.Pop),
			},
		},
		{
			label: "imported module is compiled once",
			input: "import a = foo.bar\nimport b = foo.bar\na\nb",
			expectedGlobals: [][]code.Instructions{
				{
					code.Make(code.Const, 0),
					code.Make(code.Module, 1),
				},
			},
			expectedInstructions: []code.Instructions{
				code.Make(code.GetGlobal, 0),
				code.Make(code.Pop),
				code.Make(code.GetGlobal, 0),
				code.Make(code.Pop),
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d. %s", i, tt.label), func(t *testing.T) {
			module := ast.MakeContextModule(registry.LogicalURI("foo.bar"))
			programModule, program := prepareSourceFileParsing(t, tt.input)
			resolver := newTestModuleResolver(programModule, map[registry.LogicalURI]*ast.ContextModule{
				module.Name: module,
			})

			comp := compiler.New(resolver)
			if err := comp.Compile(program); err != nil {
				t.Fatalf("compile: %s", err)
			}

			bytecode := comp.Bytecode()

			if err := testInstructions(t, tt.expectedInstructions, bytecode.Instructions); err != nil {
				t.Fatalf("testInstructions failed: %s", err)
			}
			if err := testGlobals(t, tt.expectedGlobals, bytecode.Globals); err != nil {
				t.Fatalf("testGlobals failed: %s", err)
			}
		})
	}
}

func TestModuleValueMultipleModules(t *testing.T) {
	moduleA := prepareContextModuleParsing(t, "foo.a", `
		mod a
		const pub = 1
		const _priv = 2
	`)
	moduleB := prepareContextModuleParsing(t, "bar.b", `
		mod b
		fn compute() { return 1 }
		const val = 3
		const _priv = 4
	`)
	mainModule, program := prepareSourceFileParsing(t, `
		import a = foo.a
		import b = bar.b
	`)
	resolver := newTestModuleResolver(mainModule, map[registry.LogicalURI]*ast.ContextModule{
		moduleA.Name: moduleA,
		moduleB.Name: moduleB,
	})

	comp := compiler.New(resolver)
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compile: %s", err)
	}

	bytecode := comp.Bytecode()

	aSym := program.Symbols.Symbols["a"]
	if aSym == nil || aSym.GlobalId == nil {
		t.Fatal("expected import symbol a to have a global id")
	}
	bSym := program.Symbols.Symbols["b"]
	if bSym == nil || bSym.GlobalId == nil {
		t.Fatal("expected import symbol b to have a global id")
	}

	aName, aExports := moduleValueExports(t, bytecode.Globals[*aSym.GlobalId].Instructions, bytecode.Constants)
	if aName != "foo.a" {
		t.Fatalf("unexpected module name for a: %q", aName)
	}
	if diff := cmp.Diff([]string{"pub"}, aExports); diff != "" {
		t.Fatalf("unexpected exports for a (-want +got):\n%s", diff)
	}

	bName, bExports := moduleValueExports(t, bytecode.Globals[*bSym.GlobalId].Instructions, bytecode.Constants)
	if bName != "bar.b" {
		t.Fatalf("unexpected module name for b: %q", bName)
	}
	if diff := cmp.Diff([]string{"compute", "val"}, bExports); diff != "" {
		t.Fatalf("unexpected exports for b (-want +got):\n%s", diff)
	}
}

func TestModuleDeclLookup(t *testing.T) {
	module := prepareContextModuleParsing(t, "module.test", `
		mod foo
		const value = 1
		foo.value
	`)
	resolver := newTestModuleResolver(module, nil)

	comp := compiler.New(resolver)
	if err := comp.Compile(module); err != nil {
		t.Fatalf("compile: %s", err)
	}

	bytecode := comp.Bytecode()

	// Statements are now wrapped in __init__; find it among constants.
	var initFn *runtime.CompiledFunction
	for _, c := range bytecode.Constants {
		if fn, ok := c.(*runtime.CompiledFunction); ok && fn.Symbol != nil {
			if _, isModule := fn.Symbol.Decl.(*ast.DeclModule); isModule {
				initFn = fn
				break
			}
		}
	}
	if initFn == nil {
		t.Fatalf("expected __init__ function in constants")
		return
	}
	if !hasOpcodeSequence(initFn.Instructions, []code.Opcode{code.GetGlobal, code.GetField, code.Pop}) {
		t.Fatalf("expected module lookup to compile to GetGlobal + GetField + Pop inside __init__")
	}
}

func TestDataTypeAttributes(t *testing.T) {
	module := prepareContextModuleParsing(t, "module.test", `
		attr Example { value }
		@Example(1)
		data Foo {}
	`)
	resolver := newTestModuleResolver(module, nil)

	comp := compiler.New(resolver)
	if err := comp.Compile(module); err != nil {
		t.Fatalf("compile: %s", err)
	}

	bytecode := comp.Bytecode()

	var dataType *runtime.DataType
	var annoType *runtime.AttributeType
	for _, constant := range bytecode.Constants {
		switch constant := constant.(type) {
		case *runtime.DataType:
			if constant.Symbol.Name == "Foo" {
				dataType = constant
			}
		case *runtime.AttributeType:
			if constant.Symbol.Name == "Example" {
				annoType = constant
			}
		}
	}

	if dataType == nil {
		t.Fatal("missing data type constant for Foo")
		return
	}
	if annoType == nil {
		t.Fatal("missing attribute type constant for Example")
		return
	}

	if len(dataType.Attributes) != 1 {
		t.Fatalf("unexpected attribute count: %d", len(dataType.Attributes))
	}

	typeId := runtime.TypeId(*annoType.Symbol.ConstantId)
	globalId, ok := dataType.Attributes[typeId]
	if !ok {
		t.Fatalf("missing attribute for type id %d", typeId)
	}
	if globalId >= len(bytecode.Globals) {
		t.Fatalf("attribute global id out of range: %d", globalId)
	}

	scope := bytecode.Globals[globalId]
	found := false
	for i := 0; i < len(scope.Instructions); {
		def, err := code.LookupDefinition(scope.Instructions[i])
		if err != nil {
			t.Fatalf("unknown opcode: %s", err)
		}
		operands, read := code.ReadOperands(def, scope.Instructions[i+1:])
		if code.Opcode(scope.Instructions[i]) == code.MakeAttribute {
			found = true
			if len(operands) != 1 || operands[0] != 1 {
				t.Fatalf("unexpected attribute operand: %v", operands)
			}
		}
		i += 1 + read
	}
	if !found {
		t.Fatal("attribute global missing MakeAttribute instruction")
	}
}

func TestDeclAttr(t *testing.T) {
	tests := []compilerTestCase{
		{
			label: "empty attribute declaration",
			input: `attr Example`,
			expectedConstants: []any{
				compiledAttributeType{
					name:   "Example",
					fields: []compiledField{},
				},
			},
		},
		{
			label: "attribute declaration",
			input: `attr Example { field }`,
			expectedConstants: []any{
				compiledAttributeType{
					name:   "Example",
					fields: []compiledField{{name: "field"}},
				},
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestDeclUnion(t *testing.T) {
	t.Run("empty union", func(t *testing.T) {
		tests := []compilerTestCase{
			{
				label: "empty union",
				input: `union Example`,
				expectedConstants: []any{
					compiledUnionType{
						name:        "Example",
						memberCount: 0,
					},
				},
			},
		}
		runCompilerTests(t, tests)
	})

	t.Run("union with data members resolves members", func(t *testing.T) {
		module := prepareContextModuleParsing(t, "module.test", `
			mod test
			data A
			data B
			union AB {
				A
				B
			}
		`)
		resolver := newTestModuleResolver(module, nil)

		comp := compiler.New(resolver)
		if err := comp.Compile(module); err != nil {
			t.Fatalf("compile: %s", err)
		}

		bytecode := comp.Bytecode()

		var unionType *runtime.UnionType
		dataNames := map[string]bool{}
		for _, constant := range bytecode.Constants {
			switch c := constant.(type) {
			case *runtime.UnionType:
				if c.Symbol.Name == "AB" {
					unionType = c
				}
			case *runtime.DataType:
				dataNames[c.Symbol.Name] = true
			}
		}

		if unionType == nil {
			t.Fatal("missing union type constant for AB")
			return
		}
		if !dataNames["A"] || !dataNames["B"] {
			t.Fatalf("missing data type constants, got: %v", dataNames)
		}
		if len(unionType.MemberTypeIds) != 2 {
			t.Fatalf("expected 2 members, got %d", len(unionType.MemberTypeIds))
		}
	})

	t.Run("union with inline data resolves members", func(t *testing.T) {
		module := prepareContextModuleParsing(t, "module.test", `
			mod test
			union Example {
				data Foo
				data Bar { value }
			}
		`)
		resolver := newTestModuleResolver(module, nil)

		comp := compiler.New(resolver)
		if err := comp.Compile(module); err != nil {
			t.Fatalf("compile: %s", err)
		}

		bytecode := comp.Bytecode()

		var unionType *runtime.UnionType
		dataNames := map[string]bool{}
		for _, constant := range bytecode.Constants {
			switch c := constant.(type) {
			case *runtime.UnionType:
				if c.Symbol.Name == "Example" {
					unionType = c
				}
			case *runtime.DataType:
				dataNames[c.Symbol.Name] = true
			}
		}

		if unionType == nil {
			t.Fatal("missing union type constant for Example")
			return
		}
		if !dataNames["Foo"] || !dataNames["Bar"] {
			t.Fatalf("missing data type constants, got: %v", dataNames)
		}
		if len(unionType.MemberTypeIds) != 2 {
			t.Fatalf("expected 2 members, got %d", len(unionType.MemberTypeIds))
		}
	})
}

func TestUnionMemberTypeIds(t *testing.T) {
	module := prepareContextModuleParsing(t, "module.test", `
		mod test
		data A
		data B { value }
		union AB {
			A
			B
		}
	`)
	resolver := newTestModuleResolver(module, nil)

	comp := compiler.New(resolver)
	if err := comp.Compile(module); err != nil {
		t.Fatalf("compile: %s", err)
	}

	bytecode := comp.Bytecode()

	var unionType *runtime.UnionType
	var dataA, dataB *runtime.DataType
	for _, constant := range bytecode.Constants {
		switch c := constant.(type) {
		case *runtime.UnionType:
			if c.Symbol.Name == "AB" {
				unionType = c
			}
		case *runtime.DataType:
			if c.Symbol.Name == "A" {
				dataA = c
			}
			if c.Symbol.Name == "B" {
				dataB = c
			}
		}
	}

	if unionType == nil {
		t.Fatal("missing union type constant for AB")
		return
	}
	if dataA == nil || dataB == nil {
		t.Fatal("missing data type constants for A or B")
		return
	}

	if len(unionType.MemberTypeIds) != 2 {
		t.Fatalf("expected 2 members, got %d", len(unionType.MemberTypeIds))
	}

	aTypeId := runtime.TypeId(*dataA.Symbol.ConstantId)
	bTypeId := runtime.TypeId(*dataB.Symbol.ConstantId)

	if !unionType.IsMember(aTypeId) {
		t.Errorf("expected A (type id %d) to be a member of AB", aTypeId)
	}
	if !unionType.IsMember(bTypeId) {
		t.Errorf("expected B (type id %d) to be a member of AB", bTypeId)
	}
	if unionType.IsMember(999) {
		t.Error("expected type id 999 NOT to be a member of AB")
	}
}

func TestUnionTypeAttributes(t *testing.T) {
	module := prepareContextModuleParsing(t, "module.test", `
		mod test
		attr Tag { label }
		@Tag("important")
		union Example {
			data A
			data B
		}
	`)
	resolver := newTestModuleResolver(module, nil)

	comp := compiler.New(resolver)
	if err := comp.Compile(module); err != nil {
		t.Fatalf("compile: %s", err)
	}

	bytecode := comp.Bytecode()

	var unionType *runtime.UnionType
	var attrType *runtime.AttributeType
	for _, constant := range bytecode.Constants {
		switch c := constant.(type) {
		case *runtime.UnionType:
			if c.Symbol.Name == "Example" {
				unionType = c
			}
		case *runtime.AttributeType:
			if c.Symbol.Name == "Tag" {
				attrType = c
			}
		}
	}

	if unionType == nil {
		t.Fatal("missing union type constant for Example")
		return
	}
	if attrType == nil {
		t.Fatal("missing attribute type constant for Tag")
		return
	}

	if len(unionType.Attributes) != 1 {
		t.Fatalf("expected 1 attribute on union, got %d", len(unionType.Attributes))
	}

	typeId := runtime.TypeId(*attrType.Symbol.ConstantId)
	if _, ok := unionType.Attributes[typeId]; !ok {
		t.Fatalf("missing Tag attribute on union (type id %d)", typeId)
	}
}

func TestFunctionAttributes(t *testing.T) {
	module := prepareContextModuleParsing(t, "module.test", `
		attr Job { jobName }
		@Job("Singer")
		fn greet(@Job("Vocalist") name) {}
	`)
	resolver := newTestModuleResolver(module, nil)

	comp := compiler.New(resolver)
	if err := comp.Compile(module); err != nil {
		t.Fatalf("compile: %s", err)
	}

	bytecode := comp.Bytecode()

	var funcConst *runtime.CompiledFunction
	var annoType *runtime.AttributeType
	for _, constant := range bytecode.Constants {
		switch constant := constant.(type) {
		case *runtime.CompiledFunction:
			if constant.Symbol.Name == "greet" {
				funcConst = constant
			}
		case *runtime.AttributeType:
			if constant.Symbol.Name == "Job" {
				annoType = constant
			}
		}
	}

	if funcConst == nil {
		t.Fatal("missing compiled function constant for greet")
		return
	}
	if annoType == nil {
		t.Fatal("missing attribute type constant for Job")
		return
	}

	typeId := runtime.TypeId(*annoType.Symbol.ConstantId)
	if funcConst.Attributes == nil {
		t.Fatal("missing function attributes map")
	}
	if _, ok := funcConst.Attributes[typeId]; !ok {
		t.Fatalf("missing function attribute for type id %d", typeId)
	}

	if len(funcConst.ParamAttributes) != 1 {
		t.Fatalf("unexpected param attribute length: %d", len(funcConst.ParamAttributes))
	}
	if funcConst.ParamAttributes[0] == nil {
		t.Fatal("missing param attributes map")
	}
	if _, ok := funcConst.ParamAttributes[0][typeId]; !ok {
		t.Fatalf("missing param attribute for type id %d", typeId)
	}
}

// testExternPlugin provides bindings for extern declarations used in tests.
type testExternPlugin struct{}

func (p *testExternPlugin) Module() string { return "" }

func (p *testExternPlugin) Bind(ctx runtime.BindContext, module *ast.SymbolTable, decl *ast.Symbol) runtime.RuntimeValue {
	switch decl.Name {
	case "greet":
		return runtime.MakeExternFunc(decl, func(args []runtime.RuntimeValue) (runtime.RuntimeValue, error) {
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
	module := prepareContextModuleParsing(t, "module.test", `
		attr Job { jobName }
		@Job("Singer")
		extern fn greet(@Job("Vocalist") name)
		@Job("Actor")
		extern type Person {}
	`)
	resolver := newTestModuleResolver(module, nil)

	comp := compiler.New(resolver)
	comp.RegisterPlugin(&testExternPlugin{})
	if err := comp.Compile(module); err != nil {
		t.Fatalf("compile: %s", err)
	}

	bytecode := comp.Bytecode()

	var externFunc *runtime.ExternFunc
	var externType runtime.SimpleType
	var annoType *runtime.AttributeType
	for _, constant := range bytecode.Constants {
		switch constant := constant.(type) {
		case *runtime.ExternFunc:
			if constant.Inspect() == "extern greet(#1)" {
				externFunc = constant
			}
		case runtime.SimpleType:
			if constant.Decl.Name == "Person" {
				externType = constant
			}
		case *runtime.AttributeType:
			if constant.Symbol.Name == "Job" {
				annoType = constant
			}
		}
	}

	if annoType == nil {
		t.Fatal("missing attribute type constant for Job")
		return
	}
	typeId := runtime.TypeId(*annoType.Symbol.ConstantId)
	if externFunc == nil {
		t.Fatal("missing extern fn constant for greet")
		return
	}
	if externFunc.Attributes == nil {
		t.Fatal("missing extern fn attributes map")
	}
	if _, ok := externFunc.Attributes[typeId]; !ok {
		t.Fatalf("missing extern fn attribute for type id %d", typeId)
	}
	if len(externFunc.ParamAttributes) != 1 {
		t.Fatalf("unexpected extern fn param attribute length: %d", len(externFunc.ParamAttributes))
	}
	if externFunc.ParamAttributes[0] == nil {
		t.Fatal("missing extern fn param attributes map")
	}
	if _, ok := externFunc.ParamAttributes[0][typeId]; !ok {
		t.Fatalf("missing extern fn param attribute for type id %d", typeId)
	}

	if externType.Attributes == nil {
		t.Fatal("missing extern type attributes map")
	}
	if _, ok := externType.Attributes[typeId]; !ok {
		t.Fatalf("missing extern type attribute for type id %d", typeId)
	}
}

func TestExternValueCompilation(t *testing.T) {
	// A module containing extern const must compile without
	// "unknown declaration *ast.DeclExternValue" errors.
	module := prepareContextModuleParsing(t, "module.test", `
		extern type Void {}
		extern const void
		fn answer() { return 42 }
	`)

	mainModule := prepareContextModuleParsing(t, "module.main", `mod main`)
	resolver := newTestModuleResolver(mainModule, map[registry.LogicalURI]*ast.ContextModule{
		module.Name: module,
	})

	analysis := analyzer.New(resolver)
	if errs, _ := analysis.Analyze(module, true); len(errs) > 0 {
		t.Fatalf("analysis errors: %v", errs)
	}

	// Verify that DeclExternValue received a ConstantId.
	voidSym := module.Symbols.Symbols["void"]
	if voidSym == nil || voidSym.ConstantId == nil {
		t.Fatal("expected symbol 'void' with a ConstantId in module symbols")
	}

	// Verify the module can be fully compiled (including extern const).
	comp := compiler.NewWithAnalyzer(resolver, analysis)
	comp.RegisterPlugin(&testExternPlugin{})
	if err := comp.Compile(module); err != nil {
		t.Fatalf("compile: %s", err)
	}

	// Verify the constant is a Void value.
	bytecode := comp.Bytecode()
	found := false
	for _, constant := range bytecode.Constants {
		if _, ok := constant.(runtime.Void); ok {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected a Void constant in bytecode for extern const void")
	}
}

func TestAttributeTypeHints(t *testing.T) {
	module := prepareContextModuleParsing(t, "module.test", `
		attr Meta { label }
		@Meta("Primary")
		attr Job { jobName }
	`)
	resolver := newTestModuleResolver(module, nil)

	comp := compiler.New(resolver)
	if err := comp.Compile(module); err != nil {
		t.Fatalf("compile: %s", err)
	}

	bytecode := comp.Bytecode()

	var annoType *runtime.AttributeType
	var metaType *runtime.AttributeType
	for _, constant := range bytecode.Constants {
		if at, ok := constant.(*runtime.AttributeType); ok {
			if at.Symbol.Name == "Job" {
				annoType = at
			}
			if at.Symbol.Name == "Meta" {
				metaType = at
			}
		}
	}

	if annoType == nil {
		t.Fatal("missing attribute type constant for Job")
		return
	}
	if metaType == nil {
		t.Fatal("missing attribute type constant for Meta")
		return
	}

	typeId := runtime.TypeId(*metaType.Symbol.ConstantId)
	if annoType.Attributes == nil {
		t.Fatal("missing attribute type attributes map")
	}
	if _, ok := annoType.Attributes[typeId]; !ok {
		t.Fatalf("missing attribute type attribute for type id %d", typeId)
	}
}

func runCompilerTests(t *testing.T, tests []compilerTestCase) {
	t.Helper()

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d. %s", i, tt.label), func(t *testing.T) {
			module, program := prepareSourceFileParsing(t, tt.input)

			resolver := newTestModuleResolver(module, nil)
			compiler := compiler.New(resolver)
			err := compiler.Compile(program)
			if err != nil {
				t.Fatalf("compiler error: %s", err)
			}

			bytecode := compiler.Bytecode()

			err = testInstructions(t, tt.expectedInstructions, bytecode.Instructions)
			if err != nil {
				t.Fatalf("testInstructions failed: %s", err)
			}

			err = testConstants(t, tt.expectedConstants, bytecode.Constants)
			if err != nil {
				t.Fatalf("testConstants failed: %s", err)
			}

			err = testGlobals(t, tt.expectedGlobals, bytecode.Globals)
			if err != nil {
				t.Fatalf("testGlobals failed: %s", err)
			}
		})
	}
}

func prepareSourceFileParsing(t *testing.T, input string) (*ast.ContextModule, *ast.SourceFile) {
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

func TestInitFunctionWrapping(t *testing.T) {
	t.Run("module with no statements produces no __init__", func(t *testing.T) {
		module := prepareContextModuleParsing(t, "module.test", "mod foo\nfn bar() {}")
		resolver := newTestModuleResolver(module, nil)
		comp := compiler.New(resolver)
		if err := comp.Compile(module); err != nil {
			t.Fatalf("compile: %s", err)
		}
		for _, c := range comp.Bytecode().Constants {
			if fn, ok := c.(*runtime.CompiledFunction); ok && fn.Symbol == nil {
				t.Fatalf("expected no synthetic __init__ function, but found one")
			}
		}
	})

	t.Run("module with statements wraps them in __init__", func(t *testing.T) {
		module := prepareContextModuleParsing(t, "module.test", "mod foo\n1 + 2")
		resolver := newTestModuleResolver(module, nil)
		comp := compiler.New(resolver)
		if err := comp.Compile(module); err != nil {
			t.Fatalf("compile: %s", err)
		}
		bytecode := comp.Bytecode()

		var initFn *runtime.CompiledFunction
		for _, c := range bytecode.Constants {
			if fn, ok := c.(*runtime.CompiledFunction); ok && fn.Symbol != nil {
				if _, isModule := fn.Symbol.Decl.(*ast.DeclModule); isModule {
					initFn = fn
					break
				}
			}
		}
		if initFn == nil {
			t.Fatalf("expected __init__ function in constants, got none")
			return
		}
		if !hasOpcodeSequence(initFn.Instructions, []code.Opcode{code.Const, code.Const, code.Add}) {
			t.Fatalf("expected __init__ to contain addition instructions")
		}
		if !hasOpcodeSequence(bytecode.Instructions, []code.Opcode{code.Const, code.Call, code.Pop}) {
			t.Fatalf("expected frame 0 to call __init__")
		}
	})

	t.Run("__init__ accesses module-level globals", func(t *testing.T) {
		module := prepareContextModuleParsing(t, "module.test", "mod foo\nconst x = 5\nx")
		resolver := newTestModuleResolver(module, nil)
		comp := compiler.New(resolver)
		if err := comp.Compile(module); err != nil {
			t.Fatalf("compile: %s", err)
		}
		bytecode := comp.Bytecode()

		var initFn *runtime.CompiledFunction
		for _, c := range bytecode.Constants {
			if fn, ok := c.(*runtime.CompiledFunction); ok && fn.Symbol != nil {
				if _, isModule := fn.Symbol.Decl.(*ast.DeclModule); isModule {
					initFn = fn
					break
				}
			}
		}
		if initFn == nil {
			t.Fatalf("expected __init__ function in constants, got none")
			return
		}
		if !hasOpcodeSequence(initFn.Instructions, []code.Opcode{code.GetGlobal, code.Pop}) {
			t.Fatalf("expected __init__ to reference global variable x")
		}
	})
}

func TestCompileSourceFileIncremental(t *testing.T) {
	t.Run("declaration only adds constant without init", func(t *testing.T) {
		module, src1 := prepareSourceFileParsing(t, "fn foo() {}")
		resolver := newTestModuleResolver(module, nil)
		analysis := analyzer.New(resolver)
		if errs, _ := analysis.Analyze(module, false); len(errs) > 0 {
			t.Fatalf("analyze: %s", errs[0].Error())
		}
		comp := compiler.NewWithAnalyzer(resolver, analysis)
		if err := comp.Compile(src1); err != nil {
			t.Fatalf("initial compile: %s", err)
		}

		l, err := lexer.New(staticmodule.NewSourceString("testing:///test/line2.zirr", "fn bar() {}"))
		if err != nil {
			t.Fatal(err)
		}
		src2 := parser.NewSourceParser(l, module.Decls, "line2.zirr").ParseSourceFile()
		module.AddSourceFile(src2)
		if errs := analysis.AnalyzeSourceFile(module, src2); len(errs) > 0 {
			t.Fatalf("analyze src2: %s", errs[0].Error())
		}

		prevConstantsLen := len(comp.Bytecode().Constants)
		initId, err := comp.CompileSourceFileIncremental(src2)
		if err != nil {
			t.Fatalf("incremental compile: %s", err)
		}
		if initId != -1 {
			t.Fatalf("expected no __init__ for declaration-only file, got id %d", initId)
		}
		if len(comp.Bytecode().Constants) <= prevConstantsLen {
			t.Fatalf("expected new constant for bar function")
		}
	})

	t.Run("statement produces __init__ constant", func(t *testing.T) {
		module, src1 := prepareSourceFileParsing(t, "fn foo() {}")
		resolver := newTestModuleResolver(module, nil)
		analysis := analyzer.New(resolver)
		if errs, _ := analysis.Analyze(module, false); len(errs) > 0 {
			t.Fatalf("analyze: %s", errs[0].Error())
		}
		comp := compiler.NewWithAnalyzer(resolver, analysis)
		if err := comp.Compile(src1); err != nil {
			t.Fatalf("initial compile: %s", err)
		}

		l, err := lexer.New(staticmodule.NewSourceString("testing:///test/line2.zirr", "1 + 2"))
		if err != nil {
			t.Fatal(err)
		}
		src2 := parser.NewSourceParser(l, module.Decls, "line2.zirr").ParseSourceFile()
		module.AddSourceFile(src2)
		if errs := analysis.AnalyzeSourceFile(module, src2); len(errs) > 0 {
			t.Fatalf("analyze src2: %s", errs[0].Error())
		}

		prevConstantsLen := len(comp.Bytecode().Constants)
		initId, err := comp.CompileSourceFileIncremental(src2)
		if err != nil {
			t.Fatalf("incremental compile: %s", err)
		}
		if initId < 0 {
			t.Fatalf("expected __init__ constant id >= 0, got %d", initId)
		}
		if initId < prevConstantsLen {
			t.Fatalf("expected __init__ at a new constant slot >= %d, got %d", prevConstantsLen, initId)
		}
		initFn, ok := comp.Bytecode().Constants[initId].(*runtime.CompiledFunction)
		if !ok {
			t.Fatalf("constant %d is not a CompiledFunction", initId)
		}
		// Modules with a `module X` declaration get a DeclModule symbol; without one (as in this test), Symbol is nil.
		if initFn.Symbol != nil {
			if _, isModule := initFn.Symbol.Decl.(*ast.DeclModule); !isModule {
				t.Fatalf("expected __init__ Symbol to be a DeclModule or nil, got %T", initFn.Symbol.Decl)
			}
		}
		if !hasOpcodeSequence(initFn.Instructions, []code.Opcode{code.Const, code.Const, code.Add}) {
			t.Fatalf("expected __init__ instructions to contain the addition")
		}
	})
}

func prepareContextModuleParsing(t *testing.T, uri string, input string) *ast.ContextModule {
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

func checkParserErrors(t *testing.T, p *parser.Parser, contents string) {
	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
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

func checkModuleParserErrors(t *testing.T, mp *parser.ModuleParser, contents string) {
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

func testInstructions(
	t *testing.T,
	expected []code.Instructions,
	actual code.Instructions,
) error {
	t.Helper()
	concatted := concatInstructions(expected)

	if len(actual) != len(concatted) {
		return fmt.Errorf("wrong instructions length.\nwant=%q\ngot =%q",
			concatted, actual)
	}

	for i, ins := range concatted {
		if actual[i] != ins {
			return fmt.Errorf("wrong instruction at %d.\nwant=%q\ngot =%q",
				i, concatted, actual)
		}
	}

	return nil
}

func moduleValueExports(t *testing.T, ins code.Instructions, constants []runtime.RuntimeValue) (string, []string) {
	t.Helper()

	stack := make([]runtime.RuntimeValue, 0, 8)

	for i := 0; i < len(ins); i++ {
		def, err := code.LookupDefinition(ins[i])
		if err != nil {
			t.Fatalf("lookup opcode: %s", err)
		}
		operands, read := code.ReadOperands(def, ins[i+1:])

		switch code.Opcode(ins[i]) {
		case code.Const:
			idx := operands[0]
			stack = append(stack, constants[idx])
		case code.GetGlobal:
			stack = append(stack, runtime.Void{})
		case code.Module:
			if len(stack) == 0 {
				t.Fatal("module build missing member count")
			}
			countVal := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			count, ok := countVal.(runtime.Int)
			if !ok {
				t.Fatalf("module member count must be Int, got %T", countVal)
			}
			exportNames := make([]string, 0, int(count))
			for i := 0; i < int(count); i++ {
				if len(stack) < 2 {
					t.Fatal("module exports missing name/value pair")
				}
				stack = stack[:len(stack)-1] // value
				nameVal := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				name, ok := nameVal.(runtime.String)
				if !ok {
					t.Fatalf("module export name must be String, got %T", nameVal)
				}
				exportNames = append(exportNames, string(name))
			}
			sort.Strings(exportNames)
			moduleNameVal := constants[operands[0]]
			moduleName, ok := moduleNameVal.(runtime.String)
			if !ok {
				t.Fatalf("module name must be String, got %T", moduleNameVal)
			}
			return string(moduleName), exportNames
		default:
			t.Fatalf("unexpected opcode in module build: %s", def.Name)
		}

		i += read
	}

	t.Fatal("module build instruction not found")
	return "", nil
}

func hasOpcodeSequence(ins code.Instructions, seq []code.Opcode) bool {
	if len(seq) == 0 {
		return true
	}
	match := 0
	for i := 0; i < len(ins); i++ {
		def, err := code.LookupDefinition(ins[i])
		if err != nil {
			return false
		}
		_, read := code.ReadOperands(def, ins[i+1:])
		opcode := code.Opcode(ins[i])
		if opcode == seq[match] {
			match++
			if match == len(seq) {
				return true
			}
		}
		i += read
	}
	return false
}

func concatInstructions(s []code.Instructions) code.Instructions {
	out := code.Instructions{}

	for _, ins := range s {
		out = append(out, ins...)
	}
	return out
}

func testConstants(
	t *testing.T,
	expected []any,
	actual []runtime.RuntimeValue,
) error {
	t.Helper()

	if len(actual) != len(expected) {
		return fmt.Errorf("wrong amount of constants.\nwant=%q\ngot=%q", expected, actual)
	}

	for i, cons := range expected {
		switch want := cons.(type) {
		case bool:
			got, ok := actual[i].(runtime.Bool)
			if !ok || want != bool(got) {
				return fmt.Errorf("wrong constant at %d.\nwant=%t\ngot=%q", i, want, got.Inspect())
			}
		case int:
			got, ok := actual[i].(runtime.Int)
			if !ok || want != int(got) {
				return fmt.Errorf("wrong constant at %d.\nwant=%d\ngot=%q", i, want, got)
			}
		case float64:
			got, ok := actual[i].(runtime.Float)
			if !ok || want != float64(got) {
				return fmt.Errorf("wrong constant at %d.\nwant=%f\ngot=%f", i, want, float64(got))
			}
		case rune:
			got, ok := actual[i].(runtime.Char)
			if !ok || want != rune(got) {
				return fmt.Errorf("wrong constant at %d.\nwant=%q\ngot=%q", i, string(want), got)
			}
		case string:
			got, ok := actual[i].(runtime.String)
			if !ok || want != string(got) {
				return fmt.Errorf("wrong constant at %d.\nwant=%q\ngot=%q", i, want, got)
			}
		case compiledFunction:
			got, ok := actual[i].(*runtime.CompiledFunction)
			if !ok {
				return fmt.Errorf("constant %d is not a function: %T", i, actual[i])
			}

			if got.Symbol.Name != want.name {
				return fmt.Errorf("wrong function name at %d.\nwant=%q\ngot=%q", i, want.name, got.Symbol.Name)
			}

			if got.Arity() != want.params {
				return fmt.Errorf("wrong function params at %d.\nwant=%d\ngot=%d", i, want.params, got.Arity())
			}

			err := testInstructions(t, want.ins, got.Instructions)
			if err != nil {
				return fmt.Errorf("wrong function instructions at %d: %s", i, err)
			}

		case compiledDataType:
			got, ok := actual[i].(*runtime.DataType)
			if !ok {
				return fmt.Errorf("constant %d is not a data type: %T", i, actual[i])
			}

			if got.Symbol.Name != want.name {
				return fmt.Errorf("wrong data type name at %d.\nwant=%q\ngot=%q", i, want.name, got.Symbol.Name)
			}

			if len(got.FieldSymbols) != len(want.fields) {
				return fmt.Errorf("wrong amount of fields at %d.\nwant=%d\ngot=%d", i, len(want.fields), len(got.FieldSymbols))
			}

			for j, field := range want.fields {
				if got.FieldSymbols[j].Name != field.name {
					return fmt.Errorf("wrong field name at %d.%d.\nwant=%q\ngot=%q", i, j, field.name, got.FieldSymbols[j].Name)
				}
			}
		case compiledUnionType:
			got, ok := actual[i].(*runtime.UnionType)
			if !ok {
				return fmt.Errorf("constant %d is not a union type: %T", i, actual[i])
			}

			if got.Symbol.Name != want.name {
				return fmt.Errorf("wrong union type name at %d.\nwant=%q\ngot=%q", i, want.name, got.Symbol.Name)
			}

			if len(got.MemberTypeIds) != want.memberCount {
				return fmt.Errorf("wrong member count at %d.\nwant=%d\ngot=%d", i, want.memberCount, len(got.MemberTypeIds))
			}
		case compiledAttributeType:
			got, ok := actual[i].(*runtime.AttributeType)
			if !ok {
				return fmt.Errorf("constant %d is not an attribute type: %T", i, actual[i])
			}

			if got.Symbol.Name != want.name {
				return fmt.Errorf("wrong attribute type name at %d.\nwant=%q\ngot=%q", i, want.name, got.Symbol.Name)
			}

			if len(got.FieldSymbols) != len(want.fields) {
				return fmt.Errorf("wrong amount of fields at %d.\nwant=%d\ngot=%d", i, len(want.fields), len(got.FieldSymbols))
			}

			for j, field := range want.fields {
				if got.FieldSymbols[j].Name != field.name {
					return fmt.Errorf("wrong field name at %d.%d.\nwant=%q\ngot=%q", i, j, field.name, got.FieldSymbols[j].Name)
				}
			}

			if got.Symbol.ConstantId == nil {
				return fmt.Errorf("attribute type %q has no constant id", got.Symbol.Name)
			}
			if got.TypeConstantId() != runtime.TypeId(*got.Symbol.ConstantId) {
				return fmt.Errorf("attribute type %q has mismatched type id", got.Symbol.Name)
			}

		case synthCompiledFunction:
			got, ok := actual[i].(*runtime.CompiledFunction)
			if !ok {
				return fmt.Errorf("constant %d is not a compiled function: %T", i, actual[i])
			}
			// __init__ functions carry the module's DeclModule symbol (or nil for anonymous modules).
			if got.Symbol != nil {
				if _, isModule := got.Symbol.Decl.(*ast.DeclModule); !isModule {
					return fmt.Errorf("expected synthetic __init__ function (DeclModule symbol or nil) at constant %d, got Symbol=%q (%T)", i, got.Symbol.Name, got.Symbol.Decl)
				}
			}
			if err := testInstructions(t, want.ins, got.Instructions); err != nil {
				return fmt.Errorf("wrong __init__ instructions at constant %d: %s", i, err)
			}

		default:
			got := actual[i]
			return fmt.Errorf("unhandled wanted type %T of value at %d.\nwant=%q\ngot=%q", i, want, want, got)
		}
	}
	return nil
}

type synthCompiledFunction struct {
	ins []code.Instructions
}

func testGlobals(t *testing.T,
	expected [][]code.Instructions,
	actual []*compiler.CompilationScope,
) error {
	t.Helper()

	if len(actual) != len(expected) {
		return fmt.Errorf("wrong amount of globals.\nwant=%d\ngot=%d", len(expected), len(actual))
	}

	for i, ins := range expected {
		err := testInstructions(t, ins, actual[i].Instructions)
		if err != nil {
			return fmt.Errorf("wrong global instructions at %d: %s", i, err)
		}
	}

	return nil
}

type compiledFunction struct {
	name   string
	params int
	ins    []code.Instructions
}
type compiledDataType struct {
	name   string
	fields []compiledField
}
type compiledUnionType struct {
	name        string
	memberCount int
}
type compiledAttributeType struct {
	name   string
	fields []compiledField
}

type compiledField struct {
	name string
}

type testModuleResolver struct {
	main    *ast.ContextModule
	modules map[registry.LogicalURI]*ast.ContextModule
}

func newTestModuleResolver(main *ast.ContextModule, modules map[registry.LogicalURI]*ast.ContextModule) testModuleResolver {
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

func TestResolveModuleSymbol(t *testing.T) {
	// Set up a "types" module with a data type "Wrapper".
	typesModule := prepareContextModuleParsing(t, "test.types", `
		mod types
		data Wrapper { value }
	`)
	// Set up a "mylib" module that imports Wrapper from types and has an extern fn.
	mylibModule := prepareContextModuleParsing(t, "test.mylib", `
		mod mylib
		import types = test.types { Wrapper }
		extern fn wrap(x) -> Wrapper
	`)
	// Main module imports mylib.
	mainModule, program := prepareSourceFileParsing(t, `
		import mylib = test.mylib
		mylib.wrap(42)
	`)

	modules := map[registry.LogicalURI]*ast.ContextModule{
		typesModule.Name: typesModule,
		mylibModule.Name: mylibModule,
	}
	resolver := newTestModuleResolver(mainModule, modules)

	// Create a plugin that uses ResolveModuleSymbol to look up Wrapper.
	testPlugin := &resolverTestPlugin{}

	comp := compiler.New(resolver)
	comp.RegisterPlugin(testPlugin)
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}

	// The plugin should have resolved the Wrapper symbol during Bind.
	if testPlugin.resolvedWrapper == nil {
		t.Fatal("ResolveModuleSymbol did not find Wrapper")
	}
	if testPlugin.resolvedWrapper.ConstantId == nil {
		t.Fatal("resolved Wrapper symbol has nil ConstantId")
	}
	if testPlugin.resolvedWrapper.Name != "Wrapper" {
		t.Errorf("resolved symbol name: got %q, want %q", testPlugin.resolvedWrapper.Name, "Wrapper")
	}
}

// resolverTestPlugin captures the result of ResolveModuleSymbol for assertions.
type resolverTestPlugin struct {
	resolvedWrapper *ast.Symbol
}

func (p *resolverTestPlugin) Module() string { return "mylib" }

func (p *resolverTestPlugin) Bind(ctx runtime.BindContext, module *ast.SymbolTable, decl *ast.Symbol) runtime.RuntimeValue {
	switch decl.Name {
	case "wrap":
		p.resolvedWrapper = ctx.ResolveModuleSymbol("types", "Wrapper")
		return runtime.MakeExternFunc(decl, func(args []runtime.RuntimeValue) (runtime.RuntimeValue, error) {
			return runtime.Void{}, nil
		})
	}
	return nil
}
