package cmds

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/osfs"
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
		absPath, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
		orch, err := newOrchestra(osfs.New("."), name)
		if err != nil {
			return err
		}

		ctx := context.Background()
		return runPath(ctx, orch, args[0])
	},
}
