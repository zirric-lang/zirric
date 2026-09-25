package cmds

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/codefmt"
	"code.knabel.dev/zirric-lang/zirric/pkg/lexer"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"code.knabel.dev/zirric-lang/zirric/pkg/parser"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry/staticmodule"
	"github.com/go-git/go-billy/v5"
	billyutil "github.com/go-git/go-billy/v5/util"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(fmtCmd)

	fmtCmd.Flags().BoolVarP(&fmtOpts.list, "list", "l", false, "list files whose formatting differs")
	fmtCmd.Flags().BoolVar(&fmtOpts.check, "check", false, "exit non-zero if any file needs formatting")
	fmtCmd.Flags().BoolVar(&fmtOpts.diff, "diff", false, "print a unified diff instead of writing")
	fmtCmd.Flags().BoolVar(&fmtOpts.stdin, "stdin", false, "read from stdin and write the result to stdout")
	fmtCmd.Flags().StringVar(&fmtOpts.stdinPath, "stdin-filepath", "<stdin>", "file name reported for --stdin diagnostics")
	fmtCmd.Flags().BoolVarP(&fmtOpts.write, "write", "w", false, "write result to the source file (the default)")
	fmtCmd.Flags().BoolVar(&fmtOpts.noExcludes, "no-excludes", false, "ignore @cave.FormattingExcludes from the Cavefile")
}

type fmtFlags struct {
	list       bool
	check      bool
	diff       bool
	stdin      bool
	stdinPath  string
	write      bool
	noExcludes bool
}

var fmtOpts fmtFlags

var fmtCmd = &cobra.Command{
	Use:     "fmt [path...]",
	GroupID: commandGroupProject,
	Short:   "Format code",
	Long: "Format Zirric sources in place.\n\n" +
		"Without flags, every .zirr file and Cavefile under the given paths (or the\n" +
		"current directory) is rewritten. Formatting only ever changes whitespace.",
	Args:         cobra.ArbitraryArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFS, err := cwdFS()
		if err != nil {
			return err
		}
		excludes, err := projectFormattingExcludes(projectFS, !fmtOpts.noExcludes)
		if err != nil {
			return err
		}
		// The formatter goes first: a script task may end the process itself, which would swallow what the formatter found.
		failed := false
		drift, err := runFmt(projectFS, args, fmtOpts, excludes, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		if err != nil {
			reportError(err)
			failed = true
		}
		if drift && fmtOpts.check {
			// A bare error would add a redundant "Error:" line; the paths were already printed.
			failed = true
		}

		if taskErr := runTaskAlongside(cmd, cmd.Name(), args, cmd.ErrOrStderr()); taskErr != nil {
			reportError(taskErr)
			failed = true
		}

		if failed {
			os.Exit(1)
		}
		return nil
	},
}

