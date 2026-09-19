package gitreg

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
	"code.knabel.dev/zirric-lang/zirric/pkg/world"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/memory"
)

// GitRegistry is a registry that clones repositories in specific versions.
// It produces the following structure:
//
//	 <root>/
//	 └── <canonical-package-uri>-<hash[:8]>/
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

		packages = append(packages, locals...)
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
	copy(vs, gitvs)

	sort.Slice(vs, func(i, j int) bool {
		return !version.Less(vs[i].Version(), vs[j].Version())
	})
	return vs, nil
}

// gitVersionCandidate pairs a resolvable git reference (never the symbolic HEAD itself) with the version it should be exposed as.
type gitVersionCandidate struct {
	ref *plumbing.Reference
	v   version.Version
}

func (r *GitRegistry) remotePackageVersions(ctx context.Context, repoUrl string, predicates []version.Predicate) ([]registry.Package, error) {
	rem := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: git.DefaultRemoteName,
		URLs: []string{repoUrl},
	})

	refs, err := rem.ListContext(ctx, &git.ListOptions{
		PeelingOption: git.IgnorePeeled,
	})
	if errors.Is(err, transport.ErrRepositoryNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	byName := make(map[plumbing.ReferenceName]*plumbing.Reference, len(refs))
	for _, ref := range refs {
		byName[ref.Name()] = ref
	}

	var hasTags bool
	for _, ref := range refs {
		if ref.Name().IsTag() {
			hasTags = true
			break
		}
	}

	var candidates []gitVersionCandidate
	for _, ref := range refs {
		switch {
		case ref.Name().IsTag(), ref.Name().IsBranch():
			// Branches are named candidates too (not just tags), so @cave.Version("main") resolves to a literal branch name, not only to a tagged release.
			candidates = append(candidates, gitVersionCandidate{ref, versionFromReference(ref)})
		case ref.Name() == plumbing.HEAD && !hasTags:
			// HEAD's own Hash is all-zero (unusable for CloneOptions.ReferenceName), so resolve it to its target branch and expose that as version "latest" — only when there are no tags, so a tagged repo keeps its existing behavior.
			if target, ok := byName[ref.Target()]; ok {
				candidates = append(candidates, gitVersionCandidate{target, version.ParseVerbal("latest")})
			}
		}
	}

	var pkgs []registry.Package
	for _, c := range candidates {
		shouldAdd := true
		for _, predicate := range predicates {
			if !c.v.Matches(predicate) {
				shouldAdd = false
				break
			}
		}

		if shouldAdd {
			pkgs = append(pkgs, &remoteGitPackage{
				provider:     r,
				source:       repoUrl,
				gitReference: c.ref,
				version:      c.v,
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
		ps, err := r.localPackageVersionAliasesInWorktree(ctx, packagefs, versionEntry.Name())
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

// localPackageVersionAliasesInWorktree lists every version name worktree can be discovered under: dirVersion (the on-disk cache directory's own name — how clone() named it, e.g. "latest" or "main") plus any tag pointing at the same commit. dirVersion must always be included even when no ref reproduces it (e.g. the synthetic "latest" HEAD alias has no git ref literally called "latest") — otherwise Discover() can never recognize this clone as already installed, and every later install re-clones into the same populated directory, failing with go-git's ErrRepositoryAlreadyExists.
func (r *GitRegistry) localPackageVersionAliasesInWorktree(ctx context.Context, worktree billy.Filesystem, dirVersion string) ([]registry.ResolvedPackage, error) {
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

	head, err := repo.Head()
	if err != nil {
		return nil, err
	}

	refs, err := r.relevantReferences(repo)
	if err != nil {
		return nil, err
	}

	packageName := remote.Config().URLs[0]
	newPackage := func(ref *plumbing.Reference, v version.Version) *localGitPackage {
		return &localGitPackage{
			fs: worktree,
			remoteGitPackage: &remoteGitPackage{
				provider:     r,
				source:       packageName,
				gitReference: ref,
				version:      v,
			},
		}
	}

	seen := make(map[string]bool)
	dv := version.Parse(dirVersion)
	packages := []registry.ResolvedPackage{newPackage(head, dv)}
	seen[dv.String()] = true

	for _, ref := range refs {
		if !ref.Name().IsTag() {
			continue
		}
		v := versionFromReference(ref)
		if seen[v.String()] {
			continue
		}
		seen[v.String()] = true
		packages = append(packages, newPackage(ref, v))
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
	repoPath := path.Join(packageDirName(pkg.source), pkg.version.String())
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

func packageDirName(source string) string {
	canonical := registry.CanonicalizeModuleSource(source)
	h := sha256.New()
	h.Write([]byte(source))
	hash := fmt.Sprintf("%x", h.Sum(nil))
	return string(canonical) + "-" + hash[:8]
}
