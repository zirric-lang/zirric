package cmds

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/toolchain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.Version = toolchain.Description()
	// Registered here so cobra does not claim -v for it, which is the slot a verbosity flag belongs in.
	rootCmd.Flags().Bool("version", false, "print the version this binary was built as")
	// Cobra's default prefixes the binary name, making `zirric --version` and `zirric version` disagree.
	rootCmd.SetVersionTemplate("{{.Version}}\n")
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version of this Zirric build",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), toolchain.Description())
		return err
	},
}
