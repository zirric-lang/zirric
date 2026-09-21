package codefmt

import (
	"path"
	"strings"
)

// Excludes are the path patterns @cave.FormattingExcludes declares, matched against the slash-separated path relative to the package root.
//
// "*" matches within one segment, "**" matches any number of segments including none, and a pattern naming a directory excludes everything beneath it.
type Excludes []string

// Match reports whether p, taken relative to the package root, is excluded.
func (e Excludes) Match(p string) bool {
	p = strings.TrimPrefix(path.Clean(strings.ReplaceAll(p, "\\", "/")), "./")
	if p == "." {
		return false
	}
	segments := strings.Split(p, "/")

	for _, pattern := range e {
		pattern = strings.TrimSuffix(strings.TrimPrefix(path.Clean(pattern), "./"), "/")
		if pattern == "" || pattern == "." {
			continue
		}
		if matchSegments(strings.Split(pattern, "/"), segments) {
			return true
		}
	}
	return false
}

// matchSegments treats "**" as any number of segments, and lets a fully consumed pattern match anything nested below it.
func matchSegments(pattern, segments []string) bool {
	if len(pattern) == 0 {
		return true
	}
	if pattern[0] == "**" {
		for i := 0; i <= len(segments); i++ {
			if matchSegments(pattern[1:], segments[i:]) {
				return true
			}
		}
		return false
	}
	if len(segments) == 0 {
		return false
	}
	if ok, err := path.Match(pattern[0], segments[0]); err != nil || !ok {
		return false
	}
	return matchSegments(pattern[1:], segments[1:])
}

// MatchesDir reports whether all of dir is excluded, letting a walk prune the subtree.
func (e Excludes) MatchesDir(dir string) bool {
	return e.Match(dir)
}
