package cavefile

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

// StandardLibrarySource is the canonical Source of the Zirric standard library package.
const StandardLibrarySource = "https://code.knabel.dev/zirric-lang/zirric"

type Cavefile struct {
	Package

	// The dependencies of the Cavefile
	Dependencies []Dependency

	// The tasks declared in the Cavefile
	Tasks []Task
}

type Dependency struct {
	Package

	// Module is the specific module this dependency binds to; set only for @cave.Stdlib.
	Module registry.LogicalURI

	// The version predicate of the dependency
	Predicates []version.Predicate
}

type Package struct {
	// Name of package, like "code.knabel.dev.zirric-lang.zirric"
	Name string
	// Source of the dependency like:
	// - https://code.knabel.dev/zirric-lang/zirric
	// - file:///path/to/local/package
	Source string
}
