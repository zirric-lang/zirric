package cmds

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(cavefileCmd)
	skipCavefileFetchForCmds["cavefile"] = false
}

var cavefileCmd = &cobra.Command{
	Use:   "cavefile",
	Short: "Describes the current Cavefile",
}
