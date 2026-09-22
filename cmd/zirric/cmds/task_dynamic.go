package cmds

import (
	"context"
	"fmt"
	"strconv"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// registerTaskCommands registers a cobra.Command per declared task under `task`, and again at the top level when nothing built in already answers to that name.
func registerTaskCommands() error {
	projectFS, err := cwdFS()
	if err != nil {
		return nil
	}
	orch, err := newOrchestra(projectFS, currentDirPackageName())
	if err != nil {
		return nil
	}

	seen := map[string]string{} // name/alias -> owning task's DeclName
	for _, task := range orch.Cavefile().Tasks {
		for _, name := range append([]string{task.Name}, task.Aliases...) {
			if existingDecl, ok := seen[name]; ok && existingDecl != task.DeclName {
				return fmt.Errorf("tasks %q and %q both use the name %q", existingDecl, task.DeclName, name)
			}
			seen[name] = task.DeclName
		}

		subCmd, err := newTaskCommand(task)
		if err != nil {
			return err
		}
		taskCmd.AddCommand(subCmd)

		// The built-in wins: `zirric fmt` must keep meaning the formatter, which calls the task of that name through itself.
		if rootCommandFor(task.Name) != nil {
			continue
		}
		rootTaskCmd, err := newTaskCommand(task)
		if err != nil {
			return err
		}
		rootTaskCmd.Aliases = freeAliases(task.Aliases)
		rootCmd.AddCommand(rootTaskCmd)
	}
	return nil
}

func freeAliases(aliases []string) []string {
	free := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		if rootCommandFor(alias) == nil {
			free = append(free, alias)
		}
	}
	return free
}

// newTaskCommand builds a cobra.Command for one task: TaskKindExec disables flag parsing (the script owns its own argv), TaskKindCall gets real typed flags and positional arguments.
func newTaskCommand(task cavefile.Task) (*cobra.Command, error) {
	use := task.Name
	for _, arg := range task.Args {
		use += " <" + arg.Name + ">"
	}

	cmd := &cobra.Command{
		Use:     use,
		Short:   task.Help,
		Aliases: task.Aliases,
	}

	switch task.Kind {
	case cavefile.TaskKindExec:
		cmd.DisableFlagParsing = true
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return runNamedTask(task.Name, args)
		}
	case cavefile.TaskKindCall:
		bindings, err := registerTaskFlags(cmd.Flags(), task.Flags)
		if err != nil {
			return nil, fmt.Errorf("task %q: %w", task.Name, err)
		}
		for _, arg := range task.Args {
			if arg.Type == cavefile.TaskParamTypeUnsupported {
				return nil, fmt.Errorf("task %q: argument %q has an unsupported type; only Bool, String, and Int are supported", task.Name, arg.Name)
			}
		}
		cmd.Args = cobra.ExactArgs(len(task.Args))
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			values := make(map[string]runtime.RuntimeValue, len(task.Flags)+len(task.Args))
			for _, b := range bindings {
				values[b.param.DeclName] = b.value()
			}
			for i, arg := range task.Args {
				v, err := parseTaskParamValue(arg, args[i])
				if err != nil {
					return err
				}
				values[arg.DeclName] = v
			}

			projectFS, err := cwdFS()
			if err != nil {
				return err
			}
			orch, err := newOrchestra(projectFS, currentDirPackageName())
			if err != nil {
				return err
			}
			return orch.RunTask(context.Background(), task.Name, values, nil)
		}
	default:
		return nil, fmt.Errorf("task %q is not runnable yet", task.Name)
	}

	return cmd, nil
}

// flagBinding pairs a TaskParam with its parsed pflag value.
type flagBinding struct {
	param     cavefile.TaskParam
	boolVal   *bool
	stringVal *string
	intVal    *int
}

func (b flagBinding) value() runtime.RuntimeValue {
	switch b.param.Type {
	case cavefile.TaskParamTypeBool:
		return runtime.Bool(*b.boolVal)
	case cavefile.TaskParamTypeString:
		return runtime.String(*b.stringVal)
	case cavefile.TaskParamTypeInt:
		return runtime.Int(*b.intVal)
	default:
		return runtime.Void{}
	}
}

// registerTaskFlags registers one typed pflag per TaskParam, plus one per declared alias bound to the same variable.
func registerTaskFlags(flags *pflag.FlagSet, params []cavefile.TaskParam) ([]flagBinding, error) {
	if len(params) == 0 {
		return nil, nil
	}
	bindings := make([]flagBinding, len(params))
	for i, param := range params {
		b := flagBinding{param: param}
		switch param.Type {
		case cavefile.TaskParamTypeBool:
			b.boolVal = flags.BoolP(param.Name, param.Short, false, param.Help)
			for _, alias := range param.Aliases {
				flags.BoolVar(b.boolVal, alias, false, param.Help)
			}
		case cavefile.TaskParamTypeString:
			b.stringVal = flags.StringP(param.Name, param.Short, "", param.Help)
			for _, alias := range param.Aliases {
				flags.StringVar(b.stringVal, alias, "", param.Help)
			}
		case cavefile.TaskParamTypeInt:
			b.intVal = flags.IntP(param.Name, param.Short, 0, param.Help)
			for _, alias := range param.Aliases {
				flags.IntVar(b.intVal, alias, 0, param.Help)
			}
		default:
			return nil, fmt.Errorf("flag %q has an unsupported type; only Bool, String, and Int are supported", param.Name)
		}
		bindings[i] = b
	}
	return bindings, nil
}

// parseTaskParamValue converts a raw positional argument to its declared RuntimeValue type.
func parseTaskParamValue(param cavefile.TaskParam, raw string) (runtime.RuntimeValue, error) {
	switch param.Type {
	case cavefile.TaskParamTypeBool:
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("argument %q: invalid Bool value %q", param.Name, raw)
		}
		return runtime.Bool(v), nil
	case cavefile.TaskParamTypeString:
		return runtime.String(raw), nil
	case cavefile.TaskParamTypeInt:
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("argument %q: invalid Int value %q", param.Name, raw)
		}
		return runtime.Int(v), nil
	default:
		return nil, fmt.Errorf("argument %q has an unsupported type; only Bool, String, and Int are supported", param.Name)
	}
}
