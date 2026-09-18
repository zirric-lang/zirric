package cmds

import (
	"fmt"
	"io"

	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"github.com/go-git/go-billy/v5"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
	skipCavefileFetchForCmds["init"] = true
}

const cavefileTemplate = `import cave

@cave.Dependencies()
data Dependencies {
}
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Cavefile",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFS, err := cwdFS()
		if err != nil {
			return err
		}
		return runInit(projectFS, cmd.OutOrStdout())
	},
}

func runInit(projectFS billy.Filesystem, out io.Writer) error {
	if _, err := projectFS.Stat(orchestra.DefaultCavefileName); err == nil {
		return fmt.Errorf("%s already exists", orchestra.DefaultCavefileName)
	}
	if err := writeProjectFile(projectFS, orchestra.DefaultCavefileName, []byte(cavefileTemplate)); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "created %s\n", orchestra.DefaultCavefileName)
	return err
}
