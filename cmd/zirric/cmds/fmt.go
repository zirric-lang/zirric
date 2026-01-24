package cmds

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(fmtCmd)
	skipCavefileFetchForCmds["fmt"] = false
}

var fmtCmd = &cobra.Command{
	Use:   "fmt",
	Short: "Format code",
	Args:  cobra.NoArgs,
}
