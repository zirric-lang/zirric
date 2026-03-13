package cavereg

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/fsmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
	"github.com/go-git/go-billy/v5"
)

const defaultLocalVersion = "local"

// CaveRegistry exposes the modules of a Cavefile from a filesystem root.
type CaveRegistry struct {
	pkg     cavefile.Package
	version version.Version
	fs      billy.Filesystem
}

func New(cave cavefile.Cavefile, fs billy.Filesystem) (*CaveRegistry, error) {
	if fs == nil {
		return nil, errors.New("filesystem is required")
	}
	if cave.Name == "" && cave.Source == "" {
		return nil, errors.New("cavefile must have a name or source")
	}
	if cave.Name == "" {
		cave.Name = string(registry.CanonicalizeModuleSource(cave.Source))
	}
	if cave.Source == "" {
		cave.Source = cave.Name
	}
	return &CaveRegistry{
		pkg:     cave.Package,
		version: version.Parse(defaultLocalVersion),
		fs:      fs,
	}, nil
}

func (r *CaveRegistry) Discover(ctx context.Context) ([]registry.ResolvedPackage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []registry.ResolvedPackage{&cavePackage{provider: r}}, nil
}

func (r *CaveRegistry) DiscoverPackageVersions(ctx context.Context, name string, preds ...version.Predicate) ([]registry.Package, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !r.matchesName(name) {
		return nil, nil
	}
	for _, pred := range preds {
		if !r.version.Matches(pred) {
			return nil, nil
		}
	}
	return []registry.Package{&cavePackage{provider: r}}, nil
}

func (r *CaveRegistry) matchesName(name string) bool {
	return name == r.pkg.Source || name == r.pkg.Name
}

type cavePackage struct {
	provider *CaveRegistry
}

func (p *cavePackage) Source() string {
	return p.provider.pkg.Source
}

func (p *cavePackage) Version() version.Version {
	return p.provider.version
}

func (p *cavePackage) Resolve(ctx context.Context) (registry.ResolvedPackage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *cavePackage) ResolveModules() ([]registry.ResolvedModule, error) {
	baseURI := registry.LogicalURI(p.provider.pkg.Name)
	mods, err := fsmodule.DiscoverModules(baseURI, p.provider.fs)
	if err != nil {
		return nil, err
	}

	out := make([]registry.ResolvedModule, 0, len(mods)+1)
	rootSeen := false
	for _, mod := range mods {
		if mod.URI() == baseURI {
			rootSeen = true
		}
		out = append(out, mod)
	}
	if !rootSeen {
		out = append(out, fsmodule.NewModule(baseURI, p.provider.fs))
	}
	return sortModules(out), nil
}

func sortModules(mods []registry.ResolvedModule) []registry.ResolvedModule {
	slices.SortFunc(mods, func(lhs, rhs registry.ResolvedModule) int {
		return cmp.Compare(lhs.URI(), rhs.URI())
	})
	return mods
}
