package cmds

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(initCmd)
	skipCavefileFetchForCmds["init"] = true
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Cavefile",
}
