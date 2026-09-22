package parser_test

import (
	"fmt"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
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
		{"fn() {}", "fn() {/* 0 stmts */}"},
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

// TestFloatLiteralCarriesItsToken is a regression test: MakeExprFloat took a token and dropped it, so every float literal in the tree had a zero token and anything anchored to one — a diagnostic, a hover, a stack frame — had no position to report.
func TestFloatLiteralCarriesItsToken(t *testing.T) {
	srcFile := prepareSourceFileParsing(t, "13.37")
	stmt := srcFile.Statements[0].(*ast.StmtExpr)
	source := stmt.Expr.TokenLiteral().Source
	if source == nil || source.Line <= 0 {
		t.Fatalf("expected the literal to carry a position, got %v", source)
	}
}

// TestExprGuardedMemberAccess covers `?.` and `!.`: that they parse as member accesses, that they bind like a plain dot, and that the operator survives into the tree rather than being flattened into one.
func TestExprGuardedMemberAccess(t *testing.T) {
	tests := []struct {
		input string
		want  string
		// access lists the operator of every member access in the tree, outermost first, which is the order the node walk visits them in.
		access []ast.MemberAccess
	}{
		{"user.name", "user.name", []ast.MemberAccess{ast.MemberAccessPlain}},
		{"user?.name", "user?.name", []ast.MemberAccess{ast.MemberAccessOption}},
		{"result!.value", "result!.value", []ast.MemberAccess{ast.MemberAccessResult}},
		{
			"user?.address?.city",
			"user?.address?.city",
			[]ast.MemberAccess{
				ast.MemberAccessOption,
				ast.MemberAccessOption,
			},
		},
		// A guarded access binds tighter than arithmetic, the same way a plain dot does.
		{"a?.b + c!.d", "(a?.b+c!.d)", []ast.MemberAccess{ast.MemberAccessOption, ast.MemberAccessResult}},
		// A fallback binds looser than the chain in front of it and than arithmetic behind it.
		{"a?.b ?? c", "(a?.b??c)", []ast.MemberAccess{ast.MemberAccessOption}},
		{"a!.b !! c", "(a!.b!!c)", []ast.MemberAccess{ast.MemberAccessResult}},
		{"a ?? b + 1", "(a??(b+1))", nil},
		{"a ?? b ?? c", "(a??(b??c))", nil},
		{"a ?? b == c", "((a??b)==c)", nil},
		{"readFile(path)!.value", "readFile(pathreadFile)!.value", []ast.MemberAccess{ast.MemberAccessResult}},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d.\t test: %q", i+1, tt.input), func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			if len(srcFile.Statements) != 1 {
				t.Fatalf("srcFile.Statements does not contain 1 statement, got %d", len(srcFile.Statements))
			}
			stmt, ok := srcFile.Statements[0].(*ast.StmtExpr)
			if !ok {
				t.Fatalf("srcFile.Statements[0] is not *ast.StmtExpr, got %T", srcFile.Statements[0])
			}
			if got := stmt.Expr.Expression(); got != tt.want {
				t.Errorf("wrong expression parsed\nwant:\t%q\ngot:\t%q", tt.want, got)
			}

			var got []ast.MemberAccess
			collect := func(node ast.Node) {
				if member, ok := node.(*ast.ExprMemberAccess); ok {
					got = append(got, member.Access())
				}
			}
			collect(stmt.Expr)
			stmt.Expr.EnumerateChildNodes(collect)
			if len(got) == 0 && len(tt.access) == 0 {
				return
			}
			if len(got) != len(tt.access) {
				t.Fatalf("expected %d member accesses, got %d (%v)", len(tt.access), len(got), got)
			}
			for j := range tt.access {
				if got[j] != tt.access[j] {
					t.Errorf("member access %d: expected %q, got %q", j, tt.access[j].Operator(), got[j].Operator())
				}
			}
		})
	}
}

// TestGuardedAccessIsNotAnAssignmentTarget covers a guard anywhere along the chain, not just at its end: `user?.value.name` ends in a plain dot, yet the place it would write to only exists when the guard lets the chain through.
func TestGuardedAccessIsNotAnAssignmentTarget(t *testing.T) {
	rejected := []string{
		"user?.name = 1",
		"user!.name = 1",
		"user?.name = 1",
		"user?.items[0] = 1",
		"user?.name += 1",
	}

	for _, input := range rejected {
		t.Run(input, func(t *testing.T) {
			module := ast.MakeContextModule(registry.LogicalURI("test"))
			l, err := lexer.New(staticmodule.NewSourceString("testing:///test.zirr", input))
			if err != nil {
				t.Fatal(err)
			}
			p := parser.NewSourceParser(l, module.Decls, "test.zirr")
			p.ParseSourceFile()

			errs := p.Errors()
			if len(errs) == 0 {
				t.Fatalf("expected %q to be rejected", input)
			}
			if errs[0].Summary != "invalid assignment target" {
				t.Fatalf("expected an invalid assignment target, got %q: %s", errs[0].Summary, errs[0].Details)
			}
		})
	}

	// The plain forms are still assignable.
	for _, input := range []string{"user.name = 1", "items[0] = 1", "count = 1"} {
		t.Run(input, func(t *testing.T) {
			prepareSourceFileParsing(t, input)
		})
	}
}
