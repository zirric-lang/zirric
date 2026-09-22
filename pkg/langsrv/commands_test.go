package langsrv

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// messageRecorder captures window/showMessage notifications, the only feedback path a generic LSP client (e.g. Helix) actually surfaces to the user for an executeCommand result — unlike window/logMessage, which typically only reaches a log/output channel.
type messageRecorder struct {
	messages []*protocol.ShowMessageParams
}

func (r *messageRecorder) Notify(method string, params any) {
	if method != protocol.ServerWindowShowMessage {
		return
	}
	if p, ok := params.(*protocol.ShowMessageParams); ok {
		r.messages = append(r.messages, p)
	}
}

func TestListTasksCommand(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "Cavefile", `mod proj

import tasks

@tasks.Name("greet")
@tasks.Alias(["hi"])
@tasks.Exec("greet.zirr")
data GreetTask {
}
`)
	writeFile(t, base, "greet.zirr", "mod greet\nconst greeting = \"hi\"\n")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	rec := &messageRecorder{}
	ctx := &glsp.Context{Notify: rec.Notify}

	result, err := ls.listTasksCommand(ctx)
	if err != nil {
		t.Fatalf("listTasksCommand: %v", err)
	}
	tasks, ok := result.([]cavefile.Task)
	if !ok {
		t.Fatalf("expected []cavefile.Task, got %T", result)
	}
	if len(tasks) != 1 || tasks[0].Name != "greet" {
		t.Fatalf("expected a single task named 'greet', got %+v", tasks)
	}
	if len(tasks[0].Aliases) != 1 || tasks[0].Aliases[0] != "hi" {
		t.Fatalf("expected alias 'hi', got %+v", tasks[0].Aliases)
	}

	if len(rec.messages) != 1 {
		t.Fatalf("expected exactly 1 window/showMessage notification, got %d: %+v", len(rec.messages), rec.messages)
	}
	if !strings.Contains(rec.messages[0].Message, "greet") {
		t.Errorf("expected the shown message to mention 'greet', got %q", rec.messages[0].Message)
	}
}

// TestExecuteCommandNamesIncludesPerTaskCommands checks that each Cavefile-declared task gets its own advertised "zirric.task.<name>" command, alongside the two static ones.
func TestExecuteCommandNamesIncludesPerTaskCommands(t *testing.T) {
	base := memfs.New()
	writeFile(t, base, "Cavefile", `mod proj

import tasks

@tasks.Name("greet")
@tasks.Exec("greet.zirr")
data GreetTask {
}
`)
	writeFile(t, base, "greet.zirr", "mod greet\nconst greeting = \"hi\"\n")

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setFilesystem(base, "/")

	names := ls.executeCommandNames()
	want := map[string]bool{"zirric.tasks": false, "zirric.install": false, "zirric.task.greet": false}
	for _, name := range names {
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("expected %q in executeCommandNames(), got %v", name, names)
		}
	}
}

// TestWorkspaceExecuteCommandDispatchesTaskPrefix checks that workspace/executeCommand routes any "zirric.task.<name>" command to runTaskCommand with the name extracted from the command itself, not from the arguments.
func TestWorkspaceExecuteCommandDispatchesTaskPrefix(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the zirric binary; skipped in -short mode")
	}
	binary := buildTestZirricBinary(t)

	prev := zirricExecutablePath
	zirricExecutablePath = func() (string, error) { return binary, nil }
	defer func() { zirricExecutablePath = prev }()

	root := t.TempDir()
	cave := `mod greeter

import tasks

@tasks.Name("greet")
@tasks.Exec("greet.zirr")
data GreetTask {
}
`
	greet := "mod greeter\n"
	if err := os.WriteFile(filepath.Join(root, "Cavefile"), []byte(cave), 0o644); err != nil {
		t.Fatalf("write Cavefile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "greet.zirr"), []byte(greet), 0o644); err != nil {
		t.Fatalf("write greet.zirr: %v", err)
	}

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setRoot(root)

	ctx := &glsp.Context{Notify: func(string, any) {}}
	result, err := ls.workspaceExecuteCommand(ctx, &protocol.ExecuteCommandParams{Command: "zirric.task.greet"})
	if err != nil {
		t.Fatalf("workspaceExecuteCommand: %v", err)
	}
	res, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if res["exitCode"] != 0 {
		t.Fatalf("expected exit code 0, got %v", res["exitCode"])
	}

	if _, err := ls.workspaceExecuteCommand(ctx, &protocol.ExecuteCommandParams{Command: "zirric.task."}); err == nil {
		t.Error("expected an error for an empty task name")
	}
	if _, err := ls.workspaceExecuteCommand(ctx, &protocol.ExecuteCommandParams{Command: "zirric.unknown"}); err == nil {
		t.Error("expected an error for an unknown command")
	}
}

