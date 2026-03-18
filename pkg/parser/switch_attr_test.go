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

func TestParseIsAttribute(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"val is @Numeric", "(val is @Numeric)"},
		{"1 + 2 is @Countable", "((1+2) is @Countable)"},
		{"val is @prelude.Iterable", "(val is @prelude.Iterable)"},
		{"val is @Attr1 @Attr2 @Attr3", "(val is @Attr1 @Attr2 @Attr3)"},
		{"val is @mymod.A @mymod.B", "(val is @mymod.A @mymod.B)"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d. %s", i, tt.input), func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			if len(srcFile.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(srcFile.Statements))
				return
			}
			stmt, ok := srcFile.Statements[0].(*ast.StmtExpr)
			if !ok {
				t.Fatalf("expected *ast.StmtExpr, got %T", srcFile.Statements[0])
				return
			}
			got := stmt.Expr.Expression()
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestParseIsWithStaticRef(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"val is prelude.String", "(val is prelude.String)"},
		{"x is mymod.sub.Type", "(x is mymod.sub.Type)"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d. %s", i, tt.input), func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			if len(srcFile.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(srcFile.Statements))
				return
			}
			stmt, ok := srcFile.Statements[0].(*ast.StmtExpr)
			if !ok {
				t.Fatalf("expected *ast.StmtExpr, got %T", srcFile.Statements[0])
				return
			}
			isExpr, ok := stmt.Expr.(ast.ExprIs)
			if !ok {
				t.Fatalf("expected ExprIs, got %T", stmt.Expr)
				return
			}
			// TypeRef should be a TypeExprRef, not TypeExprAttrs
			if _, isAttrs := isExpr.TypeRef.(ast.TypeExprAttrs); isAttrs {
				t.Fatal("expected type check, not attribute check")
			}
			got := isExpr.Expression()
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestParseSwitchExprWithAttributeCase(t *testing.T) {
	tests := []struct {
		label string
		input string
		want  string
	}{
		{
			label: "attribute and default",
			input: "(switch x { case is @Numeric: 1 case _: 0 })",
			want:  "(switch x { case is @Numeric: 1 case _: 0 })",
		},
		{
			label: "multiple attribute cases",
			input: "(switch x { case is @Alpha: 1 case is @Beta: 2 case _: 0 })",
			want:  "(switch x { case is @Alpha: 1 case is @Beta: 2 case _: 0 })",
		},
		{
			label: "mixed is-type and attribute",
			input: "(switch x { case is Foo: 1 case is @Bar: 2 case 42: 3 case _: 0 })",
			want:  "(switch x { case is Foo: 1 case is @Bar: 2 case 42: 3 case _: 0 })",
		},
		{
			label: "static ref in is-type",
			input: "(switch x { case is mymod.Foo: 1 case _: 0 })",
			want:  "(switch x { case is mymod.Foo: 1 case _: 0 })",
		},
		{
			label: "static ref in attribute",
			input: "(switch x { case is @mymod.Attr: 1 case _: 0 })",
			want:  "(switch x { case is @mymod.Attr: 1 case _: 0 })",
		},
		{
			label: "multiple attributes in one case",
			input: "(switch x { case is @A @B: 1 case _: 0 })",
			want:  "(switch x { case is @A @B: 1 case _: 0 })",
		},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			if len(srcFile.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(srcFile.Statements))
				return
			}
			stmt, ok := srcFile.Statements[0].(*ast.StmtExpr)
			if !ok {
				t.Fatalf("expected *ast.StmtExpr, got %T", srcFile.Statements[0])
				return
			}
			sw, ok := stmt.Expr.(*ast.ExprSwitch)
			if !ok {
				t.Fatalf("expected *ast.ExprSwitch, got %T", stmt.Expr)
				return
			}
			got := sw.Expression()
			if tt.want != got {
				t.Errorf("wrong expression\nwant: %q\ngot:  %q", tt.want, got)
			}
		})
	}
}

