package version_test

import (
	"sort"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

func TestConditionParsing(t *testing.T) {
	semvers := []string{
		"1.2.3",
		"1.2.3-alpha",
		"1.2.3-alpha.1",
		"1.2.3-0.3.7",
		"1.2.3-x.7.z.92",
		"v1.2.3",
	}
	verbals := []string{
		"main",
		"latest",
		"stable",
		"dev",
		"master",
		"develop",
		"feature/branch",
		"v1",
		"v1.2",
		"v1.2.x",
	}

	for _, text := range semvers {
		t.Run("semver: "+text, func(t *testing.T) {
			v := version.Parse(text)
			_, ok := v.(version.SemverVersion)
			if !ok {
				t.Errorf("expected %q to be a semver", text)
			}
		})
	}

	for _, text := range verbals {
		t.Run("verbal: "+text, func(t *testing.T) {
			v := version.Parse(text)
			_, ok := v.(version.VerbalVersion)
			if !ok {
				t.Errorf("expected %q to be a verbal", text)
			}
		})
	}
}

func TestVersionsLess(t *testing.T) {
	wanted := []string{
		"1.2.3-0.3.7",
		"1.2.3-alpha",
		"1.2.3",
		"v1.2.3",
		"dev",
		"develop",
		"feature/branch",
		"latest",
		"main",
		"master",
		"stable",
		"v1",
		"v1.2",
	}

	isSorted := sort.SliceIsSorted(wanted, func(i, j int) bool {
		lhs := version.Parse(wanted[i])
		rhs := version.Parse(wanted[j])
		return version.Less(lhs, rhs)
	})
	if !isSorted {
		got := make([]string, len(wanted))
		copy(got, wanted)
		sort.Slice(got, func(i, j int) bool {
			lhs := version.Parse(got[i])
			rhs := version.Parse(got[j])
			return version.Less(lhs, rhs)
		})
		t.Errorf("expected versions to be sorted, want %v, got %v", wanted, got)
	}
}
