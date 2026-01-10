package version_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/version"
)

func TestComparisonString(t *testing.T) {
	cases := []struct {
		comparison version.Comparison
		want       string
	}{
		{version.ComparisonExact, "=="},
		{version.ComparisonLessThan, "<"},
		{version.ComparisonLessThanOrEqual, "<="},
		{version.ComparisonGreaterThan, ">"},
		{version.ComparisonGreaterThanOrEqual, ">="},
		{version.ComparisonUpToNextMajor, "^"},
		{version.ComparisonUpToNextMinor, "~"},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			if c.want != c.comparison.String() {
				t.Errorf("expected %q, got %q", c.want, c.comparison.String())
			}
		})
	}
}

func TestPredicateString(t *testing.T) {
	cases := []struct {
		predicate version.Predicate
		want      string
	}{
		{
			version.Predicate{
				Comparison: version.ComparisonExact,
				Version:    version.Parse("v1.2.3"),
			},
			"==1.2.3",
		},
		{
			version.Predicate{
				Comparison: version.ComparisonExact,
				Version:    version.Parse("main"),
			},
			"main",
		},
		{
			version.Predicate{
				Comparison: version.ComparisonLessThan,
				Version:    version.Parse("v1.2.3"),
			},
			"<1.2.3",
		},
		{
			version.Predicate{
				Comparison: version.ComparisonLessThanOrEqual,
				Version:    version.Parse("v1.2.3"),
			},
			"<=1.2.3",
		},
		{
			version.Predicate{
				Comparison: version.ComparisonGreaterThan,
				Version:    version.Parse("v1.2.3"),
			},
			">1.2.3",
		},
		{
			version.Predicate{
				Comparison: version.ComparisonGreaterThanOrEqual,
				Version:    version.Parse("v1.2.3"),
			},
			">=1.2.3",
		},
		{
			version.Predicate{
				Comparison: version.ComparisonUpToNextMajor,
				Version:    version.Parse("v1.2.3"),
			},
			"^1.2.3",
		},
		{
			version.Predicate{
				Comparison: version.ComparisonUpToNextMinor,
				Version:    version.Parse("v1.2.3"),
			},
			"~1.2.3",
		},
	}

	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			if c.want != c.predicate.String() {
				t.Errorf("expected %q, got %q", c.want, c.predicate.String())
			}
		})
	}
}

func TestPredicate(t *testing.T) {
	cases := []struct {
		raw  string
		want version.Predicate
	}{
		{
			"main",
			version.Predicate{
				Comparison: version.ComparisonExact,
				Version:    version.ParseVerbal("main"),
			},
		},
		{
			"v1.2.3",
			version.Predicate{
				Comparison: version.ComparisonExact,
				Version:    must(version.ParseSemver("1.2.3")),
			},
		},
		{
			"=v1.2.3",
			version.Predicate{
				Comparison: version.ComparisonExact,
				Version:    must(version.ParseSemver("1.2.3")),
			},
		},
		{
			"==v1.2.3",
			version.Predicate{
				Comparison: version.ComparisonExact,
				Version:    must(version.ParseSemver("1.2.3")),
			},
		},
		{
			"<v1.2.3",
			version.Predicate{
				Comparison: version.ComparisonLessThan,
				Version:    must(version.ParseSemver("1.2.3")),
			},
		},
		{
			"<=v1.2.3",
			version.Predicate{
				Comparison: version.ComparisonLessThanOrEqual,
				Version:    must(version.ParseSemver("1.2.3")),
			},
		},
		{
			">v1.2.3",
			version.Predicate{
				Comparison: version.ComparisonGreaterThan,
				Version:    must(version.ParseSemver("1.2.3")),
			},
		},
		{
			">=v1.2.3",
			version.Predicate{
				Comparison: version.ComparisonGreaterThanOrEqual,
				Version:    must(version.ParseSemver("1.2.3")),
			},
		},
		{
			"^v1.2.3",
			version.Predicate{
				Comparison: version.ComparisonUpToNextMajor,
				Version:    must(version.ParseSemver("1.2.3")),
			},
		},
		{
			"~v1.2.3",
			version.Predicate{
				Comparison: version.ComparisonUpToNextMinor,
				Version:    must(version.ParseSemver("1.2.3")),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.raw, func(t *testing.T) {
			got := version.ParsePredicate(c.raw)
			if c.want.Comparison != got.Comparison {
				t.Errorf("expected same comparision %q, got %q", c.want.Comparison, got.Comparison)
			}
			if c.want.Version.String() != got.Version.String() {
				t.Errorf("expected same version %q, got %q", c.want.Version, got.Version)
			}
		})
	}
}

func must[V any](value V, err error) V {
	if err != nil {
		panic(err)
	}
	return value
}
