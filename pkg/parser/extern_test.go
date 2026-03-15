package parser_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
)

func TestParseExternDeclarations(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		expectedType string
		expectedName string
	}{
		{
			name:         "extern type without fields",
			input:        "extern type Void",
			expectedType: "*ast.DeclExternType",
			expectedName: "Void",
		},
		{
			name:         "extern type with fields",
			input:        "extern type String { length }",
			expectedType: "*ast.DeclExternType",
			expectedName: "String",
		},
		{
			name:         "extern fn without parameters",
			input:        "extern fn print()",
			expectedType: "*ast.DeclExternFunc",
			expectedName: "print",
		},
		{
			name:         "extern fn with parameters",
			input:        "extern fn add(a, b)",
			expectedType: "*ast.DeclExternFunc",
			expectedName: "add",
		},
		{
			name:         "extern let value",
			input:        "extern let myvalue",
			expectedType: "*ast.DeclExternValue",
			expectedName: "myvalue",
		},
		// Note: Cannot use reserved keywords like 'void' as identifiers
		// The lexer tokenizes them as keywords, not IDENT tokens
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			module := ast.MakeContextModule(registry.LogicalURI("test"))
			parentTable := module.Decls
			l, err := lexer.New(staticmodule.NewSourceString("testing:///test.zirr", tc.input))
			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}

			p := parser.NewSourceParser(l, parentTable, "test.zirr")
			srcFile := p.ParseSourceFile()

			// Check for parser errors
			if len(p.Errors()) > 0 {
				for _, err := range p.Errors() {
					t.Errorf("Parser error: %v", err)
				}
				return
			}

			// Check that the declaration was added to the symbol table
			if srcFile.Decls == nil || srcFile.Decls.Parent == nil || srcFile.Decls.Parent.Symbols == nil {
				t.Fatal("Decl table is nil")
			}

			symbol, exists := srcFile.Decls.Parent.Symbols[tc.expectedName]
			if !exists {
				t.Fatalf("Expected symbol %q not found in symbol table", tc.expectedName)
			}

			if symbol.Decl == nil {
				t.Fatalf("Symbol %q has nil declaration", tc.expectedName)
			}

			// Check the type of the declaration
			declType := getTypeName(symbol.Decl)
			if declType != tc.expectedType {
				t.Errorf("Expected declaration type %s, got %s", tc.expectedType, declType)
			}

			// Check the declaration name
			declName := symbol.Decl.DeclName().Value
			if declName != tc.expectedName {
				t.Errorf("Expected declaration name %s, got %s", tc.expectedName, declName)
			}

			// Verify DeclOverview shows new syntax
			if overviewable, ok := symbol.Decl.(ast.Overviewable); ok {
				overview := overviewable.DeclOverview()
				t.Logf("Declaration overview: %s", overview)

				// Basic checks for the overview format
				switch tc.expectedType {
				case "*ast.DeclExternType":
					if !contains(overview, "extern type") {
						t.Errorf("Expected overview to contain 'extern type', got: %s", overview)
					}
				case "*ast.DeclExternFunc":
					if !contains(overview, "extern fn") {
						t.Errorf("Expected overview to contain 'extern fn', got: %s", overview)
					}
				case "*ast.DeclExternValue":
					if !contains(overview, "extern let") {
						t.Errorf("Expected overview to contain 'extern let', got: %s", overview)
					}
				}
			}
		})
	}
}

func TestParseExternDeclarationErrors(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "extern without keyword",
			input: "extern SomeName",
		},
		{
			name:  "extern with invalid keyword",
			input: "extern invalid SomeName",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			module := ast.MakeContextModule(registry.LogicalURI("test"))
			parentTable := module.Decls
			l, err := lexer.New(staticmodule.NewSourceString("testing:///test.zirr", tc.input))
			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}

			p := parser.NewSourceParser(l, parentTable, "test.zirr")
			p.ParseSourceFile()

			// Should have parser errors
			if len(p.Errors()) == 0 {
				t.Error("Expected parser errors but got none")
			}
		})
	}
}

// Helper functions
func getTypeName(v interface{}) string {
	switch v.(type) {
	case *ast.DeclExternType:
		return "*ast.DeclExternType"
	case *ast.DeclExternFunc:
		return "*ast.DeclExternFunc"
	case *ast.DeclExternValue:
		return "*ast.DeclExternValue"
	default:
		return "unknown"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && findInString(s, substr)
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