// projectFormattingExcludes opens the project and, when wanted, reads the paths it excludes from formatting.
//
// A Cavefile that cannot be read is an error only when its excludes are wanted: reformatting a file it meant to exclude is worse than formatting nothing. A package this Zirric cannot build refuses either way, which is no decision about excludes.
func projectFormattingExcludes(projectFS billy.Filesystem, wantExcludes bool) (codefmt.Excludes, error) {
	path := cavefilePath
	if path == "" {
		path = orchestra.DefaultCavefileName
	}

	if _, err := projectFS.Stat(path); err != nil {
		if cavefilePath != "" {
			return nil, fmt.Errorf("cavefile not found: %s", path)
		}
		return nil, nil
	}

	if wantExcludes {
		src, err := billyutil.ReadFile(projectFS, path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		if err := checkCavefileSyntax(path, string(src)); err != nil {
			return nil, err
		}
	}

	orch, err := newOrchestra(projectFS, currentDirPackageName())
	if err != nil {
		// The file was read fine, so wrapping this as a read failure would mislead.
		var languageVersion *cavefile.LanguageVersionError
		if errors.As(err, &languageVersion) {
			return nil, err
		}
		if !wantExcludes {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if !wantExcludes {
		return nil, nil
	}
	return codefmt.Excludes(orch.Cavefile().FormattingExcludes), nil
}

// checkCavefileSyntax reports the first syntax error, since excludes cannot be trusted from a Cavefile that does not parse.
func checkCavefileSyntax(path, src string) error {
	lex, err := lexer.New(staticmodule.NewSourceString(registry.LogicalURI(path), src))
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	module := ast.MakeContextModule(registry.LogicalURI(path))
	p := parser.NewSourceParser(lex, module.Decls, path)
	p.ParseSourceFile()
	if errs := p.Errors(); len(errs) > 0 {
		return fmt.Errorf("%s is malformed, so its formatting excludes cannot be read; fix it or pass --no-excludes: %w", path, errs[0])
	}
	return nil
}

// runFmt formats each target and reports whether any differed, which --check turns into an exit code.
func runFmt(projectFS billy.Filesystem, paths []string, flags fmtFlags, excludes codefmt.Excludes, in io.Reader, out, errOut io.Writer) (bool, error) {
	if flags.stdin || (len(paths) == 1 && paths[0] == "-") {
		return formatStdin(flags, excludes, in, out, errOut)
	}

	targets, err := collectTargets(projectFS, paths, excludes, errOut)
	if err != nil {
		return false, err
	}

	drift := false
	for _, path := range targets {
		raw, err := billyutil.ReadFile(projectFS, path)
		if err != nil {
			return drift, fmt.Errorf("read %s: %w", path, err)
		}
		src := string(raw)

		formatted, err := codefmt.String(path, src, codefmt.DefaultOptions())
		if err != nil {
			_, _ = fmt.Fprintf(errOut, "%v\n", err)
			drift = true
			continue
		}
		if formatted == src {
			continue
		}
		drift = true

		switch {
		case flags.diff:
			_, _ = fmt.Fprint(out, unifiedDiff(path, src, formatted))
		case flags.list, flags.check:
			_, _ = fmt.Fprintln(out, path)
		default:
			if err := writeProjectFile(projectFS, path, []byte(formatted)); err != nil {
				return drift, err
			}
		}
	}
	return drift, nil
}

func formatStdin(flags fmtFlags, excludes codefmt.Excludes, in io.Reader, out, errOut io.Writer) (bool, error) {
	raw, err := io.ReadAll(in)
	if err != nil {
		return false, err
	}
	src := string(raw)

	// An editor piping a file names it with --stdin-filepath, so excludes apply to it too.
	if flags.stdinPath != "" && excludes.Match(flags.stdinPath) {
		_, _ = fmt.Fprintf(errOut, "%s: skipped by @cave.FormattingExcludes (use --no-excludes to format it)\n", flags.stdinPath)
		_, _ = fmt.Fprint(out, src)
		return false, nil
	}
	formatted, err := codefmt.String(flags.stdinPath, src, codefmt.DefaultOptions())
	if err != nil {
		return false, err
	}
	if flags.list || flags.check {
		if formatted != src {
			_, _ = fmt.Fprintln(out, flags.stdinPath)
			return true, nil
		}
		return false, nil
	}
	_, _ = fmt.Fprint(out, formatted)
	return formatted != src, nil
}

// isZirricSource also accepts Cavefiles, which have no extension but are ordinary Zirric.
func isZirricSource(name string) bool {
	return filepath.Ext(name) == ".zirr" || filepath.Base(name) == "Cavefile"
}

// testdata is skipped by convention, since fixtures are often misformatted on purpose.
var skippedDirs = map[string]bool{
	".git": true, "node_modules": true, "site": true, "testdata": true,
}

// collectTargets expands paths into a sorted file list. Only directory walks apply the name filter, so an explicitly named path is always included.
func collectTargets(projectFS billy.Filesystem, paths []string, excludes codefmt.Excludes, errOut io.Writer) ([]string, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	seen := map[string]bool{}
	var targets []string
	add := func(p string) {
		p = filepath.Clean(p)
		if !seen[p] {
			seen[p] = true
			targets = append(targets, p)
		}
	}

	for _, path := range paths {
		path = filepath.Clean(path)
		info, err := projectFS.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", path, err)
		}
		if !info.IsDir() {
			// An excluded path is reported rather than silently skipped, since the user asked for it by name.
			if excludes.Match(path) {
				_, _ = fmt.Fprintf(errOut, "%s: skipped by @cave.FormattingExcludes (use --no-excludes to format it)\n", path)
				continue
			}
			add(path)
			continue
		}
		err = billyutil.Walk(projectFS, path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if skippedDirs[filepath.Base(p)] || excludes.MatchesDir(p) {
					return filepath.SkipDir
				}
				return nil
			}
			if isZirricSource(p) && !excludes.Match(p) {
				add(p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(targets)
	return targets, nil
}

const diffContext = 3

// unifiedDiff renders a unified diff with the usual three lines of context.
func unifiedDiff(path, before, after string) string {
	edits := diffLines(strings.Split(before, "\n"), strings.Split(after, "\n"))

	// Keep only lines within diffContext of a change.
	keep := make([]bool, len(edits))
	for i, e := range edits {
		if e[0] == ' ' {
			continue
		}
		for j := max(0, i-diffContext); j <= min(len(edits)-1, i+diffContext); j++ {
			keep[j] = true
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n", path, path)
	oldLine, newLine := 1, 1
	for i := 0; i < len(edits); i++ {
		if !keep[i] {
			switch edits[i][0] {
			case ' ':
				oldLine++
				newLine++
			case '-':
				oldLine++
			case '+':
				newLine++
			}
			continue
		}
		end := i
		for end < len(edits) && keep[end] {
			end++
		}
		oldCount, newCount := 0, 0
		for _, e := range edits[i:end] {
			if e[0] != '+' {
				oldCount++
			}
			if e[0] != '-' {
				newCount++
			}
		}
		fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n", oldLine, oldCount, newLine, newCount)
		for _, e := range edits[i:end] {
			sb.WriteString(e)
			sb.WriteString("\n")
		}
		oldLine += oldCount
		newLine += newCount
		i = end - 1
	}
	return sb.String()
}

// diffLines walks a longest-common-subsequence table, which is small enough to build outright for Zirric sources.
func diffLines(a, b []string) []string {
	lcs := make([][]int, len(a)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}

	var out []string
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			out = append(out, " "+a[i])
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			out = append(out, "-"+a[i])
			i++
		default:
			out = append(out, "+"+b[j])
			j++
		}
	}
	for ; i < len(a); i++ {
		out = append(out, "-"+a[i])
	}
	for ; j < len(b); j++ {
		out = append(out, "+"+b[j])
	}
	return out
}
