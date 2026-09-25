package parser_test

import (
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
)

func TestParseStringInterpolation(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`"plain"`, `"plain"`},
		{`"n = \(n)"`, `"n = \(n)"`},
		{`"\(a) + \(b) = \(a + b)"`, `"\(a) + \(b) = \((a+b))"`},
		// ExprInvocation renders its callee after the arguments, which is what puts the trailing "f" here.
		{`"\(f("inner \(x)"))"`, `"\(f("inner \(x)"f))"`},
		{`"\\(literal)"`, `"\\(literal)"`},
		{`"\(if ok { a } else { b })"`, `"\((if ok { a } else { b }))"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			stmt, ok := srcFile.Statements[0].(*ast.StmtExpr)
			if !ok {
				t.Fatalf("statement is %T, want *ast.StmtExpr", srcFile.Statements[0])
			}
			if got := stmt.Expr.Expression(); got != tt.want {
				t.Errorf("wrong expression parsed\nwant:\t%q\ngot:\t%q", tt.want, got)
			}
		})
	}
}

// TestAPlainLiteralStaysAPlainLiteral keeps the node every other pass already matches on, so a string with nothing embedded costs nothing new.
func TestAPlainLiteralStaysAPlainLiteral(t *testing.T) {
	srcFile := prepareSourceFileParsing(t, `"just text"`)
	stmt := srcFile.Statements[0].(*ast.StmtExpr)
	if _, ok := stmt.Expr.(*ast.ExprString); !ok {
		t.Fatalf("expression is %T, want *ast.ExprString", stmt.Expr)
	}
}

func TestInterpolationParts(t *testing.T) {
	srcFile := prepareSourceFileParsing(t, `"a \(x) b \(y)"`)
	stmt := srcFile.Statements[0].(*ast.StmtExpr)
	expr, ok := stmt.Expr.(*ast.ExprStringInterpolation)
	if !ok {
		t.Fatalf("expression is %T, want *ast.ExprStringInterpolation", stmt.Expr)
	}
	if len(expr.Parts) != 4 {
		t.Fatalf("got %d parts, want 4", len(expr.Parts))
	}
	if expr.Parts[0].Literal != "a " || expr.Parts[2].Literal != " b " {
		t.Errorf("literal runs are %q and %q", expr.Parts[0].Literal, expr.Parts[2].Literal)
	}
	for _, i := range []int{1, 3} {
		if expr.Parts[i].Expr == nil {
			t.Errorf("part %d holds no expression", i)
		}
	}
}

// TestInterpolationIsLocatedWhereItIsWritten is what makes a diagnostic inside `\( … )` point at the source rather than at a detached fragment of it.
func TestInterpolationIsLocatedWhereItIsWritten(t *testing.T) {
	const input = "mod t\nconst s = \"a \\(value) b\"\n"
	srcFile := prepareSourceFileParsing(t, input)

	symbol := srcFile.Decls.Parent.Symbols["s"]
	if symbol == nil || symbol.Decl == nil {
		t.Fatal("expected a declaration for s")
	}
	expr := symbol.Decl.(*ast.DeclConstant).Value.(*ast.ExprStringInterpolation)
	source := expr.Parts[1].Expr.TokenLiteral().Source
	if source == nil {
		t.Fatal("the embedded expression carries no source")
	}
	if source.Line != 2 {
		t.Errorf("line = %d, want 2", source.Line)
	}
	if got := input[source.Offset : source.Offset+len("value")]; got != "value" {
		t.Errorf("offset %d points at %q, want %q", source.Offset, got, "value")
	}
	if source.Column != len(`const s = "a \(`)+1 {
		t.Errorf("column = %d, want %d", source.Column, len(`const s = "a \(`)+1)
	}
}

func TestInterpolationErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", `const s = "\()"`, "empty interpolation"},
		{"unclosed", `const s = "\(1 + 2"`, "unclosed interpolation"},
		{"line break", "const s = \"\\(1 +\n2)\"", "unclosed interpolation"},
		{"two expressions", `const s = "\(1 2)"`, "a single expression"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := lexer.New(staticmodule.NewSourceString("testing:///t.zirr", tt.input))
			if err != nil {
				t.Fatal(err)
			}
			module := ast.MakeContextModule(registry.LogicalURI("test"))
			p := parser.NewSourceParser(l, module.Decls, "t.zirr")
			p.ParseSourceFile()

			var messages []string
			for _, e := range p.Errors() {
				messages = append(messages, e.Error())
			}
			if !strings.Contains(strings.Join(messages, "\n"), tt.want) {
				t.Errorf("errors %q do not mention %q", messages, tt.want)
			}
		})
	}
}
