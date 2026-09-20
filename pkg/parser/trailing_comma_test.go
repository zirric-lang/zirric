package parser_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

// TestTrailingCommaInArrayLiteral is a regression test: parsePrattExprArrayElements
// unconditionally parsed another element after each comma, so a trailing comma before
// `]` hit the closing bracket's unrecognized-prefix error and discarded the whole
// array expression (returned nil), not just a soft parse error.
func TestTrailingCommaInArrayLiteral(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"[1, 2, 3,]", 3},
		{"[1,]", 1},
		{"[1, 2,]", 2},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			stmt, ok := srcFile.Statements[0].(*ast.StmtExpr)
			if !ok {
				t.Fatalf("expected *ast.StmtExpr, got %T", srcFile.Statements[0])
			}
			arr, ok := stmt.Expr.(*ast.ExprArray)
			if !ok {
				t.Fatalf("expected *ast.ExprArray, got %T", stmt.Expr)
			}
			if len(arr.Elements) != tt.want {
				t.Errorf("expected %d elements, got %d", tt.want, len(arr.Elements))
			}
		})
	}
}

// TestTrailingCommaInDictLiteral mirrors TestTrailingCommaInArrayLiteral for dict
// literals (parsePrattExprDictEntries had the same bug).
func TestTrailingCommaInDictLiteral(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{`["a": 1, "b": 2,]`, 2},
		{`["a": 1,]`, 1},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			stmt, ok := srcFile.Statements[0].(*ast.StmtExpr)
			if !ok {
				t.Fatalf("expected *ast.StmtExpr, got %T", srcFile.Statements[0])
			}
			dict, ok := stmt.Expr.(*ast.ExprDict)
			if !ok {
				t.Fatalf("expected *ast.ExprDict, got %T", stmt.Expr)
			}
			if len(dict.Entries) != tt.want {
				t.Errorf("expected %d entries, got %d", tt.want, len(dict.Entries))
			}
		})
	}
}

// TestTrailingCommaInCallArguments is a regression test: parsePrattExprCall's
// argument loop unconditionally parsed another argument after each comma, so a
// trailing comma before `)` recorded a spurious "unexpected )" error and appended a
// nil argument to the call.
func TestTrailingCommaInCallArguments(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"f(1, 2, 3,)", 3},
		{"f(1,)", 1},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			stmt, ok := srcFile.Statements[0].(*ast.StmtExpr)
			if !ok {
				t.Fatalf("expected *ast.StmtExpr, got %T", srcFile.Statements[0])
			}
			inv, ok := stmt.Expr.(*ast.ExprInvocation)
			if !ok {
				t.Fatalf("expected *ast.ExprInvocation, got %T", stmt.Expr)
			}
			if len(inv.Arguments) != tt.want {
				t.Errorf("expected %d arguments, got %d", tt.want, len(inv.Arguments))
			}
			for i, arg := range inv.Arguments {
				if arg == nil {
					t.Errorf("argument %d is nil", i)
				}
			}
		})
	}
}

// TestTrailingCommaInFunctionParameters confirms trailing commas in function
// declaration and closure parameter lists — already handled correctly by
// parseDeclParameterListWithInsert before this change — continue to work.
func TestTrailingCommaInFunctionParameters(t *testing.T) {
	t.Run("fn f(a, b,) { a }", func(t *testing.T) {
		srcFile := prepareSourceFileParsing(t, "fn f(a, b,) { a }")
		symbol, ok := srcFile.Decls.Parent.Symbols["f"]
		if !ok || symbol.Decl == nil {
			t.Fatal("expected function declaration")
		}
		decl, ok := symbol.Decl.(*ast.DeclFunc)
		if !ok {
			t.Fatalf("declaration is %T, want *ast.DeclFunc", symbol.Decl)
		}
		if len(decl.Impl.Parameters) != 2 {
			t.Errorf("expected 2 params, got %d", len(decl.Impl.Parameters))
		}
	})

	t.Run("fn(a, b,) { a }", func(t *testing.T) {
		srcFile := prepareSourceFileParsing(t, "fn(a, b,) { a }")
		stmt, ok := srcFile.Statements[0].(*ast.StmtExpr)
		if !ok {
			t.Fatalf("expected *ast.StmtExpr, got %T", srcFile.Statements[0])
		}
		fnExpr, ok := stmt.Expr.(*ast.ExprFunc)
		if !ok {
			t.Fatalf("expected *ast.ExprFunc, got %T", stmt.Expr)
		}
		if len(fnExpr.Parameters) != 2 {
			t.Errorf("expected 2 params, got %d", len(fnExpr.Parameters))
		}
	})
}
