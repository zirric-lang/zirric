package langsrv

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"code.knabel.dev/zirric-lang/zirric/pkg/pkgmanager"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const (
	commandTasks   = "zirric.tasks"
	commandInstall = "zirric.install"
	// commandTaskPrefix identifies a specific Cavefile task to run: the full command is commandTaskPrefix+task.Name (e.g. "zirric.task.greet"), so a client's command palette can list/invoke each task individually instead of going through one generic "run a task" command with a name argument.
	commandTaskPrefix = "zirric.task."
)

// executeCommandNames returns the commands to advertise as ExecuteCommandProvider.Commands: the two static ones, plus one commandTaskPrefix+name entry per Cavefile-declared task, if any are known yet.
func (ls *zirricLangserver) executeCommandNames() []string {
	names := []string{commandTasks, commandInstall}
	if ls.orch != nil {
		for _, task := range ls.orch.Cavefile().Tasks {
			names = append(names, commandTaskPrefix+task.Name)
		}
	}
	return names
}

// zirricExecutablePath is a var (not a direct os.Executable() call) so tests can point it at a real built zirric binary instead of the go-test binary.
var zirricExecutablePath = os.Executable

func (ls *zirricLangserver) workspaceExecuteCommand(ctx *glsp.Context, params *protocol.ExecuteCommandParams) (any, error) {
	switch {
	case params.Command == commandTasks:
		return ls.listTasksCommand(ctx)
	case params.Command == commandInstall:
		return ls.installDependenciesCommand(ctx)
	case strings.HasPrefix(params.Command, commandTaskPrefix):
		name := strings.TrimPrefix(params.Command, commandTaskPrefix)
		if name == "" {
			return nil, fmt.Errorf("unknown command %q", params.Command)
		}
		return ls.runTaskCommand(ctx, name, params.Arguments)
	default:
		return nil, fmt.Errorf("unknown command %q", params.Command)
	}
}

// listTasksCommand also sends a window/showMessage summary, since a generic LSP client (e.g. Helix) doesn't render an executeCommand's raw JSON result to the user anywhere on its own.
func (ls *zirricLangserver) listTasksCommand(ctx *glsp.Context) (any, error) {
	if ls.orch == nil {
		ls.showMessage(ctx, protocol.MessageTypeInfo, "No workspace root; no tasks available.")
		return []cavefile.Task{}, nil
	}
	tasks := ls.orch.Cavefile().Tasks
	if len(tasks) == 0 {
		ls.showMessage(ctx, protocol.MessageTypeInfo, "No tasks declared in Cavefile.")
		return tasks, nil
	}
	names := make([]string, len(tasks))
	for i, task := range tasks {
		names[i] = task.Name
	}
	ls.showMessage(ctx, protocol.MessageTypeInfo, "Tasks: %s", strings.Join(names, ", "))
	return tasks, nil
}

// installDependenciesCommand builds a separate, writable Orchestra over the real on-disk project filesystem (not the unsaved-edits overlay), since ls.orch/ls.resolver are always read-only. Safe to run in-process: unlike task execution, installing never runs user Zirric code through the VM.
func (ls *zirricLangserver) installDependenciesCommand(ctx *glsp.Context) (any, error) {
	if ls.rootPath == "" {
		return nil, fmt.Errorf("zirric.install: no workspace root")
	}

	registryFS, err := orchestra.DefaultRegistryFS()
	if err != nil {
		return nil, fmt.Errorf("zirric.install: %w", err)
	}

	orch, err := orchestra.New(orchestra.Config{
		ProjectFS:   osfs.New(ls.rootPath),
		RegistryFS:  registryFS,
		PackageName: filepath.Base(ls.rootPath),
	})
	if err != nil {
		return nil, fmt.Errorf("zirric.install: %w", err)
	}

	resolver, err := orch.NewResolver(orchestra.WithInstallProgress(func(evt pkgmanager.InstallEvent) {
		ls.logMessage(ctx, "installed %s (%s)", evt.Dependency.Name, evt.Package.Source())
	}))
	if err != nil {
		return nil, fmt.Errorf("zirric.install: %w", err)
	}

	installed, installErr := resolver.EnsureInstalled(context.Background())

	// Refresh so newly-installed imports light up immediately, regardless of install success.
	ls.moduleCacheMu.Lock()
	ls.moduleCache = make(map[string]*moduleCacheEntry)
	ls.moduleCacheMu.Unlock()
	if ls.resolver != nil {
		ls.resolver.InvalidateModules()
	}
	_ = ls.refreshDiagnostics(ctx)

	if installErr != nil {
		ls.showMessage(ctx, protocol.MessageTypeError, "zirric.install: %v", installErr)
		return nil, fmt.Errorf("zirric.install: %w", installErr)
	}
	ls.showMessage(ctx, protocol.MessageTypeInfo, "Installed %d package(s).", len(installed))
	return map[string]any{"installed": len(installed)}, nil
}

// runTaskCommand is the one deliberate exception to "the LSP never executes user code", and only ever fires from an explicit command. It shells out to `zirric task run` rather than calling orchestra.RunTask in-process, since the VM's os plugin hardcodes real stdio and calls os.Exit, which would corrupt the LSP's JSON-RPC stream or kill it outright.
func (ls *zirricLangserver) runTaskCommand(ctx *glsp.Context, name string, args []any) (any, error) {
	if ls.rootPath == "" {
		return nil, fmt.Errorf("zirric.task.%s: no workspace root", name)
	}

	var execArgs []string
	if len(args) > 0 {
		raw, ok := args[0].([]any)
		if !ok {
			return nil, fmt.Errorf("zirric.task.%s: first argument must be an array of strings", name)
		}
		for _, a := range raw {
			s, ok := a.(string)
			if !ok {
				return nil, fmt.Errorf("zirric.task.%s: task arguments must be strings", name)
			}
			execArgs = append(execArgs, s)
		}
	}

	self, err := zirricExecutablePath()
	if err != nil {
		return nil, fmt.Errorf("zirric.task.%s: could not locate zirric executable: %w", name, err)
	}

	cmdArgs := append([]string{"task", "run", name}, execArgs...)
	cmd := exec.Command(self, cmdArgs...)
	cmd.Dir = ls.rootPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("zirric.task.%s: %w", name, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("zirric.task.%s: %w", name, err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("zirric.task.%s: failed to start: %w", name, err)
	}

	relay := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			ls.logMessage(ctx, "[task %s] %s", name, scanner.Text())
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); relay(stdout) }()
	go func() { defer wg.Done(); relay(stderr) }()
	wg.Wait()

	exitCode := 0
	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			ls.showMessage(ctx, protocol.MessageTypeError, "zirric.task.%s: failed: %v", name, err)
			return nil, fmt.Errorf("zirric.task.%s: failed: %w", name, err)
		}
		exitCode = exitErr.ExitCode()
	}

	if exitCode == 0 {
		ls.showMessage(ctx, protocol.MessageTypeInfo, "Task %q completed.", name)
	} else {
		ls.showMessage(ctx, protocol.MessageTypeWarning, "Task %q exited with code %d.", name, exitCode)
	}
	return map[string]any{"exitCode": exitCode}, nil
}
