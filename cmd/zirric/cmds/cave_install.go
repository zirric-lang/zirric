package cmds

import (
	"context"
	"fmt"
	"io"

	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"code.knabel.dev/zirric-lang/zirric/pkg/pkgmanager"
	"github.com/spf13/cobra"
)

func init() {
	caveCmd.AddCommand(caveInstallCmd)
}

var caveInstallCmd = &cobra.Command{
	Use:     "install",
	Aliases: []string{"i"},
	Short:   "Install the dependencies the Cavefile declares",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFS, err := cwdFS()
		if err != nil {
			return err
		}
		orch, err := newOrchestra(projectFS, currentDirPackageName())
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		resolver, err := orch.NewResolver(orchestra.WithInstallProgress(func(evt pkgmanager.InstallEvent) {
			printInstalled(out, evt)
		}))
		if err != nil {
			return err
		}
		_, err = resolver.EnsureInstalled(context.Background())
		return err
	},
}

func printInstalled(w io.Writer, evt pkgmanager.InstallEvent) {
	fmt.Fprintf(w, "✓ %s (%s)\n", evt.Dependency.Name, evt.Package.Source())
}
