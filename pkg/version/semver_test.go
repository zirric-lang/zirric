package version_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

func TestSemverParsingSuccess(t *testing.T) {
	table := []struct {
		input    string
		expected *version.SemverVersion
	}{
		{"v1.2.3", &version.SemverVersion{
			Major: 1,
			Minor: 2,
			Patch: 3,
		}},
		{"1.2.3", &version.SemverVersion{
			Major: 1,
			Minor: 2,
			Patch: 3,
		}},
		{"1.2.3-alpha", &version.SemverVersion{
			Major:                 1,
			Minor:                 2,
			Patch:                 3,
			PreReleaseIdentifiers: []string{"alpha"},
		}},
		{"1.2.3-alpha.1", &version.SemverVersion{
			Major:                 1,
			Minor:                 2,
			Patch:                 3,
			PreReleaseIdentifiers: []string{"alpha", "1"},
		}},
		{"1.2.3-0.3.7", &version.SemverVersion{
			Major:                 1,
			Minor:                 2,
			Patch:                 3,
			PreReleaseIdentifiers: []string{"0", "3", "7"},
		}},
		{"1.2.3-x.7.z.92", &version.SemverVersion{
			Major:                 1,
			Minor:                 2,
			Patch:                 3,
			PreReleaseIdentifiers: []string{"x", "7", "z", "92"},
		}},
		{"1.2.3+20130313144700", &version.SemverVersion{
			Major:            1,
			Minor:            2,
			Patch:            3,
			BuildIdentifiers: []string{"20130313144700"},
		}},
		{"1.2.3-beta+exp.sha.5114f85", &version.SemverVersion{
			Major:                 1,
			Minor:                 2,
			Patch:                 3,
			PreReleaseIdentifiers: []string{"beta"},
			BuildIdentifiers:      []string{"exp", "sha", "5114f85"},
		}},
		{"0.0.0", &version.SemverVersion{
			Major: 0,
			Minor: 0,
			Patch: 0,
		}},
	}

	for _, test := range table {
		t.Run(test.input, func(t *testing.T) {
			actual, err := version.ParseSemver(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if actual.Major != test.expected.Major {
				t.Errorf("expected major %d, got %d", test.expected.Major, actual.Major)
			}
			if actual.Minor != test.expected.Minor {
				t.Errorf("expected minor %d, got %d", test.expected.Minor, actual.Minor)
			}
			if actual.Patch != test.expected.Patch {
				t.Errorf("expected patch %d, got %d", test.expected.Patch, actual.Patch)
			}
			if len(actual.PreReleaseIdentifiers) != len(test.expected.PreReleaseIdentifiers) {
				t.Errorf("expected pre-release %v, got %v", test.expected.PreReleaseIdentifiers, actual.PreReleaseIdentifiers)
			}
			if len(actual.BuildIdentifiers) != len(test.expected.BuildIdentifiers) {
				t.Errorf("expected build %v, got %v", test.expected.BuildIdentifiers, actual.BuildIdentifiers)
			}
		})
	}
}

func TestSemverParsingFailure(t *testing.T) {
	table := []string{
		"",
		"1",
		"1.2",
		"1.2.3-",
		"1.2.3+",
		"1.2.3-+",
		"1.2.3-+1",
		"1.2.3-+1.2",
		"1.2.3-+1.2.3",
		"1.2.3-+",
		"1.2.3-+1",
		"1.2.3-+1.2",
		"1.2.3-+1.2.3",
		"..",
		"...",
		"a.b.c",
		"0.a.b",
		"0.0.a",
		"0.0.0-",
	}

	for _, test := range table {
		t.Run(test, func(t *testing.T) {
			_, err := version.ParseSemver(test)
			if err != version.ErrInvalidVersion {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestSemverCompareSorting(t *testing.T) {
	sortedStrs := []string{
		"0.0.0",
		"0.0.1-alpha",
		"0.0.1",
		"1.0.0-alpha",
		"1.0.0-alpha.1",
		"1.0.0-alpha.beta",
		"1.0.0-beta",
		"1.0.0-beta.2",
		"1.0.0-beta.11",
		"1.0.0-rc.1",
		"1.0.0",
		"1.0.1-alpha.1",
		"1.0.1",
		"1.1.0",
	}
	sortedVersions := make([]version.SemverVersion, len(sortedStrs))
	for i, str := range sortedStrs {
		v, err := version.ParseSemver(str)
		if err != nil {
			t.Fatalf("cannot parse version %q: %v", str, err)
		}
		sortedVersions[i] = v
	}
	unsortedVersions := make([]version.SemverVersion, len(sortedVersions))
	copy(unsortedVersions, sortedVersions)
	rand.Shuffle(len(unsortedVersions), func(i, j int) {
		unsortedVersions[i], unsortedVersions[j] = unsortedVersions[j], unsortedVersions[i]
	})
	sort.Slice(unsortedVersions, func(i, j int) bool {
		return unsortedVersions[i].Compare(unsortedVersions[j]) < 0
	})
	if !reflect.DeepEqual(sortedVersions, unsortedVersions) {
		t.Errorf("expected %v, got %v", sortedVersions, unsortedVersions)
	}
}

func TestSemverCompareCases(t *testing.T) {
	table := []struct {
		name     string
		lhs, rhs string
		want     int
	}{
		{"=", "1.2.3", "1.2.3", 0},
		{"= with build", "1.2.3+build", "1.2.3+build", 0},
		{"= with pre-release", "1.2.3-alpha", "1.2.3-alpha", 0},
		{"= with pre-release and build", "1.2.3-alpha+build", "1.2.3-alpha+build", 0},
		{"= ignores build", "1.2.3+build", "1.2.3", 0},
		{"pre-release < release", "1.2.3-alpha", "1.2.3", -1},
		{"pre-release < release with build", "1.2.3-alpha", "1.2.3+build", -1},
		{"pre-release a < pre-release b", "1.2.3-alpha", "1.2.3-beta", -1},
		{"pre-release a < pre-release b with build", "1.2.3-alpha", "1.2.3-beta+build", -1},
		{"pre-release a with build < pre-release b with build", "1.2.3-alpha+build", "1.2.3-beta+build", -1},
		{"major < major", "1.2.3", "2.0.0", -1},
		{"major < major with build", "1.2.3", "2.0.0+build", -1},
		{"minor < minor", "1.2.3", "1.3.0", -1},
		{"minor < minor with build", "1.2.3", "1.3.0+build", -1},
		{"patch < patch", "1.2.3", "1.2.4", -1},
		{"patch < patch with build", "1.2.3", "1.2.4+build", -1},
		{"major < major with pre-release", "1.2.3", "2.0.0-alpha", -1},
		{"major < major with pre-release and build", "1.2.3", "2.0.0-alpha+build", -1},
		{"minor < minor with pre-release", "1.2.3", "1.3.0-alpha", -1},
		{"minor < minor with pre-release and build", "1.2.3", "1.3.0-alpha+build", -1},
		{"patch < patch with pre-release", "1.2.3", "1.2.4-alpha", -1},
		{"patch < patch with pre-release and build", "1.2.3", "1.2.4-alpha+build", -1},
		{"major < major with pre-release b", "1.2.3-alpha", "2.0.0", -1},
		{"major < major with pre-release b and build", "1.2.3-alpha", "2.0.0+build", -1},
		{"minor < minor with pre-release b", "1.2.3-alpha", "1.3.0", -1},
		{"minor < minor with pre-release b and build", "1.2.3-alpha", "1.3.0+build", -1},
		{"patch < patch with pre-release b", "1.2.3-alpha", "1.2.4", -1},
		{"pre-release numeric < pre-release numeric", "1.2.3-alpha.1", "1.2.3-alpha.2", -1},
		{"pre-release numeric < pre-release numeric multiple digits", "1.2.3-alpha.5", "1.2.3-alpha.10", -1},
	}

	for _, test := range table {
		t.Run(fmt.Sprintf("%s (%s,%s)", test.name, test.lhs, test.rhs), func(t *testing.T) {
			lhs, err := version.ParseSemver(test.lhs)
			if err != nil {
				t.Fatalf("cannot parse left version %q: %v", test.lhs, err)
			}
			rhs, err := version.ParseSemver(test.rhs)
			if err != nil {
				t.Fatalf("cannot parse right version %q: %v", test.rhs, err)
			}

			got := lhs.Compare(rhs)
			if got != test.want {
				t.Errorf("expected %d, got %d", test.want, got)
			}

			got = rhs.Compare(lhs)
			if got != -test.want {
				t.Errorf("not associative: expected %d, got %d", -test.want, got)
			}
		})
	}
}

func TestSemverCompareEmpty(t *testing.T) {
	var emptyVersion version.SemverVersion
	if got := emptyVersion.Compare(emptyVersion); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}

	if got := emptyVersion.Compare(version.SemverVersion{}); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}

	if got := (&version.SemverVersion{}).Compare(emptyVersion); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}

	if got := (&version.SemverVersion{}).Compare(version.SemverVersion{}); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}

	if got := emptyVersion.Compare(version.SemverVersion{Major: 1}); got != -1 {
		t.Errorf("expected -1, got %d", got)
	}

	if got := (&version.SemverVersion{Major: 1}).Compare(emptyVersion); got != 1 {
		t.Errorf("expected 1, got %d", got)
	}
}

func TestSemverString(t *testing.T) {
	table := []string{
		"v1.2.3",
		"1.2.3",
		"1.2.3+build",
		"1.2.3-alpha",
		"1.2.3-alpha+build",
		"1.2.3-alpha.1",
		"1.2.3-alpha.1+build",
		"1.2.3-alpha.10",
		"1.2.3-alpha.10+build",
		"1.2.3-beta.2",
		"1.2.3-beta.2+build",
		"1.2.3-beta.5",
		"1.2.3-beta.5+build",
		"1.2.3-beta.9",
	}

	for _, str := range table {
		t.Run(str, func(t *testing.T) {
			v, err := version.ParseSemver(str)
			if err != nil {
				t.Fatalf("cannot parse version %q: %v", str, err)
			}
			if str[0] == 'v' {
				str = str[1:]
			}
			if v.String() != str {
				t.Errorf("expected %q, got %q", str, v.String())
			}
		})
	}
}

func TestSemverStringEmpty(t *testing.T) {
	var emptyVersion version.SemverVersion
	if emptyVersion.String() != "0.0.0" {
		t.Errorf("expected empty string, got %q", emptyVersion.String())
	}
}

func TestIsPrerelease(t *testing.T) {
	table := []struct {
		version string
		want    bool
	}{
		{"1.2.3", false},
		{"1.2.3+build", false},
		{"1.2.3-alpha", true},
		{"1.2.3-alpha+build", true},
		{"1.2.3-alpha.1", true},
		{"1.2.3-alpha.1+build", true},
		{"1.2.3-alpha.10", true},
		{"1.2.3-alpha.10+build", true},
		{"1.2.3-beta.2", true},
		{"1.2.3-beta.2+build", true},
		{"1.2.3-beta.5", true},
		{"1.2.3-beta.5+build", true},
		{"1.2.3-beta.9", true},
		{"0.0.0", true},
		{"0.0.0+build", true},
		{"0.0.0-alpha", true},
		{"0.0.1", false},
		{"0.1.0", false},
		{"1.0.0", false},
	}

	for _, test := range table {
		t.Run(test.version, func(t *testing.T) {
			v, err := version.ParseSemver(test.version)
			if err != nil {
				t.Fatalf("cannot parse version %q: %v", test.version, err)
			}
			if got := v.IsPreRelease(); got != test.want {
				t.Errorf("expected %t, got %t", test.want, got)
			}
		})
	}

	t.Run("empty", func(t *testing.T) {
		var emptyVersion version.SemverVersion
		if !emptyVersion.IsPreRelease() {
			t.Errorf("expected true, got false")
		}
	})
}

func TestSemverMatches(t *testing.T) {
	type testcase struct {
		version string
		want    bool
	}
	table := []struct {
		comparision version.Comparison
		reference   string
		test        []testcase
	}{
		{version.ComparisonExact, "1.2.3", []testcase{
			{"1.2.3", true},
			{"1.2.4", false},
			{"1.2.3+build", false},
			{"1.2.3-alpha", false},
			{"1.2.3-alpha+build", false},
			{"1.2.3-alpha.1", false},
			{"1.2.3-alpha.1+build", false},
		}},
		{version.ComparisonUpToNextMajor, "1.2.3", []testcase{
			{"1.2.3", true},
			{"1.2.4", true},
			{"1.3.0", true},
			{"2.0.0", false},
			{"1.2.3+build", true},
			{"1.2.3-alpha", false},
			{"1.2.3-alpha+build", false},
			{"1.2.3-alpha.1", false},
			{"1.2.3-alpha.1+build", false},
			{"1.2.4+build", true},
			{"1.2.4-alpha", true},
			{"1.2.4-alpha+build", true},
			{"1.2.4-alpha.1", true},
			{"1.2.4-alpha.1+build", true},
		}},
		{version.ComparisonUpToNextMinor, "1.2.3", []testcase{
			{"1.2.3", true},
			{"1.2.4", true},
			{"1.3.0", false},
			{"2.0.0", false},
			{"1.2.3+build", true},
			{"1.2.3-alpha", false},
			{"1.2.3-alpha+build", false},
			{"1.2.3-alpha.1", false},
			{"1.2.3-alpha.1+build", false},
			{"1.2.4+build", true},
			{"1.2.4-alpha", true},
			{"1.2.4-alpha+build", true},
			{"1.2.4-alpha.1", true},
			{"1.2.4-alpha.1+build", true},
		}},
		{version.ComparisonUpToNextMajor, "0.2.3", []testcase{
			{"0.2.3", true},
			{"0.2.4", true},
			{"0.3.0", false},
			{"1.0.0", false},
			{"0.2.3+build", true},
			{"0.2.3-alpha", false},
			{"0.2.3-alpha+build", false},
		}},
		{version.ComparisonUpToNextMinor, "0.2.3-alpha.1", []testcase{
			{"0.2.3", true},
			{"0.2.4", false},
			{"0.3.0", false},
			{"1.0.0", false},
			{"0.2.3+build", true},
			{"0.2.3-alpha", false},
			{"0.2.3-alpha.2", true},
			{"0.2.3-beta.1", true},
			{"0.2.3-rc.1", true},
		}},
		{version.ComparisonGreaterThan, "1.2.3", []testcase{
			{"1.2.3", false},
			{"1.2.4", true},
			{"1.3.0", true},
			{"2.0.0", true},
			{"1.2.3+build", false},
			{"1.2.3-alpha", false},
			{"1.2.3-alpha+build", false},
			{"1.2.3-alpha.1", false},
		}},
		{version.ComparisonGreaterThanOrEqual, "1.2.3", []testcase{
			{"1.2.3", true},
			{"1.2.4", true},
			{"1.3.0", true},
			{"2.0.0", true},
			{"1.2.3+build", true},
			{"1.2.3-alpha", false},
			{"1.2.3-alpha+build", false},
			{"1.2.3-alpha.1", false},
		}},
		{version.ComparisonLessThan, "1.2.3", []testcase{
			{"1.2.3", false},
			{"1.2.2", true},
			{"1.1.0", true},
			{"0.0.0", true},
			{"1.2.3+build", false},
			{"1.2.3-alpha", true},
			{"1.2.3-alpha+build", true},
			{"1.2.3-alpha.1", true},
		}},
		{version.ComparisonLessThanOrEqual, "1.2.3", []testcase{
			{"1.2.3", true},
			{"1.2.2", true},
			{"1.1.0", true},
			{"0.0.0", true},
			{"1.2.3+build", true},
			{"1.2.3-alpha", true},
			{"1.2.3-alpha+build", true},
			{"1.2.3-alpha.1", true},
		}},
	}

	for _, test := range table {
		t.Run(fmt.Sprintf("%s %s", test.comparision, test.reference), func(t *testing.T) {
			ref, err := version.ParseSemver(test.reference)
			if err != nil {
				t.Fatalf("cannot parse reference version %q: %v", test.reference, err)
			}

			cond := version.Predicate{
				Comparison: test.comparision,
				Version:    ref,
			}

			for _, test := range test.test {
				t.Run(test.version, func(t *testing.T) {
					v, err := version.ParseSemver(test.version)
					if err != nil {
						t.Fatalf("cannot parse test version %q: %v", test.version, err)
					}
					if v.Matches(cond) != test.want {
						t.Errorf("expected %v, got %v", test.want, !test.want)
					}
				})
			}
		})
	}
}
