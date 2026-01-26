package cmds

import (
	"context"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
	skipCavefileFetchForCmds["run"] = false
}

var runCmd = &cobra.Command{
	Use:   "run [script]",
	Short: "Run a Zirric program",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		orch, err := newOrchestra()
		if err != nil {
			return err
		}

		ctx := context.Background()
		cave := cavefile.Cavefile{}
		return runPath(ctx, orch, args[0], cave)
	},
}
