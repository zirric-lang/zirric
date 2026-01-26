package parser_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

func TestImportDecl(t *testing.T) {
	tests := []struct {
		input   string
		alias   string
		module  []string
		members []string
	}{
		{"import alias = foo.bar { one, two }", "alias", []string{"foo", "bar"}, []string{"one", "two"}},
		{"import alias = foo.bar", "alias", []string{"foo", "bar"}, nil},
		{"import foo.bar { one }", "bar", []string{"foo", "bar"}, []string{"one"}},
		{"import foo.bar", "bar", []string{"foo", "bar"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)

			sym, ok := srcFile.Decls.Symbols[tt.alias]
			if !ok {
				t.Fatalf("alias %q symbol not found", tt.alias)
			}

			decl, ok := sym.Decl.(*ast.DeclImport)
			if !ok {
				t.Fatalf("symbol is %T, want *ast.DeclImport", sym.Decl)
			}

			if len(decl.ModuleName) != len(tt.module) {
				t.Fatalf("unexpected module name length: %v", decl.ModuleName)
			}
			for i, m := range tt.module {
				if decl.ModuleName[i].Value != m {
					t.Fatalf("unexpected module name: %v", decl.ModuleName)
				}
			}

			if len(decl.Members) != len(tt.members) {
				t.Fatalf("unexpected members: %v", decl.Members)
			}
			for i, m := range tt.members {
				if decl.Members[i].DeclName().Value != m {
					t.Fatalf("unexpected members: %v", decl.Members)
				}
			}

			if decl.Alias.Value != tt.alias {
				t.Fatalf("unexpected alias %q", decl.Alias.Value)
			}
		})
	}
}
