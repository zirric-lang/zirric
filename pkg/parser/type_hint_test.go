package parser_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

func lookupDeclSymbol(t *testing.T, sf *ast.SourceFile, name string) *ast.DeclSymbol {
	t.Helper()
	sym, ok := sf.Decls.Parent.Symbols[name]
	if !ok {
		if sym2, ok2 := sf.Decls.Symbols[name]; ok2 {
			return sym2
		}
		t.Fatalf("symbol %q not found in declaration table", name)
	}
	return sym
}

func TestParseFieldWithTypeHint(t *testing.T) {
	input := `data Foo { name: String }`
	sf := prepareSourceFileParsing(t, input)

	sym := lookupDeclSymbol(t, sf, "Foo")
	decl, ok := sym.Decl.(*ast.DeclData)
	if !ok {
		t.Fatalf("expected DeclData, got %T", sym.Decl)
	}
	if len(decl.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(decl.Fields))
	}
	field := decl.Fields[0]
	if field.Name.Value != "name" {
		t.Errorf("field name: want 'name', got %q", field.Name.Value)
	}
	if field.TypeHint == nil {
		t.Fatal("expected type hint on field, got nil")
	}
	ref, ok := field.TypeHint.(ast.TypeExprRef)
	if !ok {
		t.Fatalf("expected TypeExprRef type hint, got %T", field.TypeHint)
	}
	if ref.Reference[0].Value != "String" {
		t.Errorf("type hint: want 'String', got %q", ref.Reference[0].Value)
	}
}

func TestParseFieldWithoutTypeHint(t *testing.T) {
	input := `data Foo { name }`
	sf := prepareSourceFileParsing(t, input)

	sym := lookupDeclSymbol(t, sf, "Foo")
	decl := sym.Decl.(*ast.DeclData)
	field := decl.Fields[0]
	if field.TypeHint != nil {
		t.Errorf("expected nil type hint, got %v", field.TypeHint)
	}
}

func TestParseParameterWithTypeHint(t *testing.T) {
	input := `fn greet(name: String) {}`
	sf := prepareSourceFileParsing(t, input)

	sym := lookupDeclSymbol(t, sf, "greet")
	decl, ok := sym.Decl.(*ast.DeclFunc)
	if !ok {
		t.Fatalf("expected DeclFunc, got %T", sym.Decl)
	}
	if len(decl.Impl.Parameters) != 1 {
		t.Fatalf("expected 1 param, got %d", len(decl.Impl.Parameters))
	}
	param := decl.Impl.Parameters[0]
	if param.Name.Value != "name" {
		t.Errorf("param name: want 'name', got %q", param.Name.Value)
	}
	if param.TypeHint == nil {
		t.Fatal("expected type hint on param, got nil")
	}
	ref := param.TypeHint.(ast.TypeExprRef)
	if ref.Reference[0].Value != "String" {
		t.Errorf("type hint: want 'String', got %q", ref.Reference[0].Value)
	}
}

func TestParseConstWithTypeHint(t *testing.T) {
	input := `const x: Int = 42`
	sf := prepareSourceFileParsing(t, input)

	sym := lookupDeclSymbol(t, sf, "x")
	decl, ok := sym.Decl.(*ast.DeclConstant)
	if !ok {
		t.Fatalf("expected DeclConstant, got %T", sym.Decl)
	}
	if decl.Name.Value != "x" {
		t.Errorf("name: want 'x', got %q", decl.Name.Value)
	}
	if decl.TypeHint == nil {
		t.Fatal("expected type hint, got nil")
	}
	ref := decl.TypeHint.(ast.TypeExprRef)
	if ref.Reference[0].Value != "Int" {
		t.Errorf("type hint: want 'Int', got %q", ref.Reference[0].Value)
	}
}

func TestParseVarWithTypeHint(t *testing.T) {
	input := `fn f() { var x: Int = 42 }`
	sf := prepareSourceFileParsing(t, input)

	sym := lookupDeclSymbol(t, sf, "f")
	fn := sym.Decl.(*ast.DeclFunc)
	block := fn.Impl.Impl
	if len(block) == 0 {
		t.Fatal("expected at least 1 statement in function body")
	}
	decl, ok := block[0].(*ast.DeclVariable)
	if !ok {
		t.Fatalf("expected DeclVariable, got %T", block[0])
	}
	if decl.TypeHint == nil {
		t.Fatal("expected type hint, got nil")
	}
	ref := decl.TypeHint.(ast.TypeExprRef)
	if ref.Reference[0].Value != "Int" {
		t.Errorf("type hint: want 'Int', got %q", ref.Reference[0].Value)
	}
}

