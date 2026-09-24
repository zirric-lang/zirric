package parser_test

import (
	"fmt"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

func TestParseExprIs(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"x is Int", "(x is Int)"},
		{"1 + 2 is String", "((1+2) is String)"},
		{"a is Bool", "(a is Bool)"},
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
			got := stmt.Expr.Expression()
			if tt.want != got {
				t.Errorf("wrong expression\nwant: %q\ngot:  %q", tt.want, got)
			}
		})
	}
}

func TestParseExprSwitch(t *testing.T) {
	// Wrap in () so switch is parsed as expression via Pratt prefix parser
	tests := []struct {
		label string
		input string
		want  string
	}{
		{
			label: "is-type and default",
			input: "(switch x { case is Int: 1 case _: 0 })",
			want:  "(switch x { case is Int: 1 case _: 0 })",
		},
		{
			label: "value and default",
			input: "(switch x { case 42: 1 case _: 0 })",
			want:  "(switch x { case 42: 1 case _: 0 })",
		},
		{
			label: "multiple is-type cases",
			input: "(switch x { case is Int: 1 case is String: 2 case _: 0 })",
			want:  "(switch x { case is Int: 1 case is String: 2 case _: 0 })",
		},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			if len(srcFile.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(srcFile.Statements))
			}
			stmt, ok := srcFile.Statements[0].(*ast.StmtExpr)
			if !ok {
				t.Fatalf("expected *ast.StmtExpr, got %T", srcFile.Statements[0])
			}
			sw, ok := stmt.Expr.(*ast.ExprSwitch)
			if !ok {
				t.Fatalf("expected *ast.ExprSwitch, got %T", stmt.Expr)
			}
			got := sw.Expression()
			if tt.want != got {
				t.Errorf("wrong expression\nwant: %q\ngot:  %q", tt.want, got)
			}
		})
	}
}

func TestParseExprSwitchCaseKinds(t *testing.T) {
	srcFile := prepareSourceFileParsing(t, "(switch x { case is Int: 1 case 42: 2 case _: 3 })")
	if len(srcFile.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(srcFile.Statements))
	}
	stmt := srcFile.Statements[0].(*ast.StmtExpr)
	sw := stmt.Expr.(*ast.ExprSwitch)

	if len(sw.Cases) != 3 {
		t.Fatalf("expected 3 cases, got %d", len(sw.Cases))
	}
	if sw.Cases[0].Kind != ast.SwitchCaseIsType {
		t.Errorf("case 0: expected SwitchCaseIsType, got %d", sw.Cases[0].Kind)
	}
	if sw.Cases[0].TypeRef == nil || sw.Cases[0].TypeRef.TypeExpression() != "Int" {
		t.Errorf("case 0: expected type ref Int, got %v", sw.Cases[0].TypeRef)
	}
	if sw.Cases[1].Kind != ast.SwitchCaseValue {
		t.Errorf("case 1: expected SwitchCaseValue, got %d", sw.Cases[1].Kind)
	}
	if sw.Cases[2].Kind != ast.SwitchCaseDefault {
		t.Errorf("case 2: expected SwitchCaseDefault, got %d", sw.Cases[2].Kind)
	}
}

func TestParseStmtSwitch(t *testing.T) {
	tests := []struct {
		label     string
		input     string
		caseCount int
	}{
		{
			label:     "two cases with default",
			input:     "switch x {\n case is Int:\n  const y = 1\n case _:\n  const z = 0\n}",
			caseCount: 2,
		},
		{
			label:     "three cases",
			input:     "switch x {\n case is Int:\n  const a = 1\n case is String:\n  const b = 2\n case _:\n  const c = 0\n}",
			caseCount: 3,
		},
		{
			label:     "value case",
			input:     "switch x {\n case 42:\n  const a = 1\n case _:\n  const b = 0\n}",
			caseCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			if len(srcFile.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(srcFile.Statements))
			}
			sw, ok := srcFile.Statements[0].(ast.StmtSwitch)
			if !ok {
				t.Fatalf("expected ast.StmtSwitch, got %T", srcFile.Statements[0])
			}
			if len(sw.Cases) != tt.caseCount {
				t.Fatalf("expected %d cases, got %d", tt.caseCount, len(sw.Cases))
			}
		})
	}
}

