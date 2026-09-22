package cmds

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// runTaskAlongside runs the project task called name in addition to a built-in command's own work, returning nil when there is no such task or it was left unrun.
//
// It runs only when it declares every flag that was set: one with no --check would write where the caller asked only to look.
func runTaskAlongside(cmd *cobra.Command, name string, args []string, errOut io.Writer) error {
	projectFS, err := cwdFS()
	if err != nil {
		return nil
	}
	orch, err := newOrchestra(projectFS, currentDirPackageName())
	if err != nil {
		return nil
	}
	task, ok := orch.FindTask(name)
	if !ok {
		return nil
	}

	if refused := unacceptedFlags(cmd, task); len(refused) > 0 {
		note(errOut, "task %q takes no %s, so it was not run", task.Name, strings.Join(refused, " or "))
		return nil
	}

	switch task.Kind {
	case cavefile.TaskKindExec:
		return orch.RunTask(context.Background(), task.Name, nil, taskArgv(cmd, args))
	case cavefile.TaskKindCall:
		values, ok := taskCallValues(task, cmd, args)
		if !ok {
			note(errOut, "task %q takes %d argument(s) rather than %d, so it was not run", task.Name, len(task.Args), len(args))
			return nil
		}
		return orch.RunTask(context.Background(), task.Name, values, nil)
	}
	return nil
}

// note writes an aside about work that was deliberately not done.
func note(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, "note: "+format+"\n", args...)
}

// runTaskInstead runs the project task called name in place of a built-in command's own work, reporting whether one was declared.
//
// It is parsed exactly as `zirric task <name>` would parse it.
func runTaskInstead(cmd *cobra.Command, name string, args []string) (bool, error) {
	projectFS, err := cwdFS()
	if err != nil {
		return false, nil
	}
	orch, err := newOrchestra(projectFS, currentDirPackageName())
	if err != nil {
		return false, nil
	}
	task, ok := orch.FindTask(name)
	if !ok {
		return false, nil
	}

	taskCmd, err := newTaskCommand(task)
	if err != nil {
		return true, err
	}
	taskCmd.SetArgs(args)
	taskCmd.SetIn(cmd.InOrStdin())
	taskCmd.SetOut(cmd.OutOrStdout())
	taskCmd.SetErr(cmd.ErrOrStderr())
	taskCmd.SilenceErrors = true
	taskCmd.SilenceUsage = true
	return true, taskCmd.Execute()
}

// unacceptedFlags lists the flags the caller set that the task does not declare.
func unacceptedFlags(cmd *cobra.Command, task cavefile.Task) []string {
	var refused []string
	// LocalNonPersistentFlags leaves out --cavefile and the rest the root command contributes.
	cmd.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed || taskAcceptsFlag(task, f.Name) {
			return
		}
		refused = append(refused, "--"+f.Name)
	})
	return refused
}

func taskAcceptsFlag(task cavefile.Task, name string) bool {
	for _, param := range task.Flags {
		if param.Name == name || slices.Contains(param.Aliases, name) {
			return true
		}
	}
	return false
}

// taskArgv rebuilds the argv a script task reads from os.args().
func taskArgv(cmd *cobra.Command, args []string) []string {
	var argv []string
	cmd.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed {
			return
		}
		if f.Value.Type() == "bool" && f.Value.String() == "true" {
			argv = append(argv, "--"+f.Name)
			return
		}
		argv = append(argv, "--"+f.Name+"="+f.Value.String())
	})
	return append(argv, args...)
}

// taskCallValues builds the fields a called task is constructed from, reporting false when the arguments cannot fill what it declares.
func taskCallValues(task cavefile.Task, cmd *cobra.Command, args []string) (map[string]runtime.RuntimeValue, bool) {
	if len(task.Args) != len(args) {
		return nil, false
	}
	values := make(map[string]runtime.RuntimeValue, len(task.Flags)+len(task.Args))
	for _, param := range task.Flags {
		values[param.DeclName] = taskFlagValue(param, cmd)
	}
	for i, param := range task.Args {
		value, err := parseTaskParamValue(param, args[i])
		if err != nil {
			return nil, false
		}
		values[param.DeclName] = value
	}
	return values, true
}

// taskFlagValue falls back to the zero value the task would see if it were run on its own.
func taskFlagValue(param cavefile.TaskParam, cmd *cobra.Command) runtime.RuntimeValue {
	flag := cmd.Flags().Lookup(param.Name)
	for _, alias := range param.Aliases {
		if flag != nil {
			break
		}
		flag = cmd.Flags().Lookup(alias)
	}
	if flag == nil || !flag.Changed {
		return zeroTaskValue(param.Type)
	}
	value, err := parseTaskParamValue(param, flag.Value.String())
	if err != nil {
		return zeroTaskValue(param.Type)
	}
	return value
}

func zeroTaskValue(paramType cavefile.TaskParamType) runtime.RuntimeValue {
	switch paramType {
	case cavefile.TaskParamTypeBool:
		return runtime.Bool(false)
	case cavefile.TaskParamTypeString:
		return runtime.String("")
	case cavefile.TaskParamTypeInt:
		return runtime.Int(0)
	default:
		return runtime.Void{}
	}
}
