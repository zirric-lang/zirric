package cavereg_test

import (
	"context"
	"path/filepath"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/cavereg"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
)

func TestCaveRegistryIncludesRootModule(t *testing.T) {
	fs := memfs.New()
	writeFile(t, fs, "moduleA/alpha.zirr", "mod a")
	writeFile(t, fs, "moduleB/beta.zirr", "mod b")

	provider, err := cavereg.New(cavefile.Cavefile{
		Package: cavefile.Package{
			Name:   "example",
			Source: "file:///example",
		},
	}, fs)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	pkgs, err := provider.Discover(context.Background())
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}

	resolved, err := pkgs[0].Resolve(context.Background())
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	mods, err := resolved.ResolveModules()
	if err != nil {
		t.Fatalf("resolve modules: %v", err)
	}

	moduleURIs := collectModuleURIs(mods)
	ensureModule(t, moduleURIs, "example")
	ensureModule(t, moduleURIs, "example.modulea")
	ensureModule(t, moduleURIs, "example.moduleb")

	root := findModule(mods, registry.LogicalURI("example"))
	if root == nil {
		t.Fatalf("expected root module to be discovered")
	}
	rootSources, err := root.Sources()
	if err != nil {
		t.Fatalf("root sources: %v", err)
	}
	if len(rootSources) != 0 {
		t.Fatalf("expected root module to have no sources, got %d", len(rootSources))
	}
}

func writeFile(t *testing.T, fs billy.Filesystem, path string, content string) {
	t.Helper()
	if dir := filepath.Dir(path); dir != "." {
		if err := fs.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir all: %v", err)
		}
	}
	file, err := fs.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = file.Write([]byte(content))
	closeErr := file.Close()
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if closeErr != nil {
		t.Fatalf("close: %v", closeErr)
	}
}

func collectModuleURIs(mods []registry.ResolvedModule) map[string]struct{} {
	out := make(map[string]struct{}, len(mods))
	for _, mod := range mods {
		out[string(mod.URI())] = struct{}{}
	}
	return out
}

func ensureModule(t *testing.T, mods map[string]struct{}, uri string) {
	t.Helper()
	if _, ok := mods[uri]; !ok {
		t.Fatalf("expected module %s to be discovered", uri)
	}
}

func findModule(mods []registry.ResolvedModule, uri registry.LogicalURI) registry.ResolvedModule {
	for _, mod := range mods {
		if mod.URI() == uri {
			return mod
		}
	}
	return nil
}
