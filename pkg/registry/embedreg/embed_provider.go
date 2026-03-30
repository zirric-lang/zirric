package embedreg

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"slices"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

type FSConfig struct {
	Name string
	FS   fs.FS
}

type EmbedRegistry struct {
	pkg         cavefile.Package
	version     version.Version
	filesystems []FSConfig
}

func New(pkg cavefile.Package, ver version.Version, filesystems ...FSConfig) (*EmbedRegistry, error) {
	if pkg.Name == "" {
		return nil, errors.New("base uri is required")
	}
	if pkg.Source == "" {
		return nil, errors.New("source is required")
	}
	if ver == nil {
		return nil, errors.New("version is required")
	}
	if len(filesystems) == 0 {
		return nil, errors.New("at least one filesystem is required")
	}
	return &EmbedRegistry{
		pkg:         pkg,
		version:     ver,
		filesystems: filesystems,
	}, nil
}

func (r *EmbedRegistry) Discover(ctx context.Context) ([]registry.ResolvedPackage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []registry.ResolvedPackage{&embedPackage{provider: r}}, nil
}

func (r *EmbedRegistry) DiscoverPackageVersions(ctx context.Context, name string, preds ...version.Predicate) ([]registry.Package, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if name != string(r.pkg.Name) {
		return nil, nil
	}
	for _, pred := range preds {
		if !r.version.Matches(pred) {
			return nil, nil
		}
	}
	return []registry.Package{&embedPackage{provider: r}}, nil
}

type embedPackage struct {
	provider *EmbedRegistry
}

func (p *embedPackage) Source() string {
	return p.provider.pkg.Source
}

func (p *embedPackage) Version() version.Version {
	return p.provider.version
}

func (p *embedPackage) Resolve(ctx context.Context) (registry.ResolvedPackage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *embedPackage) ResolveModules() ([]registry.ResolvedModule, error) {
	return p.provider.discoverModules()
}

type embedModule struct {
	uri     registry.LogicalURI
	sources []registry.Source
}

func (m *embedModule) URI() registry.LogicalURI {
	return m.uri
}

func (m *embedModule) Sources() ([]registry.Source, error) {
	return m.sources, nil
}

type embedSource struct {
	fs   fs.FS
	path string
	uri  registry.LogicalURI
}

func (s *embedSource) URI() registry.LogicalURI {
	return s.uri
}

func (s *embedSource) Read() ([]byte, error) {
	return fs.ReadFile(s.fs, s.path)
}

func (r *EmbedRegistry) discoverModules() ([]registry.ResolvedModule, error) {
	modules := make(map[registry.LogicalURI]*embedModule)
	for _, cfg := range r.filesystems {
		moduleBase := filepath.Join(cfg.Name)
		err := fs.WalkDir(cfg.FS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".zirr" {
				return nil
			}

			dir := filepath.Dir(path)
			if dir == "." {
				dir = ""
			}
			moduleURI := registry.JoinModuleURI("", filepath.Join(moduleBase, dir))
			mod := modules[moduleURI]
			if mod == nil {
				mod = &embedModule{uri: moduleURI}
				modules[moduleURI] = mod
			}

			sourcePath := filepath.ToSlash(filepath.Join(cfg.Name, path))
			sourceURI := registry.LogicalURI(sourcePath)
			mod.sources = append(mod.sources, &embedSource{
				fs:   cfg.FS,
				path: path,
				uri:  sourceURI,
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	out := make([]registry.ResolvedModule, 0, len(modules))
	for _, mod := range modules {
		slices.SortFunc(mod.sources, func(lhs, rhs registry.Source) int {
			return cmpURI(lhs.URI(), rhs.URI())
		})
		out = append(out, mod)
	}
	slices.SortFunc(out, func(lhs, rhs registry.ResolvedModule) int {
		return cmpURI(lhs.URI(), rhs.URI())
	})
	return out, nil
}

func cmpURI(lhs, rhs registry.LogicalURI) int {
	if lhs < rhs {
		return -1
	}
	if lhs > rhs {
		return 1
	}
	return 0
}
