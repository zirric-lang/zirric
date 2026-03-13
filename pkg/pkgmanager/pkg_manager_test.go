package pkgmanager

import (
	"context"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

type mockRegistry struct {
	pkgs []registry.ResolvedPackage
}

func (m mockRegistry) Discover(ctx context.Context) ([]registry.ResolvedPackage, error) {
	return m.pkgs, nil
}

func (m mockRegistry) DiscoverPackageVersions(ctx context.Context, name string, preds ...version.Predicate) ([]registry.Package, error) {
	return nil, nil
}

type mockResolvedPackage struct {
	source string
	ver    version.Version
}

func (p mockResolvedPackage) Source() string           { return p.source }
func (p mockResolvedPackage) Version() version.Version { return p.ver }
func (p mockResolvedPackage) Resolve(ctx context.Context) (registry.ResolvedPackage, error) {
	return p, nil
}
func (p mockResolvedPackage) ResolveModules() ([]registry.ResolvedModule, error) { return nil, nil }

func TestPkgManagerInstallationTaskRun(t *testing.T) {
	ver := version.SemverVersion{Major: 1, Minor: 0, Patch: 0}
	dep := cavefile.Dependency{
		Package: cavefile.Package{Source: "example/pkg"},
		Predicates: []version.Predicate{
			version.Predicate{
				Comparison: version.ComparisonExact,
				Version:    ver,
			},
		},
	}
	pot := cavefile.Cavefile{Dependencies: []cavefile.Dependency{dep}}
	pkg := mockResolvedPackage{source: dep.Source, ver: ver}
	pm := &PackageManager{registries: []registry.Provider{mockRegistry{pkgs: []registry.ResolvedPackage{pkg}}}}
	task := pm.Install(pot)
	completed, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(completed) != 1 {
		t.Fatalf("expected 1 completed package, got %d", len(completed))
	}
	if completed[0].Source() != dep.Source {
		t.Fatalf("expected source %s, got %s", dep.Source, completed[0].Source())
	}
}
