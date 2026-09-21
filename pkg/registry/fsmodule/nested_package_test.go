package fsmodule_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/fsmodule"
	"github.com/go-git/go-billy/v5/memfs"
	billyutil "github.com/go-git/go-billy/v5/util"
)

func TestDiscoverModulesSkipsNestedProjects(t *testing.T) {
	fs := memfs.New()
	write := func(path string, content string) {
		t.Helper()
		if err := billyutil.WriteFile(fs, path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	write("main.zirr", "mod main\n")
	write("lib/lib.zirr", "mod lib\n")
	// A directory that declares a package of its own: its modules belong to it, and it resolves its own dependencies.
	write("examples/project/Cavefile", "import cave\n")
	write("examples/project/person.zirr", "mod person\n")

	mods, err := fsmodule.DiscoverModules(registry.LogicalURI("root"), fs)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}

	found := map[string]bool{}
	for _, mod := range mods {
		found[string(mod.URI())] = true
	}
	if !found["root"] {
		t.Errorf("expected the root module, got %v", found)
	}
	if !found["root.lib"] {
		t.Errorf("expected root.lib, got %v", found)
	}
	for uri := range found {
		if uri == "root.examples.project" {
			t.Errorf("a nested project must not be walked as part of the package, got %v", found)
		}
	}
}

func TestDiscoverModulesKeepsTheRootWithItsOwnCavefile(t *testing.T) {
	// The package being walked has a Cavefile of its own, and that must not exclude it from itself.
	fs := memfs.New()
	if err := billyutil.WriteFile(fs, "Cavefile", []byte("import cave\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := billyutil.WriteFile(fs, "main.zirr", []byte("mod main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	mods, err := fsmodule.DiscoverModules(registry.LogicalURI("root"), fs)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(mods) != 1 || string(mods[0].URI()) != "root" {
		t.Errorf("expected just the root module, got %v", mods)
	}
}
