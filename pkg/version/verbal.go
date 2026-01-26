package version

import "strings"

type VerbalVersion struct {
	Verbal string
}

func ParseVerbal(input string) VerbalVersion {
	return VerbalVersion{
		Verbal: strings.TrimSpace(input),
	}
}

// String implements the fmt.Stringer interface.
func (v VerbalVersion) String() string {
	return v.Verbal
}

// IsPreRelease implements the Version interface.
func (v VerbalVersion) IsPreRelease() bool {
	return true
}

// Matches implements the Version interface.
func (v VerbalVersion) Matches(cond Predicate) bool {
	switch cond.Comparison {
	case ComparisonExact,
		ComparisonLessThanOrEqual,
		ComparisonGreaterThanOrEqual,
		ComparisonUpToNextMajor,
		ComparisonUpToNextMinor:
		return v.Verbal == cond.Version.String()
	default:
		return false
	}
}
