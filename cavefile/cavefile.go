package cavefile

import "code.knabel.dev/zirric-lang/zirric/version"

type Cavefile struct {
	Dependencies []Dependency
}

type Dependency struct {
	ImportName string
	Source     string
	Predicate  version.Predicate
}
