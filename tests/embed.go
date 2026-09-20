package tests

import "embed"

// Each submodule is listed rather than matched with a wildcard, so that a _t directory is never shipped as part of the standard library.
//
//go:embed *.zirr assert/*.zirr runner/*.zirr tap/*.zirr
var FS embed.FS
