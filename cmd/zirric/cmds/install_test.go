package cmds

import (
	"context"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/pkgmanager"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

func TestPrintInstalled(t *testing.T) {
	var buf strings.Builder
	evt := pkgmanager.InstallEvent{
		Dependency: cavefile.Dependency{Package: cavefile.Package{Name: "tests"}},
		Package:    &stubResolvedPackage{source: "https://example.com/pkg"},
	}
	printInstalled(&buf, evt)

	want := "✓ tests (https://example.com/pkg)\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

type stubResolvedPackage struct {
	source string
}

func (p *stubResolvedPackage) Source() string           { return p.source }
func (p *stubResolvedPackage) Version() version.Version { return version.Parse("local") }
func (p *stubResolvedPackage) Resolve(context.Context) (registry.ResolvedPackage, error) {
	return p, nil
}
func (p *stubResolvedPackage) ResolveModules() ([]registry.ResolvedModule, error) { return nil, nil }
