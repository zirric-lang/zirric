package langsrv

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-git/go-billy/v5"
)

type overlayFS struct {
	base billy.Filesystem
	docs *documentStore
}

func newOverlayFS(base billy.Filesystem, docs *documentStore) billy.Filesystem {
	return &overlayFS{base: base, docs: docs}
}

func (fs *overlayFS) Create(filename string) (billy.File, error) {
	return fs.base.Create(filename)
}

func (fs *overlayFS) Open(filename string) (billy.File, error) {
	filename = cleanPath(filename)
	if snapshot, ok := fs.docs.Snapshot(filename); ok {
		return newMemFile(filename, snapshot.Text), nil
	}

	return fs.base.Open(filename)
}

func (fs *overlayFS) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	if flag&(os.O_WRONLY|os.O_RDWR|os.O_TRUNC|os.O_CREATE|os.O_APPEND) != 0 {
		return fs.base.OpenFile(filename, flag, perm)
	}

	return fs.Open(filename)
}

func (fs *overlayFS) Stat(filename string) (os.FileInfo, error) {
	filename = cleanPath(filename)
	if snapshot, ok := fs.docs.Snapshot(filename); ok {
		return memFileInfo{name: filepath.Base(filename), size: int64(len(snapshot.Text)), modTime: snapshot.ModTime}, nil
	}

	return fs.base.Stat(filename)
}

func (fs *overlayFS) Rename(oldpath, newpath string) error {
	return fs.base.Rename(oldpath, newpath)
}

func (fs *overlayFS) Remove(filename string) error {
	return fs.base.Remove(filename)
}

func (fs *overlayFS) Join(elem ...string) string {
	return fs.base.Join(elem...)
}

func (fs *overlayFS) TempFile(dir, prefix string) (billy.File, error) {
	return fs.base.TempFile(dir, prefix)
}

func (fs *overlayFS) ReadDir(path string) ([]os.FileInfo, error) {
	path = cleanPath(path)

	entries, err := fs.base.ReadDir(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	entryMap := make(map[string]os.FileInfo, len(entries))
	for _, entry := range entries {
		entryMap[entry.Name()] = entry
	}

	for _, docPath := range fs.docs.Paths() {
		if filepath.Dir(docPath) != path {
			continue
		}

		base := filepath.Base(docPath)
		if _, exists := entryMap[base]; exists {
			continue
		}

		if snapshot, ok := fs.docs.Snapshot(docPath); ok {
			entryMap[base] = memFileInfo{
				name:    base,
				size:    int64(len(snapshot.Text)),
				modTime: snapshot.ModTime,
			}
		}
	}

	if len(entryMap) == 0 && err != nil {
		return nil, err
	}

	merged := make([]os.FileInfo, 0, len(entryMap))
	for _, entry := range entryMap {
		merged = append(merged, entry)
	}

	// ReadDir must return entries sorted by filename.
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Name() < merged[j].Name()
	})
	return merged, nil
}

func (fs *overlayFS) MkdirAll(filename string, perm os.FileMode) error {
	return fs.base.MkdirAll(filename, perm)
}

func (fs *overlayFS) Lstat(filename string) (os.FileInfo, error) {
	return fs.base.Lstat(filename)
}

func (fs *overlayFS) Symlink(target, link string) error {
	return fs.base.Symlink(target, link)
}

func (fs *overlayFS) Readlink(link string) (string, error) {
	return fs.base.Readlink(link)
}

func (fs *overlayFS) Chroot(path string) (billy.Filesystem, error) {
	chroot, ok := fs.base.(billy.Chroot)
	if !ok {
		return nil, billy.ErrNotSupported
	}

	base, err := chroot.Chroot(path)
	if err != nil {
		return nil, err
	}

	return &overlayFS{base: base, docs: fs.docs}, nil
}

func (fs *overlayFS) Root() string {
	chroot, ok := fs.base.(billy.Chroot)
	if !ok {
		return ""
	}

	return chroot.Root()
}

func (fs *overlayFS) Capabilities() billy.Capability {
	capable, ok := fs.base.(billy.Capable)
	if !ok {
		return billy.DefaultCapabilities
	}
	return capable.Capabilities()
}

type memFile struct {
	name   string
	reader *bytes.Reader
}

func newMemFile(name string, contents string) *memFile {
	return &memFile{
		name:   name,
		reader: bytes.NewReader([]byte(contents)),
	}
}

func (f *memFile) Name() string {
	return f.name
}

func (f *memFile) Read(p []byte) (int, error) {
	return f.reader.Read(p)
}

func (f *memFile) ReadAt(p []byte, off int64) (int, error) {
	return f.reader.ReadAt(p, off)
}

func (f *memFile) Seek(offset int64, whence int) (int64, error) {
	return f.reader.Seek(offset, whence)
}

func (f *memFile) Write(_ []byte) (int, error) {
	return 0, billy.ErrReadOnly
}

func (f *memFile) Close() error {
	return nil
}

func (f *memFile) Lock() error {
	return billy.ErrNotSupported
}

func (f *memFile) Unlock() error {
	return billy.ErrNotSupported
}

func (f *memFile) Truncate(_ int64) error {
	return billy.ErrReadOnly
}

type memFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (fi memFileInfo) Name() string {
	return fi.name
}

func (fi memFileInfo) Size() int64 {
	return fi.size
}

func (fi memFileInfo) Mode() os.FileMode {
	return 0o644
}

func (fi memFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi memFileInfo) IsDir() bool {
	return false
}

func (fi memFileInfo) Sys() any {
	return nil
}