func TestParseFuncReturnType(t *testing.T) {
	input := `fn add(a: Int, b: Int) -> Int {}`
	sf := prepareSourceFileParsing(t, input)

	sym := lookupDeclSymbol(t, sf, "add")
	decl := sym.Decl.(*ast.DeclFunc)
	if decl.ReturnType == nil {
		t.Fatal("expected return type, got nil")
	}
	ref := decl.ReturnType.(ast.TypeExprRef)
	if ref.Reference[0].Value != "Int" {
		t.Errorf("return type: want 'Int', got %q", ref.Reference[0].Value)
	}
}

func TestParseFieldMethodReturnType(t *testing.T) {
	input := `data Foo { greet(name: String) -> String }`
	sf := prepareSourceFileParsing(t, input)

	sym := lookupDeclSymbol(t, sf, "Foo")
	decl := sym.Decl.(*ast.DeclData)
	field := decl.Fields[0]
	if field.Name.Value != "greet" {
		t.Errorf("field name: want 'greet', got %q", field.Name.Value)
	}
	if len(field.Parameters) != 1 {
		t.Fatalf("expected 1 param, got %d", len(field.Parameters))
	}
	if field.TypeHint == nil {
		t.Fatal("expected return type hint on field method, got nil")
	}
	ref := field.TypeHint.(ast.TypeExprRef)
	if ref.Reference[0].Value != "String" {
		t.Errorf("return type: want 'String', got %q", ref.Reference[0].Value)
	}
}

func TestParseFieldWithAttrAndType(t *testing.T) {
	input := `
		attr Default { value }
		data Config {
			@Default("hello") name: String
		}
	`
	sf := prepareSourceFileParsing(t, input)

	sym := lookupDeclSymbol(t, sf, "Config")
	decl := sym.Decl.(*ast.DeclData)
	field := decl.Fields[0]
	if field.Name.Value != "name" {
		t.Errorf("field name: want 'name', got %q", field.Name.Value)
	}
	if len(field.Attributes) != 1 {
		t.Fatalf("expected 1 attribute, got %d", len(field.Attributes))
	}
	if field.TypeHint == nil {
		t.Fatal("expected type hint, got nil")
	}
	ref := field.TypeHint.(ast.TypeExprRef)
	if ref.Reference[0].Value != "String" {
		t.Errorf("type: want 'String', got %q", ref.Reference[0].Value)
	}
}

func TestParseDottedTypeHint(t *testing.T) {
	input := `data Foo { name: prelude.String }`
	sf := prepareSourceFileParsing(t, input)

	sym := lookupDeclSymbol(t, sf, "Foo")
	decl := sym.Decl.(*ast.DeclData)
	field := decl.Fields[0]
	if field.TypeHint == nil {
		t.Fatal("expected type hint, got nil")
	}
	ref, ok := field.TypeHint.(ast.TypeExprRef)
	if !ok {
		t.Fatalf("expected TypeExprRef, got %T", field.TypeHint)
	}
	if len(ref.Reference) != 2 {
		t.Fatalf("expected 2-part reference, got %d", len(ref.Reference))
	}
	if ref.Reference[0].Value != "prelude" {
		t.Errorf("first part: want 'prelude', got %q", ref.Reference[0].Value)
	}
	if ref.Reference[1].Value != "String" {
		t.Errorf("second part: want 'String', got %q", ref.Reference[1].Value)
	}
}

// TestParseTypeShorthands covers `T?` and `T!`, including the adjacency rule that keeps a hint from swallowing the line after it.
func TestParseTypeShorthands(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`data Foo { name: String? }`, "String?"},
		{`data Foo { name: String! }`, "String!"},
		{`data Foo { name: [Int]? }`, "[Int]?"},
		{`data Foo { name: a.User? }`, "a.User?"},
		{`data Foo { name: fn() -> Int? }`, "fn() -> Int?"},
		// The shorthands stack and read left to right.
		{`data Foo { name: String?! }`, "String?!"},
		// A space between the type and the suffix means the suffix is not one.
		{`data Foo { name: String }`, "String"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			sf := prepareSourceFileParsing(t, tt.input)
			decl := lookupDeclSymbol(t, sf, "Foo").Decl.(*ast.DeclData)
			hint := decl.Fields[0].TypeHint
			if hint == nil {
				t.Fatal("expected a type hint")
			}
			if got := hint.TypeExpression(); got != tt.want {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		})
	}
}

// TestTypeHintDoesNotSwallowTheNextLine is a regression guard: `!` starts plenty of statements, so a return type has to end at the end of its own line.
func TestTypeHintDoesNotSwallowTheNextLine(t *testing.T) {
	sf := prepareSourceFileParsing(t, "extern fn ready() -> Bool\nconst notReady = !ready()")

	decl, ok := lookupDeclSymbol(t, sf, "ready").Decl.(*ast.DeclExternFunc)
	if !ok {
		t.Fatalf("expected DeclExternFunc, got %T", lookupDeclSymbol(t, sf, "ready").Decl)
	}
	if got := decl.ReturnType.TypeExpression(); got != "Bool" {
		t.Errorf("expected the return type to end at Bool, got %q", got)
	}
}
