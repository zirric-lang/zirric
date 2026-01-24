package cmds

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(lintCmd)
	skipCavefileFetchForCmds["lint"] = false
}

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Run linters",
	Args:  cobra.NoArgs,
}
