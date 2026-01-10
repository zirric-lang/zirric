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
