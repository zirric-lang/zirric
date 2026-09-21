package vm_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/compiler"
	"code.knabel.dev/zirric-lang/zirric/pkg/vm"
)

// A binding declared inside a block at module scope lives in the top-level frame's locals.
// The main module's instructions are inlined into that frame, so its local slots have to be carried across with them — otherwise MainLocals stays 0 and the SetLocal reaches past an empty locals slice at runtime.
func TestTopLevelBlockLocals(t *testing.T) {
	tests := []struct {
		label string
		input string
	}{
		{"const in if", "if true {\n\tconst r = 1\n}\n"},
		{"var in if", "if true {\n\tvar r = 1\n}\n"},
		{"const in switch case", "switch 1 {\ncase 1:\n\tconst r = 2\ncase _:\n}\n"},
		{"const in for", "for {\n\tconst r = 1\n\tbreak\n}\n"},
		{"nested blocks", "if true {\n\tconst a = 1\n\tif true {\n\t\tconst b = 2\n\t}\n}\n"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			module := prepareContextModuleParsing(t, "module.test", tt.input)
			resolver := newTestModuleResolver(module)

			comp := compiler.New(resolver)
			if err := comp.Compile(module); err != nil {
				t.Fatalf("compiler error: %s", err)
			}

			bytecode := comp.Bytecode()
			if bytecode.MainLocals < 1 {
				t.Fatalf("MainLocals = %d, want at least 1 for a block-scoped binding", bytecode.MainLocals)
			}

			if err := vm.New(bytecode).Run(); err != nil {
				t.Fatalf("vm error: %s", err)
			}
		})
	}
}
