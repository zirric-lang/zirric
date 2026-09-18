package orchestra

import (
	cavefs "code.knabel.dev/zirric-lang/zirric/cave"
	fmtfs "code.knabel.dev/zirric-lang/zirric/fmt"
	futurefs "code.knabel.dev/zirric-lang/zirric/future"
	iofs "code.knabel.dev/zirric-lang/zirric/io"
	osfs "code.knabel.dev/zirric-lang/zirric/os"
	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/pkgmanager"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/embedreg"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
	preludefs "code.knabel.dev/zirric-lang/zirric/prelude"
	"github.com/go-git/go-billy/v5"
)

const (
	defaultStandardLibraryName    = "code.knabel.dev.zirric_lang.zirric"
	defaultStandardLibraryVersion = "latest"
)

var StandardLibraryDependency = cavefile.Dependency{
	Package: cavefile.Package{
		Name:   defaultStandardLibraryName,
		Source: cavefile.StandardLibrarySource,
	},
	Predicates: []version.Predicate{
		version.Predicate{
			Comparison: version.ComparisonExact,
			Version:    version.Parse(defaultStandardLibraryVersion),
		},
	},
}

func DefaultStdlibProvider() (*embedreg.EmbedRegistry, error) {
	return embedreg.New(
		StandardLibraryDependency.Package,
		version.Parse("latest"),
		embedreg.FSConfig{Name: "prelude", FS: preludefs.FS},
		embedreg.FSConfig{Name: "cave", FS: cavefs.FS},
		embedreg.FSConfig{Name: "future", FS: futurefs.FS},
		embedreg.FSConfig{Name: "io", FS: iofs.FS},
		embedreg.FSConfig{Name: "fmt", FS: fmtfs.FS},
		embedreg.FSConfig{Name: "os", FS: osfs.FS},
	)
}

func withDefaultStdlibRegistry() pkgmanager.Option {
	return func(pm *pkgmanager.PackageManager, fs billy.Filesystem) error {
		stdlib, err := DefaultStdlibProvider()
		if err != nil {
			return err
		}
		return pkgmanager.WithRegistry(stdlib)(pm, fs)
	}
}

func ensureStandardLibraryDependency(cave cavefile.Cavefile) cavefile.Cavefile {
	for _, dep := range cave.Dependencies {
		if dep.Module != "" {
			continue
		}
		if dep.Name == StandardLibraryDependency.Name || dep.Source == StandardLibraryDependency.Source {
			return cave
		}
	}
	cave.Dependencies = append(cave.Dependencies, StandardLibraryDependency)
	return cave
}
