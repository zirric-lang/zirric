package cmds

import (
	"context"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(taskCmd)
}

var taskCmd = &cobra.Command{
	Use:     "task [name] [args...]",
	GroupID: commandGroupProject,
	Aliases: []string{"tasks"},
	Short:   "List the declared tasks, or run one by name",
	Args:    cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFS, err := cwdFS()
		if err != nil {
			return err
		}
		orch, err := newOrchestra(projectFS, currentDirPackageName())
		if err != nil {
			return err
		}
		// A declared task is a subcommand of its own, so anything reaching here names none.
		if len(args) > 0 {
			return fmt.Errorf("unknown task %q", args[0])
		}
		return printTasks(cmd.OutOrStdout(), orch.Cavefile().Tasks)
	},
}

func printTasks(w io.Writer, tasks []cavefile.Task) error {
	if len(tasks) == 0 {
		_, err := fmt.Fprintln(w, "no tasks declared")
		return err
	}
	width := 0
	for _, task := range tasks {
		// Counted in runes: the name is padded to sit in a column, not to occupy so many bytes.
		if count := utf8.RuneCountInString(task.Name); count > width {
			width = count
		}
	}
	for _, task := range tasks {
		line := strings.TrimRight(fmt.Sprintf("%-*s  %s", width, task.Name, task.Help), " ")
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

func runNamedTask(name string, execArgs []string) error {
	projectFS, err := cwdFS()
	if err != nil {
		return err
	}
	orch, err := newOrchestra(projectFS, currentDirPackageName())
	if err != nil {
		return err
	}
	return orch.RunTask(context.Background(), name, nil, execArgs)
}
