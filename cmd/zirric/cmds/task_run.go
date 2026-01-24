package cmds

import "github.com/spf13/cobra"

func init() {
	taskCmd.AddCommand(taskRunCmd)
}

var taskRunCmd = &cobra.Command{
	Use:   "run <task-name> [args...]",
	Short: "Run a task",
	Args:  cobra.MinimumNArgs(1),
}
