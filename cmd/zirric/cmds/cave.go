package cmds

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(caveCmd)
}

var caveCmd = &cobra.Command{
	Use:     "cave",
	GroupID: commandGroupProject,
	Aliases: []string{"cv"},
	Short:   "Work with the project's Cavefile",
}
