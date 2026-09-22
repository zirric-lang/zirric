package orchestra_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	mv "github.com/metal-stack/v"
)

// newTestOrchestraOrError is newTestOrchestra for tests that care about the error.
func newTestOrchestraOrError(projectFS billy.Filesystem, name string, ignoreLanguageVersion bool) (*orchestra.Orchestra, error) {
	return orchestra.New(orchestra.Config{
		ProjectFS:             projectFS,
		RegistryFS:            memfs.New(),
		PackageName:           name,
		IgnoreLanguageVersion: ignoreLanguageVersion,
	})
}

func projectWithCavefile(t *testing.T) billy.Filesystem {
	t.Helper()
	projectFS := memfs.New()
	writeFile(t, projectFS, orchestra.DefaultCavefileName, "mod myapp\n\nimport cave\n\n@cave.Package()\ndata Deps {}\n")
	return projectFS
}

func runMain(t *testing.T, projectFS billy.Filesystem, path string) error {
	t.Helper()
	orch := newTestOrchestra(t, projectFS, "myapp")
	return orch.RunFile(context.Background(), path)
}

func TestModuleDeclarationIsRequired(t *testing.T) {
	projectFS := projectWithCavefile(t)
	writeFile(t, projectFS, "flow/node/main.zirr", "const x = 1\n")

	err := runMain(t, projectFS, "flow/node/main.zirr")
	if err == nil {
		t.Fatal("expected a file without a mod declaration to be rejected")
	}
	if !strings.Contains(err.Error(), "missing module declaration") {
		t.Errorf("error = %v, want it to report a missing module declaration", err)
	}
	if !strings.Contains(err.Error(), "mod myapp.flow.node") {
		t.Errorf("error = %v, want it to name the module the file belongs to", err)
	}
}

func TestModuleDeclarationMustBeFullyQualified(t *testing.T) {
	projectFS := projectWithCavefile(t)
	writeFile(t, projectFS, "flow/node/main.zirr", "mod node\n")

	err := runMain(t, projectFS, "flow/node/main.zirr")
	if err == nil {
		t.Fatal("expected an unqualified mod declaration to be rejected")
	}
	if !strings.Contains(err.Error(), "wrong module name") {
		t.Errorf("error = %v, want it to report the wrong module name", err)
	}
}

func TestModuleDeclarationBindsLastSegment(t *testing.T) {
	projectFS := projectWithCavefile(t)
	writeFile(t, projectFS, "flow/node/main.zirr", "mod myapp.flow.node\n\nconst value = 1\n\nfn same() {\n\treturn node.value\n}\n")

	if err := runMain(t, projectFS, "flow/node/main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func TestModuleDeclarationAliasBindsItsOwnName(t *testing.T) {
	projectFS := projectWithCavefile(t)
	writeFile(t, projectFS, "flow/node/main.zirr", "mod here = myapp.flow.node\n\nconst value = 1\n\nfn same() {\n\treturn here.value\n}\n")

	if err := runMain(t, projectFS, "flow/node/main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func TestModuleDeclarationsMustAgreeAcrossFiles(t *testing.T) {
	projectFS := projectWithCavefile(t)
	writeFile(t, projectFS, "flow/one.zirr", "mod myapp.flow\n")
	writeFile(t, projectFS, "flow/two.zirr", "mod myapp.other\n")

	orch := newTestOrchestra(t, projectFS, "myapp")
	err := orch.RunModulePath(context.Background(), "flow")
	if err == nil {
		t.Fatal("expected files of one module declaring different names to be rejected")
	}
	if !strings.Contains(err.Error(), "conflicting module declaration") && !strings.Contains(err.Error(), "wrong module name") {
		t.Errorf("error = %v, want it to report the disagreement", err)
	}
}

// A project with no Cavefile has no base to qualify a module path against, so a loose script keeps working whatever it declares.
func TestModuleDeclarationUncheckedWithoutCavefile(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, "sub/main.zirr", "mod anything\n")

	orch := newTestOrchestra(t, projectFS, "loose")
	if err := orch.RunFile(context.Background(), "sub/main.zirr"); err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func TestCavefileMustDeclareItsModule(t *testing.T) {
	projectFS := memfs.New()
	writeFile(t, projectFS, orchestra.DefaultCavefileName, "import cave\n\n@cave.Package()\ndata Deps {}\n")

	_, err := orchestra.New(orchestra.Config{
		ProjectFS:   projectFS,
		RegistryFS:  memfs.New(),
		PackageName: "my-app",
	})
	if err == nil {
		t.Fatal("expected a Cavefile without a mod declaration to be rejected")
	}
	if !strings.Contains(err.Error(), "declares no module") {
		t.Errorf("error = %v, want it to report the missing module declaration", err)
	}
	if !strings.Contains(err.Error(), "mod my_app") {
		t.Errorf("error = %v, want it to suggest a module path from the project directory", err)
	}
}

// A @cave.LanguageVersion this Zirric does not satisfy stops the project from being opened at all, so no command builds, formats or installs it.
func TestLanguageVersionRefusesOpeningTheProject(t *testing.T) {
	projectFS := languageVersionProject(t)

	_, err := newTestOrchestraOrError(projectFS, "myapp", false)
	var mismatch *cavefile.LanguageVersionError
	if !errors.As(err, &mismatch) {
		t.Fatalf("err = %v, want *cavefile.LanguageVersionError", err)
	}
	if mismatch.Package != "myapp" || mismatch.Dependency {
		t.Errorf("refused %+v, want the package myapp", mismatch)
	}
}

// Reading the manifest is the exception, since that report is how the refusal is explained.
func TestLanguageVersionStillReadable(t *testing.T) {
	projectFS := languageVersionProject(t)

	orch, err := newTestOrchestraOrError(projectFS, "myapp", true)
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}
	if got := orch.Cavefile().LanguageVersion; got != ">=9.0.0" {
		t.Errorf("LanguageVersion = %q, want %q", got, ">=9.0.0")
	}
}

// languageVersionProject needs a Zirric this is not, stamping a version the way a release build's ldflags would.
func languageVersionProject(t *testing.T) billy.Filesystem {
	t.Helper()
	previous := mv.Version
	mv.Version = "1.0.0"
	t.Cleanup(func() { mv.Version = previous })

	projectFS := memfs.New()
	writeFile(t, projectFS, orchestra.DefaultCavefileName, "mod myapp\n\nimport cave\n\n@cave.Package()\n@cave.LanguageVersion(\">=9.0.0\")\ndata Deps {}\n")
	return projectFS
}
