package cmds

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Execute() error {
	delimiter := slices.Index(os.Args, "--")

	var cmdArgs []string
	if delimiter != -1 {
		cmdArgs = os.Args[1:delimiter]
	} else {
		cmdArgs = os.Args[1:]
	}

	cmdArgs = extractCavefileFlag(cmdArgs)
	rootCmd.SetArgs(cmdArgs)

	if err := loadCavefileIfNeeded(cmdArgs); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return err
	}

	return rootCmd.Execute()
}

var skipCavefileFetchForCmds = map[string]bool{}

var cavefilePath string

var rootCmd = &cobra.Command{
	Use: "zirric",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cavefilePath, "cavefile", "", "path to the Cavefile, overriding autodetection (must precede the subcommand)")
}

// extractCavefileFlag pulls a leading --cavefile flag out of args into cavefilePath and returns the remainder. Some task commands disable Cobra's own flag parsing, so this must run before Cobra ever sees the args.
func extractCavefileFlag(args []string) []string {
	fs := pflag.NewFlagSet("peek", pflag.ContinueOnError)
	fs.ParseErrorsAllowlist = pflag.ParseErrorsAllowlist{UnknownFlags: true}
	fs.SetInterspersed(false)
	fs.Usage = func() {}
	fs.SetOutput(io.Discard)
	fs.StringVar(&cavefilePath, "cavefile", "", "")
	if err := fs.Parse(args); err != nil {
		return args
	}
	return fs.Args()
}

// loadCavefileIfNeeded registers one dynamic subcommand per declared task under `task run` and `x`; a missing Cavefile is tolerated, but a task-level problem (name collision, unsupported type) is a hard error.
func loadCavefileIfNeeded(args []string) error {
	if len(args) == 0 {
		return nil
	}
	target := args[0]
	if target == cobra.ShellCompRequestCmd || target == cobra.ShellCompNoDescRequestCmd {
		// completion requests prefix the real command path with __complete/__completeNoDesc
		if len(args) < 2 {
			return nil
		}
		target = args[1]
	}
	if skipCavefileFetchForCmds[target] {
		return nil
	}
	if target != "task" && target != "tasks" && target != "x" {
		return nil
	}
	return registerTaskCommands()
}
