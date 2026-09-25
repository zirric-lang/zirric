package cmds

import (
	"os/exec"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"github.com/go-git/go-billy/v5/memfs"
	billyutil "github.com/go-git/go-billy/v5/util"
)

func TestRunCaveNew_CreatesCavefile(t *testing.T) {
	projectFS := memfs.New()
	var buf strings.Builder
	opts := caveNewOptions{modulePath: "code.knabel.dev.example.my_app"}
	if err := runCaveNew(projectFS, opts, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := billyutil.ReadFile(projectFS, orchestra.DefaultCavefileName)
	if err != nil {
		t.Fatalf("read Cavefile: %v", err)
	}
	want := `mod code.knabel.dev.example.my_app

import cave

@cave.Package()
data MyApp {
}
`
	if string(content) != want {
		t.Errorf("created Cavefile =\n%s\nwant\n%s", content, want)
	}
	if !strings.Contains(buf.String(), "created") {
		t.Errorf("expected confirmation output, got %q", buf.String())
	}
}

func TestRunCaveNew_WritesEveryDeclaredAttribute(t *testing.T) {
	projectFS := memfs.New()
	opts := caveNewOptions{
		modulePath:       "myapp",
		gitURL:           "https://code.knabel.dev/example/myapp",
		version:          "1.2.3",
		languageVersion:  ">=0.1.0",
		description:      `a "quoted" description`,
		documentationURL: "https://example.com/docs",
	}
	if err := runCaveNew(projectFS, opts, &strings.Builder{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := billyutil.ReadFile(projectFS, orchestra.DefaultCavefileName)
	if err != nil {
		t.Fatalf("read Cavefile: %v", err)
	}
	want := `mod myapp

import cave

@cave.Package()
@cave.Git("https://code.knabel.dev/example/myapp")
@cave.Version("1.2.3")
@cave.LanguageVersion(">=0.1.0")
@cave.Description("a \"quoted\" description")
@cave.Documentation("https://example.com/docs")
data Myapp {
}
`
	if string(content) != want {
		t.Errorf("created Cavefile =\n%s\nwant\n%s", content, want)
	}
}

func TestRunCaveNew_RefusesToOverwriteExistingCavefile(t *testing.T) {
	projectFS := memfs.New()
	if err := billyutil.WriteFile(projectFS, orchestra.DefaultCavefileName, []byte("mod existing\n"), 0o644); err != nil {
		t.Fatalf("write existing Cavefile: %v", err)
	}

	var buf strings.Builder
	err := runCaveNew(projectFS, caveNewOptions{modulePath: "myapp"}, &buf)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	content, readErr := billyutil.ReadFile(projectFS, orchestra.DefaultCavefileName)
	if readErr != nil {
		t.Fatalf("read Cavefile: %v", readErr)
	}
	if string(content) != "mod existing\n" {
		t.Errorf("expected the existing Cavefile to be left untouched, got %q", content)
	}
}

func TestResolveCaveNewOptions_NeedsSomethingToNameThePackage(t *testing.T) {
	// t.TempDir() is not a Git checkout, so nothing names the package on its behalf.
	_, err := resolveCaveNewOptions(caveNewOptions{}, t.TempDir())
	if err == nil {
		t.Fatal("expected an error when neither --mod nor a Git remote names the package")
	}
	if !strings.Contains(err.Error(), "--mod") {
		t.Errorf("error = %v, want it to name the flag that would fix it", err)
	}
}

func TestResolveCaveNewOptions_RejectsAModulePathThatIsNotOne(t *testing.T) {
	_, err := resolveCaveNewOptions(caveNewOptions{modulePath: "not a path"}, t.TempDir())
	if err == nil {
		t.Fatal("expected an error for a module path that is not dot-separated identifiers")
	}
}

func TestResolveCaveNewOptions_DerivesNameAndURLFromTheGitRemote(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "remote", "add", "origin", "git@code.knabel.dev:zirric-lang/zirric.git")

	opts, err := resolveCaveNewOptions(caveNewOptions{}, dir)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if opts.gitURL != "https://code.knabel.dev/zirric-lang/zirric" {
		t.Errorf("gitURL = %q, want the https form without .git", opts.gitURL)
	}
	if opts.modulePath != "code.knabel.dev.zirric_lang.zirric" {
		t.Errorf("modulePath = %q, want it derived from the remote", opts.modulePath)
	}
}

func TestResolveCaveNewOptions_FlagsWinOverTheGitRemote(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "remote", "add", "origin", "https://code.knabel.dev/zirric-lang/zirric.git")

	opts, err := resolveCaveNewOptions(caveNewOptions{modulePath: "chosen.by.hand", gitURL: "https://example.com/other"}, dir)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if opts.modulePath != "chosen.by.hand" || opts.gitURL != "https://example.com/other" {
		t.Errorf("resolved %+v, want the flags to stand", opts)
	}
}

func TestNormalizeGitURL(t *testing.T) {
	cases := map[string]string{
		"git@code.knabel.dev:zirric-lang/zirric.git":       "https://code.knabel.dev/zirric-lang/zirric",
		"ssh://git@code.knabel.dev/zirric-lang/zirric.git": "https://code.knabel.dev/zirric-lang/zirric",
		"git://code.knabel.dev/zirric-lang/zirric.git":     "https://code.knabel.dev/zirric-lang/zirric",
		"https://code.knabel.dev/zirric-lang/zirric.git":   "https://code.knabel.dev/zirric-lang/zirric",
		"https://code.knabel.dev/zirric-lang/zirric/":      "https://code.knabel.dev/zirric-lang/zirric",
		"": "",
	}
	for input, want := range cases {
		if got := normalizeGitURL(input); got != want {
			t.Errorf("normalizeGitURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestPackageDataName(t *testing.T) {
	cases := map[string]string{
		"myapp":                              "Myapp",
		"code.knabel.dev.example.my_app":     "MyApp",
		"code.knabel.dev.zirric_lang.zirric": "Zirric",
		"a.b.flow_node_graph":                "FlowNodeGraph",
	}
	for input, want := range cases {
		if got := packageDataName(input); got != want {
			t.Errorf("packageDataName(%q) = %q, want %q", input, got, want)
		}
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestRunCaveNew_WelcomeSignposts(t *testing.T) {
	projectFS := memfs.New()
	var buf strings.Builder
	opts := caveNewOptions{modulePath: "code.knabel.dev.example.my_app"}
	if err := runCaveNew(projectFS, opts, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		"███",
		"created Cavefile",
		"welcome to Zirric",
		"zirric cave install",
		"zirric ci github",
		"zirric ci forgejo",
		"https://zirric.knabel.dev/guides/getting-started",
		"https://zirric.knabel.dev/tooling/zirric-cli",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in the welcome, got\n%s", want, out)
		}
	}
}

// A strings.Builder is not a terminal, so nothing styled may reach it: a redirected `zirric cave new` must not collect escape sequences as text.
func TestRunCaveNew_WelcomeIsPlainWhenNotATerminal(t *testing.T) {
	projectFS := memfs.New()
	var buf strings.Builder
	opts := caveNewOptions{modulePath: "code.knabel.dev.example.my_app"}
	if err := runCaveNew(projectFS, opts, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("welcome carried escape sequences into a non-terminal writer:\n%q", buf.String())
	}
}
