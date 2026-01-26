package registry

import (
	"path/filepath"
	"strings"
)

// CanonicalizeModuleSource maps a module source to its canonical URI.
func CanonicalizeModuleSource(source string) LogicalURI {
	canonical := strings.ToLower(source)
	canonical = strings.TrimPrefix(canonical, "https://")
	canonical = strings.TrimRight(canonical, "/")
	canonical = strings.TrimSuffix(canonical, ".git")
	canonical = strings.TrimRight(canonical, "/")
	canonical = strings.ReplaceAll(canonical, "/", ".")
	canonical = strings.ReplaceAll(canonical, "-", "_")
	return LogicalURI(canonical)
}

func CanonicalizeModulePath(modulePath string) string {
	canonical := strings.TrimSpace(modulePath)
	if canonical == "" || canonical == "." {
		return ""
	}
	canonical = filepath.ToSlash(canonical)
	canonical = strings.TrimPrefix(canonical, "./")
	canonical = strings.Trim(canonical, "/")
	if canonical == "" || canonical == "." {
		return ""
	}
	canonical = strings.ToLower(canonical)
	canonical = strings.ReplaceAll(canonical, "-", "_")
	canonical = strings.ReplaceAll(canonical, "/", ".")
	return canonical
}

// JoinModuleURI appends a module path to a canonical module URI.
func JoinModuleURI(base LogicalURI, modulePath string) LogicalURI {
	canonical := CanonicalizeModulePath(modulePath)
	if canonical == "" {
		return base
	}
	if base == "" {
		return LogicalURI(canonical)
	}
	return base + "." + LogicalURI(canonical)
}
