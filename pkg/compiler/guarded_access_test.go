package compiler_test

import (
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/compiler"
)

// TestGuardedAccessRejections covers the two shapes of guarded access the compiler cannot give a meaning to, which are worth reporting rather than leaving to fail at runtime.
func TestGuardedAccessRejections(t *testing.T) {
	const types = `
	union Option {
		data Some {
			value
		}
		data None
	}
	data Holder {
		field
	}
	`

	tests := []struct {
		label string
		input string
		want  string
	}{
		{
			label: "!. outside a function has nothing to return from",
			input: types + `
			const value = Some(Holder(1))!.field
			`,
			want: "!. must be inside a function",
		},
		{
			label: "a call cannot be the end of an optional chain",
			input: types + `
			fn call(holder) {
				return holder?.field(1)
			}
			`,
			want: "a call cannot be part of an optional chain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			module, program := prepareSourceFileParsing(t, tt.input)
			comp := compiler.New(newTestModuleResolver(module, nil))
			err := comp.Compile(program)
			if err == nil {
				t.Fatalf("expected %q, compiled without error", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected an error containing %q, got %q", tt.want, err)
			}
		})
	}
}
