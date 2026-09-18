package gitreg_test

import (
	"cmp"
	"context"
	"net/http"
	"slices"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/gitreg"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
	"github.com/go-git/go-billy/v5/memfs"
)

const (
	zirricGitRepo = "https://code.knabel.dev/zirric-lang/zirric-package-fixture"
	zirricVersion = "0.0.1"
)

func TestIntegrationGitRegistryResolveLatestZirricInMemory(t *testing.T) {
	resp, err := http.Get("https://github.com")
	if err != nil {
		t.Skipf("unable to connect to GitHub. Are you connected to the internet? %s", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		t.Skipf("invalid response from GitHub (%s)", resp.Status)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reg := gitreg.New(memfs.New())
	pkgs, err := reg.DiscoverPackageVersions(ctx, zirricGitRepo)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) < 1 {
		t.Errorf("expected at least one package")
	}
	slices.SortFunc(pkgs, func(lhs registry.Package, rhs registry.Package) int {
		res := cmp.Compare(lhs.Source(), rhs.Source())
		if res != 0 {
			return res
		}
		return version.Compare(lhs.Version(), rhs.Version())
	})
	pkg := pkgs[0]

	if pkg.Source() != zirricGitRepo {
		t.Errorf("expected package name to be code.knabel.dev/zirric-lang/zirric-package-fixture, got %s", pkg.Source())
	}
	if pkg.Version().String() != zirricVersion {
		t.Errorf("expected package version to be v%s, got %s", zirricVersion, pkg.Version())
	}

	localPkg, err := pkg.Resolve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if localPkg.Source() != zirricGitRepo {
		t.Errorf("expected package local path to be /code.knabel.dev/zirric-lang/zirric-package-fixture, got %s", localPkg.Source())
	}
	if localPkg.Version().String() != zirricVersion {
		t.Errorf("expected package version to be v%s, got %s", zirricVersion, localPkg.Version())
	}

	if pkg.Source() != localPkg.Source() {
		t.Errorf("expected package source to match local package source, got %s and %s", pkg.Source(), localPkg.Source())
	}
}

func TestDiscoverPackageVersionsNoMatchForNonGitLocalPath(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reg := gitreg.New(memfs.New())
	pkgs, err := reg.DiscoverPackageVersions(ctx, "file://"+t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("expected no packages, got %+v", pkgs)
	}
}

// func TestIntegrationGitRegistryResolveSecondLatestZirric(t *testing.T) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	reg := gitreg.NewGitRegistry(memfs.New())

// 	packages, err := reg.Discover(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if len(packages) > 0 {
// 		t.Errorf("expected no packages, got %d", len(packages))
// 	}

// 	versions, err := reg.DiscoverPackageVersions(ctx, zirricGitRepo)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if len(versions) < 32 {
// 		t.Errorf("expected no versions, got %d: %v", len(versions), versions)
// 	}

// 	packages, err = reg.Discover(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if len(packages) > 0 {
// 		t.Errorf("expected no packages, got %d", len(packages))
// 	}

// 	predicate := version.Predicate{
// 		Comparison: version.ComparisonUpToNextMajor,
// 		Version:    version.SemverVersion{Major: 0, Minor: 0, Patch: 1},
// 	}
// 	pkg, err := reg.ResolveLatest(ctx, zirricGitRepo, predicate)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if pkg.Name != zirricGitRepo {
// 		t.Errorf("expected package name to be github.com/vknabel/lithia, got %s", pkg.Name)
// 	}

// 	packages, err = reg.Discover(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if len(packages) < 1 {
// 		t.Errorf("expected at least one package, got %d, %v", len(packages), packages)
// 	}
// 	for i, pkg := range packages {
// 		if pkg.Name != zirricGitRepo {
// 			t.Errorf("expected package %d name to be %s, got %s", i, zirricGitRepo, packages[i].Name)
// 		}
// 	}
// }

// func (r *GitProvider) pickLatestVersion(versions []version.Version) (int, version.Version) {
// 	mapping := make([]int, len(versions))
// 	for i := range mapping {
// 		mapping[i] = i
// 	}
// 	sort.Slice(mapping, func(i, j int) bool {
// 		return !version.Less(versions[mapping[i]], versions[mapping[j]])
// 	})

// 	for i := range mapping {
// 		candidate := versions[mapping[i]]
// 		if !candidate.IsPreRelease() {
// 			return mapping[i], candidate
// 		}
// 	}

// 	for i := range mapping {
// 		candidate := versions[mapping[i]]
// 		return mapping[i], candidate
// 	}

// 	return -1, nil
// }
