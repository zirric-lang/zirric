package cmds

import (
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	billyutil "github.com/go-git/go-billy/v5/util"
)

func TestRunCIWorkflow_Forgejo(t *testing.T) {
	projectFS := memfs.New()
	var buf strings.Builder
	opts := ciWorkflowOptions{setupAction: forgejoForge.defaultSetupAction, branch: "main", runsOn: "ubuntu-latest"}
	if err := runCIWorkflow(projectFS, forgejoForge, opts, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := billyutil.ReadFile(projectFS, ".forgejo/workflows/zirric.yaml")
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}
	want := `name: Zirric

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7

      - name: Setup Zirric
        uses: https://code.knabel.dev/zirric-lang/setup-action@v1

      - name: Check formatting
        run: zirric fmt --check --diff

      - name: Run tests
        run: zirric test
`
	if string(content) != want {
		t.Errorf("created workflow =\n%s\nwant\n%s", content, want)
	}
	if !strings.Contains(buf.String(), "created .forgejo/workflows/zirric.yaml") {
		t.Errorf("expected confirmation output, got %q", buf.String())
	}
}

// GitHub rejects a fully qualified URL in `uses`, so its workflow must name the mirror instead.
func TestRunCIWorkflow_GitHubUsesMirroredAction(t *testing.T) {
	projectFS := memfs.New()
	opts := ciWorkflowOptions{setupAction: githubForge.defaultSetupAction, branch: "main", runsOn: "ubuntu-latest"}
	if err := runCIWorkflow(projectFS, githubForge, opts, &strings.Builder{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := billyutil.ReadFile(projectFS, ".github/workflows/zirric.yaml")
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}
	if !strings.Contains(string(content), "uses: zirric-lang/setup-action@v1") {
		t.Errorf("expected the mirrored action, got\n%s", content)
	}
	if strings.Contains(string(content), "https://") {
		t.Errorf("GitHub does not resolve a URL in uses, got\n%s", content)
	}
}

func TestRunCIWorkflow_HonoursFlags(t *testing.T) {
	projectFS := memfs.New()
	opts := ciWorkflowOptions{setupAction: "acme/setup@v2", branch: "trunk", runsOn: "self-hosted"}
	if err := runCIWorkflow(projectFS, githubForge, opts, &strings.Builder{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := billyutil.ReadFile(projectFS, ".github/workflows/zirric.yaml")
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}
	for _, want := range []string{"branches: [trunk]", "runs-on: self-hosted", "uses: acme/setup@v2"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("expected %q in\n%s", want, content)
		}
	}
	// The branch is written to both triggers, so a single substitution would be a silent half-rename.
	if got := strings.Count(string(content), "branches: [trunk]"); got != 2 {
		t.Errorf("branch substituted %d times, want 2", got)
	}
}

func TestRunCIWorkflow_RefusesToOverwrite(t *testing.T) {
	projectFS := memfs.New()
	opts := ciWorkflowOptions{setupAction: githubForge.defaultSetupAction, branch: "main", runsOn: "ubuntu-latest"}
	if err := runCIWorkflow(projectFS, githubForge, opts, &strings.Builder{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := runCIWorkflow(projectFS, githubForge, opts, &strings.Builder{})
	if err == nil {
		t.Fatal("expected an error for an existing workflow")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("expected the error to name --force, got %q", err)
	}

	opts.force = true
	opts.branch = "trunk"
	if err := runCIWorkflow(projectFS, githubForge, opts, &strings.Builder{}); err != nil {
		t.Fatalf("--force should overwrite: %v", err)
	}
	content, err := billyutil.ReadFile(projectFS, ".github/workflows/zirric.yaml")
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}
	if !strings.Contains(string(content), "branches: [trunk]") {
		t.Errorf("expected the overwritten content, got\n%s", content)
	}
}
