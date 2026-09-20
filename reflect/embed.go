package reflect

import "embed"

// Only the module's own sources and its packages submodule: a wildcard would also ship _t, whose test modules would then shadow a project's own.
//
//go:embed *.zirr packages/*.zirr
var FS embed.FS
