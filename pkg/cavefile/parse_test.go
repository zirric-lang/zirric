package cavefile_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	billyutil "github.com/go-git/go-billy/v5/util"
)

// testdataCavefile is the content written into the in-memory projectFS as "Cavefile".
const testdataCavefile = `mod myproject

import cave
import tasks

@cave.Package()
data Dependencies {
  @cave.Stdlib("tests")
  tests

  @cave.Stdlib("prelude")
  prelude

  @cave.Local("../some-local-package")
  helpers

  @cave.Git("https://code.knabel.dev/zirric-lang/zirric")
  @cave.Version("latest")
  zirric
}
`

func parseCavefileInMem(t *testing.T, content string) (cavefile.Cavefile, error) {
	t.Helper()
	projectFS := memfs.New()
	if err := billyutil.WriteFile(projectFS, "Cavefile", []byte(content), 0o644); err != nil {
		t.Fatalf("write Cavefile: %v", err)
	}
	registryFS := memfs.New()
	return orchestra.ParseCavefile(context.Background(), projectFS, registryFS, "Cavefile")
}

func TestParseFullCavefile(t *testing.T) {
	cave, err := parseCavefileInMem(t, testdataCavefile)
	if err != nil {
		t.Logf("parse warning (non-fatal): %v", err)
	}
	if len(cave.Dependencies) != 4 {
		t.Fatalf("expected 4 dependencies, got %d: %+v", len(cave.Dependencies), cave.Dependencies)
	}
	byName := map[string]struct{ Source string }{}
	for _, d := range cave.Dependencies {
		byName[d.Name] = struct{ Source string }{d.Source}
	}

	if d, ok := byName["tests"]; !ok {
		t.Error("missing 'tests' dependency")
	} else if d.Source != cavefile.StandardLibrarySource {
		t.Errorf("tests source = %q, want %q", d.Source, cavefile.StandardLibrarySource)
	}

	if d, ok := byName["prelude"]; !ok {
		t.Error("missing 'prelude' dependency")
	} else if d.Source != cavefile.StandardLibrarySource {
		t.Errorf("prelude source = %q, want %q", d.Source, cavefile.StandardLibrarySource)
	}

	if _, ok := byName["helpers"]; !ok {
		t.Error("missing 'helpers' dependency")
	}

	if d, ok := byName["zirric"]; !ok {
		t.Error("missing 'zirric' dependency")
	} else {
		if d.Source != "https://code.knabel.dev/zirric-lang/zirric" {
			t.Errorf("zirric source = %q, want %q", d.Source, "https://code.knabel.dev/zirric-lang/zirric")
		}
	}
}

func TestParseStdlibDependencyModule(t *testing.T) {
	src := `import cave

@cave.Package()
data Dependencies {
  @cave.Stdlib("cave")
  caveModule
}
`
	cave, _ := parseCavefileInMem(t, src)
	if len(cave.Dependencies) != 1 {
		t.Fatalf("expected 1 dep, got %d: %+v", len(cave.Dependencies), cave.Dependencies)
	}
	dep := cave.Dependencies[0]
	if dep.Name != "caveModule" {
		t.Errorf("Name = %q, want %q", dep.Name, "caveModule")
	}
	if dep.Module != "cave" {
		t.Errorf("Module = %q, want %q", dep.Module, "cave")
	}
}

func TestParseNameFromModuleDeclaration(t *testing.T) {
	cave, _ := parseCavefileInMem(t, testdataCavefile)
	if cave.ModulePath != "myproject" {
		t.Errorf("ModulePath = %q, want %q", cave.ModulePath, "myproject")
	}
	if cave.Name != "myproject" {
		t.Errorf("Name = %q, want %q", cave.Name, "myproject")
	}
}

func TestParseFallbackName(t *testing.T) {
	cave, _ := parseCavefileInMem(t, `import cave

@cave.Package()
data Dependencies {
}
`)
	// Without a mod declaration the name falls back to the memfs root basename.
	if cave.ModulePath != "" {
		t.Errorf("ModulePath = %q, want empty", cave.ModulePath)
	}
	if cave.Name == "" {
		t.Error("expected non-empty package name")
	}
}

func TestParsePackageMetadata(t *testing.T) {
	src := `mod code.knabel.dev.zirric_lang.zirric

import cave

@cave.Package()
@cave.Git("https://code.knabel.dev/zirric-lang/zirric")
@cave.Version("1.2.3")
@cave.LanguageVersion("^0.1.0")
@cave.Description("The package itself")
@cave.Documentation("https://zirric.knabel.dev")
data Zirric {
}
`
	cave, _ := parseCavefileInMem(t, src)
	wantName := "code.knabel.dev.zirric_lang.zirric"
	if cave.Name != wantName {
		t.Errorf("Name = %q, want %q", cave.Name, wantName)
	}
	wantSource := "https://code.knabel.dev/zirric-lang/zirric"
	if cave.Source != wantSource {
		t.Errorf("Source = %q, want %q", cave.Source, wantSource)
	}
	if cave.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", cave.Version, "1.2.3")
	}
	if cave.LanguageVersion != "^0.1.0" {
		t.Errorf("LanguageVersion = %q, want %q", cave.LanguageVersion, "^0.1.0")
	}
	if cave.Description != "The package itself" {
		t.Errorf("Description = %q, want %q", cave.Description, "The package itself")
	}
	if cave.Documentation != "https://zirric.knabel.dev" {
		t.Errorf("Documentation = %q, want %q", cave.Documentation, "https://zirric.knabel.dev")
	}
}

