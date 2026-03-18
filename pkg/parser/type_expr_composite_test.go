package parser_test

import (
	"fmt"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

func getDataDecl(t *testing.T, srcFile *ast.SourceFile, name string) *ast.DeclData {
	t.Helper()
	sym := lookupDeclSymbol(t, srcFile, name)
	data, ok := sym.Decl.(*ast.DeclData)
	if !ok {
		t.Fatalf("expected *ast.DeclData, got %T", sym.Decl)
	}
	return data
}

func getFuncDecl(t *testing.T, srcFile *ast.SourceFile, name string) *ast.DeclFunc {
	t.Helper()
	sym := lookupDeclSymbol(t, srcFile, name)
	fn, ok := sym.Decl.(*ast.DeclFunc)
	if !ok {
		t.Fatalf("expected *ast.DeclFunc, got %T", sym.Decl)
	}
	return fn
}

func TestParseTypeHintArray(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"data Foo { items: [Int] }", "[Int]"},
		{"data Foo { items: [String] }", "[String]"},
		{"data Foo { items: [[Int]] }", "[[Int]]"},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d. %s", i, tt.input), func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			data := getDataDecl(t, srcFile, "Foo")
			if len(data.Fields) == 0 {
				t.Fatal("expected at least one field")
			}
			if data.Fields[0].TypeHint == nil {
				t.Fatal("expected type hint on field")
			}
			got := data.Fields[0].TypeHint.TypeExpression()
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestParseTypeHintDict(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"data Foo { entries: [String: Int] }", "[String: Int]"},
		{"data Foo { entries: [String: [Int]] }", "[String: [Int]]"},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d. %s", i, tt.input), func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			data := getDataDecl(t, srcFile, "Foo")
			if len(data.Fields) == 0 {
				t.Fatal("expected at least one field")
			}
			if data.Fields[0].TypeHint == nil {
				t.Fatal("expected type hint on field")
			}
			got := data.Fields[0].TypeHint.TypeExpression()
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestParseTypeHintFunc(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"data Foo { cb: fn(a) }", "fn(a)"},
		{"data Foo { cb: fn(a, b) }", "fn(a, b)"},
		{"data Foo { cb: fn(a: Int) -> String }", "fn(a: Int) -> String"},
		{"data Foo { cb: fn() -> Bool }", "fn() -> Bool"},
		{"data Foo { cb: fn() }", "fn()"},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d. %s", i, tt.input), func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			data := getDataDecl(t, srcFile, "Foo")
			if len(data.Fields) == 0 {
				t.Fatal("expected at least one field")
			}
			if data.Fields[0].TypeHint == nil {
				t.Fatal("expected type hint on field")
			}
			got := data.Fields[0].TypeHint.TypeExpression()
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestParseTypeHintInFunctionParam(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"fn foo(x: [Int]) {}", "[Int]"},
		{"fn foo(x: [String: Int]) {}", "[String: Int]"},
		{"fn foo(x: fn(a) -> Int) {}", "fn(a) -> Int"},
		{"fn foo(x: @Numeric) {}", "@Numeric"},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d. %s", i, tt.input), func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			fn := getFuncDecl(t, srcFile, "foo")
			if len(fn.Impl.Parameters) == 0 {
				t.Fatal("expected at least one parameter")
			}
			if fn.Impl.Parameters[0].TypeHint == nil {
				t.Fatal("expected type hint on parameter")
			}
			got := fn.Impl.Parameters[0].TypeHint.TypeExpression()
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestParseTypeHintReturnType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"fn foo() -> [Int] {}", "[Int]"},
		{"fn foo() -> [String: Int] {}", "[String: Int]"},
		{"fn foo() -> fn(a) -> Int {}", "fn(a) -> Int"},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d. %s", i, tt.input), func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			fn := getFuncDecl(t, srcFile, "foo")
			if fn.ReturnType == nil {
				t.Fatal("expected return type hint")
			}
			got := fn.ReturnType.TypeExpression()
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestParseTypeHintInIsExpr(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"x is [Int]", "(x is [Int])"},
		{"x is [String: Int]", "(x is [String: Int])"},
		{"x is fn(a) -> String", "(x is fn(a) -> String)"},
		{"x is @Marker", "(x is @Marker)"},
		{"x is @A @B", "(x is @A @B)"},
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
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestParseTypeHintInSwitchExpr(t *testing.T) {
	tests := []struct {
		label string
		input string
		want  string
	}{
		{
			label: "array type in case",
			input: "(switch x { case is [Int]: 1 case _: 0 })",
			want:  "(switch x { case is [Int]: 1 case _: 0 })",
		},
		{
			label: "dict type in case",
			input: "(switch x { case is [String: Int]: 1 case _: 0 })",
			want:  "(switch x { case is [String: Int]: 1 case _: 0 })",
		},
		{
			label: "fn type in case",
			input: "(switch x { case is fn(a) -> Int: 1 case _: 0 })",
			want:  "(switch x { case is fn(a) -> Int: 1 case _: 0 })",
		},
		{
			label: "mixed type expressions",
			input: "(switch x { case is [Int]: 1 case is @Marker: 2 case is Foo: 3 case _: 0 })",
			want:  "(switch x { case is [Int]: 1 case is @Marker: 2 case is Foo: 3 case _: 0 })",
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
			got := stmt.Expr.Expression()
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestParseNestedCompositeTypes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"data Foo { cbs: [fn(a) -> Int] }", "[fn(a) -> Int]"},
		{"data Foo { items: [String: [Int]] }", "[String: [Int]]"},
		{"data Foo { cb: fn() -> [String] }", "fn() -> [String]"},
		{"data Foo { cb: fn(d: [String: Int]) }", "fn(d: [String: Int])"},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d. %s", i, tt.input), func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)
			data := getDataDecl(t, srcFile, "Foo")
			if len(data.Fields) == 0 {
				t.Fatal("expected at least one field")
			}
			if data.Fields[0].TypeHint == nil {
				t.Fatal("expected type hint on field")
			}
			got := data.Fields[0].TypeHint.TypeExpression()
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
