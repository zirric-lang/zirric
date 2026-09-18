package cmds

import (
	"context"

	"github.com/spf13/cobra"
)

func init() {
	taskCmd.AddCommand(taskRunCmd)
}

var taskRunCmd = &cobra.Command{
	Use:                "run <task-name> [args...]",
	Short:              "Run a task",
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNamedTask(args[0], args[1:])
	},
}

func runNamedTask(name string, execArgs []string) error {
	projectFS, err := cwdFS()
	if err != nil {
		return err
	}
	orch, err := newOrchestra(projectFS, currentDirPackageName())
	if err != nil {
		return err
	}
	return orch.RunTask(context.Background(), name, nil, execArgs)
}
