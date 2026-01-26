package gitreg

import (
	"context"

	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/fsmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
	"github.com/go-git/go-billy/v5"
)

type localGitPackage struct {
	*remoteGitPackage
	fs billy.Filesystem
}

// Source implements registry.Package
func (p *localGitPackage) Source() string {
	return p.source
}

// Version implements registry.Package
func (p *localGitPackage) Version() version.Version {
	return p.version
}

// Resolve implements registry.Package
func (p *localGitPackage) Resolve(ctx context.Context) (registry.ResolvedPackage, error) {
	return p, nil
}

// ResolveModules implements registry.ResolvedPackage
func (p *localGitPackage) ResolveModules() ([]registry.ResolvedModule, error) {
	baseURI := registry.CanonicalizeModuleSource(p.source)
	fsmods, err := fsmodule.DiscoverModules(baseURI, p.fs)
	if err != nil {
		return nil, err
	}

	mods := make([]registry.ResolvedModule, len(fsmods))
	for i, m := range fsmods {
		mods[i] = m
	}
	return mods, nil
}
