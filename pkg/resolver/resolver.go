package resolver

import (
	"context"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
)

// ModuleResolver provides access to resolved modules without requiring
// dependencies on orchestration or registry internals.
type ModuleResolver interface {
	MainModule() *ast.ContextModule
	ResolveModule(ctx context.Context, name registry.LogicalURI) (*ast.ContextModule, error)
}
