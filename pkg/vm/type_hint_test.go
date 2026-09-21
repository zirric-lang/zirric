package vm_test

import (
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/compiler"
)

// TestNonAttributeTypeRejectedOnDataField verifies that using a non-attribute
// type (like Bool) as @Bool() on a data field produces a compile error.
func TestNonAttributeTypeRejectedOnDataField(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name: "Bool on field",
			input: `
				extern type Bool {}
				attr Marker {}
				@Marker()
				data Foo {
					@Bool()
					flag
				}
			`,
			wantErr: `not an attribute: Bool is a`,
		},
		{
			name: "String on field",
			input: `
				extern type String {}
				data Foo {
					@String()
					name
				}
			`,
			wantErr: `not an attribute: String is a`,
		},
		{
			name: "data type on field",
			input: `
				data Bar {}
				data Foo {
					@Bar()
					field
				}
			`,
			wantErr: `not an attribute: Bar is a`,
		},
		{
			name: "union type on field",
			input: `
				data A {}
				data B {}
				union AB { A B }
				data Foo {
					@AB()
					field
				}
			`,
			wantErr: `not an attribute: AB is a`,
		},
		{
			name: "non-attr on constant",
			input: `
				extern type Bool {}
				@Bool()
				const flag = 0 == 0
			`,
			wantErr: `not an attribute: Bool is a`,
		},
		{
			name: "non-attr on variable",
			input: `
				extern type Int {}
				fn example() {
					@Int()
					var x = 42
				}
			`,
			wantErr: `not an attribute: Int is a`,
		},
		{
			name: "non-attr on extern type field",
			input: `
				extern type Int {}
				extern type Array {
					@Int()
					length
				}
			`,
			wantErr: `not an attribute: Int is a`,
		},
		{
			name: "non-attr on attr field",
			input: `
				extern type String {}
				attr Doc {
					@String()
					description
				}
			`,
			wantErr: `not an attribute: String is a`,
		},
		{
			name: "valid attr on field accepted",
			input: `
				attr Marker {}
				@Marker()
				data Foo {
					@Marker()
					field
				}
			`,
			wantErr: "", // should compile fine
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mainModule, program := prepareSourceFileParsing(t, tt.input)
			resolver := newTestModuleResolver(mainModule)

			comp := compiler.New(resolver)
			err := comp.Compile(program)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected compile error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

// TestFieldTypeHintParsesCorrectly verifies that `: Type` syntax
// on fields, parameters, constants and variables parses and compiles.
func TestFieldTypeHintParsesCorrectly(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "field with type hint",
			input: `
				extern type String {}
				data Greeter {
					greeting: String
				}
				Greeter("hello").greeting
			`,
		},
		{
			name: "parameter with type hint",
			input: `
				extern type String {}
				fn greet(name: String) {
					return name
				}
				greet("world")
			`,
		},
		{
			name: "const with type hint",
			input: `
				extern type String {}
				const name: String = "hello"
				name
			`,
		},
		{
			name: "var with type hint",
			input: `
				extern type Int {}
				fn example() {
					var x: Int = 42
					return x
				}
				example()
			`,
		},
		{
			name: "field method with return type",
			input: `
				extern type String {}
				data Greeter {
					greet(name: String) -> String
				}
				Greeter(fn(n) { return "Hi " + n }).greet("world")
			`,
		},
		{
			name: "function with return type",
			input: `
				extern type Int {}
				fn add(a: Int, b: Int) -> Int {
					return a + b
				}
				add(1, 2)
			`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mainModule, program := prepareSourceFileParsing(t, tt.input)
			resolver := newTestModuleResolver(mainModule)

			comp := compiler.New(resolver)
			if err := comp.Compile(program); err != nil {
				t.Fatalf("compile error: %v", err)
			}
		})
	}
}

// TestFieldTypeHintWithAttribute verifies that fields can have
// both attributes and type hints simultaneously.
func TestFieldTypeHintWithAttribute(t *testing.T) {
	input := `
		attr Default {
			value
		}
		extern type String {}
		data Config {
			@Default("hello") name: String
		}
		Config("world").name
	`

	mainModule, program := prepareSourceFileParsing(t, input)
	resolver := newTestModuleResolver(mainModule)

	comp := compiler.New(resolver)
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compile error: %v", err)
	}
}