func TestParseSwitchStmtWithAttributeCase(t *testing.T) {
	input := `fn test(val) {
		switch val {
			case is @Numeric:
				return 1
			case is @Iterable:
				return 2
			case _:
				return 0
		}
	}`
	srcFile := prepareSourceFileParsing(t, input)
	// fn declarations are hoisted into the module DeclTable by the parser
	fn := findDeclFunc(t, srcFile, "test")
	if fn == nil {
		return
	}
	if len(fn.Impl.Impl) == 0 {
		t.Fatal("expected function body")
		return
	}
	sw, ok := fn.Impl.Impl[0].(ast.StmtSwitch)
	if !ok {
		t.Fatalf("expected StmtSwitch, got %T", fn.Impl.Impl[0])
		return
	}
	if len(sw.Cases) != 3 {
		t.Fatalf("expected 3 cases, got %d", len(sw.Cases))
		return
	}
	if sw.Cases[0].Kind != ast.SwitchCaseIsType {
		t.Fatalf("case 0: expected SwitchCaseIsType, got %d", sw.Cases[0].Kind)
	}
	if sw.Cases[0].TypeRef == nil {
		t.Fatal("case 0: expected TypeRef to be set")
	}
	if sw.Cases[0].TypeRef.TypeExpression() != "@Numeric" {
		t.Fatalf("expected '@Numeric', got %q", sw.Cases[0].TypeRef.TypeExpression())
	}
	if sw.Cases[1].Kind != ast.SwitchCaseIsType {
		t.Fatalf("case 1: expected SwitchCaseIsType, got %d", sw.Cases[1].Kind)
	}
	if sw.Cases[1].TypeRef == nil {
		t.Fatal("case 1: expected TypeRef to be set")
	}
	if sw.Cases[1].TypeRef.TypeExpression() != "@Iterable" {
		t.Fatalf("expected '@Iterable', got %q", sw.Cases[1].TypeRef.TypeExpression())
	}
	if sw.Cases[2].Kind != ast.SwitchCaseDefault {
		t.Fatalf("case 2: expected SwitchCaseDefault, got %d", sw.Cases[2].Kind)
	}
}

// findDeclFunc locates a DeclFunc by name, checking statements and hoisted decl table (recursively).
func findDeclFunc(t *testing.T, srcFile *ast.SourceFile, name string) *ast.DeclFunc {
	t.Helper()
	for _, stmt := range srcFile.Statements {
		if fn, ok := stmt.(*ast.DeclFunc); ok && fn.Name.Value == name {
			return fn
		}
	}
	fn := findInDeclTable(srcFile.Decls, name)
	if fn != nil {
		return fn
	}
	t.Fatalf("function %q not found", name)
	return nil
}

func findInDeclTable(dt *ast.DeclTable, name string) *ast.DeclFunc {
	if dt == nil {
		return nil
	}
	if ds, ok := dt.Symbols[name]; ok {
		if fn, ok := ds.Decl.(*ast.DeclFunc); ok {
			return fn
		}
	}
	return findInDeclTable(dt.Parent, name)
}

func TestParseAttributeInstanceRequiresParens(t *testing.T) {
	// Attribute application on declarations requires parentheses: @Marker() not @Marker
	input := `
	attr Marker
	@Marker
	data Foo
	`
	module := ast.MakeContextModule(registry.LogicalURI("test"))
	parentTable := module.Decls
	l, err := lexer.New(staticmodule.NewSourceString("testing:///test.zirr", input))
	if err != nil {
		t.Fatal(err)
	}
	p := parser.NewSourceParser(l, parentTable, "test.zirr")
	_ = p.ParseSourceFile()
	if len(p.Errors()) == 0 {
		t.Fatal("expected parser error for @Marker without parentheses, but got none")
	}
}

func TestParseAttributeInstanceWithEmptyParens(t *testing.T) {
	// @Marker() with empty parens should parse successfully
	input := `
	attr Marker
	@Marker()
	data Foo
	`
	srcFile := prepareSourceFileParsing(t, input)
	// Declarations are hoisted into Parent DeclTable
	if srcFile.Decls == nil || srcFile.Decls.Parent == nil || srcFile.Decls.Parent.Symbols["Foo"] == nil {
		t.Fatal("expected Foo declaration")
	}
}
