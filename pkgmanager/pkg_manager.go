package pkgmanager

import (
	"github.com/go-git/go-billy/v5"
	"code.knabel.dev/zirric-lang/zirric/cavefile"
	"code.knabel.dev/zirric-lang/zirric/registry"
	"code.knabel.dev/zirric-lang/zirric/registry/gitreg"
)

type PackageManager struct {
	registries []registry.Provider
}

func New(fs billy.Filesystem) (*PackageManager, error) {
	gitregfs, err := fs.Chroot("git")
	if err != nil {
		return nil, err
	}
	return &PackageManager{
		registries: []registry.Provider{
			gitreg.New(gitregfs),
		},
	}, nil
}

func (pm *PackageManager) Install(cf cavefile.Cavefile) *InstallationTask {
	return &InstallationTask{
		pkgmanager: pm,
		cave:       cf,
	}
}
