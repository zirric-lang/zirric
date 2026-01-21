package pkgmanager

import (
	"context"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/cavefile"
	"code.knabel.dev/zirric-lang/zirric/registry"
	"code.knabel.dev/zirric-lang/zirric/version"
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
	dep := cavefile.Dependency{Source: "example/pkg", Predicate: version.Predicate{Comparison: version.ComparisonExact, Version: ver}}
	pot := cavefile.Cavefile{Dependencies: []cavefile.Dependency{dep}}
	pkg := mockResolvedPackage{source: dep.Source, ver: ver}
	pm := &PackageManager{registries: []registry.Provider{mockRegistry{pkgs: []registry.ResolvedPackage{pkg}}}}
	task := pm.Install(pot)
	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(task.completed) != 1 {
		t.Fatalf("expected 1 completed package, got %d", len(task.completed))
	}
	if task.completed[0].Source() != dep.Source {
		t.Fatalf("expected source %s, got %s", dep.Source, task.completed[0].Source())
	}
}
