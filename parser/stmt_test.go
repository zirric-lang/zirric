package parser_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/ast"
)

func TestParseStatementElseIf(t *testing.T) {
	tests := []struct {
		input     string
		elseIfLen int
		elseLen   int
	}{
		{"if true { return 1 } else if false { return 2 } else { return 3 }", 1, 1},
		{"if true { return 1 } else if false { return 2 }", 1, 0},
		{"if true { return 1 } else if false { return 2 } else if true { return 3 } else { return 4 }", 2, 1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)

			if len(srcFile.Statements) != 1 {
				t.Fatalf("expected one statement, got %d", len(srcFile.Statements))
			}
			stmt, ok := srcFile.Statements[0].(ast.StmtIf)
			if !ok {
				t.Fatalf("statement is %T, want ast.StmtIf", srcFile.Statements[0])
			}
			if len(stmt.ElseIf) != tt.elseIfLen {
				t.Errorf("expected %d else-if, got %d", tt.elseIfLen, len(stmt.ElseIf))
			}
			if len(stmt.ElseBlock) != tt.elseLen {
				t.Errorf("expected else block with %d stmt, got %d", tt.elseLen, len(stmt.ElseBlock))
			}
		})
	}
}

func TestParseStatementForInfinite(t *testing.T) {
	srcFile := prepareSourceFileParsing(t, "for { break }")

	if len(srcFile.Statements) != 1 {
		t.Fatalf("expected one statement, got %d", len(srcFile.Statements))
	}
	stmt, ok := srcFile.Statements[0].(ast.StmtFor)
	if !ok {
		t.Fatalf("statement is %T, want ast.StmtFor", srcFile.Statements[0])
	}
	if stmt.Condition != nil {
		t.Fatalf("expected no condition, got %T", stmt.Condition)
	}
	if stmt.CollectionIdent != nil {
		t.Fatalf("expected no collection identifier, got %T", stmt.CollectionIdent)
	}
	if stmt.CollectionExpr != nil {
		t.Fatalf("expected no collection expr, got %T", stmt.CollectionExpr)
	}
	if len(stmt.Body) != 1 {
		t.Fatalf("expected one body statement, got %d", len(stmt.Body))
	}
	if _, ok := stmt.Body[0].(ast.StmtBreak); !ok {
		t.Fatalf("body statement is %T, want ast.StmtBreak", stmt.Body[0])
	}
}

func TestParseStatementForCondition(t *testing.T) {
	srcFile := prepareSourceFileParsing(t, "for true { continue }")

	if len(srcFile.Statements) != 1 {
		t.Fatalf("expected one statement, got %d", len(srcFile.Statements))
	}
	stmt, ok := srcFile.Statements[0].(ast.StmtFor)
	if !ok {
		t.Fatalf("statement is %T, want ast.StmtFor", srcFile.Statements[0])
	}
	if stmt.Condition == nil {
		t.Fatal("expected condition expression")
	}
	if stmt.CollectionIdent != nil {
		t.Fatalf("expected no collection identifier, got %T", stmt.CollectionIdent)
	}
	if stmt.CollectionExpr != nil {
		t.Fatalf("expected no collection expr, got %T", stmt.CollectionExpr)
	}
	if len(stmt.Body) != 1 {
		t.Fatalf("expected one body statement, got %d", len(stmt.Body))
	}
	if _, ok := stmt.Body[0].(ast.StmtContinue); !ok {
		t.Fatalf("body statement is %T, want ast.StmtContinue", stmt.Body[0])
	}
}

func TestParseStatementForCollection(t *testing.T) {
	srcFile := prepareSourceFileParsing(t, "for item <- items { break }")

	if len(srcFile.Statements) != 1 {
		t.Fatalf("expected one statement, got %d", len(srcFile.Statements))
	}
	stmt, ok := srcFile.Statements[0].(ast.StmtFor)
	if !ok {
		t.Fatalf("statement is %T, want ast.StmtFor", srcFile.Statements[0])
	}
	if stmt.Condition != nil {
		t.Fatalf("expected no condition, got %T", stmt.Condition)
	}
	if stmt.CollectionIdent == nil {
		t.Fatal("expected collection identifier")
	}
	if stmt.CollectionIdent.Value != "item" {
		t.Fatalf("expected collection identifier item, got %q", stmt.CollectionIdent.Value)
	}
	if stmt.CollectionExpr == nil {
		t.Fatal("expected collection expression")
	}
	if _, ok := stmt.CollectionExpr.(*ast.ExprIdentifier); !ok {
		t.Fatalf("collection expression is %T, want *ast.ExprIdentifier", stmt.CollectionExpr)
	}
	if len(stmt.Body) != 1 {
		t.Fatalf("expected one body statement, got %d", len(stmt.Body))
	}
	if _, ok := stmt.Body[0].(ast.StmtBreak); !ok {
		t.Fatalf("body statement is %T, want ast.StmtBreak", stmt.Body[0])
	}
}
