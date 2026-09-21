package parser_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

// A return value must share the line with its `return`: a newline or a comment makes the return bare.
func TestParseReturnValueRequiresSameLine(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantValue bool
		wantStmts int
	}{
		{"value on same line", "return 1", true, 1},
		{"negative value on same line", "return -1", true, 1},
		{"bare return before brace", "return", false, 1},
		{"bare return then newline value", "return\n\t1", false, 2},
		{"bare return then newline negative", "return\n\t-1", false, 2},
		{"bare return then newline call", "return\n\tf()", false, 2},
		{"comment ends the return", "return // done", false, 1},
		{"comment then value on next line", "return // done\n\t1", false, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, "fn sample() {\n\t"+tt.body+"\n}")

			decl := lookupFuncDecl(t, srcFile, "sample")
			if len(decl.Impl.Impl) != tt.wantStmts {
				t.Fatalf("want %d statements in body, got %d", tt.wantStmts, len(decl.Impl.Impl))
			}
			ret, ok := decl.Impl.Impl[0].(*ast.StmtReturn)
			if !ok {
				t.Fatalf("first statement is %T, want *ast.StmtReturn", decl.Impl.Impl[0])
			}
			if got := ret.Expr != nil; got != tt.wantValue {
				t.Errorf("return has value = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

func lookupFuncDecl(t *testing.T, srcFile *ast.SourceFile, name string) *ast.DeclFunc {
	t.Helper()

	sym, ok := srcFile.Decls.Resolve(name)
	if !ok {
		t.Fatalf("no declaration named %q", name)
	}
	decl, ok := sym.Decl.(*ast.DeclFunc)
	if !ok {
		t.Fatalf("declaration %q is %T, want *ast.DeclFunc", name, sym.Decl)
	}
	return decl
}
