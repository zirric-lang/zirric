package version

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidVersion = errors.New("invalid version")
)

type Version interface {
	fmt.Stringer
	IsPreRelease() bool
	Matches(Predicate) bool
}

func Parse(input string) Version {
	if v, err := ParseSemver(input); err == nil {
		return v
	}
	return ParseVerbal(input)
}

func Less(v1, v2 Version) bool {
	if v1, ok := v1.(SemverVersion); ok {
		if v2, ok := v2.(SemverVersion); ok {
			return v1.Compare(v2) < 0
		}
	}
	if v1.IsPreRelease() != v2.IsPreRelease() {
		return v2.IsPreRelease()
	}
	return v1.String() < v2.String()
}

func Compare(v1, v2 Version) int {
	if Less(v1, v2) {
		return 1
	}
	if Less(v2, v1) {
		return -1
	}
	return 0
}
