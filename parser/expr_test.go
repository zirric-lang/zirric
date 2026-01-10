package parser_test

import (
	"fmt"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/ast"
)

func TestExprIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"example", "example"},
		{" other ", "other"},
		{"42", "42"},
		// {"0b11", "3"}, // TODO
		{"13.37", "13.370000"},
		{"!true", "(!true)"},
		{"-3", "(-3)"},
		{"(-3)", "(-3)"},
		{"(if x { y } else { z })", "(if x { y } else { z })"},
		{"(if x { y } else if e { e1 } else { z })", "(if x { y } else if e { e1 } else { z })"},
		{"(if x { y } else if e { e1 } else if f { f1 } else { z })", "(if x { y } else if e { e1 } else if f { f1 } else { z })"},
		{"json.Null", "json.Null"},
		{"[42 + 1337]", "[(42+1337)]"},
		{"[42 + 1337: 12 - 34]", "[(42+1337): (12-34)]"},
		{"[42 + 1337: 12 - 34, 2: 3]", "[(42+1337): (12-34), 2: 3]"},
		{"true", "true"},
		{"false", "false"},
		{"'a'", "'a'"},
		{"'\\n'", "'\\n'"},
		{"'\\''", "'\\''"},
		{"'\\\\'", "'\\\\'"},
		{"[1, 2]", "[1, 2]"},
		{"some()", "some(some)"},
		{"call(1, 2)", "call(1, 2call)"},
		{"{}", "{->/* 0 stmts */}"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d.\t test: %q", i+1, tt.input), func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			if len(srcFile.Statements) != 1 {
				t.Fatalf("srcFile.Statements does not contain %d statements, got %d\n", 1, len(srcFile.Statements))
			}

			stmt, ok := srcFile.Statements[0].(*ast.StmtExpr)
			if !ok {
				t.Fatalf("srcFile.Statements[0] is not *ast.StmtExpr, got %T", srcFile.Statements[0])
			}

			got := stmt.Expr.Expression()
			if tt.want != got {
				t.Errorf("wrong expression parsed\nwant:\t%q\ngot:\t%q", tt.want, got)
			}
		})
	}
}
