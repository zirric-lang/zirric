package cmds

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(testCmd)
	skipCavefileFetchForCmds["test"] = false
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run tests",
	Args:  cobra.NoArgs,
}
