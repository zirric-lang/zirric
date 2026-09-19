package gitreg_test

import (
	"context"
	"os"
	"testing"
	"time"

	"code.knabel.dev/zirric-lang/zirric/pkg/registry/gitreg"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// initLocalRepo creates a real, local (network-free) git repository with one commit and one tag, usable as a "file://" dependency source.
func initLocalRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
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
	hash, err := wt.Commit("init", &git.CommitOptions{Author: sig})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := repo.CreateTag("v1.0.0", hash, nil); err != nil {
		t.Fatalf("create tag: %v", err)
	}
	return dir
}

// TestDiscoverAfterCloneDoesNotPanic is a regression test: localPackageVersionAliasesInWorktree pre-sized its result slice to len(refs) but only filled tag slots, leaving relevantReferences' HEAD-ref slot as a nil registry.ResolvedPackage that panicked any caller calling a method on it.
func TestDiscoverAfterCloneDoesNotPanic(t *testing.T) {
	remote := initLocalRepo(t)
	source := "file://" + remote

	registryFS := osfs.New(t.TempDir())
	reg := gitreg.New(registryFS)

	ctx := context.Background()
	versions, err := reg.DiscoverPackageVersions(ctx, source)
	if err != nil {
		t.Fatalf("discover package versions: %v", err)
	}
	if len(versions) == 0 {
		t.Fatal("expected at least one version")
	}
	if _, err := versions[0].Resolve(ctx); err != nil {
		t.Fatalf("resolve: %v", err)
	}

	pkgs, err := reg.Discover(ctx)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("expected at least one discovered package")
	}
	for i, pkg := range pkgs {
		if pkg == nil {
			t.Fatalf("pkgs[%d] is nil", i)
		}
		if pkg.Source() != source {
			t.Errorf("pkgs[%d].Source() = %q, want %q", i, pkg.Source(), source)
		}
	}
}
