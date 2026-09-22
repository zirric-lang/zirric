package gitreg_test

import (
	"context"
	"os"
	"testing"
	"time"

	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/gitreg"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// initLocalRepoNoTags creates a real, local git repository with one commit and no tags — the shape of a fresh project that hasn't cut a release yet. Returns the repo path and its default branch's short name (e.g. "master").
func initLocalRepoNoTags(t *testing.T) (dir string, branch string) {
	t.Helper()
	dir = t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("plain init: %v", err)
	}
	if err := os.WriteFile(dir+"/greet.zirr", []byte("mod dep\nfn greet() {}\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	if _, err := wt.Add("greet.zirr"); err != nil {
		t.Fatalf("add: %v", err)
	}
	sig := &object.Signature{Name: "test", Email: "test@test.com", When: time.Now()}
	if _, err := wt.Commit("init", &git.CommitOptions{Author: sig}); err != nil {
		t.Fatalf("commit: %v", err)
	}
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	return dir, head.Name().Short()
}

// TestDiscoverPackageVersionsLatestWithoutTags is a regression test: @cave.Version("latest") against a tag-less repo used to resolve to zero versions, since remotePackageVersions only ever considered tag refs.
func TestDiscoverPackageVersionsLatestWithoutTags(t *testing.T) {
	remote, _ := initLocalRepoNoTags(t)
	source := "file://" + remote

	registryFS := osfs.New(t.TempDir())
	reg := gitreg.New(registryFS)
	ctx := context.Background()

	pkgs, err := reg.DiscoverPackageVersions(ctx, source, version.ParsePredicate("latest"))
	if err != nil {
		t.Fatalf("discover package versions: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected exactly one package for \"latest\", got %d: %+v", len(pkgs), pkgs)
	}
	if pkgs[0].Version().String() != "latest" {
		t.Errorf("expected version %q, got %q", "latest", pkgs[0].Version())
	}

	resolved, err := pkgs[0].Resolve(ctx)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Source() != source {
		t.Errorf("resolved.Source() = %q, want %q", resolved.Source(), source)
	}
}

// TestDiscoverPackageVersionsWithoutPredicateNoTags checks that a tag-less repo's single commit is resolvable both by its branch's own name and by the synthesized "latest" alias.
func TestDiscoverPackageVersionsWithoutPredicateNoTags(t *testing.T) {
	remote, branch := initLocalRepoNoTags(t)
	source := "file://" + remote

	registryFS := osfs.New(t.TempDir())
	reg := gitreg.New(registryFS)
	ctx := context.Background()

	pkgs, err := reg.DiscoverPackageVersions(ctx, source)
	if err != nil {
		t.Fatalf("discover package versions: %v", err)
	}
	versions := make(map[string]bool, len(pkgs))
	for _, pkg := range pkgs {
		versions[pkg.Version().String()] = true
	}
	if !versions[branch] {
		t.Errorf("expected a candidate named after the branch %q, got %v", branch, versions)
	}
	if !versions["latest"] {
		t.Errorf("expected a \"latest\" candidate, got %v", versions)
	}
}

// TestDiscoverPackageVersionsBranchNameWithoutTags is a regression test: @cave.Version("<branch>") (e.g. "main") against a repo with no tags used to resolve to zero versions, since branches were never considered version candidates.
func TestDiscoverPackageVersionsBranchNameWithoutTags(t *testing.T) {
	remote, branch := initLocalRepoNoTags(t)
	source := "file://" + remote

	registryFS := osfs.New(t.TempDir())
	reg := gitreg.New(registryFS)
	ctx := context.Background()

	pkgs, err := reg.DiscoverPackageVersions(ctx, source, version.ParsePredicate(branch))
	if err != nil {
		t.Fatalf("discover package versions: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected exactly one package for branch %q, got %d: %+v", branch, len(pkgs), pkgs)
	}
	if pkgs[0].Version().String() != branch {
		t.Errorf("expected version %q, got %q", branch, pkgs[0].Version())
	}

	resolved, err := pkgs[0].Resolve(ctx)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Source() != source {
		t.Errorf("resolved.Source() = %q, want %q", resolved.Source(), source)
	}
}

// TestReinstallBranchClonedPackageDoesNotFail is a regression test: a branch-based clone (e.g. under the synthetic "latest" label, which has no git ref of that literal name) was invisible to Discover(), so InstallationTask's tryResolveAvailable never recognized it as already installed, and every subsequent "zirric install" re-cloned into the same populated directory and failed with go-git's ErrRepositoryAlreadyExists. This mirrors production exactly: a fresh orchestra/resolver per call, like separate CLI invocations sharing one on-disk registry.
func TestReinstallBranchClonedPackageDoesNotFail(t *testing.T) {
	remote, branch := initLocalRepoNoTags(t)
	source := "file://" + remote

	registryRoot := t.TempDir()
	projectRoot := t.TempDir()
	if err := os.WriteFile(projectRoot+"/main.zirr", []byte("mod proj\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	writeCavefile := func(predicate string) {
		content := `mod proj

import cave

@cave.Package()
data Deps {
  @cave.Git("` + source + `")
  @cave.Version("` + predicate + `")
  dep
}
`
		if err := os.WriteFile(projectRoot+"/Cavefile", []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	install := func(predicate string) {
		t.Helper()
		writeCavefile(predicate)
		orch, err := orchestra.New(orchestra.Config{
			ProjectFS:   osfs.New(projectRoot),
			RegistryFS:  osfs.New(registryRoot),
			PackageName: "proj",
		})
		if err != nil {
			t.Fatalf("new orchestra: %v", err)
		}
		resolver, err := orch.NewResolver()
		if err != nil {
			t.Fatalf("new resolver: %v", err)
		}
		if _, err := resolver.EnsureInstalled(context.Background()); err != nil {
			t.Fatalf("ensure installed (%s): %v", predicate, err)
		}
	}

	install("latest")
	install("latest") // simulates running `zirric install` again unchanged
	install(branch)   // simulates switching @cave.Version("latest") to @cave.Version(branch)
	install(branch)
}

// TestDiscoverPackageVersionsWithTagsHasNoPhantomLatest checks that the HEAD/"latest" synthesis is scoped to tag-less repos: a repo with a tag never grows an extra "latest" candidate alongside it, even though its branch is still resolvable by its own name.
func TestDiscoverPackageVersionsWithTagsHasNoPhantomLatest(t *testing.T) {
	remote := initLocalRepo(t)
	source := "file://" + remote

	registryFS := osfs.New(t.TempDir())
	reg := gitreg.New(registryFS)
	ctx := context.Background()

	pkgs, err := reg.DiscoverPackageVersions(ctx, source)
	if err != nil {
		t.Fatalf("discover package versions: %v", err)
	}
	for _, pkg := range pkgs {
		if pkg.Version().String() == "latest" {
			t.Errorf("expected no \"latest\" candidate for a repo with tags, got %+v", pkgs)
		}
	}

	tagged, err := reg.DiscoverPackageVersions(ctx, source, version.ParsePredicate("1.0.0"))
	if err != nil {
		t.Fatalf("discover package versions: %v", err)
	}
	if len(tagged) != 1 {
		t.Fatalf("expected exactly one package for tag 1.0.0, got %d: %+v", len(tagged), tagged)
	}
}
