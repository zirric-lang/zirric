package parser_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

func TestModuleDecl(t *testing.T) {
	tests := []struct {
		input    string
		name     string
		path     []string
		hasAlias bool
		overview string
	}{
		{"mod app", "app", []string{"app"}, false, "mod app"},
		{"mod code.knabel.dev.example.flow.node", "node", []string{"code", "knabel", "dev", "example", "flow", "node"}, false, "mod code.knabel.dev.example.flow.node"},
		{"mod local = code.knabel.dev.example.flow.node", "local", []string{"code", "knabel", "dev", "example", "flow", "node"}, true, "mod local = code.knabel.dev.example.flow.node"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			srcFile := prepareSourceFileParsing(t, tt.input)

			sym, ok := srcFile.Decls.Symbols[tt.name]
			if !ok {
				t.Fatalf("module binds no %q, got %v", tt.name, srcFile.Decls.Symbols)
			}
			decl, ok := sym.Decl.(*ast.DeclModule)
			if !ok {
				t.Fatalf("symbol is %T, want *ast.DeclModule", sym.Decl)
			}

			if len(decl.Path) != len(tt.path) {
				t.Fatalf("path = %v, want %v", decl.Path, tt.path)
			}
			for i, segment := range tt.path {
				if decl.Path[i].Value != segment {
					t.Fatalf("path = %v, want %v", decl.Path, tt.path)
				}
			}
			if decl.HasAlias != tt.hasAlias {
				t.Errorf("HasAlias = %v, want %v", decl.HasAlias, tt.hasAlias)
			}
			if got := decl.DeclOverview(); got != tt.overview {
				t.Errorf("DeclOverview = %q, want %q", got, tt.overview)
			}
		})
	}
}
