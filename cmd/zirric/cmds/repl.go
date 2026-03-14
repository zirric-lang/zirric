package cmds

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/compiler"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
	"code.knabel.dev/zirric-lang/zirric/pkg/vm"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(replCmd)
	skipCavefileFetchForCmds["repl"] = false
}

type replState struct {
	module   *ast.ContextModule
	resolver *orchestra.ModuleResolver
	analysis *analyzer.Analyzer
	comp     *compiler.Compiler
	machine  *vm.VM
	lineIdx  int
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

		resolver, err := orch.NewResolver()
		if err != nil {
			return err
		}

		module, err := orch.ParseFile(ctx, replPath, resolver)
		if err != nil {
			return err
		}

		analysis := analyzer.New(resolver)
		if errs, _ := analysis.Analyze(module, true); len(errs) > 0 {
			return fmt.Errorf("%s", errs[0].Error())
		}

		comp := compiler.NewWithAnalyzer(resolver, analysis)
		if err := comp.Compile(module); err != nil {
			return err
		}

		bytecode := comp.Bytecode()
		machine := vm.New(bytecode)
		if err := machine.Run(); err != nil {
			return err
		}

		state := &replState{
			module:   module,
			resolver: resolver,
			analysis: analysis,
			comp:     comp,
			machine:  machine,
		}

		reader := bufio.NewReader(os.Stdin)

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

			result, compileErr := replEvalLine(state, line)
			if compileErr != nil {
				if _, err := fmt.Fprintf(os.Stderr, "error: %s\n", compileErr); err != nil {
					return err
				}
			} else {
				if result == nil {
					if _, err := fmt.Fprintln(os.Stdout, "- void"); err != nil {
						return err
					}
				} else {
					if _, err := fmt.Fprintf(os.Stdout, "- %s\n", result.Inspect()); err != nil {
						return err
					}
				}
			}

			if isEOF {
				return nil
			}
		}
	},
}

func replEvalLine(state *replState, line string) (runtime.RuntimeValue, error) {
	state.lineIdx++
	lineURI := registry.LogicalURI(fmt.Sprintf("repl://line-%d", state.lineIdx))

	src := staticmodule.NewSourceString(lineURI, line)
	lex, err := lexer.New(src)
	if err != nil {
		return nil, err
	}

	// Snapshot symbol map keys before parsing; the parser may insert declarations
	// into module.Decls even when it ultimately returns an error.
	declsBefore := snapshotMapKeys(state.module.Decls.Symbols)

	prs := parser.NewSourceParser(lex, state.module.Decls, string(lineURI))
	file := prs.ParseSourceFile()
	if len(prs.Errors()) > 0 {
		removeAddedKeys(state.module.Decls.Symbols, declsBefore)
		return nil, prs.Errors()[0]
	}

	state.module.AddSourceFile(file)

	// Snapshot module.Symbols keys and analyzer ID counters before analysis so that
	// any IDs allocated during a failed attempt can be reclaimed on rollback.
	symbolsBefore := snapshotMapKeys(state.module.Symbols.Symbols)
	analyzerSnap := state.analysis.Snapshot()

	rollback := func() {
		state.module.Files = state.module.Files[:len(state.module.Files)-1]
		removeAddedKeys(state.module.Decls.Symbols, declsBefore)
		removeAddedKeys(state.module.Symbols.Symbols, symbolsBefore)
		state.analysis.Restore(analyzerSnap)
	}

	if errs := state.analysis.AnalyzeSourceFile(state.module, file); len(errs) > 0 {
		rollback()
		return nil, fmt.Errorf("%s", errs[0].Error())
	}

	prevGlobalsLen := len(state.comp.Bytecode().Globals)
	prevConstantsLen := len(state.comp.Bytecode().Constants)

	initConstantId, err := state.comp.CompileSourceFileIncremental(file)
	if err != nil {
		rollback()
		return nil, err
	}

	bytecode := state.comp.Bytecode()
	state.machine.ExtendGlobals(bytecode.Globals[prevGlobalsLen:])
	state.machine.ExtendConstants(bytecode.Constants[prevConstantsLen:])

	if initConstantId < 0 {
		return nil, nil
	}

	initFn := bytecode.Constants[initConstantId]
	return state.machine.CallFunction(initFn)
}

// snapshotMapKeys returns the current set of keys in m.
func snapshotMapKeys[V any](m map[string]V) map[string]struct{} {
	s := make(map[string]struct{}, len(m))
	for k := range m {
		s[k] = struct{}{}
	}
	return s
}

// removeAddedKeys deletes from m any key that was not present in before.
func removeAddedKeys[V any](m map[string]V, before map[string]struct{}) {
	for k := range m {
		if _, existed := before[k]; !existed {
			delete(m, k)
		}
	}
}
