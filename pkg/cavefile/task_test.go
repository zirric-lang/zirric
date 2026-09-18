package cavefile_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
)

func TestParseExecTask(t *testing.T) {
	src := `import future.tasks

@tasks.Name("generate")
@tasks.Help("Generates something")
@tasks.Exec("tasks/generate.zirr")
data GenerateTask {
  @tasks.Flag()
  @tasks.Name("dry")
  @tasks.Short('d')
  @tasks.Help("If true, only simulates the generation")
  isDryRun: Bool

  @tasks.Arg()
  positional: String
}
`
	cave, _ := parseCavefileInMem(t, src)
	if len(cave.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d: %+v", len(cave.Tasks), cave.Tasks)
	}
	task := cave.Tasks[0]

	if task.Name != "generate" {
		t.Errorf("Name = %q, want %q", task.Name, "generate")
	}
	if task.DeclName != "GenerateTask" {
		t.Errorf("DeclName = %q, want %q", task.DeclName, "GenerateTask")
	}
	if task.Help != "Generates something" {
		t.Errorf("Help = %q, want %q", task.Help, "Generates something")
	}
	if task.Kind != cavefile.TaskKindExec {
		t.Errorf("Kind = %v, want TaskKindExec", task.Kind)
	}
	if task.Exec != "tasks/generate.zirr" {
		t.Errorf("Exec = %q, want %q", task.Exec, "tasks/generate.zirr")
	}

	if len(task.Flags) != 1 {
		t.Fatalf("expected 1 flag, got %+v", task.Flags)
	}
	flag := task.Flags[0]
	if flag.Name != "dry" || flag.Short != "d" || flag.Help != "If true, only simulates the generation" {
		t.Errorf("unexpected flag: %+v", flag)
	}
	if flag.Type != cavefile.TaskParamTypeBool {
		t.Errorf("flag Type = %v, want TaskParamTypeBool", flag.Type)
	}
	if flag.DeclName != "isDryRun" {
		t.Errorf("flag DeclName = %q, want %q", flag.DeclName, "isDryRun")
	}

	if len(task.Args) != 1 {
		t.Fatalf("expected 1 arg, got %+v", task.Args)
	}
	if task.Args[0].Name != "positional" {
		t.Errorf("Args[0].Name = %q, want %q", task.Args[0].Name, "positional")
	}
	if task.Args[0].Type != cavefile.TaskParamTypeString {
		t.Errorf("Args[0].Type = %v, want TaskParamTypeString", task.Args[0].Type)
	}
}

func TestParseTaskParamTypes(t *testing.T) {
	src := `import future.tasks

@tasks.Call(fn(opts: BuildTask) {})
data BuildTask {
  @tasks.Flag()
  dry: Bool

  @tasks.Flag()
  target: String

  @tasks.Flag()
  count: Int

  @tasks.Flag()
  untyped

  @tasks.Flag()
  unsupported: [String]
}
`
	cave, _ := parseCavefileInMem(t, src)
	if len(cave.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d: %+v", len(cave.Tasks), cave.Tasks)
	}
	flags := cave.Tasks[0].Flags
	if len(flags) != 5 {
		t.Fatalf("expected 5 flags, got %d: %+v", len(flags), flags)
	}

	want := map[string]cavefile.TaskParamType{
		"dry":         cavefile.TaskParamTypeBool,
		"target":      cavefile.TaskParamTypeString,
		"count":       cavefile.TaskParamTypeInt,
		"untyped":     cavefile.TaskParamTypeUnsupported,
		"unsupported": cavefile.TaskParamTypeUnsupported,
	}
	for _, flag := range flags {
		if got, wantType := flag.Type, want[flag.Name]; got != wantType {
			t.Errorf("%s.Type = %v, want %v", flag.Name, got, wantType)
		}
	}
}

func TestParseTaskDefaultNameAndAliases(t *testing.T) {
	src := `import future.tasks

@tasks.Alias(["b", "compile"])
@tasks.Exec("build.zirr")
data BuildTask {
}
`
	cave, _ := parseCavefileInMem(t, src)
	if len(cave.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d: %+v", len(cave.Tasks), cave.Tasks)
	}
	task := cave.Tasks[0]
	if task.Name != "buildtask" {
		t.Errorf("Name = %q, want %q", task.Name, "buildtask")
	}
	if task.DeclName != "BuildTask" {
		t.Errorf("DeclName = %q, want %q", task.DeclName, "BuildTask")
	}
	if len(task.Aliases) != 2 || task.Aliases[0] != "b" || task.Aliases[1] != "compile" {
		t.Errorf("Aliases = %+v, want [b compile]", task.Aliases)
	}
}

func TestParseNoTasks(t *testing.T) {
	cave, _ := parseCavefileInMem(t, testdataCavefile)
	if len(cave.Tasks) != 0 {
		t.Errorf("expected no standalone tasks, got %+v", cave.Tasks)
	}
}

func TestParseTestdataFileTasks(t *testing.T) {
	cave, err := parseCavefileFromTestdata(t)
	if err != nil {
		t.Logf("parse warning (non-fatal): %v", err)
	}
	if len(cave.Tasks) != 1 {
		t.Fatalf("expected 1 task from testdata, got %d: %+v", len(cave.Tasks), cave.Tasks)
	}
	task := cave.Tasks[0]
	if task.Name != "generate" || task.Kind != cavefile.TaskKindExec || task.Exec != "tasks/generate.zirr" {
		t.Errorf("unexpected task from testdata: %+v", task)
	}
}
