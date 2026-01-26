package version_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

func TestVerbalVersionParsingUntransformed(t *testing.T) {
	cases := []string{
		"main",
		"latest",
		"stable",
		"dev",
		"master",
		"develop",
		"feature/branch",
	}
	comparisions := []struct {
		comparison version.Comparison
		want       bool
	}{
		{version.ComparisonExact, true},
		{version.ComparisonLessThan, false},
		{version.ComparisonLessThanOrEqual, true},
		{version.ComparisonGreaterThan, false},
		{version.ComparisonGreaterThanOrEqual, true},
		{version.ComparisonUpToNextMajor, true},
		{version.ComparisonUpToNextMinor, true},
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			v := version.ParseVerbal(c)
			if c != v.String() {
				t.Errorf("expected %q, got %q", c, v.String())
			}
			if !v.IsPreRelease() {
				t.Errorf("expected %q to be a pre-release", c)
			}

			for _, comp := range comparisions {
				t.Run(comp.comparison.String(), func(t *testing.T) {
					if comp.want != v.Matches(version.Predicate{
						Comparison: comp.comparison,
						Version:    v,
					}) {
						t.Errorf("expected %q to match %q", c, comp.comparison)
					}
				})
			}
		})
	}
}

func TestVerbalVersionParsingTrimmed(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{" main ", "main"},
		{" latest ", "latest"},
		{" stable ", "stable"},
		{" dev ", "dev"},
		{" master ", "master"},
		{" develop ", "develop"},
		{" feature/branch ", "feature/branch"},
		{"main", "main"},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			v := version.ParseVerbal(c.input)
			if c.want != v.String() {
				t.Errorf("expected %q, got %q", c.want, v.String())
			}
			if !v.IsPreRelease() {
				t.Errorf("expected %q to be a pre-release", c.want)
			}
		})
	}
}
