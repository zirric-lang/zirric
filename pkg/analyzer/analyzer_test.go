package analyzer_test

import (
	"context"
	"fmt"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
)

func TestAnalyzerResolvesIdentifierFreeSymbols(t *testing.T) {
	module := parseModule(t, "test", `
		module test
		const x = 1
		fn example() {
			return x
		}
	`)
	a := analyzer.New(nil)
	if errs, _ := a.Analyze(module, false); len(errs) > 0 {
		t.Fatalf("analysis errors: %v", errs)
	}

	fn := findDeclFunc(module, "example")
	if fn == nil || fn.Impl == nil || fn.Impl.Symbols == nil {
		t.Fatal("expected function symbols to be resolved")
	}
	if len(fn.Impl.Symbols.FreeSymbols) != 1 || fn.Impl.Symbols.FreeSymbols[0].Name != "x" {
		t.Fatalf("expected one free symbol for x, got %+v", fn.Impl.Symbols.FreeSymbols)
	}

	id := findExprIdentifier(module, "x")
	if id == nil || id.Symbol == nil {
		t.Fatal("expected identifier x to be resolved to a symbol")
	}
	if id.Symbol.Scope != ast.FreeScope {
		t.Fatalf("expected identifier x to resolve to free symbol, got %v", id.Symbol.Scope)
	}
	if id.Symbol.Parent == nil || id.Symbol.Parent.Decl == nil {
		t.Fatal("expected free symbol to reference global declaration")
	}
	if decl, ok := id.Symbol.Parent.Decl.(*ast.DeclConstant); !ok || decl.Name.Value != "x" {
		t.Fatalf("expected free symbol parent to be global x, got %T", id.Symbol.Parent.Decl)
	}
}

func TestAnalyzerResolvesAttributeReferences(t *testing.T) {
	module := parseModule(t, "test", `
		module test
		attr Type { value }
		data String { value }
		@Type(String)
		data Example { name }
	`)
	a := analyzer.New(nil)
	if errs, _ := a.Analyze(module, false); len(errs) > 0 {
		t.Fatalf("analysis errors: %v", errs)
	}

	sym := module.Symbols.Symbols["Type"]
	if sym == nil || sym.Decl == nil {
		t.Fatal("expected Type attribute to be declared")
	}
	if _, ok := sym.Decl.(*ast.DeclAttr); !ok {
		t.Fatalf("expected Type to be attribute, got %T", sym.Decl)
	}
	if len(sym.Usages) == 0 {
		t.Fatal("expected attribute Type to be referenced at least once")
	}

	id := findExprIdentifier(module, "String")
	if id == nil || id.Symbol == nil {
		t.Fatal("expected attribute argument String to resolve")
	}
	if decl, ok := id.Symbol.Decl.(*ast.DeclData); !ok || decl.Name.Value != "String" {
		t.Fatalf("expected String identifier to resolve to data String, got %T", id.Symbol.Decl)
	}
}

func TestAnalyzerValidatesImports(t *testing.T) {
	module := parseModule(t, "test", `
		module test
		import missing
	`)
	a := analyzer.New(stubResolver{})
	errs, _ := a.Analyze(module, false)
	if len(errs) == 0 {
		t.Fatal("expected an analysis error for missing module")
	}
	if errs[0].Summary != "unknown module" {
		t.Fatalf("expected unknown module error, got %q", errs[0].Summary)
	}
}

func TestAnalyzerValidatesUnionMemberStaticRefs(t *testing.T) {
	module := parseModule(t, "test", `
		module test
		union Optional { Missing }
	`)
	a := analyzer.New(nil)
	errs, _ := a.Analyze(module, false)
	if len(errs) == 0 {
		t.Fatal("expected an analysis error for missing union member")
	}
	if errs[0].Summary != "unknown reference" {
		t.Fatalf("expected unknown reference error, got %q", errs[0].Summary)
	}
}

func TestAnalyzerResolvesImportedModulePrefixes(t *testing.T) {
	imported := parseModule(t, "foo.bar", `
		module bar
		data Value { field }
	`)
	module := parseModule(t, "test", `
		module test
		import foo.bar
		union Example { foo.bar.Value }
	`)
	a := analyzer.New(mapResolver{modules: map[registry.LogicalURI]*ast.ContextModule{
		imported.Name: imported,
	}})
	errs, _ := a.Analyze(module, false)
	if len(errs) > 0 {
		t.Fatalf("unexpected analysis errors: %v", errs)
	}
}

func TestAnalyzerRejectsDeepImportedRefs(t *testing.T) {
	imported := parseModule(t, "foo.bar", `
		module bar
		data Value { field }
	`)
	module := parseModule(t, "test", `
		module test
		import foo.bar
		union Example { foo.bar.Value.field }
	`)
	a := analyzer.New(mapResolver{modules: map[registry.LogicalURI]*ast.ContextModule{
		imported.Name: imported,
	}})
	errs, _ := a.Analyze(module, false)
	if len(errs) == 0 {
		t.Fatal("expected analysis errors for deep imported reference")
	}
	if errs[0].Summary != "unknown reference" {
		t.Fatalf("expected unknown reference error, got %q", errs[0].Summary)
	}
}

func parseModule(t *testing.T, uri string, input string) *ast.ContextModule {
	t.Helper()

	moduleURI := registry.LogicalURI(uri)
	src := staticmodule.NewSourceString(moduleURI.Join("main.zirr"), input)
	mod := staticmodule.NewModule(moduleURI, []registry.Source{src})
	mp := parser.NewModuleParse(mod)
	ctxMod, err := mp.Parse(mod)
	if err != nil {
		t.Fatalf("parse module: %v", err)
	}
	if len(mp.Errors()) > 0 {
		t.Fatalf("parse errors: %v", mp.Errors())
	}
	return ctxMod
}

func findExprIdentifier(module *ast.ContextModule, name string) *ast.ExprIdentifier {
	var found *ast.ExprIdentifier
	walkNode(module, func(child ast.Node) {
		if found != nil {
			return
		}
		if id, ok := child.(*ast.ExprIdentifier); ok && id.Name.Value == name {
			found = id
		}
	})
	return found
}

func findDeclFunc(module *ast.ContextModule, name string) *ast.DeclFunc {
	var found *ast.DeclFunc
	walkNode(module, func(child ast.Node) {
		if found != nil {
			return
		}
		if fn, ok := child.(*ast.DeclFunc); ok && fn.Name.Value == name {
			found = fn
		}
	})
	return found
}

func walkNode(node ast.Node, visit func(ast.Node)) {
	if node == nil {
		return
	}
	visit(node)
	node.EnumerateChildNodes(func(child ast.Node) {
		walkNode(child, visit)
	})
}

type stubResolver struct{}

func (stubResolver) MainModule() *ast.ContextModule {
	return nil
}

func (stubResolver) ResolveModule(_ context.Context, _ registry.LogicalURI) (*ast.ContextModule, error) {
	return nil, fmt.Errorf("missing module")
}

type mapResolver struct {
	modules map[registry.LogicalURI]*ast.ContextModule
}

func (mapResolver) MainModule() *ast.ContextModule {
	return nil
}

func (mr mapResolver) ResolveModule(_ context.Context, name registry.LogicalURI) (*ast.ContextModule, error) {
	if mod, ok := mr.modules[name]; ok {
		return mod, nil
	}
	return nil, fmt.Errorf("missing module %s", name)
}
