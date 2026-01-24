package cmds

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(xCmd)
	skipCavefileFetchForCmds["x"] = false
}

var xCmd = &cobra.Command{
	Use:   "x <task-name> [args...]",
	Short: "Shorthand for `task run`",
	Args:  cobra.MinimumNArgs(1),
}
