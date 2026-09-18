package orchestra_test

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"code.knabel.dev/zirric-lang/zirric/pkg/pkgmanager"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
)

func TestNewFallsBackToSyntheticCavefileWhenNil(t *testing.T) {
	projectFS := memfs.New()
	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:   projectFS,
		RegistryFS:  memfs.New(),
		PackageName: "myproj",
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}

	cave := orch.Cavefile()
	if cave.Name != "myproj" {
		t.Errorf("Name = %q, want %q", cave.Name, "myproj")
	}
	wantSource := "file://" + projectFS.Root()
	if cave.Source != wantSource {
		t.Errorf("Source = %q, want %q", cave.Source, wantSource)
	}
}

func TestNewUsesProvidedCavefileForProjectIdentity(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "greet.zirr", "mod mypkg\nconst greeting = \"hi\"\n")

	cave := cavefile.Cavefile{
		Package: cavefile.Package{Name: "mypkg", Source: "file://" + projectFS.Root()},
	}

	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:   projectFS,
		RegistryFS:  memfs.New(),
		PackageName: "ignored-when-cavefile-is-set",
		Cavefile:    &cave,
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}

	if got := orch.Cavefile().Name; got != "mypkg" {
		t.Errorf("Name = %q, want %q", got, "mypkg")
	}

	if err := orch.RunFile(context.Background(), "greet.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func TestNewInjectsStdlibDependencyIntoProvidedCavefile(t *testing.T) {
	projectFS := memfs.New()
	cave := cavefile.Cavefile{
		Package: cavefile.Package{Name: "proj", Source: "file://" + projectFS.Root()},
	}

	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:   projectFS,
		RegistryFS:  memfs.New(),
		PackageName: "proj",
		Cavefile:    &cave,
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}

	found := false
	for _, dep := range orch.Cavefile().Dependencies {
		if dep.Source == orchestra.StandardLibraryDependency.Source {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected stdlib dependency to be injected, got %+v", orch.Cavefile().Dependencies)
	}

	// New must treat Config.Cavefile as a value to copy from, not modify in place.
	if len(cave.Dependencies) != 0 {
		t.Errorf("caller's Cavefile was mutated: %+v", cave.Dependencies)
	}
}

func TestNewUsesCavefilePathOverride(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, orchestra.DefaultCavefileName, `import cave

@cave.Dependencies()
data Dependencies {
  @cave.Stdlib("io")
  io
}
`)
	writeFile(t, projectFS, "Cavefile.alt", `import cave

@cave.Dependencies()
data Dependencies {
  @cave.Stdlib("os")
  os
}
`)

	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:    projectFS,
		RegistryFS:   memfs.New(),
		PackageName:  "proj",
		CavefilePath: "Cavefile.alt",
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}

	var names []string
	for _, dep := range orch.Cavefile().Dependencies {
		names = append(names, dep.Name)
	}
	if !slices.Contains(names, "os") {
		t.Errorf("expected the 'os' dependency from Cavefile.alt, got %v", names)
	}
	if slices.Contains(names, "io") {
		t.Errorf("expected the default Cavefile to be ignored, got %v", names)
	}
}

func TestNewCavefilePathOverrideMissingIsAnError(t *testing.T) {
	projectFS := memfs.New()

	_, err := orchestra.New(orchestra.Config{
		ProjectFS:    projectFS,
		RegistryFS:   memfs.New(),
		PackageName:  "proj",
		CavefilePath: "does-not-exist.Cavefile",
	})
	if err == nil {
		t.Fatal("expected an error for a missing CavefilePath override")
	}
}

func TestNewAutoDetectsCavefileFromProjectRoot(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, orchestra.DefaultCavefileName, `import cave

@cave.Dependencies()
data Dependencies {
  @cave.Stdlib("io")
  io
}
`)

	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:   projectFS,
		RegistryFS:  memfs.New(),
		PackageName: "ignored-when-cavefile-is-detected",
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}

	var found bool
	for _, dep := range orch.Cavefile().Dependencies {
		if dep.Name != "io" {
			continue
		}
		found = true
		if dep.Source != cavefile.StandardLibrarySource {
			t.Errorf("io source = %q, want %q", dep.Source, cavefile.StandardLibrarySource)
		}
	}
	if !found {
		t.Fatalf("expected an 'io' dependency detected from the Cavefile, got %+v", orch.Cavefile().Dependencies)
	}
}

func TestStdlibDependencyModuleResolves(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, orchestra.DefaultCavefileName, `import cave

@cave.Dependencies()
data Dependencies {
  @cave.Stdlib("io")
  io
}
`)

	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:   projectFS,
		RegistryFS:  memfs.New(),
		PackageName: "proj",
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}

	var mod string
	for _, dep := range orch.Cavefile().Dependencies {
		if dep.Name == "io" {
			mod = string(dep.Module)
		}
	}
	if mod != "io" {
		t.Fatalf("Module = %q, want %q", mod, "io")
	}

	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	if _, err := resolver.ResolveModule(context.Background(), "io"); err != nil {
		t.Fatalf("resolve module %q: %v", mod, err)
	}
}

// Resolves under "helpers" (declared in the Cavefile), not whatever name the registry would derive from the directory itself.
func TestLocalDependencyResolvesUnderDeclaredName(t *testing.T) {
	testdataDir, err := filepath.Abs("testdata/localdep")
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}

	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:   osfs.New(testdataDir),
		RegistryFS:  memfs.New(),
		PackageName: "localdep",
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}

	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	mod, err := resolver.ResolveModule(context.Background(), "helpers")
	if err != nil {
		t.Fatalf("resolve module %q: %v", "helpers", err)
	}
	if len(mod.Files) == 0 {
		t.Fatal("expected the helpers module to have source files")
	}

	if err := orch.RunFile(context.Background(), "main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func TestEnsureInstalledResolvesDeclaredDependencies(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, orchestra.DefaultCavefileName, `import cave

@cave.Dependencies()
data Dependencies {
  @cave.Stdlib("io")
  io
}
`)

	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:   projectFS,
		RegistryFS:  memfs.New(),
		PackageName: "proj",
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}

	resolver, err := orch.NewResolver()
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	pkgs, err := resolver.EnsureInstalled(context.Background())
	if err != nil {
		t.Fatalf("ensure installed: %v", err)
	}

	found := false
	for _, pkg := range pkgs {
		if pkg.Source() == cavefile.StandardLibrarySource {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the stdlib package among installed packages, got %+v", pkgs)
	}
}

func TestWithInstallProgressReportsDependenciesBeforeProject(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, orchestra.DefaultCavefileName, `import cave

@cave.Dependencies()
data Dependencies {
  @cave.Stdlib("io")
  io
}
`)

	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:   projectFS,
		RegistryFS:  memfs.New(),
		PackageName: "proj",
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}

	var names []string
	resolver, err := orch.NewResolver(orchestra.WithInstallProgress(func(evt pkgmanager.InstallEvent) {
		names = append(names, evt.Dependency.Name)
	}))
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	if _, err := resolver.EnsureInstalled(context.Background()); err != nil {
		t.Fatalf("ensure installed: %v", err)
	}

	if len(names) < 2 {
		t.Fatalf("expected at least 2 install events, got %v", names)
	}
	if last := names[len(names)-1]; last != orch.Cavefile().Name {
		t.Fatalf("expected the project (%q) last, got %v", orch.Cavefile().Name, names)
	}
	if names[0] != "io" {
		t.Fatalf("expected the declared dependency first, got %v", names)
	}
}