func TestParseDependencyMetadata(t *testing.T) {
	src := `mod myproject

import cave

@cave.Package()
data Dependencies {
  @cave.Git("https://example.com/example/example")
  @cave.Version("^1.2.3")
  @cave.LanguageVersion(">=0.1.0")
  @cave.Description("An example dependency")
  @cave.Documentation("https://example.com/docs")
  example
}
`
	cave, _ := parseCavefileInMem(t, src)
	if len(cave.Dependencies) != 1 {
		t.Fatalf("expected 1 dep, got %d: %+v", len(cave.Dependencies), cave.Dependencies)
	}
	dep := cave.Dependencies[0]
	if dep.LanguageVersion != ">=0.1.0" {
		t.Errorf("LanguageVersion = %q, want %q", dep.LanguageVersion, ">=0.1.0")
	}
	if dep.Description != "An example dependency" {
		t.Errorf("Description = %q, want %q", dep.Description, "An example dependency")
	}
	if dep.Documentation != "https://example.com/docs" {
		t.Errorf("Documentation = %q, want %q", dep.Documentation, "https://example.com/docs")
	}
	if len(dep.Predicates) != 1 || dep.Predicates[0].String() != "^1.2.3" {
		t.Errorf("Predicates = %v, want [^1.2.3]", dep.Predicates)
	}
}

func TestParseModuleWithoutGitAttribute(t *testing.T) {
	// A package that names no Git repository has no source, and its name is the one its mod declares.
	cave, _ := parseCavefileInMem(t, testdataCavefile)
	if cave.Source != "" {
		t.Errorf("Source = %q, want empty", cave.Source)
	}
	if cave.Name != "myproject" {
		t.Errorf("Name = %q, want %q", cave.Name, "myproject")
	}
}

func TestParsePackageViaAlias(t *testing.T) {
	// @cave.Package and @cave.Git must resolve through import aliases just like every other cave attribute (regression guard: the package lookup must not hardcode the "cave" prefix).
	src := `mod mymodule

import x = cave

@x.Package()
@x.Git("https://code.knabel.dev/zirric-lang/zirric")
data Deps {
}
`
	cave, _ := parseCavefileInMem(t, src)
	wantSource := "https://code.knabel.dev/zirric-lang/zirric"
	if cave.Source != wantSource {
		t.Errorf("Source = %q, want %q", cave.Source, wantSource)
	}
}

func TestParseNoCavefileBlock(t *testing.T) {
	cave, _ := parseCavefileInMem(t, "mod mypackage\n")
	if len(cave.Dependencies) != 0 {
		t.Errorf("expected 0 deps, got %d", len(cave.Dependencies))
	}
}

func TestParseAttributeAliasingIrrelevant(t *testing.T) {
	// Using an explicit alias (x = cave) should still work because we resolve the alias via the import declaration
	src := `mod mymodule

import x = cave

@x.Package()
data Dependencies {
  @x.Stdlib("prelude")
  prelude
}
`
	cave, _ := parseCavefileInMem(t, src)
	if len(cave.Dependencies) != 1 {
		t.Fatalf("expected 1 dep, got %d: %+v", len(cave.Dependencies), cave.Dependencies)
	}
	if cave.Dependencies[0].Source != cavefile.StandardLibrarySource {
		t.Errorf("source = %q, want %q", cave.Dependencies[0].Source, cavefile.StandardLibrarySource)
	}
}

func TestParseNoNameCollision(t *testing.T) {
	// An import with a coincidentally-named attribute should NOT be treated as cave.Package because the alias maps to a different module URI
	src := `mod mymodule

import notcave = tasks
import cave

@notcave.Package()
data WrongDependencies {
  tasks
}

@cave.Package()
data RealDependencies {
  @cave.Stdlib("prelude")
  prelude
}
`
	cave, _ := parseCavefileInMem(t, src)
	if len(cave.Dependencies) != 1 {
		t.Fatalf("expected 1 dep (only from cave.Package), got %d: %+v", len(cave.Dependencies), cave.Dependencies)
	}
	if cave.Dependencies[0].Name != "prelude" {
		t.Errorf("dependency name = %q, want %q", cave.Dependencies[0].Name, "prelude")
	}
}

func TestParseTestdataFile(t *testing.T) {
	cave, parseErr := parseCavefileFromTestdata(t)
	if parseErr != nil {
		t.Logf("parse warning (non-fatal): %v", parseErr)
	}
	if len(cave.Dependencies) != 3 {
		t.Errorf("expected 3 dependencies from testdata, got %d: %+v", len(cave.Dependencies), cave.Dependencies)
	}
}

// parseCavefileFromTestdata parses ../orchestra/testdata/Cavefile using the real project filesystem.
func parseCavefileFromTestdata(t *testing.T) (cavefile.Cavefile, error) {
	t.Helper()
	if _, err := os.Stat("../orchestra/testdata/Cavefile"); err != nil {
		t.Skip("testdata Cavefile not found:", err)
	}

	testdataDir, _ := filepath.Abs("../orchestra/testdata")
	projectFS := osfs.New(testdataDir)
	registryFS := memfs.New()
	return orchestra.ParseCavefile(context.Background(), projectFS, registryFS, "Cavefile")
}
