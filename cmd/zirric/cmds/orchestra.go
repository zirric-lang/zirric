package cmds

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	billyutil "github.com/go-git/go-billy/v5/util"
)

func defaultRegistryFS() (billy.Filesystem, error) {
	zirricPath, _ := os.LookupEnv("ZIRRIC_PATH")
	if zirricPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not determine default ZIRRIC_PATH: %w", err)
		}
		zirricPath = filepath.Join(home, ".zirric")
	}

	registryRoot := filepath.Join(zirricPath, "registry")
	if override, ok := os.LookupEnv("ZIRRIC_REGISTRY"); ok {
		registryRoot = override
	}
	if err := os.MkdirAll(registryRoot, 0o755); err != nil {
		return nil, err
	}
	return osfs.New(registryRoot), nil
}

func newOrchestra(projectFS billy.Filesystem, packageName string) (*orchestra.Orchestra, error) {
	registryFS, err := defaultRegistryFS()
	if err != nil {
		return nil, err
	}

	return orchestra.New(orchestra.Config{
		ProjectFS:    projectFS,
		RegistryFS:   registryFS,
		PackageName:  packageName,
		CavefilePath: cavefilePath,
	})
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
