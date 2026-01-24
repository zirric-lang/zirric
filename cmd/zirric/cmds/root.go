package cmds

import (
	"os"
	"slices"

	"github.com/spf13/cobra"
)

func Execute() error {
	delimiter := slices.Index(os.Args, "--")

	if delimiter != -1 {
		loadCavefileIfNeeded(os.Args[1:delimiter])
	} else {
		loadCavefileIfNeeded(os.Args[1:])
	}

	rootCmd.SetArgs(os.Args[1:delimiter])
	return rootCmd.Execute()
}

var skipCavefileFetchForCmds = map[string]bool{}

var rootCmd = &cobra.Command{
	Use: "zirric",
}

func loadCavefileIfNeeded(args []string) {
	if len(args) > 0 {
		if skipCavefileFetchForCmds[args[0]] {
			return
		}
	}

	// TODO: load Cavefile
}
