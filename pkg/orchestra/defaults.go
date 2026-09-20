package orchestra

import (
	arraysfs "code.knabel.dev/zirric-lang/zirric/arrays"
	bytesfs "code.knabel.dev/zirric-lang/zirric/bytes"
	cavefs "code.knabel.dev/zirric-lang/zirric/cave"
	clockfs "code.knabel.dev/zirric-lang/zirric/clock"
	dictsfs "code.knabel.dev/zirric-lang/zirric/dicts"
	errorsfs "code.knabel.dev/zirric-lang/zirric/errors"
	fmtfs "code.knabel.dev/zirric-lang/zirric/fmt"
	fsfs "code.knabel.dev/zirric-lang/zirric/fs"
	funfs "code.knabel.dev/zirric-lang/zirric/fun"
	futurefs "code.knabel.dev/zirric-lang/zirric/future"
	iofs "code.knabel.dev/zirric-lang/zirric/io"
	jsonfs "code.knabel.dev/zirric-lang/zirric/json"
	mathfs "code.knabel.dev/zirric-lang/zirric/math"
	optionsfs "code.knabel.dev/zirric-lang/zirric/options"
	osfs "code.knabel.dev/zirric-lang/zirric/os"
	pathsfs "code.knabel.dev/zirric-lang/zirric/paths"
	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/pkgmanager"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/embedreg"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
	preludefs "code.knabel.dev/zirric-lang/zirric/prelude"
	randomfs "code.knabel.dev/zirric-lang/zirric/random"
	rangesfs "code.knabel.dev/zirric-lang/zirric/ranges"
	reflectfs "code.knabel.dev/zirric-lang/zirric/reflect"
	resultsfs "code.knabel.dev/zirric-lang/zirric/results"
	scriptsfs "code.knabel.dev/zirric-lang/zirric/scripts"
	stringsfs "code.knabel.dev/zirric-lang/zirric/strings"
	tasksfs "code.knabel.dev/zirric-lang/zirric/tasks"
	testsfs "code.knabel.dev/zirric-lang/zirric/tests"
	timefs "code.knabel.dev/zirric-lang/zirric/time"
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
		embedreg.FSConfig{Name: "tasks", FS: tasksfs.FS},
		embedreg.FSConfig{Name: "io", FS: iofs.FS},
		embedreg.FSConfig{Name: "fmt", FS: fmtfs.FS},
		embedreg.FSConfig{Name: "os", FS: osfs.FS},
		embedreg.FSConfig{Name: "scripts", FS: scriptsfs.FS},
		embedreg.FSConfig{Name: "bytes", FS: bytesfs.FS},
		embedreg.FSConfig{Name: "ranges", FS: rangesfs.FS},
		embedreg.FSConfig{Name: "strings", FS: stringsfs.FS},
		embedreg.FSConfig{Name: "arrays", FS: arraysfs.FS},
		embedreg.FSConfig{Name: "dicts", FS: dictsfs.FS},
		embedreg.FSConfig{Name: "fun", FS: funfs.FS},
		embedreg.FSConfig{Name: "options", FS: optionsfs.FS},
		embedreg.FSConfig{Name: "results", FS: resultsfs.FS},
		embedreg.FSConfig{Name: "errors", FS: errorsfs.FS},
		embedreg.FSConfig{Name: "reflect", FS: reflectfs.FS},
		embedreg.FSConfig{Name: "tests", FS: testsfs.FS},
		embedreg.FSConfig{Name: "math", FS: mathfs.FS},
		embedreg.FSConfig{Name: "paths", FS: pathsfs.FS},
		embedreg.FSConfig{Name: "fs", FS: fsfs.FS},
		embedreg.FSConfig{Name: "random", FS: randomfs.FS},
		embedreg.FSConfig{Name: "time", FS: timefs.FS},
		embedreg.FSConfig{Name: "clock", FS: clockfs.FS},
		embedreg.FSConfig{Name: "json", FS: jsonfs.FS},
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
