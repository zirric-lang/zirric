package codefmt

import "testing"

func TestFromLSP(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]any
		want Options
	}{
		{"empty keeps defaults", map[string]any{}, DefaultOptions()},
		{
			"insertSpaces switches to spaces",
			map[string]any{"insertSpaces": true, "tabSize": float64(2)},
			Options{UseTabs: false, TabWidth: 2, MaxBlankLines: 1, InsertFinalNewline: true},
		},
		{
			"json numbers arrive as float64",
			map[string]any{"tabSize": float64(8)},
			Options{UseTabs: true, TabWidth: 8, MaxBlankLines: 1, InsertFinalNewline: true},
		},
		{"int tabSize also works", map[string]any{"tabSize": 3}, Options{UseTabs: true, TabWidth: 3, MaxBlankLines: 1, InsertFinalNewline: true}},
		{"wrong types are ignored", map[string]any{"tabSize": "four", "insertSpaces": "yes"}, DefaultOptions()},
		{"unknown keys are ignored", map[string]any{"somethingElse": true}, DefaultOptions()},
		{"nonpositive tabSize is ignored", map[string]any{"tabSize": float64(0)}, DefaultOptions()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultOptions().FromLSP(tt.in); got != tt.want {
				t.Errorf("want %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestIndentUnit(t *testing.T) {
	if got := (Options{UseTabs: true}).indentUnit(); got != "\t" {
		t.Errorf("want tab, got %q", got)
	}
	if got := (Options{UseTabs: false, TabWidth: 2}).indentUnit(); got != "  " {
		t.Errorf("want two spaces, got %q", got)
	}
	if got := (Options{UseTabs: false}).indentUnit(); got != "    " {
		t.Errorf("zero TabWidth should fall back to the default, got %q", got)
	}
}

func TestSpacesIndentOutput(t *testing.T) {
	opts := DefaultOptions().FromLSP(map[string]any{"insertSpaces": true, "tabSize": float64(2)})
	got, err := String("t.zirr", "fn f() {\nconst a = 1\n}\n", opts)
	if err != nil {
		t.Fatal(err)
	}
	want := "fn f() {\n  const a = 1\n}\n"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}
