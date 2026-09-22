package cmds

import (
	"errors"
	"io"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/codefmt"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	billyutil "github.com/go-git/go-billy/v5/util"
	mv "github.com/metal-stack/v"
)

const unformatted = "fn f() {\n      const x = 1+2\n}\n"
const formatted = "fn f() {\n\tconst x = 1 + 2\n}\n"

func fmtFixture(t *testing.T) billy.Filesystem {
	t.Helper()
	fs := memfs.New()
	write := func(path, content string) {
		t.Helper()
		if err := billyutil.WriteFile(fs, path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("main.zirr", unformatted)
	write("clean.zirr", formatted)
	write("Cavefile", "mod fmtfixture\n\nimport cave\n\n@cave.Package()\ndata Dependencies {\n      prelude\n}\n")
	write("nested/deep.zirr", unformatted)
	write("notes.txt", "not zirric\n")
	return fs
}

func runFmtTest(t *testing.T, fs billy.Filesystem, paths []string, flags fmtFlags, stdin string) (bool, string, string) {
	t.Helper()
	return runFmtExcl(t, fs, paths, flags, nil, stdin)
}

func runFmtExcl(t *testing.T, fs billy.Filesystem, paths []string, flags fmtFlags, excludes codefmt.Excludes, stdin string) (bool, string, string) {
	t.Helper()
	var out, errOut strings.Builder
	drift, err := runFmt(fs, paths, flags, excludes, strings.NewReader(stdin), &out, &errOut)
	if err != nil {
		t.Fatalf("runFmt: %v", err)
	}
	return drift, out.String(), errOut.String()
}

func readFile(t *testing.T, fs billy.Filesystem, path string) string {
	t.Helper()
	b, err := billyutil.ReadFile(fs, path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRunFmt_WritesInPlaceByDefault(t *testing.T) {
	fs := fmtFixture(t)
	drift, out, _ := runFmtTest(t, fs, nil, fmtFlags{}, "")

	if !drift {
		t.Error("expected drift to be reported")
	}
	if out != "" {
		t.Errorf("expected no output when writing, got %q", out)
	}
	if got := readFile(t, fs, "main.zirr"); got != formatted {
		t.Errorf("main.zirr not formatted: %q", got)
	}
	if got := readFile(t, fs, "nested/deep.zirr"); got != formatted {
		t.Errorf("nested/deep.zirr not formatted: %q", got)
	}
	if got := readFile(t, fs, "Cavefile"); !strings.Contains(got, "\tprelude") {
		t.Errorf("Cavefile not formatted: %q", got)
	}
	if got := readFile(t, fs, "notes.txt"); got != "not zirric\n" {
		t.Errorf("non-Zirric file was touched: %q", got)
	}
}

func TestRunFmt_CheckDoesNotWrite(t *testing.T) {
	fs := fmtFixture(t)
	drift, out, _ := runFmtTest(t, fs, nil, fmtFlags{check: true}, "")

	if !drift {
		t.Error("expected drift")
	}
	if got := readFile(t, fs, "main.zirr"); got != unformatted {
		t.Error("--check must not rewrite files")
	}
	for _, want := range []string{"main.zirr", "nested/deep.zirr", "Cavefile"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in listing, got %q", want, out)
		}
	}
	if strings.Contains(out, "clean.zirr") {
		t.Errorf("already-formatted file should not be listed, got %q", out)
	}
}

func TestRunFmt_NoDriftOnFormattedTree(t *testing.T) {
	fs := fmtFixture(t)
	runFmtTest(t, fs, nil, fmtFlags{}, "")

	drift, out, _ := runFmtTest(t, fs, nil, fmtFlags{check: true}, "")
	if drift {
		t.Errorf("expected no drift after formatting, got listing %q", out)
	}
}

func TestRunFmt_ListOnly(t *testing.T) {
	fs := fmtFixture(t)
	_, out, _ := runFmtTest(t, fs, []string{"main.zirr"}, fmtFlags{list: true}, "")
	if strings.TrimSpace(out) != "main.zirr" {
		t.Errorf("want main.zirr listed, got %q", out)
	}
	if readFile(t, fs, "main.zirr") != unformatted {
		t.Error("--list must not rewrite files")
	}
}

func TestRunFmt_Diff(t *testing.T) {
	fs := fmtFixture(t)
	_, out, _ := runFmtTest(t, fs, []string{"main.zirr"}, fmtFlags{diff: true}, "")
	for _, want := range []string{"--- main.zirr", "+++ main.zirr", "@@", "-      const x = 1+2", "+\tconst x = 1 + 2"} {
		if !strings.Contains(out, want) {
			t.Errorf("diff missing %q, got:\n%s", want, out)
		}
	}
	if readFile(t, fs, "main.zirr") != unformatted {
		t.Error("--diff must not rewrite files")
	}
}

func TestRunFmt_Stdin(t *testing.T) {
	fs := fmtFixture(t)
	drift, out, _ := runFmtTest(t, fs, nil, fmtFlags{stdin: true, stdinPath: "<stdin>"}, unformatted)
	if !drift {
		t.Error("expected drift")
	}
	if out != formatted {
		t.Errorf("want %q, got %q", formatted, out)
	}
}

// An explicitly named path is formatted whatever it is called; only walks apply the name filter.
func TestRunFmt_ExplicitPathBypassesNameFilter(t *testing.T) {
	fs := memfs.New()
	if err := billyutil.WriteFile(fs, "scratch.txt", []byte(unformatted), 0o644); err != nil {
		t.Fatal(err)
	}
	runFmtTest(t, fs, []string{"scratch.txt"}, fmtFlags{}, "")
	if got := readFile(t, fs, "scratch.txt"); got != formatted {
		t.Errorf("explicitly named file not formatted: %q", got)
	}
}

func TestRunFmt_ReportsUnformattableWithoutWriting(t *testing.T) {
	fs := memfs.New()
	broken := "fn f() { const c = 'unterminated\n}\n"
	if err := billyutil.WriteFile(fs, "broken.zirr", []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	drift, _, errOut := runFmtTest(t, fs, []string{"broken.zirr"}, fmtFlags{}, "")
	if !drift {
		t.Error("an unformattable file should count as drift")
	}
	if !strings.Contains(errOut, "broken.zirr") {
		t.Errorf("expected a diagnostic naming the file, got %q", errOut)
	}
	if got := readFile(t, fs, "broken.zirr"); got != broken {
		t.Errorf("unformattable file must be left alone, got %q", got)
	}
}

func TestCollectTargets_SkipsTestdata(t *testing.T) {
	fs := memfs.New()
	for _, p := range []string{"a.zirr", "testdata/fixture.zirr", ".git/x.zirr", "sub/b.zirr"} {
		if err := billyutil.WriteFile(fs, p, []byte(formatted), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := collectTargets(fs, nil, nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.zirr", "sub/b.zirr"}
	if len(got) != len(want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("want %v, got %v", want, got)
		}
	}
}

func TestRunFmt_SkipsExcludedPathsInWalk(t *testing.T) {
	fs := memfs.New()
	for _, p := range []string{"main.zirr", "vendor/dep.zirr", "gen/a.generated.zirr"} {
		if err := billyutil.WriteFile(fs, p, []byte(unformatted), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	excludes := codefmt.Excludes{"vendor/**", "**/*.generated.zirr"}

	runFmtExcl(t, fs, nil, fmtFlags{}, excludes, "")

	if got := readFile(t, fs, "main.zirr"); got != formatted {
		t.Errorf("main.zirr should have been formatted: %q", got)
	}
	for _, p := range []string{"vendor/dep.zirr", "gen/a.generated.zirr"} {
		if got := readFile(t, fs, p); got != unformatted {
			t.Errorf("%s should have been left alone: %q", p, got)
		}
	}
}

// An excluded file must not count as drift, or --check would fail forever.
func TestRunFmt_ExcludedFilesAreNotDrift(t *testing.T) {
	fs := memfs.New()
	if err := billyutil.WriteFile(fs, "vendor/dep.zirr", []byte(unformatted), 0o644); err != nil {
		t.Fatal(err)
	}
	drift, out, _ := runFmtExcl(t, fs, nil, fmtFlags{check: true}, codefmt.Excludes{"vendor/**"}, "")
	if drift {
		t.Errorf("excluded files must not report drift, got listing %q", out)
	}
}

// Naming an excluded file explicitly still skips it, but says so, since a silent no-op on a typed path is confusing.
func TestRunFmt_ExplicitExcludedPathIsReported(t *testing.T) {
	fs := memfs.New()
	if err := billyutil.WriteFile(fs, "vendor/dep.zirr", []byte(unformatted), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, errOut := runFmtExcl(t, fs, []string{"vendor/dep.zirr"}, fmtFlags{}, codefmt.Excludes{"vendor/**"}, "")

	if !strings.Contains(errOut, "vendor/dep.zirr") || !strings.Contains(errOut, "--no-excludes") {
		t.Errorf("expected a diagnostic pointing at --no-excludes, got %q", errOut)
	}
	if got := readFile(t, fs, "vendor/dep.zirr"); got != unformatted {
		t.Error("explicitly named excluded file should still be skipped")
	}
}

// Passing no excludes is what --no-excludes does at the call site.
func TestRunFmt_NoExcludesFormatsEverything(t *testing.T) {
	fs := memfs.New()
	if err := billyutil.WriteFile(fs, "vendor/dep.zirr", []byte(unformatted), 0o644); err != nil {
		t.Fatal(err)
	}
	runFmtExcl(t, fs, nil, fmtFlags{}, nil, "")
	if got := readFile(t, fs, "vendor/dep.zirr"); got != formatted {
		t.Errorf("without excludes the file should be formatted: %q", got)
	}
}

// An editor piping a vendored file through --stdin must not bypass excludes.
func TestRunFmt_StdinHonoursExcludes(t *testing.T) {
	fs := memfs.New()
	flags := fmtFlags{stdin: true, stdinPath: "vendor/dep.zirr"}
	drift, out, errOut := runFmtExcl(t, fs, nil, flags, codefmt.Excludes{"vendor/**"}, unformatted)

	if drift {
		t.Error("an excluded stdin path must not report drift")
	}
	if out != unformatted {
		t.Errorf("excluded stdin should be echoed unchanged, got %q", out)
	}
	if !strings.Contains(errOut, "vendor/dep.zirr") {
		t.Errorf("expected a diagnostic, got %q", errOut)
	}
}

func TestRunFmt_StdinFormatsNonExcludedPath(t *testing.T) {
	fs := memfs.New()
	flags := fmtFlags{stdin: true, stdinPath: "main.zirr"}
	_, out, _ := runFmtExcl(t, fs, nil, flags, codefmt.Excludes{"vendor/**"}, unformatted)
	if out != formatted {
		t.Errorf("want %q, got %q", formatted, out)
	}
}

// `zirric fmt` stops for a package this Zirric cannot build rather than reformatting sources written for a language it does not have.
// --no-excludes is no way around it: it says nothing about which Zirric may build the package.
func TestProjectFormattingExcludes_RefusesLanguageVersionMismatch(t *testing.T) {
	// Stands in for the ldflags a release build stamps.
	previous := mv.Version
	mv.Version = "1.0.0"
	t.Cleanup(func() { mv.Version = previous })

	fs := memfs.New()
	if err := billyutil.WriteFile(fs, "Cavefile", []byte("mod myapp\n\nimport cave\n\n@cave.Package()\n@cave.LanguageVersion(\">=9.0.0\")\ndata Deps {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := projectFormattingExcludes(fs, true)
	var mismatch *cavefile.LanguageVersionError
	if !errors.As(err, &mismatch) {
		t.Fatalf("err = %v, want *cavefile.LanguageVersionError", err)
	}
	// The Cavefile read fine, so the refusal must not arrive dressed as a read failure.
	if strings.Contains(err.Error(), "read Cavefile") {
		t.Errorf("err = %q, want it to report the refusal on its own", err)
	}

	if _, err := projectFormattingExcludes(fs, false); !errors.As(err, &mismatch) {
		t.Fatalf("err under --no-excludes = %v, want the same refusal", err)
	}
}
