package cmds

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(replCmd)
	skipCavefileFetchForCmds["repl"] = false
}

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Start the Zirric REPL (Read-Eval-Print Loop)",
}
