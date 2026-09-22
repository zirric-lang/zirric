package cmds

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/diag"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	billyutil "github.com/go-git/go-billy/v5/util"
)

func newOrchestra(projectFS billy.Filesystem, packageName string) (*orchestra.Orchestra, error) {
	return newOrchestraWith(projectFS, packageName, false)
}

// newReadingOrchestra opens a project only to report what its Cavefile says, which must work even for a package this Zirric cannot build — that report is how the refusal is explained.
func newReadingOrchestra(projectFS billy.Filesystem, packageName string) (*orchestra.Orchestra, error) {
	return newOrchestraWith(projectFS, packageName, true)
}

func newOrchestraWith(projectFS billy.Filesystem, packageName string, ignoreLanguageVersion bool) (*orchestra.Orchestra, error) {
	registryFS, err := orchestra.DefaultRegistryFS()
	if err != nil {
		return nil, err
	}

	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:             projectFS,
		RegistryFS:            registryFS,
		PackageName:           packageName,
		CavefilePath:          cavefilePath,
		IgnoreLanguageVersion: ignoreLanguageVersion,
	})
	if err != nil {
		return nil, err
	}
	// A warning does not stop a build, but it still has to be seen, so it is written where errors go rather than mixed into a program's own output.
	orch.ReportWarnings = func(warnings analyzer.AnalysisErrors) {
		fmt.Fprintln(os.Stderr, "Warning:", diag.Render(warnings, readSourceForDiagnostic))
	}
	return orch, nil
}

// cwdFS returns a filesystem rooted at the current working directory's absolute path.
// osfs.New(".") would work too, but its Root() stays the literal string ".", which breaks anything
// deriving a project identity from it (e.g. ParseCavefile's fallback package name).
func cwdFS() (billy.Filesystem, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return osfs.New(wd), nil
}

func currentDirPackageName() string {
	wd, err := os.Getwd()
	if err != nil {
		return "app"
	}
	return filepath.Base(wd)
}

func runPath(ctx context.Context, orch *orchestra.Orchestra, path string, args []string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return orch.RunModulePath(ctx, filepath.ToSlash(path))
	}
	return orch.RunFileWithArgs(ctx, filepath.ToSlash(path), args)
}

func writeProjectFile(projectFS billy.Filesystem, path string, contents []byte) error {
	if path == "" {
		return errors.New("path is required")
	}

	if err := projectFS.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := billyutil.WriteFile(projectFS, path, contents, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
