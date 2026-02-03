package cmds

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	billyutil "github.com/go-git/go-billy/v5/util"
)

func newOrchestra(projectFS billy.Filesystem) (*orchestra.Orchestra, error) {
	zirricPath, _ := os.LookupEnv("ZIRRIC_PATH")
	if zirricPath == "" {
		zirricPath = "~/.zirric"
	}

	registryRoot := filepath.Join(zirricPath, "registry")
	if override, ok := os.LookupEnv("ZIRRIC_REGISTRY"); ok {
		registryRoot = override
	}
	if err := os.MkdirAll(registryRoot, 0o755); err != nil {
		return nil, err
	}

	return orchestra.New(orchestra.Config{
		ProjectFS:      projectFS,
		ProjectBaseURI: registry.CanonicalizeModuleSource("project"),
		RegistryFS:     osfs.New(registryRoot),
	})
}

func runPath(ctx context.Context, orch *orchestra.Orchestra, path string, cave cavefile.Cavefile) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return orch.RunModulePath(ctx, filepath.ToSlash(path), cave)
	}
	return orch.RunFile(ctx, filepath.ToSlash(path), cave)
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
