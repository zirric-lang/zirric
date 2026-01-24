package version

import (
	"regexp"
	"strconv"
	"strings"
)

type SemverVersion struct {
	Major                 int
	Minor                 int
	Patch                 int
	PreReleaseIdentifiers []string
	BuildIdentifiers      []string
}

var semverRegex = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

func ParseSemver(input string) (SemverVersion, error) {
	captures := semverRegex.FindStringSubmatch(input)
	if captures == nil || len(captures) != 6 {
		return SemverVersion{}, ErrInvalidVersion
	}
	majorStr, minorStr, patchStr := captures[1], captures[2], captures[3]
	majorU, err := strconv.Atoi(majorStr)
	if err != nil {
		return SemverVersion{}, err
	}
	minorU, err := strconv.Atoi(minorStr)
	if err != nil {
		return SemverVersion{}, err
	}
	patchU, err := strconv.Atoi(patchStr)
	if err != nil {
		return SemverVersion{}, err
	}
	preReleaseIdentifierStr := captures[4]
	buildIdentifierStr := captures[5]
	var preReleaseIdentifiers []string
	if preReleaseIdentifierStr != "" {
		preReleaseIdentifiers = strings.Split(preReleaseIdentifierStr, ".")
	}
	var buildIdentifiers []string
	if buildIdentifierStr != "" {
		buildIdentifiers = strings.Split(buildIdentifierStr, ".")
	}
	return SemverVersion{
		Major:                 int(majorU),
		Minor:                 int(minorU),
		Patch:                 int(patchU),
		PreReleaseIdentifiers: preReleaseIdentifiers,
		BuildIdentifiers:      buildIdentifiers,
	}, nil
}

// String implements the fmt.Stringer interface.
func (v SemverVersion) String() string {
	var preReleaseIdentifierStr string
	if len(v.PreReleaseIdentifiers) > 0 {
		preReleaseIdentifierStr = "-" + strings.Join(v.PreReleaseIdentifiers, ".")
	}
	var buildIdentifierStr string
	if len(v.BuildIdentifiers) > 0 {
		buildIdentifierStr = "+" + strings.Join(v.BuildIdentifiers, ".")
	}
	return strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor) + "." + strconv.Itoa(v.Patch) + preReleaseIdentifierStr + buildIdentifierStr
}

// IsPreRelease implements the Version interface.
func (v SemverVersion) IsPreRelease() bool {
	if len(v.PreReleaseIdentifiers) > 0 {
		return true
	}
	if v.Major == 0 && v.Minor == 0 && v.Patch == 0 {
		return true
	}
	return false
}

// Matches implements the Version interface.
func (v SemverVersion) Matches(cond Predicate) bool {
	var ref SemverVersion
	if cond.Version == nil {
		ref = SemverVersion{}
	} else if rv, ok := cond.Version.(SemverVersion); ok {
		ref = rv
	} else {
		return false
	}

	switch cond.Comparison {
	case ComparisonExact:
		return v.String() == ref.String()
	case ComparisonLessThan:
		return v.Compare(ref) < 0
	case ComparisonLessThanOrEqual:
		return v.Compare(ref) <= 0
	case ComparisonGreaterThan:
		return v.Compare(ref) > 0
	case ComparisonGreaterThanOrEqual:
		return v.Compare(ref) >= 0
	case ComparisonUpToNextMajor:
		if v.Matches(Predicate{Comparison: ComparisonLessThan, Version: ref}) {
			return false
		}
		if ref.Major == 0 {
			return v.Major == 0 && v.Minor == ref.Minor
		}
		return v.Major == ref.Major && ref.Minor <= v.Minor
	case ComparisonUpToNextMinor:
		if v.Matches(Predicate{Comparison: ComparisonLessThan, Version: ref}) {
			return false
		}
		if ref.Major == 0 {
			return v.Major == 0 && v.Minor == ref.Minor && v.Patch == ref.Patch
		}
		return v.Major == ref.Major && v.Minor == ref.Minor && ref.Patch <= v.Patch
	}
	return false
}

func (v SemverVersion) Compare(rhs SemverVersion) int {
	if v.Major > rhs.Major {
		return 1
	} else if v.Major < rhs.Major {
		return -1
	}

	if v.Minor > rhs.Minor {
		return 1
	} else if v.Minor < rhs.Minor {
		return -1
	}

	if v.Patch > rhs.Patch {
		return 1
	} else if v.Patch < rhs.Patch {
		return -1
	}

	if len(v.PreReleaseIdentifiers) > 0 && len(rhs.PreReleaseIdentifiers) == 0 {
		return -1
	} else if len(v.PreReleaseIdentifiers) == 0 && len(rhs.PreReleaseIdentifiers) > 0 {
		return 1
	} else if len(v.PreReleaseIdentifiers) == 0 && len(rhs.PreReleaseIdentifiers) == 0 {
		return 0
	}

	for i := 0; i < len(v.PreReleaseIdentifiers) && i < len(rhs.PreReleaseIdentifiers); i++ {
		if v.PreReleaseIdentifiers[i] == rhs.PreReleaseIdentifiers[i] {
			continue
		}
		lhsInt, err := strconv.Atoi(v.PreReleaseIdentifiers[i])
		if err == nil {
			rhsInt, err := strconv.Atoi(rhs.PreReleaseIdentifiers[i])
			if err == nil {
				if lhsInt > rhsInt {
					return 1
				} else if lhsInt < rhsInt {
					return -1
				}
			}
		}
		if v.PreReleaseIdentifiers[i] > rhs.PreReleaseIdentifiers[i] {
			return 1
		} else if v.PreReleaseIdentifiers[i] < rhs.PreReleaseIdentifiers[i] {
			return -1
		}
	}
	if len(v.PreReleaseIdentifiers) > len(rhs.PreReleaseIdentifiers) {
		return 1
	} else if len(v.PreReleaseIdentifiers) < len(rhs.PreReleaseIdentifiers) {
		return -1
	}
	return 0
}
