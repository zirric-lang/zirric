package codefmt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// corpusFiles collects every real Zirric source, Cavefiles included.
func corpusFiles(t *testing.T) []string {
	t.Helper()

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "site":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".zirr" || d.Name() == "Cavefile" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no corpus files found")
	}
	return paths
}

func TestCorpusInvariants(t *testing.T) {
	opts := DefaultOptions()
	for _, path := range corpusFiles(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			src := string(raw)

			// The scanner must account for every byte, or nothing else is safe.
			if _, err := scan(path, src); err != nil {
				t.Fatalf("scan: %v", err)
			}

			out, err := String(path, src, opts)
			if err != nil {
				t.Fatalf("format: %v", err)
			}
			if err := Equivalent(path, src, out); err != nil {
				t.Fatalf("not equivalent: %v", err)
			}

			again, err := String(path, out, opts)
			if err != nil {
				t.Fatalf("reformat: %v", err)
			}
			if again != out {
				t.Errorf("not idempotent:\n%s", firstLineDiff(out, again))
			}
		})
	}
}

func firstLineDiff(a, b string) string {
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < len(al) && i < len(bl); i++ {
		if al[i] != bl[i] {
			return "line " + itoa(i+1) + ":\n  first  %q\n  second %q" +
				"\n  " + quote(al[i]) + "\n  " + quote(bl[i])
		}
	}
	return "line counts differ: " + itoa(len(al)) + " vs " + itoa(len(bl))
}

func quote(s string) string { return "\"" + s + "\"" }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
