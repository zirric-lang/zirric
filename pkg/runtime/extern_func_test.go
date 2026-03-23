package runtime

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

func TestMakeNativeFunc(t *testing.T) {
	t.Run("creates callable with correct arity", func(t *testing.T) {
		fn := MakeNativeFunc("write", 1, func(args []RuntimeValue) (RuntimeValue, error) {
			return Int(len(args)), nil
		})
		if fn.Arity() != 1 {
			t.Errorf("arity: got %d, want 1", fn.Arity())
		}
	})

	t.Run("inspect includes name and arity", func(t *testing.T) {
		fn := MakeNativeFunc("read", 2, func(args []RuntimeValue) (RuntimeValue, error) {
			return Void{}, nil
		})
		expected := "extern read(#2)"
		if fn.Inspect() != expected {
			t.Errorf("Inspect: got %q, want %q", fn.Inspect(), expected)
		}
	})

	t.Run("impl is invokable", func(t *testing.T) {
		fn := MakeNativeFunc("add", 2, func(args []RuntimeValue) (RuntimeValue, error) {
			a := args[0].(Int)
			b := args[1].(Int)
			return a + b, nil
		})
		result, err := fn.Impl([]RuntimeValue{Int(3), Int(7)})
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if result != Int(10) {
			t.Errorf("result: got %v, want 10", result)
		}
	})

	t.Run("type constant id is Func", func(t *testing.T) {
		fn := MakeNativeFunc("test", 0, func(args []RuntimeValue) (RuntimeValue, error) {
			return Void{}, nil
		})
		if fn.TypeConstantId() != BuiltinTypeIds["Func"] {
			t.Errorf("TypeConstantId: got %d, want %d", fn.TypeConstantId(), BuiltinTypeIds["Func"])
		}
	})

	t.Run("has no symbol", func(t *testing.T) {
		fn := MakeNativeFunc("test", 0, func(args []RuntimeValue) (RuntimeValue, error) {
			return Void{}, nil
		})
		if fn.symbol != nil {
			t.Error("expected nil symbol for native func")
		}
	})
}

func TestMakeExternFunc(t *testing.T) {
	t.Run("creates from symbol with correct arity", func(t *testing.T) {
		sym := makeExternFuncSymbol("greet", 1)
		fn := MakeExternFunc(sym, func(args []RuntimeValue) (RuntimeValue, error) {
			return String("hello"), nil
		})
		if fn.Arity() != 1 {
			t.Errorf("arity: got %d, want 1", fn.Arity())
		}
		if fn.Inspect() != "extern greet(#1)" {
			t.Errorf("Inspect: got %q, want %q", fn.Inspect(), "extern greet(#1)")
		}
	})

	t.Run("panics for non-extern decl", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for non-extern decl")
			}
		}()
		sym := &ast.Symbol{
			Name: "bad",
			Decl: ast.MakeDeclData(token.Token{}, ast.MakeIdentifier(token.Token{Literal: "bad"})),
		}
		MakeExternFunc(sym, func(args []RuntimeValue) (RuntimeValue, error) {
			return Void{}, nil
		})
	})
}

func makeExternFuncSymbol(name string, paramCount int) *ast.Symbol {
	params := make([]ast.DeclParameter, paramCount)
	for i := range params {
		params[i] = ast.DeclParameter{
			Name: ast.MakeIdentifier(token.Token{Literal: "p"}),
		}
	}
	decl := &ast.DeclExternFunc{
		Token:      token.Token{},
		Name:       ast.MakeIdentifier(token.Token{Literal: name}),
		Parameters: params,
	}
	return &ast.Symbol{
		Name: name,
		Decl: decl,
	}
}
