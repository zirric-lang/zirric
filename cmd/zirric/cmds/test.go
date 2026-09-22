package cmds

import (
	"context"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(testCmd)
}

// defaultTestProgram is what `zirric test` runs when a project declares no task of its own.
const defaultTestProgram = `import runner = tests.runner

runner.runT()
`

var testCmd = &cobra.Command{
	Use:   "test [args...]",
	Short: "Run the project's tests",
	Long: "Run the project's tests.\n\n" +
		"A project that declares a `test` task runs that instead, arguments and all.\n" +
		"Otherwise every @tests.Test function in a module whose name ends in _t is run.",
	Args: cobra.ArbitraryArgs,
	// A declared task owns its own flags, and they must reach it rather than be parsed away here.
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFS, err := cwdFS()
		if err != nil {
			return err
		}
		orch, err := newOrchestra(projectFS, currentDirPackageName())
		if err != nil {
			return err
		}

		ran, err := runTaskInstead(cmd, cmd.Name(), args)
		if ran {
			return err
		}
		return orch.RunSource(context.Background(), "test.zirr", defaultTestProgram)
	},
}
