package codefmt

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata golden files")

// Golden cases pair testdata/NAME.in.zirr with NAME.out.zirr. Regenerate with `task gen:golden`.
func TestGolden(t *testing.T) {
	ins, err := filepath.Glob("testdata/*.in.zirr")
	if err != nil {
		t.Fatal(err)
	}
	if len(ins) == 0 {
		t.Fatal("no golden inputs found")
	}

	for _, in := range ins {
		name := strings.TrimSuffix(filepath.Base(in), ".in.zirr")
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(in)
			if err != nil {
				t.Fatal(err)
			}
			src := string(raw)
			out := filepath.Join("testdata", name+".out.zirr")

			got, err := String(in, src, DefaultOptions())
			if err != nil {
				t.Fatalf("format: %v", err)
			}

			if *update {
				if err := os.WriteFile(out, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}

			wantRaw, err := os.ReadFile(out)
			if err != nil {
				t.Fatalf("missing golden (run with -update): %v", err)
			}
			if want := string(wantRaw); got != want {
				t.Errorf("formatted output differs\n--- want ---\n%s\n--- got ---\n%s", want, got)
			}

			// Formatting the golden again must be a no-op.
			again, err := String(out, got, DefaultOptions())
			if err != nil {
				t.Fatalf("reformat: %v", err)
			}
			if again != got {
				t.Errorf("not idempotent\n--- once ---\n%s\n--- twice ---\n%s", got, again)
			}

			// And no token or comment may have been harmed.
			if err := Equivalent(in, src, got); err != nil {
				t.Errorf("not equivalent: %v", err)
			}
		})
	}
}
