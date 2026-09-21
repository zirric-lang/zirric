package codefmt

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzFormat asserts the two properties that matter: formatting never panics, and whenever it succeeds it preserves every token and comment.
func FuzzFormat(f *testing.F) {
	seeds, _ := filepath.Glob("testdata/*.zirr")
	for _, seed := range seeds {
		if b, err := os.ReadFile(seed); err == nil {
			f.Add(string(b))
		}
	}
	f.Add("fn f() { return -1 }")
	f.Add("switch v {\ncase _:\n\tx\n}")

	f.Fuzz(func(t *testing.T, src string) {
		out, err := String("fuzz.zirr", src, DefaultOptions())
		if err != nil {
			if out != src {
				t.Fatalf("declined but did not return the input unchanged")
			}
			return
		}
		if err := Equivalent("fuzz.zirr", src, out); err != nil {
			t.Fatalf("formatted output is not equivalent: %v", err)
		}
	})
}
