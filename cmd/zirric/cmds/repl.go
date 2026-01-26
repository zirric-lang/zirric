package cmds

import (
	"bytes"
	"context"
	"io"
	"os"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(replCmd)
	skipCavefileFetchForCmds["repl"] = false
}

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Start the Zirric REPL (Read-Eval-Print Loop)",
	RunE: func(cmd *cobra.Command, args []string) error {
		orch, err := newOrchestra()
		if err != nil {
			return err
		}

		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(input)) == 0 {
			return nil
		}

		replSource := append([]byte("module repl\n"), input...)
		if err := writeProjectFile(".zirric/repl.zirr", replSource); err != nil {
			return err
		}

		ctx := context.Background()
		cave := cavefile.Cavefile{}
		return orch.RunFile(ctx, ".zirric/repl.zirr", cave)
	},
}
