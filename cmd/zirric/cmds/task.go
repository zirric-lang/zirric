package cmds

import (
	"fmt"
	"io"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(taskCmd)
	skipCavefileFetchForCmds["task"] = false
}

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "List available tasks",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFS, err := cwdFS()
		if err != nil {
			return err
		}
		orch, err := newOrchestra(projectFS, currentDirPackageName())
		if err != nil {
			return err
		}
		return printTasks(cmd.OutOrStdout(), orch.Cavefile().Tasks)
	},
}

func printTasks(w io.Writer, tasks []cavefile.Task) error {
	if len(tasks) == 0 {
		_, err := fmt.Fprintln(w, "no tasks declared")
		return err
	}
	for _, task := range tasks {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", task.Name, task.Help); err != nil {
			return err
		}
	}
	return nil
}
