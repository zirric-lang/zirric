package resolver

import (
	"context"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
)

// ModuleResolver provides access to resolved modules without requiring
// dependencies on orchestration or registry internals.
type ModuleResolver interface {
	MainModule() *ast.ContextModule
	ResolveModule(ctx context.Context, name registry.LogicalURI) (*ast.ContextModule, error)
}

// CavefileProvider is an optional ModuleResolver capability for reading the manifest the project declared itself with.
// It is optional so the stub resolvers used in tests need not implement it; reflect reports no Cavefile for a resolver that does not.
type CavefileProvider interface {
	MainPackageCavefile() cavefile.Cavefile
}

// MainPackageLister is an optional ModuleResolver capability for enumerating every module the project's own package declares, including ones no import reaches.
// It is optional so the stub resolvers used in tests need not implement it; reflect reports an empty package for resolvers that do not.
type MainPackageLister interface {
	MainPackageName() string
	MainPackageModules(ctx context.Context) ([]registry.LogicalURI, error)
}

// DeclaredPackageBase is an optional ModuleResolver capability reporting the module path the project's Cavefile declares with its own `mod`.
// It is empty for a loose script or the REPL, where a file's `mod` has no base to be held to.
type DeclaredPackageBase interface {
	DeclaredPackageBase() registry.LogicalURI
}
