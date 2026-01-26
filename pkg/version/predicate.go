package version

import (
	"strings"
)

// Predicate constraints a Version.
//
// Examples:
//   - "main"
//   - "~1.2.3"
//   - "^1.2.3"
//   - ">=1.2.3"
//   - ">= 1.0.0"
//   - "< 1.0.0"
//   - ">= 1.0.0"
//   - ">= 1.0.0"
type Predicate struct {
	Comparison Comparison
	Version    Version
}
type Comparison int

const (
	ComparisonExact Comparison = iota
	ComparisonLessThan
	ComparisonLessThanOrEqual
	ComparisonGreaterThan
	ComparisonGreaterThanOrEqual
	ComparisonUpToNextMajor
	ComparisonUpToNextMinor
)

func (c Comparison) String() string {
	switch c {
	case ComparisonExact:
		return "=="
	case ComparisonLessThan:
		return "<"
	case ComparisonLessThanOrEqual:
		return "<="
	case ComparisonGreaterThan:
		return ">"
	case ComparisonGreaterThanOrEqual:
		return ">="
	case ComparisonUpToNextMajor:
		return "^"
	case ComparisonUpToNextMinor:
		return "~"
	}
	return ""
}

func (p Predicate) String() string {
	if _, ok := p.Version.(VerbalVersion); ok && p.Comparison == ComparisonExact {
		return p.Version.String()
	}
	return p.Comparison.String() + p.Version.String()
}

func ParsePredicate(s string) Predicate {
	prefixes := []struct {
		prefix string
		comp   Comparison
	}{
		{"==", ComparisonExact},
		{"<=", ComparisonLessThanOrEqual},
		{">=", ComparisonGreaterThanOrEqual},
		{"=", ComparisonExact},
		{"<", ComparisonLessThan},
		{">", ComparisonGreaterThan},
		{"^", ComparisonUpToNextMajor},
		{"~", ComparisonUpToNextMinor},
	}

	for _, pref := range prefixes {
		if strings.HasPrefix(s, pref.prefix) {
			v := Parse(s[len(pref.prefix):])
			return Predicate{
				Comparison: pref.comp,
				Version:    v,
			}
		}
	}
	return Predicate{
		Comparison: ComparisonExact,
		Version:    Parse(s),
	}
}
