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
	"github.com/go-git/go-billy/v5/osfs"
)

func newOrchestra() (*orchestra.Orchestra, error) {
	const registryRoot = ".zirric"
	if err := os.MkdirAll(registryRoot, 0o755); err != nil {
		return nil, err
	}

	return orchestra.New(orchestra.Config{
		ProjectFS:      osfs.New("."),
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

func writeProjectFile(path string, contents []byte) error {
	if path == "" {
		return errors.New("path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
