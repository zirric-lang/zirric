package parser_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

func TestParseExternConstWithTypeHint(t *testing.T) {
	input := `extern const void: Void`
	srcFile := prepareSourceFileParsing(t, input)

	sym := lookupDeclSymbol(t, srcFile, "void")
	decl, ok := sym.Decl.(*ast.DeclExternValue)
	if !ok {
		t.Fatalf("expected *ast.DeclExternValue, got %T", sym.Decl)
	}
	if decl.Name.Value != "void" {
		t.Errorf("name: want 'void', got %q", decl.Name.Value)
	}
	if decl.TypeHint == nil {
		t.Fatal("expected type hint, got nil")
	}
	ref, ok := decl.TypeHint.(ast.TypeExprRef)
	if !ok {
		t.Fatalf("expected TypeExprRef, got %T", decl.TypeHint)
	}
	if ref.Reference[0].Value != "Void" {
		t.Errorf("type: want 'Void', got %q", ref.Reference[0].Value)
	}
}

func TestParseExternConstWithoutTypeHint(t *testing.T) {
	input := `extern const myvalue`
	srcFile := prepareSourceFileParsing(t, input)

	sym := lookupDeclSymbol(t, srcFile, "myvalue")
	decl := sym.Decl.(*ast.DeclExternValue)
	if decl.TypeHint != nil {
		t.Errorf("expected nil type hint, got %v", decl.TypeHint)
	}
}

func TestParseExternConstWithQualifiedType(t *testing.T) {
	input := `extern const value: prelude.String`
	srcFile := prepareSourceFileParsing(t, input)

	sym := lookupDeclSymbol(t, srcFile, "value")
	decl := sym.Decl.(*ast.DeclExternValue)
	if decl.TypeHint == nil {
		t.Fatal("expected type hint, got nil")
	}
	ref := decl.TypeHint.(ast.TypeExprRef)
	if len(ref.Reference) != 2 {
		t.Fatalf("expected 2-part reference, got %d", len(ref.Reference))
	}
	if ref.Reference[0].Value != "prelude" || ref.Reference[1].Value != "String" {
		t.Errorf("type: want 'prelude.String', got %q", ref.TypeExpression())
	}
}
