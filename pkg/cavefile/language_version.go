package cavefile

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

// LanguageVersionError reports that the running Zirric does not satisfy a @cave.LanguageVersion.
type LanguageVersionError struct {
	Package   string
	Predicate string
	Toolchain version.Version
	// Dependency changes nothing but how the refusal reads.
	Dependency bool
}

func (e *LanguageVersionError) Error() string {
	subject := "package"
	if e.Dependency {
		subject = "dependency"
	}
	return fmt.Sprintf("%s %q needs Zirric %s, but this is %s", subject, e.Package, e.Predicate, e.Toolchain)
}

// CheckLanguageVersion holds the running Zirric against what pkg declares with @cave.LanguageVersion.
//
// A nil toolchain passes everything: an unstamped build cannot be told apart from one that is too old, and refusing it would leave no way to develop Zirric itself.
func CheckLanguageVersion(pkg Package, toolchain version.Version) error {
	return checkLanguageVersion(pkg, toolchain, false)
}

// CheckDependencyLanguageVersion is CheckLanguageVersion for a dependency, differing only in how the refusal reads.
func CheckDependencyLanguageVersion(dep Dependency, toolchain version.Version) error {
	return checkLanguageVersion(dep.Package, toolchain, true)
}

func checkLanguageVersion(pkg Package, toolchain version.Version, dependency bool) error {
	if pkg.LanguageVersion == "" || toolchain == nil {
		return nil
	}
	if toolchain.Matches(version.ParsePredicate(pkg.LanguageVersion)) {
		return nil
	}
	return &LanguageVersionError{
		Package:    pkg.Name,
		Predicate:  pkg.LanguageVersion,
		Toolchain:  toolchain,
		Dependency: dependency,
	}
}
