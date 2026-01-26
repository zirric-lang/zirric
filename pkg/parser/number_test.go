package parser_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

func TestNumberLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"42", 42},
		{"0xFFF", 0xFFF},       // 4095
		{"0x8899aa", 0x8899aa}, // 8956330
		{"0777", 0777},         // 511 (octal)
		{"0b101010", 0b101010}, // 42 (binary)
		{"0B100011", 0b100011}, // 35 (binary uppercase)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			if len(srcFile.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(srcFile.Statements))
			}

			stmt, ok := srcFile.Statements[0].(*ast.StmtExpr)
			if !ok {
				t.Fatalf("expected *ast.StmtExpr, got %T", srcFile.Statements[0])
			}

			intExpr, ok := stmt.Expr.(*ast.ExprInt)
			if !ok {
				t.Fatalf("expected *ast.ExprInt, got %T", stmt.Expr)
			}

			if intExpr.Literal != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, intExpr.Literal)
			}
		})
	}
}

func TestFloatLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"3.14", 3.14},
		{"2e10", 2e10},
		{"1.5e-3", 1.5e-3},
		{"3.14E+2", 3.14e+2},
		{"42.0", 42.0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			if len(srcFile.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(srcFile.Statements))
			}

			stmt, ok := srcFile.Statements[0].(*ast.StmtExpr)
			if !ok {
				t.Fatalf("expected *ast.StmtExpr, got %T", srcFile.Statements[0])
			}

			floatExpr, ok := stmt.Expr.(*ast.ExprFloat)
			if !ok {
				t.Fatalf("expected *ast.ExprFloat, got %T", stmt.Expr)
			}

			if floatExpr.Literal != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, floatExpr.Literal)
			}
		})
	}
}
