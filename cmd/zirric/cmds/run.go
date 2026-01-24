package cmds

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(runCmd)
	skipCavefileFetchForCmds["run"] = false
}

var runCmd = &cobra.Command{
	Use:   "run [script]",
	Short: "Run a Zirric program",
	Args:  cobra.ExactArgs(1),
}
