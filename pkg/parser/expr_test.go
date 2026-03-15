package parser_test

import (
	"fmt"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
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

func TestParseExprFor(t *testing.T) {
	tests := []struct {
		input           string
		condition       bool
		collectionIdent string
		collectionExpr  string
		decls           int
		stmts           int
		lastStmtType    string
	}{
		{"const result = for false { 1 }", true, "", "", 0, 1, "*ast.StmtExpr"},
		{"const result = for false { const value = 1 value }", true, "", "", 1, 1, "*ast.StmtExpr"},
		{"const result = for item <- items { item }", false, "item", "items", 0, 1, "*ast.StmtExpr"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			symbol, ok := srcFile.Decls.Parent.Symbols["result"]
			if !ok || symbol.Decl == nil {
				t.Fatal("expected result declaration")
			}
			decl, ok := symbol.Decl.(*ast.DeclConstant)
			if !ok {
				t.Fatalf("declaration is %T, want *ast.DeclConstant", symbol.Decl)
			}
			expr, ok := decl.Value.(ast.ExprFor)
			if !ok {
				t.Fatalf("expression is %T, want ast.ExprFor", decl.Value)
			}
			if tt.condition && expr.Condition == nil {
				t.Fatal("expected condition expression")
			}
			if !tt.condition && expr.Condition != nil {
				t.Fatalf("expected no condition, got %T", expr.Condition)
			}
			if tt.collectionIdent != "" {
				if expr.CollectionIdent == nil {
					t.Fatal("expected collection identifier")
				}
				if expr.CollectionIdent.Value != tt.collectionIdent {
					t.Fatalf("expected collection identifier %q, got %q", tt.collectionIdent, expr.CollectionIdent.Value)
				}
				idExpr, ok := expr.CollectionExpr.(*ast.ExprIdentifier)
				if !ok {
					t.Fatalf("collection expression is %T, want *ast.ExprIdentifier", expr.CollectionExpr)
				}
				if idExpr.Name.Value != tt.collectionExpr {
					t.Fatalf("expected collection expr %q, got %q", tt.collectionExpr, idExpr.Name.Value)
				}
			} else if expr.CollectionIdent != nil || expr.CollectionExpr != nil {
				t.Fatalf("expected no collection, got %v", expr.CollectionIdent)
			}
			if len(expr.Body.Decls) != tt.decls {
				t.Fatalf("expected decls len %d, got %d", tt.decls, len(expr.Body.Decls))
			}
			if len(expr.Body.Stmts) != tt.stmts {
				t.Fatalf("expected stmts len %d, got %d", tt.stmts, len(expr.Body.Stmts))
			}
			if len(expr.Body.Stmts) > 0 {
				last := expr.Body.Stmts[len(expr.Body.Stmts)-1]
				if got := fmt.Sprintf("%T", last); got != tt.lastStmtType {
					t.Fatalf("expected last stmt %s, got %s", tt.lastStmtType, got)
				}
			}
		})
	}
}
