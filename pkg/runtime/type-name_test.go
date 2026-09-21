package runtime_test

import (
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
)

func TestTypeNameSpellsTypesTheWayTheLanguageDoes(t *testing.T) {
	// Every message that names what it was given goes through here, so a Go type name leaking out would leak everywhere.
	for _, tt := range []struct {
		value runtime.RuntimeValue
		want  string
	}{
		{runtime.Int(1), "Int"},
		{runtime.Float(1.5), "Float"},
		{runtime.String("a"), "String"},
		{runtime.Bool(true), "Bool"},
		{runtime.Char('a'), "Char"},
		{runtime.Byte(1), "Byte"},
		{runtime.Binary{1, 2}, "Binary"},
		{runtime.Void{}, "Void"},
		{runtime.Array{}, "Array"},
		{runtime.Dict{}, "Dict"},
		{runtime.Duration(1), "Duration"},
		{runtime.Instant(1), "Instant"},
		{runtime.Timestamp(1), "Timestamp"},
	} {
		if got := runtime.TypeName(tt.value); got != tt.want {
			t.Errorf("expected %q, got %q", tt.want, got)
		}
	}
}

func TestTypeNameOfADataValueIsItsOwnName(t *testing.T) {
	value := &runtime.DataValue{TypeName: "Person"}
	if got := runtime.TypeName(value); got != "Person" {
		t.Errorf("expected %q, got %q", "Person", got)
	}
}

func TestTypeNameNeverLeaksAGoType(t *testing.T) {
	for _, value := range []runtime.RuntimeValue{
		runtime.Int(1), runtime.String("a"), runtime.Array{}, runtime.Dict{}, runtime.Void{},
		&runtime.DataValue{TypeName: "Person"}, runtime.MakeModuleValue("m", nil),
	} {
		if name := runtime.TypeName(value); strings.Contains(name, "runtime.") {
			t.Errorf("%v is named with a Go type: %q", value, name)
		}
	}
	// Nothing at all still has to be describable.
	if got := runtime.TypeName(nil); got != "nothing" {
		t.Errorf("expected %q, got %q", "nothing", got)
	}
}
