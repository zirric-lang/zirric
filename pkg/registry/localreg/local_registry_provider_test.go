package localreg_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/localreg"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

func TestDiscoverPackageVersionsResolvesExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greet.zirr"), "mod local\nconst greeting = \"hi\"\n")

	provider := localreg.New()
	pkgs, err := provider.DiscoverPackageVersions(context.Background(), "file://"+dir)
	if err != nil {
		t.Fatalf("discover package versions: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}

	resolved, err := pkgs[0].Resolve(context.Background())
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	mods, err := resolved.ResolveModules()
	if err != nil {
		t.Fatalf("resolve modules: %v", err)
	}

	wantURI := registry.LogicalURI(filepath.Base(dir))
	found := false
	for _, mod := range mods {
		if mod.URI() == wantURI {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected module %q, got %+v", wantURI, mods)
	}
}

func TestDiscoverPackageVersionsMissingDirectory(t *testing.T) {
	provider := localreg.New()
	pkgs, err := provider.DiscoverPackageVersions(context.Background(), "file:///does/not/exist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("expected no packages, got %+v", pkgs)
	}
}

func TestDiscoverPackageVersionsIgnoresNonFileScheme(t *testing.T) {
	provider := localreg.New()
	pkgs, err := provider.DiscoverPackageVersions(context.Background(), "https://example.com/pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("expected no packages, got %+v", pkgs)
	}
}

func TestDiscoverPackageVersionsFiltersByPredicate(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greet.zirr"), "mod local\n")

	provider := localreg.New()
	pkgs, err := provider.DiscoverPackageVersions(context.Background(), "file://"+dir, version.ParsePredicate(">1.0.0"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("expected the local version to fail a semver predicate, got %+v", pkgs)
	}
}

func TestDiscover(t *testing.T) {
	provider := localreg.New()
	pkgs, err := provider.Discover(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("expected no locally cached packages, got %+v", pkgs)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir all: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
