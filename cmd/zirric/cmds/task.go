package cmds

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(taskCmd)
	skipCavefileFetchForCmds["task"] = false
}

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "List available tasks",
}
