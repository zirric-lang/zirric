package localreg

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/fsmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
	"github.com/go-git/go-billy/v5/osfs"
)

const sourceScheme = "file://"

const localVersion = "local"

// LocalRegistry resolves file:// dependency sources directly from the real filesystem; the target doesn't need to be a Git repository.
type LocalRegistry struct{}

func New() *LocalRegistry {
	return &LocalRegistry{}
}

func (r *LocalRegistry) Discover(ctx context.Context) ([]registry.ResolvedPackage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *LocalRegistry) DiscoverPackageVersions(ctx context.Context, name string, preds ...version.Predicate) ([]registry.Package, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, ok := strings.CutPrefix(name, sourceScheme)
	if !ok {
		return nil, nil
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return nil, nil
	}
	ver := version.Parse(localVersion)
	if !version.MatchesAll(ver, preds...) {
		return nil, nil
	}
	return []registry.Package{&localPackage{source: name, path: path, version: ver}}, nil
}

type localPackage struct {
	source  string
	path    string
	version version.Version
}

func (p *localPackage) Source() string {
	return p.source
}

func (p *localPackage) Version() version.Version {
	return p.version
}

func (p *localPackage) Resolve(ctx context.Context) (registry.ResolvedPackage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *localPackage) ResolveModules() ([]registry.ResolvedModule, error) {
	baseURI := registry.LogicalURI(registry.CanonicalizeModulePath(filepath.Base(p.path)))
	fsmods, err := fsmodule.DiscoverModules(baseURI, osfs.New(p.path))
	if err != nil {
		return nil, err
	}
	mods := make([]registry.ResolvedModule, len(fsmods))
	for i, m := range fsmods {
		mods[i] = m
	}
	return mods, nil
}
