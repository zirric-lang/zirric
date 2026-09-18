package orchestra

import (
	"context"
	"io"
	"path/filepath"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/pkgmanager"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/cavereg"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
	"github.com/go-git/go-billy/v5"
)

// ParseCavefile reads and parses the Cavefile at cavefilePath within projectFS.
// It creates a bootstrap resolver with only the standard library so that
// attribute declarations in cave can be resolved for type checking.
// The analyzer is run on both cave and the Cavefile module so that
// identifier references and attribute types are fully resolved.
func ParseCavefile(ctx context.Context, projectFS billy.Filesystem, registryFS billy.Filesystem, cavefilePath string) (cavefile.Cavefile, error) {
	bootstrapCave := ensureStandardLibraryDependency(cavefile.Cavefile{
		Package: cavefile.Package{
			Name:   "_bootstrap",
			Source: "file://" + projectFS.Root(),
		},
	})

	caveReg, err := cavereg.New(bootstrapCave, projectFS)
	if err != nil {
		return cavefile.Cavefile{}, err
	}

	pm, err := pkgmanager.New(
		registryFS,
		pkgmanager.WithRegistry(caveReg),
		withDefaultStdlibRegistry(),
	)
	if err != nil {
		return cavefile.Cavefile{}, err
	}

	resolver, err := NewModuleResolver(pm, bootstrapCave, ReadOnly())
	if err != nil {
		return cavefile.Cavefile{}, err
	}

	// Parse the Cavefile as a standalone module (not part of the project's module tree)
	src, err := readFile(projectFS, cavefilePath)
	if err != nil {
		return cavefile.Cavefile{}, err
	}
	cavefileURI := registry.LogicalURI("Cavefile")
	staticSrc := staticmodule.NewSource(cavefileURI, src)
	mod := staticmodule.NewModule(cavefileURI, []registry.Source{staticSrc})

	prelude, err := resolver.Prelude(ctx)
	if err != nil {
		return cavefile.Cavefile{}, err
	}

	// parseResolvedModule handles parse errors gracefully (returns partial results)
	cavefileMod, _ := parseResolvedModule(mod, prelude)

	// Resolve cave and tasks so attribute declarations are available for type checking
	caveMod, err := resolver.ResolveModule(ctx, "cave")
	if err != nil {
		return cavefile.Cavefile{}, err
	}
	tasksMod, err := resolver.ResolveModule(ctx, "tasks")
	if err != nil {
		return cavefile.Cavefile{}, err
	}

	// Run the analyzer on all three modules; errors are tolerated (Cavefile may not
	// be fully type-correct at parse time, but attribute references are resolved)
	analysis := analyzer.New(resolver)
	analysis.Analyze(caveMod, false)
	analysis.Analyze(tasksMod, false)
	analysis.Analyze(cavefileMod, false)

	fallbackName := filepath.Base(projectFS.Root())
	result := cavefile.Parse(cavefileMod, caveMod, tasksMod, fallbackName, projectFS.Root())
	return result, nil
}

func readFile(fs billy.Filesystem, path string) ([]byte, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