func TestParseStmtSwitchCaseBodies(t *testing.T) {
	srcFile := prepareSourceFileParsing(t, `switch x {
case is Int:
const y = 1
y + 2
case _:
0
}`)
	if len(srcFile.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(srcFile.Statements))
	}
	sw, ok := srcFile.Statements[0].(ast.StmtSwitch)
	if !ok {
		t.Fatalf("expected ast.StmtSwitch, got %T", srcFile.Statements[0])
	}
	if len(sw.Cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(sw.Cases))
	}
	// First case body should have 2 statements (const decl + expr stmt)
	if len(sw.Cases[0].Body) != 2 {
		t.Errorf("case 0: expected 2 body stmts, got %d", len(sw.Cases[0].Body))
	}
	if len(sw.Cases[1].Body) != 1 {
		t.Errorf("case 1: expected 1 body stmt, got %d", len(sw.Cases[1].Body))
	}
}

// TestParseExprForSwitchWithBreakAndContinue is a regression test: parseExprForBlock's top-level statement dispatch only special-cased if/break/continue, so a `switch` inside an expr-for body fell through to parseExpr, parsing it as an ExprSwitch whose case bodies require a single expression — break/continue can't appear there at all, so this used to fail with a syntax error.
func TestParseExprForSwitchWithBreakAndContinue(t *testing.T) {
	srcFile := prepareSourceFileParsing(t, `(for x <- items {
switch x {
case is Int:
continue
case is String:
break
case _:
x
}
})`)
	if len(srcFile.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(srcFile.Statements))
	}
	stmt, ok := srcFile.Statements[0].(*ast.StmtExpr)
	if !ok {
		t.Fatalf("expected *ast.StmtExpr, got %T", srcFile.Statements[0])
	}
	forExpr, ok := stmt.Expr.(ast.ExprFor)
	if !ok {
		t.Fatalf("expected ast.ExprFor, got %T", stmt.Expr)
	}
	if len(forExpr.Body.Stmts) != 1 {
		t.Fatalf("expected 1 statement in the for-expr body, got %d", len(forExpr.Body.Stmts))
	}
	sw, ok := forExpr.Body.Stmts[0].(ast.StmtSwitch)
	if !ok {
		t.Fatalf("expected ast.StmtSwitch, got %T", forExpr.Body.Stmts[0])
	}
	if len(sw.Cases) != 3 {
		t.Fatalf("expected 3 cases, got %d", len(sw.Cases))
	}
	if len(sw.Cases[0].Body) != 1 {
		t.Fatalf("case 0 (continue): expected 1 body stmt, got %d", len(sw.Cases[0].Body))
	}
	if _, ok := sw.Cases[0].Body[0].(ast.StmtContinue); !ok {
		t.Errorf("case 0: expected ast.StmtContinue, got %T", sw.Cases[0].Body[0])
	}
	if len(sw.Cases[1].Body) != 1 {
		t.Fatalf("case 1 (break): expected 1 body stmt, got %d", len(sw.Cases[1].Body))
	}
	if _, ok := sw.Cases[1].Body[0].(ast.StmtBreak); !ok {
		t.Errorf("case 1: expected ast.StmtBreak, got %T", sw.Cases[1].Body[0])
	}
}

func TestParseStmtSwitchWithReturn(t *testing.T) {
	// return is allowed inside a switch when the switch is inside a function
	srcFile := prepareSourceFileParsing(t, `fn example(x) {
switch x {
case is Int:
return 1
case _:
return 0
}
}`)
	// fn produces a declaration, not a statement
	if len(srcFile.Statements) != 0 {
		t.Fatalf("expected 0 statements (fn is a decl), got %d", len(srcFile.Statements))
	}
}
