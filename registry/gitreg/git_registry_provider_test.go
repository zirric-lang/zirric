package gitreg_test

import (
	"cmp"
	"context"
	"net/http"
	"slices"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"code.knabel.dev/zirric-lang/zirric/registry"
	"code.knabel.dev/zirric-lang/zirric/registry/gitreg"
	"code.knabel.dev/zirric-lang/zirric/version"
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
	pkgs, err := reg.DiscoverPackageVersions(ctx, "https://github.com/vknabel/lithia")
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

	if pkg.Source() != "https://github.com/vknabel/lithia" {
		t.Errorf("expected package name to be github.com/vknabel/lithia, got %s", pkg.Source())
	}
	if pkg.Version().String() != "0.0.19" {
		t.Errorf("expected package version to be v0.0.19, got %s", pkg.Version())
	}

	localPkg, err := pkg.Resolve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if localPkg.Source() != "https://github.com/vknabel/lithia" {
		t.Errorf("expected package local path to be /github.com/vknabel/lithia, got %s", localPkg.Source())
	}
	if localPkg.Version().String() != "0.0.19" {
		t.Errorf("expected package version to be v0.0.19, got %s", localPkg.Version())
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

// 	versions, err := reg.DiscoverPackageVersions(ctx, "https://github.com/vknabel/lithia")
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
// 	pkg, err := reg.ResolveLatest(ctx, "https://github.com/vknabel/lithia", predicate)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if pkg.Name != "https://github.com/vknabel/lithia" {
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
// 		if pkg.Name != "https://github.com/vknabel/lithia" {
// 			t.Errorf("expected package %d name to be https://github.com/vknabel/lithia, got %s", i, packages[i].Name)
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
