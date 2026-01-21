package gitreg

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode"

	"code.knabel.dev/zirric-lang/zirric/registry"
	"code.knabel.dev/zirric-lang/zirric/version"
	"code.knabel.dev/zirric-lang/zirric/world"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/memory"
)

// GitRegistry is a registry that clones repositories in specific versions.
// It produces the following structure:
//
//	 <root>/
//	 └── <package>/
//		 └── <version>/
//			 ├── Cavefile
//	 		 └── <submodule>/
//
// TODO: Add some in-memory caching to avoid discovering remote versions on every run.
// TODO: How to handle git commits and branch names from remote?
type GitRegistry struct {
	rootfs billy.Filesystem

	remoteStorage     func() storage.Storer
	repositoryStorage func(worktree billy.Filesystem) (storage.Storer, error)
}

type Option func(*GitRegistry)

func New(regrootfs billy.Filesystem, opts ...Option) *GitRegistry {
	reg := &GitRegistry{
		rootfs: regrootfs,
	}
	WithDefaultOptions()(reg)

	for _, opt := range opts {
		opt(reg)
	}
	return reg
}

// Discover implements Registry
func (r *GitRegistry) Discover(ctx context.Context) ([]registry.ResolvedPackage, error) {
	repoEntries, err := r.rootfs.ReadDir(".")
	if err != nil && !errors.Is(err, world.ErrNotExist) {
		return nil, err
	}
	var errs []error
	var packages []registry.ResolvedPackage
	for _, repoEntry := range repoEntries {
		if !repoEntry.IsDir() {
			continue
		}
		unversionedPackageFS, err := r.rootfs.Chroot(repoEntry.Name())
		if err != nil {
			errs = append(errs, err)
			continue
		}
		locals, err := r.localPackageVersionClones(ctx, unversionedPackageFS, nil)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, local := range locals {
			packages = append(packages, local)
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return packages, nil
}

// DiscoverPackageVersions implements Registry
func (r *GitRegistry) DiscoverPackageVersions(ctx context.Context, repoUrl string, predicates ...version.Predicate) ([]registry.Package, error) {
	gitvs, err := r.remotePackageVersions(ctx, repoUrl, predicates)
	if err != nil {
		return nil, err
	}
	vs := make([]registry.Package, len(gitvs))
	for i, gitv := range gitvs {
		vs[i] = gitv
	}
	sort.Slice(vs, func(i, j int) bool {
		return !version.Less(vs[i].Version(), vs[j].Version())
	})
	return vs, nil
}

func (r *GitRegistry) remotePackageVersions(ctx context.Context, repoUrl string, predicates []version.Predicate) ([]registry.Package, error) {
	rem := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: git.DefaultRemoteName,
		URLs: []string{repoUrl},
	})

	refs, err := rem.ListContext(ctx, &git.ListOptions{
		PeelingOption: git.IgnorePeeled,
	})
	if err != nil {
		return nil, err
	}

	var pkgs []registry.Package
	for _, ref := range refs {
		if !ref.Name().IsTag() {
			continue
		}
		v := versionFromReference(ref)
		shouldAdd := true
		for _, predicate := range predicates {
			if !v.Matches(predicate) {
				shouldAdd = false
				break
			}
		}

		if shouldAdd {
			pkgs = append(pkgs, &remoteGitPackage{
				provider:     r,
				source:       repoUrl,
				gitReference: ref,
				version:      v,
			})
		}
	}
	return pkgs, nil
}

func (r *GitRegistry) localPackageVersionClones(ctx context.Context, unversionedPackageFS billy.Filesystem, preds []version.Predicate) ([]registry.ResolvedPackage, error) {
	var providables []registry.ResolvedPackage
	var errs []error
	versionEntries, err := unversionedPackageFS.ReadDir(".")
	if err != nil && !errors.Is(err, world.ErrNotExist) {
		errs = append(errs, err)
	}
	for _, versionEntry := range versionEntries {
		packagefs, err := unversionedPackageFS.Chroot(versionEntry.Name())
		if err != nil {
			errs = append(errs, err)
			continue
		}
		ps, err := r.localPackageVersionAliasesInWorktree(ctx, packagefs)
		if err != nil {
			errs = append(errs, err)
		}
		providables = append(providables, ps...)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return providables, nil
}

func (r *GitRegistry) localPackageVersionAliasesInWorktree(ctx context.Context, worktree billy.Filesystem) ([]registry.ResolvedPackage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	storer, err := r.repositoryStorage(worktree)
	if err != nil {
		return nil, err
	}

	repo, err := git.Open(storer, worktree)
	if err != nil {
		return nil, err
	}

	remote, err := repo.Remote(git.DefaultRemoteName)
	if err != nil {
		return nil, err
	}

	refs, err := r.relevantReferences(repo)
	if err != nil {
		return nil, err
	}

	packages := make([]registry.ResolvedPackage, len(refs))
	packageName := remote.Config().URLs[0]
	for i, ref := range refs {
		if !ref.Name().IsTag() {
			continue
		}
		packages[i] = &localGitPackage{
			fs: worktree,
			remoteGitPackage: &remoteGitPackage{
				provider:     r,
				source:       packageName,
				gitReference: ref,
				version:      versionFromReference(ref),
			},
		}
	}
	return packages, nil
}

func (r *GitRegistry) relevantReferences(repo *git.Repository) ([]*plumbing.Reference, error) {
	ref, err := repo.Head()
	if err != nil {
		return nil, err
	}

	versions := []*plumbing.Reference{ref}

	tags, err := repo.Tags()
	if err != nil {
		return nil, err
	}

	err = tags.ForEach(func(tag *plumbing.Reference) error {
		revHash, err := repo.ResolveRevision(plumbing.Revision(tag.Name()))
		if err != nil {
			return err
		}
		if *revHash == ref.Hash() {
			versions = append(versions, tag)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return versions, nil
}

func (r *GitRegistry) clone(ctx context.Context, pkg *remoteGitPackage) (registry.ResolvedPackage, error) {
	repoPath := path.Join(mangle(pkg.source), mangle(pkg.version.String()))
	err := r.rootfs.MkdirAll(repoPath, 0755)
	if err != nil {
		return nil, err
	}

	worktreefs, err := r.rootfs.Chroot(repoPath)
	if err != nil {
		return nil, err
	}
	storer, err := r.repositoryStorage(worktreefs)
	if err != nil {
		return nil, err
	}
	_, err = git.CloneContext(ctx, storer, worktreefs, &git.CloneOptions{
		URL:               pkg.source,
		RemoteName:        git.DefaultRemoteName,
		SingleBranch:      true,
		Depth:             1,
		ReferenceName:     pkg.gitReference.Name(),
		Tags:              git.AllTags,
		RecurseSubmodules: git.NoRecurseSubmodules,
	})
	if err != nil {
		return nil, err
	}
	return &localGitPackage{
		remoteGitPackage: pkg,
		fs:               worktreefs,
	}, nil
}

func versionFromReference(ref *plumbing.Reference) version.Version {
	return version.Parse(strings.TrimSuffix(ref.Name().Short(), "^{}"))
}

func mangle(str string) string {
	var mangled string
	for _, r := range str {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '.' {
			mangled += string(r)
		} else if len(mangled) > 0 && mangled[len(mangled)-1] != '-' {
			mangled += "-"
		}
	}

	h := sha256.New()
	h.Write([]byte(str))
	hash := fmt.Sprintf("%x", h.Sum(nil))
	return mangled + "-" + hash[:8]
}
