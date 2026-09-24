package cavefile

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

// StandardLibrarySource is the canonical Source of the Zirric standard library package.
const StandardLibrarySource = "https://code.knabel.dev/zirric-lang/zirric"

type Cavefile struct {
	Package

	// Docs is the comment written above the Cavefile's own `mod` declaration.
	Docs string

	Dependencies []Dependency

	Tasks []Task

	// Path patterns from @cave.FormattingExcludes, relative to the package root.
	FormattingExcludes []string
}

type Dependency struct {
	Package

	// Docs is the comment written above the field that declares the dependency.
	Docs string

	// Module is the specific module this dependency binds to; set only for @cave.Stdlib.
	Module registry.LogicalURI

	Predicates []version.Predicate
}

type Package struct {
	// Name of package, like "code.knabel.dev.zirric-lang.zirric"
	Name string
	// Source of the dependency like:
	// - https://code.knabel.dev/zirric-lang/zirric
	// - file:///path/to/local/package
	Source string

	// ModulePath is the fully qualified module path the package's own `mod` declares, and the base every one of its modules is named under. It is empty when nothing declares one, as for a project without a Cavefile.
	ModulePath string

	// Version is the package's own version, from @cave.Version on its @cave.Package() declaration. A dependency constrains which versions it accepts through Predicates instead.
	Version string

	// Description from @cave.Description.
	Description string

	// Documentation is the URL of the package's documentation, from @cave.Documentation.
	Documentation string

	// LanguageVersion constrains the Zirric toolchain itself rather than the package, from @cave.LanguageVersion.
	LanguageVersion string
}
