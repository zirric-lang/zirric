package gitreg

import (
	"context"

	"github.com/go-git/go-git/v5/plumbing"
	"code.knabel.dev/zirric-lang/zirric/registry"
	"code.knabel.dev/zirric-lang/zirric/version"
)

type remoteGitPackage struct {
	provider *GitRegistry

	source       string
	gitReference *plumbing.Reference
	version      version.Version
}

// Source implements registry.Package
func (p *remoteGitPackage) Source() string {
	return p.source
}

// Version implements registry.Package
func (p *remoteGitPackage) Version() version.Version {
	return p.version
}

// Resolve implements registry.Package
func (p *remoteGitPackage) Resolve(ctx context.Context) (registry.ResolvedPackage, error) {
	return p.provider.clone(ctx, p)
}
