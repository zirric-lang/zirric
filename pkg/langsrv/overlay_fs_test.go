package langsrv

import (
	"io"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
)

func TestOverlayFSReadsOverlayAndDisk(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "root/disk.zirr", "disk")

	docs := newDocumentStore()
	docs.Open("root/unsaved.zirr", 1, "mem")

	fs := newOverlayFS(base, docs)

	memFile, err := fs.Open("root/unsaved.zirr")
	if err != nil {
		t.Fatalf("open overlay file: %v", err)
	}
	memContents := readAll(t, memFile)
	if memContents != "mem" {
		t.Fatalf("unexpected overlay content: %q", memContents)
	}

	diskFile, err := fs.Open("root/disk.zirr")
	if err != nil {
		t.Fatalf("open disk file: %v", err)
	}
	diskContents := readAll(t, diskFile)
	if diskContents != "disk" {
		t.Fatalf("unexpected disk content: %q", diskContents)
	}
}

func TestOverlayFSReadDirIncludesOverlay(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "root/disk.zirr", "disk")

	docs := newDocumentStore()
	docs.Open("root/unsaved.zirr", 1, "mem")

	fs := newOverlayFS(base, docs)
	entries, err := fs.ReadDir("root")
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}

	names := make(map[string]bool, len(entries))
	for _, entry := range entries {
		names[entry.Name()] = true
	}
	if !names["disk.zirr"] {
		t.Fatalf("expected disk.zirr in ReadDir")
	}
	if !names["unsaved.zirr"] {
		t.Fatalf("expected unsaved.zirr in ReadDir")
	}
}

func TestOverlayFSStatUsesOverlay(t *testing.T) {
	base := memfs.New()
	docs := newDocumentStore()
	docs.Open("root/unsaved.zirr", 1, "mem")

	fs := newOverlayFS(base, docs)
	info, err := fs.Stat("root/unsaved.zirr")
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != 3 {
		t.Fatalf("expected size 3, got %d", info.Size())
	}
}

func writeFile(t *testing.T, fs billy.Filesystem, path string, contents string) {
	t.Helper()
	if err := fs.MkdirAll("root", 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	file, err := fs.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := file.Write([]byte(contents)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func readAll(t *testing.T, file billy.File) string {
	t.Helper()
	defer func() {
		_ = file.Close()
	}()
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(data)
}