func TestInstallDependenciesCommand(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.zirr"), []byte("mod main\nconst x = 1\n"), 0o644); err != nil {
		t.Fatalf("write main.zirr: %v", err)
	}

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setRoot(root)
	ls.openDocs["main.zirr"] = ls.fileURI("main.zirr")

	diagRec := &diagnosticRecorder{}
	msgRec := &messageRecorder{}
	ctx := &glsp.Context{Notify: func(method string, params any) {
		diagRec.Notify(method, params)
		msgRec.Notify(method, params)
	}}

	if _, err := ls.installDependenciesCommand(ctx); err != nil {
		t.Fatalf("installDependenciesCommand: %v", err)
	}

	// refreshDiagnostics runs asynchronously; use the deterministic variant to assert the result.
	if err := ls.refreshDiagnosticsSync(ctx); err != nil {
		t.Fatalf("refresh diagnostics: %v", err)
	}
	if diags := diagRec.DiagnosticsFor(ls.fileURI("main.zirr")); len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %+v", diags)
	}

	if len(msgRec.messages) != 1 {
		t.Fatalf("expected exactly 1 window/showMessage notification, got %d: %+v", len(msgRec.messages), msgRec.messages)
	}
}

// buildTestZirricBinary compiles the real zirric CLI so runTaskCommand's subprocess has a real binary to exec, instead of the go-test binary os.Executable() would return.
func buildTestZirricBinary(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	out := filepath.Join(t.TempDir(), "zirric")
	cmd := exec.Command("go", "build", "-o", out, "code.knabel.dev/zirric-lang/zirric/cmd/zirric")
	cmd.Dir = filepath.Join(wd, "..", "..")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build zirric binary: %v\n%s", err, output)
	}
	return out
}

func TestRunTaskCommandSpawnsSubprocess(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the zirric binary; skipped in -short mode")
	}
	binary := buildTestZirricBinary(t)

	prev := zirricExecutablePath
	zirricExecutablePath = func() (string, error) { return binary, nil }
	defer func() { zirricExecutablePath = prev }()

	root := t.TempDir()
	cave := `mod greeter

import tasks

@tasks.Name("greet")
@tasks.Exec("greet.zirr")
data GreetTask {
}
`
	greet := `mod greeter
import io
import os
import bytes

io.Writer(os.stdout()).write(os.stdout(), bytes.fromString("hello from task\n"))
`
	if err := os.WriteFile(filepath.Join(root, "Cavefile"), []byte(cave), 0o644); err != nil {
		t.Fatalf("write Cavefile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "greet.zirr"), []byte(greet), 0o644); err != nil {
		t.Fatalf("write greet.zirr: %v", err)
	}

	ls := zirricLangserver{
		docs:     newDocumentStore(),
		diagURIs: make(map[protocol.DocumentUri]struct{}),
		openDocs: make(map[string]protocol.DocumentUri),
	}
	ls.setRoot(root)

	var messages []string
	msgRec := &messageRecorder{}
	ctx := &glsp.Context{Notify: func(method string, params any) {
		msgRec.Notify(method, params)
		if method != protocol.ServerWindowLogMessage {
			return
		}
		if p, ok := params.(*protocol.LogMessageParams); ok {
			messages = append(messages, p.Message)
		}
	}}

	result, err := ls.runTaskCommand(ctx, "greet", nil)
	if err != nil {
		t.Fatalf("runTaskCommand: %v", err)
	}
	res, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if res["exitCode"] != 0 {
		t.Fatalf("expected exit code 0, got %v (log: %v)", res["exitCode"], messages)
	}

	found := false
	for _, m := range messages {
		if strings.Contains(m, "hello from task") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected relayed task output, got messages: %v", messages)
	}

	if len(msgRec.messages) != 1 {
		t.Fatalf("expected exactly 1 window/showMessage notification, got %d: %+v", len(msgRec.messages), msgRec.messages)
	}
	if !strings.Contains(msgRec.messages[0].Message, "greet") {
		t.Errorf("expected the shown message to mention the task name, got %q", msgRec.messages[0].Message)
	}
}
