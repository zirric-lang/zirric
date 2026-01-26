package pkgmanager

import (
	"context"
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/gitreg"
	"github.com/go-git/go-billy/v5"
)

type PackageManager struct {
	registries []registry.Provider
}

type Option func(*PackageManager, billy.Filesystem) error

func New(fs billy.Filesystem, opts ...Option) (*PackageManager, error) {
	pm := &PackageManager{
		registries: []registry.Provider{},
	}
	for _, opt := range opts {
		if err := opt(pm, fs); err != nil {
			return nil, err
		}
	}
	return pm, nil
}

func (pm *PackageManager) Install(cf cavefile.Cavefile) *InstallationTask {
	return &InstallationTask{
		pkgmanager: pm,
		cave:       cf,
	}
}

func (pm *PackageManager) ResolveModule(ctx context.Context, cave cavefile.Cavefile, uri registry.LogicalURI) (registry.ResolvedModule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	task := pm.Install(cave)
	pkgs, err := task.Run(ctx)
	if err != nil {
		return nil, err
	}
	for _, pkg := range pkgs {
		mods, err := pkg.ResolveModules()
		if err != nil {
			return nil, err
		}
		for _, mod := range mods {
			if mod.URI() == uri {
				return mod, nil
			}
		}
	}
	return nil, fmt.Errorf("module %q not found", uri)
}

func WithRegistry(registry registry.Provider) Option {
	return func(pm *PackageManager, fs billy.Filesystem) error {
		pm.registries = append(pm.registries, registry)
		return nil
	}
}

func WithGitRegistry() Option {
	return func(pm *PackageManager, fs billy.Filesystem) error {
		gitregfs, err := fs.Chroot("git")
		if err != nil {
			return err
		}

		return WithRegistry(gitreg.New(gitregfs))(pm, fs)
	}
}
