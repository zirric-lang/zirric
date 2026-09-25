package cmds

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:                "run <script> [args...]",
	GroupID:            commandGroupCode,
	Short:              "Run a Zirric program",
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		absPath, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
		projectFS, err := cwdFS()
		if err != nil {
			return err
		}
		orch, err := newOrchestra(projectFS, name)
		if err != nil {
			return err
		}

		ctx := context.Background()
		return runPath(ctx, orch, args[0], args[1:])
	},
}
