package codefmt

import "testing"

func TestExcludesMatch(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		path     string
		want     bool
	}{
		{"empty excludes nothing", nil, "a.zirr", false},
		{"exact file", []string{"a.zirr"}, "a.zirr", true},
		{"exact file elsewhere", []string{"a.zirr"}, "sub/a.zirr", false},
		{"star within a segment", []string{"*.zirr"}, "a.zirr", true},
		{"star does not cross separators", []string{"*.zirr"}, "sub/a.zirr", false},
		{"doublestar crosses separators", []string{"**/*.zirr"}, "a/b/c.zirr", true},
		{"doublestar matches zero segments", []string{"**/*.zirr"}, "a.zirr", true},
		{"directory subtree", []string{"vendor/**"}, "vendor/pkg/a.zirr", true},
		{"directory subtree root file", []string{"vendor/**"}, "vendor/a.zirr", true},
		{"bare directory excludes subtree", []string{"vendor"}, "vendor/pkg/a.zirr", true},
		{"directory does not match a sibling", []string{"vendor"}, "vendored/a.zirr", false},
		{"suffix at any depth", []string{"**/*.generated.zirr"}, "src/deep/x.generated.zirr", true},
		{"suffix does not match other files", []string{"**/*.generated.zirr"}, "src/deep/x.zirr", false},
		{"middle doublestar", []string{"src/**/gen/*.zirr"}, "src/a/b/gen/x.zirr", true},
		{"middle doublestar zero segments", []string{"src/**/gen/*.zirr"}, "src/gen/x.zirr", true},
		{"Cavefile by name", []string{"**/Cavefile"}, "examples/project/Cavefile", true},
		{"leading ./ is ignored", []string{"./vendor/**"}, "vendor/a.zirr", true},
		{"path leading ./ is ignored", []string{"vendor/**"}, "./vendor/a.zirr", true},
		{"trailing slash is ignored", []string{"vendor/"}, "vendor/a.zirr", true},
		{"one of several patterns", []string{"a/**", "b/**"}, "b/x.zirr", true},
		{"no pattern matches", []string{"a/**", "b/**"}, "c/x.zirr", false},
		{"windows separators are normalized", []string{"vendor/**"}, `vendor\a.zirr`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Excludes(tt.patterns).Match(tt.path); got != tt.want {
				t.Errorf("Excludes(%q).Match(%q) = %v, want %v", tt.patterns, tt.path, got, tt.want)
			}
		})
	}
}

// A directory that is wholly excluded can be pruned during a walk.
func TestExcludesMatchesDir(t *testing.T) {
	e := Excludes{"vendor/**", "**/gen"}
	for _, dir := range []string{"vendor", "src/gen"} {
		if !e.MatchesDir(dir) {
			t.Errorf("want %q pruned", dir)
		}
	}
	for _, dir := range []string{"src", "vendored"} {
		if e.MatchesDir(dir) {
			t.Errorf("want %q kept", dir)
		}
	}
}

// A malformed pattern must not match everything or panic.
func TestExcludesIgnoresBadPattern(t *testing.T) {
	if (Excludes{"["}).Match("a.zirr") {
		t.Error("a malformed pattern should not match")
	}
	if (Excludes{""}).Match("a.zirr") {
		t.Error("an empty pattern should not match")
	}
}
