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
const testdataCavefile = `import cave
import tasks

@cave.Dependencies()
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

@cave.Dependencies()
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

func TestParseFallbackName(t *testing.T) {
	cave, _ := parseCavefileInMem(t, testdataCavefile)
	// Name should be non-empty (fallback to the memfs root basename)
	if cave.Name == "" {
		t.Error("expected non-empty package name")
	}
}

func TestParsePackageFromURL(t *testing.T) {
	src := `import cave

@cave.Package("https://code.knabel.dev/zirric-lang/zirric")
mod mymodule
` + testdataCavefile
	cave, _ := parseCavefileInMem(t, src)
	wantName := "code.knabel.dev.zirric_lang.zirric"
	if cave.Name != wantName {
		t.Errorf("Name = %q, want %q", cave.Name, wantName)
	}
	wantSource := "https://code.knabel.dev/zirric-lang/zirric"
	if cave.Source != wantSource {
		t.Errorf("Source = %q, want %q", cave.Source, wantSource)
	}
}

func TestParseModuleWithoutPackageAttribute(t *testing.T) {
	// A `mod` declaration without @cave.Package should leave Source empty
	// and fall back to fallbackName, exactly like having no mod decl at all.
	src := `import cave

mod mymodule
` + testdataCavefile
	cave, _ := parseCavefileInMem(t, src)
	if cave.Source != "" {
		t.Errorf("Source = %q, want empty", cave.Source)
	}
	if cave.Name == "" {
		t.Error("expected fallback name to be non-empty")
	}
}

func TestParsePackageViaAlias(t *testing.T) {
	// @cave.Package must resolve through import aliases just like every
	// other cave attribute (regression guard: extractPackageSource
	// must not hardcode the "cave" prefix).
	src := `import x = cave

@x.Package("https://code.knabel.dev/zirric-lang/zirric")
mod mymodule
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
	// Using an explicit alias (x = cave) should still work
	// because we resolve the alias via the import declaration
	src := `import x = cave

@x.Dependencies()
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
	// An import with a coincidentally-named attribute should NOT be treated as cave.Dependencies
	// because the alias maps to a different module URI
	src := `import notcave = tasks
import cave

@notcave.Dependencies()
data WrongDependencies {
  tasks
}

@cave.Dependencies()
data RealDependencies {
  @cave.Stdlib("prelude")
  prelude
}
`
	cave, _ := parseCavefileInMem(t, src)
	if len(cave.Dependencies) != 1 {
		t.Fatalf("expected 1 dep (only from cave.Dependencies), got %d: %+v", len(cave.Dependencies), cave.Dependencies)
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
