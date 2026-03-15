package ast_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

func makeIdentifier(name string) ast.Identifier {
	return ast.Identifier{
		Token: token.Token{Type: token.IDENT, Literal: name},
		Value: name,
	}
}

func TestSymbolTableInsertCreatesSymbol(t *testing.T) {
	module := ast.MakeContextModule(registry.LogicalURI("test"))
	table := ast.MakeModuleSymbolTable(module)

	decl := &ast.DeclVariable{
		Token: token.Token{Type: token.VAR, Literal: "var"},
		Name:  makeIdentifier("answer"),
	}

	sym := table.Insert(decl)
	if sym == nil { // nolint:staticcheck
		t.Fatalf("expected symbol, got nil")
	}

	if sym.Name != "answer" { // nolint:staticcheck
		t.Fatalf("expected symbol name to be %q, got %q", "answer", sym.Name)
	}
	if sym.Index != 0 { // nolint:staticcheck
		t.Fatalf("expected first symbol to have index 0, got %d", sym.Index)
	}
	gotDecl, ok := sym.Decl.(*ast.DeclVariable) // nolint:staticcheck
	if !ok || gotDecl != decl {
		t.Fatalf("expected symbol to reference original declaration")
	}

	ident := makeIdentifier("answer")
	ref := table.Lookup("answer", ident)
	if ref != sym {
		t.Fatalf("lookup should return the same symbol instance")
	}
	if len(sym.Usages) != 1 {
		t.Fatalf("expected lookup to record exactly one usage, got %d", len(sym.Usages))
	}
	usage, ok := sym.Usages[0].Node.(ast.Identifier)
	if !ok {
		t.Fatalf("expected usage node to be an Identifier, got %T", sym.Usages[0].Node)
	}
	if !cmp.Equal(usage, ident) {
		t.Fatalf("usage identifier mismatch (-want +got):\n%s", cmp.Diff(ident, usage))
	}
}

func TestSymbolTableInsertReportsRedeclaration(t *testing.T) {
	module := ast.MakeContextModule(registry.LogicalURI("test"))
	table := ast.MakeModuleSymbolTable(module)

	decl := &ast.DeclVariable{
		Token: token.Token{Type: token.VAR, Literal: "var"},
		Name:  makeIdentifier("value"),
	}

	first := table.Insert(decl)
	second := table.Insert(decl)
	if first != second {
		t.Fatalf("expected redeclaration to return the existing symbol")
	}

	if len(first.Errs) != 1 {
		t.Fatalf("expected one error on redeclaration, got %d", len(first.Errs))
	}
	if len(first.Usages) != 1 {
		t.Fatalf("expected redeclaration to record a usage, got %d", len(first.Usages))
	}
	usageDecl, ok := first.Usages[0].Node.(*ast.DeclVariable)
	if !ok || usageDecl != decl {
		t.Fatalf("expected usage node to be the redeclared declaration")
	}
}

func TestLookupCreatesPlaceholderForUndefinedSymbol(t *testing.T) {
	module := ast.MakeContextModule(registry.LogicalURI("test"))
	table := ast.MakeModuleSymbolTable(module)

	ident := makeIdentifier("unknown")
	sym := table.Lookup("unknown", ident)
	if sym == nil { // nolint:staticcheck
		t.Fatalf("expected placeholder symbol, got nil")
	}
	if sym.Decl != nil { // nolint:staticcheck

		t.Fatalf("expected placeholder symbol to have no declaration")
	}
	if len(sym.Usages) != 1 { // nolint:staticcheck
		t.Fatalf("expected placeholder to record a usage, got %d", len(sym.Usages))
	}
	recorded, ok := sym.Usages[0].Node.(ast.Identifier)
	if !ok {
		t.Fatalf("expected usage node to be an Identifier, got %T", sym.Usages[0].Node)
	}
	if !cmp.Equal(recorded, ident) {
		t.Fatalf("usage identifier mismatch (-want +got):\n%s", cmp.Diff(ident, recorded))
	}
}

func TestChildSymbolTableCreatesFreeSymbol(t *testing.T) {
	module := ast.MakeContextModule(registry.LogicalURI("test"))
	parent := ast.MakeModuleSymbolTable(module)
	child := parent.MakeChild(nil)

	decl := &ast.DeclVariable{
		Token: token.Token{Type: token.VAR, Literal: "var"},
		Name:  makeIdentifier("capture"),
	}

	original := parent.Insert(decl)
	resolved := child.Lookup("capture", makeIdentifier("capture"))
	if resolved == nil { // nolint:staticcheck
		t.Fatalf("expected resolved symbol, got nil")
	}

	if resolved == original {
		t.Fatalf("expected child lookup to create a free symbol")
	}
	if resolved.Scope != ast.FreeScope { // nolint:staticcheck
		t.Fatalf("expected free symbol scope, got %q", resolved.Scope)
	}
	if resolved.Parent != original {
		t.Fatalf("expected free symbol to reference original symbol")
	}
	if len(child.FreeSymbols) != 1 || child.FreeSymbols[0] != original {
		t.Fatalf("expected original symbol to be tracked as free")
	}
}

func TestNextAnonymousFunctionName(t *testing.T) {
	module := ast.MakeContextModule(registry.LogicalURI("test"))
	table := ast.MakeModuleSymbolTable(module)

	if name := table.NextAnonymousFunctionName(); name != "fn#1" {
		t.Fatalf("expected first anonymous function name to be fn#1, got %s", name)
	}
	if name := table.NextAnonymousFunctionName(); name != "fn#2" {
		t.Fatalf("expected second anonymous function name to be fn#2, got %s", name)
	}
}
