package fsmodule

import (
	"fmt"
	"io"
	"path/filepath"

	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"github.com/go-git/go-billy/v5"
)

type FSSource struct {
	name       string
	logicalURI registry.LogicalURI
	fs         billy.Filesystem
}

// URI implements registry.Source.
func (f *FSSource) URI() registry.LogicalURI {
	return f.logicalURI
}

// Read implements registry.Source.
func (f *FSSource) Read() ([]byte, error) {
	file, err := f.fs.Open(f.name)
	if err != nil {
		return nil, fmt.Errorf("failed to read source %q from %q, %w", f.logicalURI, filepath.Join(f.fs.Root(), f.name), err)
	}
	return io.ReadAll(file)
}
