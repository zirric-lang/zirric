package cmds

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(docsCmd)
	skipCavefileFetchForCmds["docs"] = false
}

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate documentation",
	Args:  cobra.NoArgs,
}
