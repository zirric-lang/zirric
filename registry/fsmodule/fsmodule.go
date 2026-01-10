package fsmodule

import (
	"cmp"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/go-git/go-billy/v5"
	billyutil "github.com/go-git/go-billy/v5/util"
	"code.knabel.dev/zirric-lang/zirric/registry"
)

type FSModule struct {
	logicalURI registry.LogicalURI
	fs         billy.Filesystem
	sources    []registry.Source
}

func NewModule(logicalURI registry.LogicalURI, fs billy.Filesystem) *FSModule {
	return &FSModule{
		logicalURI: logicalURI,
		fs:         fs,
	}
}

func (m *FSModule) URI() registry.LogicalURI {
	return m.logicalURI
}

func (m *FSModule) Sources() ([]registry.Source, error) {
	return m.sources, nil
}

func (m *FSModule) String() string {
	return fmt.Sprintf("%s: (%d files)", m.logicalURI, len(m.sources))
}

func DiscoverModules(base registry.LogicalURI, fs billy.Filesystem) ([]*FSModule, error) {
	rootSrcs := make(map[registry.LogicalURI]*FSModule)

	// TODO: replace glob with cusom logic
	// matches, err := billyutil.Glob(fs, "**/*.zirr")
	matches, err := recursiveGlob(fs, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to discover modules of %q, %w", base, err)
	}

	for _, m := range matches {
		name := filepath.Base(m)

		moduleURI := base.Join(filepath.Dir(m))
		mod, ok := rootSrcs[moduleURI]
		if !ok {
			subfs, err := fs.Chroot(filepath.Dir(m))
			if err != nil {
				return nil, fmt.Errorf("failed to prepare file system of %q, %w", base, err)
			}
			mod = &FSModule{
				logicalURI: moduleURI,
				fs:         subfs,
				sources:    []registry.Source{},
			}
			rootSrcs[moduleURI] = mod
		}
		mod.sources = append(mod.sources, &FSSource{
			name:       name,
			logicalURI: base.Join(m),
			fs:         mod.fs,
		})
	}

	mods := make([]*FSModule, 0, len(rootSrcs))
	for _, m := range rootSrcs {
		mods = append(mods, m)
	}
	slices.SortFunc(mods, func(lhs, rhs *FSModule) int {
		return cmp.Compare(lhs.URI(), rhs.URI())
	})
	return mods, nil
}

func recursiveGlob(fsys billy.Filesystem, maxDepth int) ([]string, error) {
	var (
		sources []string
		folder  = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_-]+$")
		ext     = ".zirr"
	)

	err := billyutil.Walk(fsys, ".", func(path string, info fs.FileInfo, err error) error {
		if path == "" || path == "." {
			return nil
		}
		if info.IsDir() {
			if maxDepth > 0 && len(strings.Split(path, string(filepath.Separator))) > maxDepth {
				return filepath.SkipDir
			}
			ok := folder.MatchString(filepath.Base(path))
			if !ok {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) == ext {
			sources = append(sources, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return sources, nil
}
