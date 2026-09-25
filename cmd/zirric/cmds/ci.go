package cmds

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(ciCmd)
}

var ciCmd = &cobra.Command{
	Use:     "ci",
	GroupID: commandGroupProject,
	Short:   "Generate continuous integration workflows",
	Long: "Generate continuous integration workflows.\n\n" +
		"A forge is named rather than detected, because a repository is often pushed to more than one.",
}
