package cavefile

import "code.knabel.dev/zirric-lang/zirric/pkg/version"

type Cavefile struct {
	Package

	// The dependencies of the Cavefile
	Dependencies []Dependency
}

type Dependency struct {
	Package

	// The version predicate of the dependency
	Predicate version.Predicate
}

type Package struct {
	// Name of package, like "code.knabel.dev.zirric-lang.zirric"
	Name string
	// Source of the dependency like:
	// - https://code.knabel.dev/zirric-lang/zirric
	// - file:///path/to/local/package
	Source string
}
