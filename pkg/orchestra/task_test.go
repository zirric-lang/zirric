package orchestra_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
)

func newTaskdemoOrchestra(t *testing.T) *orchestra.Orchestra {
	t.Helper()
	testdataDir, err := filepath.Abs("testdata/taskdemo")
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:   osfs.New(testdataDir),
		RegistryFS:  memfs.New(),
		PackageName: "taskdemo",
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}
	return orch
}

func TestRunTaskExecutesExecTask(t *testing.T) {
	orch := newTaskdemoOrchestra(t)
	if err := orch.RunTask(context.Background(), "greet", nil, nil); err != nil {
		t.Fatalf("run task: %v", err)
	}
}

func TestRunTaskByAlias(t *testing.T) {
	orch := newTaskdemoOrchestra(t)
	if err := orch.RunTask(context.Background(), "hi", nil, nil); err != nil {
		t.Fatalf("run task by alias: %v", err)
	}
}

func TestRunTaskUnknownName(t *testing.T) {
	orch := newTaskdemoOrchestra(t)
	if err := orch.RunTask(context.Background(), "nope", nil, nil); err == nil {
		t.Fatal("expected an error for an unknown task")
	}
}

func TestRunTaskExecTaskRestoresOSArgsAfterExecArgsRewrite(t *testing.T) {
	orch := newTaskdemoOrchestra(t)
	before := append([]string{}, os.Args...)

	if err := orch.RunTask(context.Background(), "greet", nil, []string{"extra", "args"}); err != nil {
		t.Fatalf("run task: %v", err)
	}

	if !reflect.DeepEqual(os.Args, before) {
		t.Errorf("os.Args not restored: got %v, want %v", os.Args, before)
	}
}

func newTaskcallOrchestraForRunTask(t *testing.T) *orchestra.Orchestra {
	t.Helper()
	testdataDir, err := filepath.Abs("testdata/taskcall")
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:   osfs.New(testdataDir),
		RegistryFS:  memfs.New(),
		PackageName: "taskcall",
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}
	return orch
}

func TestRunTaskExecutesCallTask(t *testing.T) {
	orch := newTaskcallOrchestraForRunTask(t)
	values := map[string]runtime.RuntimeValue{
		"dry":    runtime.Bool(false),
		"target": runtime.String("release"),
		"count":  runtime.Int(1),
	}
	if err := orch.RunTask(context.Background(), "build", values, nil); err != nil {
		t.Fatalf("run task: %v", err)
	}
}

func TestRunTaskExecutesCallTaskWithCavefilePathOverride(t *testing.T) {
	testdataDir, err := filepath.Abs("testdata/taskcall")
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:    osfs.New(testdataDir),
		RegistryFS:   memfs.New(),
		PackageName:  "taskcall",
		CavefilePath: "Cavefile.alt",
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}

	values := map[string]runtime.RuntimeValue{
		"dry":    runtime.Bool(false),
		"target": runtime.String("release"),
		"count":  runtime.Int(1),
	}
	if err := orch.RunTask(context.Background(), "build", values, nil); err != nil {
		t.Fatalf("run task: %v", err)
	}
}

func TestFindTask(t *testing.T) {
	orch := newTaskdemoOrchestra(t)
	task, ok := orch.FindTask("greet")
	if !ok {
		t.Fatal("expected to find the greet task")
	}
	if task.Exec != "greet.zirr" {
		t.Errorf("Exec = %q, want %q", task.Exec, "greet.zirr")
	}

	if _, ok := orch.FindTask("nope"); ok {
		t.Error("expected no task named 'nope'")
	}
}
