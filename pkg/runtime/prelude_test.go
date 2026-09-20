package runtime

import "testing"

func TestPreludeBool(t *testing.T) {
	var p Prelude
	b := p.Bool(true)
	if b.Inspect() != "true" {
		t.Fatalf("expected Inspect true, got %s", b.Inspect())
	}
	if b.TypeConstantId() != typeIdBool {
		t.Fatalf("expected typeIdBool, got %d", b.TypeConstantId())
	}
}

func TestPreludeInspect(t *testing.T) {
	var p Prelude
	sym := makeExternFuncSymbol("_inspect", 1)
	val := p.Bind(nil, nil, sym)
	fn, ok := val.(*ExternFunc)
	if !ok {
		t.Fatalf("expected *ExternFunc, got %T", val)
	}

	tests := []struct {
		name string
		arg  RuntimeValue
		want string
	}{
		{"Int", Int(5), "5"},
		{"String", String("hi"), `"hi"`},
		{"Array", Array{Int(1), Int(2)}, "[1, 2]"},
		{"Bool", Bool(true), "true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fn.Impl(nil, []RuntimeValue{tt.arg})
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			str, ok := result.(String)
			if !ok {
				t.Fatalf("expected String result, got %T", result)
			}
			if string(str) != tt.want {
				t.Errorf("_inspect(%v): got %q, want %q", tt.arg, str, tt.want)
			}
		})
	}
}

func TestByteInspectIsHex(t *testing.T) {
	tests := []struct {
		b    Byte
		want string
	}{
		{Byte(0), "0"},
		{Byte(15), "f"},
		{Byte(255), "ff"},
	}
	for _, tt := range tests {
		if got := tt.b.Inspect(); got != tt.want {
			t.Errorf("Byte(%d).Inspect() = %q, want %q", tt.b, got, tt.want)
		}
	}
}

func TestByteTypeConstantId(t *testing.T) {
	if got := Byte(0).TypeConstantId(); got != typeIdByte {
		t.Fatalf("expected typeIdByte, got %d", got)
	}
}

func TestByteLookupHasNoMembers(t *testing.T) {
	if got := Byte(42).Lookup("anything"); got != nil {
		t.Fatalf("expected Lookup to return nil, got %v", got)
	}
}

func TestByteDistinctFromBinaryTypeId(t *testing.T) {
	if Byte(0).TypeConstantId() == Binary(nil).TypeConstantId() {
		t.Fatal("expected Byte and Binary to have distinct type constant ids")
	}
}
