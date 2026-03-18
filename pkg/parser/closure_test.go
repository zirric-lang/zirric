package parser_test

import (
	"fmt"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

func TestParseFnClosureExpression(t *testing.T) {
	tests := []struct {
		input      string
		paramCount int
		want       string
	}{
		{"fn() { 42 }", 0, "fn() {/* 1 stmts */}"},
		{"fn(x) { x }", 1, "fn(x) {/* 1 stmts */}"},
		{"fn(a, b) { a + b }", 2, "fn(a, b) {/* 1 stmts */}"},
		{"fn(x) { return x }", 1, "fn(x) {/* 1 stmts */}"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d. %s", i, tt.input), func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			if len(srcFile.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(srcFile.Statements))
			}
			stmt, ok := srcFile.Statements[0].(*ast.StmtExpr)
			if !ok {
				t.Fatalf("expected *ast.StmtExpr, got %T", srcFile.Statements[0])
			}
			fnExpr, ok := stmt.Expr.(*ast.ExprFunc)
			if !ok {
				t.Fatalf("expected *ast.ExprFunc, got %T", stmt.Expr)
			}
			if len(fnExpr.Parameters) != tt.paramCount {
				t.Errorf("expected %d params, got %d", tt.paramCount, len(fnExpr.Parameters))
			}
			got := fnExpr.Expression()
			if got != tt.want {
				t.Errorf("Expression()\nwant: %q\ngot:  %q", tt.want, got)
			}
		})
	}
}

func TestParseFnClosureWithTypedParams(t *testing.T) {
	tests := []struct {
		input     string
		paramName string
		typeName  string
	}{
		{"fn(x: Int) { x }", "x", "Int"},
		{"fn(name: String) { name }", "name", "String"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d. %s", i, tt.input), func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			stmt := srcFile.Statements[0].(*ast.StmtExpr)
			fnExpr := stmt.Expr.(*ast.ExprFunc)
			if len(fnExpr.Parameters) != 1 {
				t.Fatalf("expected 1 param, got %d", len(fnExpr.Parameters))
			}
			param := fnExpr.Parameters[0]
			if param.Name.Value != tt.paramName {
				t.Errorf("param name: want %q, got %q", tt.paramName, param.Name.Value)
			}
			if param.TypeHint == nil {
				t.Fatal("expected type hint on param")
			}
			ref, ok := param.TypeHint.(ast.TypeExprRef)
			if !ok {
				t.Fatalf("expected TypeExprRef, got %T", param.TypeHint)
			}
			if ref.Reference[0].Value != tt.typeName {
				t.Errorf("type: want %q, got %q", tt.typeName, ref.Reference[0].Value)
			}
		})
	}
}

func TestParseFnClosureWithReturnType(t *testing.T) {
	input := `const f = fn(x: Int) -> String { "hello" }`
	srcFile := prepareSourceFileParsing(t, input)

	sym := lookupDeclSymbol(t, srcFile, "f")
	decl := sym.Decl.(*ast.DeclConstant)
	fnExpr, ok := decl.Value.(*ast.ExprFunc)
	if !ok {
		t.Fatalf("expected *ast.ExprFunc, got %T", decl.Value)
	}
	if len(fnExpr.Parameters) != 1 {
		t.Fatalf("expected 1 param, got %d", len(fnExpr.Parameters))
	}
	if fnExpr.Parameters[0].TypeHint == nil {
		t.Fatal("expected type hint on param")
	}
	if fnExpr.ReturnType == nil {
		t.Fatal("expected return type on closure")
	}
	ref, ok := fnExpr.ReturnType.(ast.TypeExprRef)
	if !ok {
		t.Fatalf("expected TypeExprRef return type, got %T", fnExpr.ReturnType)
	}
	if ref.Reference[0].Value != "String" {
		t.Errorf("return type: want 'String', got %q", ref.Reference[0].Value)
	}
}

func TestParseFnClosureMultipleTypedParams(t *testing.T) {
	input := `fn(a: Int, b: String) { a }`
	srcFile := prepareSourceFileParsing(t, input)
	stmt := srcFile.Statements[0].(*ast.StmtExpr)
	fnExpr := stmt.Expr.(*ast.ExprFunc)
	if len(fnExpr.Parameters) != 2 {
		t.Fatalf("expected 2 params, got %d", len(fnExpr.Parameters))
	}

	ref0 := fnExpr.Parameters[0].TypeHint.(ast.TypeExprRef)
	if ref0.Reference[0].Value != "Int" {
		t.Errorf("param 0 type: want Int, got %q", ref0.Reference[0].Value)
	}
	ref1 := fnExpr.Parameters[1].TypeHint.(ast.TypeExprRef)
	if ref1.Reference[0].Value != "String" {
		t.Errorf("param 1 type: want String, got %q", ref1.Reference[0].Value)
	}
}

func TestParseFnClosureAsArgument(t *testing.T) {
	input := `call(fn(x) { x + 1 })`
	srcFile := prepareSourceFileParsing(t, input)
	if len(srcFile.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(srcFile.Statements))
	}
	stmt := srcFile.Statements[0].(*ast.StmtExpr)
	inv, ok := stmt.Expr.(*ast.ExprInvocation)
	if !ok {
		t.Fatalf("expected *ast.ExprInvocation, got %T", stmt.Expr)
	}
	if len(inv.Arguments) != 1 {
		t.Fatalf("expected 1 argument, got %d", len(inv.Arguments))
	}
	_, ok = inv.Arguments[0].(*ast.ExprFunc)
	if !ok {
		t.Fatalf("expected argument to be *ast.ExprFunc, got %T", inv.Arguments[0])
	}
}
