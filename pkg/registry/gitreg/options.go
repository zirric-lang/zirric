package gitreg

import (
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
)

func WithDefaultOptions() func(*GitRegistry) {
	return func(reg *GitRegistry) {
		WithPlainRepositoryStorage()(reg)
		WithRemoteStorageInMemory()(reg)
	}
}

func WithRemoteStorage(newStorer func() storage.Storer) func(*GitRegistry) {
	return func(reg *GitRegistry) {
		reg.remoteStorage = newStorer
	}
}

func WithRemoteStorageInMemory() func(*GitRegistry) {
	return WithRemoteStorage(func() storage.Storer {
		return memory.NewStorage()
	})
}

func WithRepositoryStorage(newStorer func(worktree billy.Filesystem) (storage.Storer, error)) func(*GitRegistry) {
	return func(reg *GitRegistry) {
		reg.repositoryStorage = newStorer
	}
}

func WithPlainRepositoryStorage() func(*GitRegistry) {
	return WithRepositoryStorage(func(worktree billy.Filesystem) (storage.Storer, error) {
		dot, err := worktree.Chroot(git.GitDirName)
		if err != nil {
			return nil, err
		}
		return filesystem.NewStorage(dot, cache.NewObjectLRUDefault()), nil
	})
}
