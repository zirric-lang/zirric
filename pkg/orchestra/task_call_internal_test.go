package orchestra

import (
	"context"
	"path/filepath"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
)

func newTaskcallOrchestra(t *testing.T) *Orchestra {
	t.Helper()
	testdataDir, err := filepath.Abs("testdata/taskcall")
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	orch, err := New(Config{
		ProjectFS:   osfs.New(testdataDir),
		RegistryFS:  memfs.New(),
		PackageName: "taskcall",
	})
	if err != nil {
		t.Fatalf("new orchestra: %v", err)
	}
	return orch
}

func TestResolveCallFunctionInvokesWithFieldOrderIndependentOfDeclOrder(t *testing.T) {
	orch := newTaskcallOrchestra(t)
	task, ok := orch.FindTask("build")
	if !ok {
		t.Fatal("expected to find the build task")
	}
	if task.Kind != cavefile.TaskKindCall {
		t.Fatalf("Kind = %v, want TaskKindCall", task.Kind)
	}

	machine, dt, fn, err := orch.resolveCallFunction(context.Background(), task)
	if err != nil {
		t.Fatalf("resolveCallFunction: %v", err)
	}
	if fn == nil {
		t.Fatal("expected a non-nil function")
	}

	dataValue, err := makeTaskDataValue(dt, map[string]runtime.RuntimeValue{
		"dry":    runtime.Bool(true),
		"target": runtime.String("release"),
		"count":  runtime.Int(3),
	})
	if err != nil {
		t.Fatalf("makeTaskDataValue: %v", err)
	}

	result, err := machine.CallFunction(fn, dataValue)
	if err != nil {
		t.Fatalf("call function: %v", err)
	}
	s, ok := result.(runtime.String)
	if !ok || s != "release" {
		t.Errorf("result = %#v, want runtime.String(\"release\")", result)
	}
}

func TestMakeTaskDataValueMissingField(t *testing.T) {
	orch := newTaskcallOrchestra(t)
	task, _ := orch.FindTask("build")

	_, dt, _, err := orch.resolveCallFunction(context.Background(), task)
	if err != nil {
		t.Fatalf("resolveCallFunction: %v", err)
	}

	_, err = makeTaskDataValue(dt, map[string]runtime.RuntimeValue{
		"dry":    runtime.Bool(true),
		"target": runtime.String("release"),
		// "count" intentionally omitted
	})
	if err == nil {
		t.Fatal("expected an error for a missing field value")
	}
}
