package cmds

import (
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"github.com/go-git/go-billy/v5/memfs"
	billyutil "github.com/go-git/go-billy/v5/util"
)

func TestRunInit_CreatesCavefile(t *testing.T) {
	projectFS := memfs.New()
	var buf strings.Builder
	if err := runInit(projectFS, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := billyutil.ReadFile(projectFS, orchestra.DefaultCavefileName)
	if err != nil {
		t.Fatalf("read Cavefile: %v", err)
	}
	if !strings.Contains(string(content), "@cave.Dependencies()") {
		t.Errorf("expected the created Cavefile to declare @cave.Dependencies(), got %q", content)
	}
	if !strings.Contains(buf.String(), "created") {
		t.Errorf("expected confirmation output, got %q", buf.String())
	}
}

func TestRunInit_RefusesToOverwriteExistingCavefile(t *testing.T) {
	projectFS := memfs.New()
	if err := billyutil.WriteFile(projectFS, orchestra.DefaultCavefileName, []byte("mod existing\n"), 0o644); err != nil {
		t.Fatalf("write existing Cavefile: %v", err)
	}

	var buf strings.Builder
	err := runInit(projectFS, &buf)
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
