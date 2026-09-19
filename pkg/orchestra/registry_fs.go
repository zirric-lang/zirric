package orchestra

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
)

// DefaultRegistryFS returns the shared on-disk package registry cache: $ZIRRIC_REGISTRY, or $ZIRRIC_PATH/registry, or ~/.zirric/registry.
func DefaultRegistryFS() (billy.Filesystem, error) {
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
