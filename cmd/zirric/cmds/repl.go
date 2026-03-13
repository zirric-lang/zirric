package cmds

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/vm"
	"github.com/go-git/go-billy/v5/memfs"
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
		tmpfs := memfs.New()
		orch, err := newOrchestra(tmpfs, "repl")
		if err != nil {
			return err
		}

		ctx := context.Background()
		replPath := "repl.zirr"
		if err := writeProjectFile(tmpfs, replPath, []byte("module repl\n")); err != nil {
			return err
		}

		reader := bufio.NewReader(os.Stdin)
		resolver, err := orch.NewResolver()
		if err != nil {
			return err
		}

		for {
			if _, err := fmt.Fprint(os.Stdout, "> "); err != nil {
				return err
			}

			line, err := reader.ReadString('\n')
			isEOF := errors.Is(err, io.EOF)
			if err != nil && !isEOF {
				return err
			}

			line = strings.TrimRight(line, "\r\n")
			if strings.TrimSpace(line) == "" {
				if isEOF {
					return nil
				}
				continue
			}

			file, err := tmpfs.OpenFile(replPath, os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				return err
			}
			_, writeErr := fmt.Fprintln(file, line)
			closeErr := file.Close()
			if writeErr != nil {
				return writeErr
			}
			if closeErr != nil {
				return closeErr
			}

			module, err := orch.ParseFile(ctx, replPath, resolver)
			if err != nil {
				return err
			}
			bytecode, err := orch.Compile(module, resolver)
			if err != nil {
				return err
			}
			fmt.Println(bytecode.Instructions)
			machine := vm.New(bytecode)
			if err := machine.Run(); err != nil {
				return err
			}

			// TODO:
			// Actually we need to start reading here...
			// Everything up to now should be preserved between runs.
			// We need to parse...
			// We need to analyze... (this might be tricky)
			// We need to compile... (eventually do a sub-slice of the instructions)
			// We need to run...
			// Is it safe to call everything twice?

			result := machine.LastPoppedStackElem()
			if result == nil {
				if _, err := fmt.Fprintln(os.Stdout, "- void"); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(os.Stdout, "- %s\n", result.Inspect()); err != nil {
					return err
				}
			}

			if isEOF {
				return nil
			}
		}
	},
}
