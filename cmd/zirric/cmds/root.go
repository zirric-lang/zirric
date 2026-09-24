package cmds

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/diag"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
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
	rootCmd.SilenceUsage = true
	// Errors are printed here rather than by cobra, so that one carrying a position can be shown with the line it refers to.
	rootCmd.SilenceErrors = true

	if err := loadCavefileIfNeeded(cmdArgs); err != nil {
		reportError(err)
		return err
	}
	rootCmd.SetArgs(resolveScriptShorthand(cmdArgs))

	if err := rootCmd.Execute(); err != nil {
		reportError(err)
		return err
	}
	return nil
}

// reportError writes an error to stderr, with the source line it refers to when it names one.
func reportError(err error) {
	fmt.Fprintln(os.Stderr, "Error:", diag.Render(err, readSourceForDiagnostic))
}

// readSourceForDiagnostic reads the file an error's position names.
//
// A position carries a logical module URI rather than a path — "myproject/main.zirr" for a file that sits at "main.zirr" — so the leading segments are dropped one at a time until something reads. This is best effort by design: the excerpt is an extra, and an error that cannot find its source still prints its message and position.
func readSourceForDiagnostic(file string) ([]byte, error) {
	path := file
	if idx := strings.Index(path, "://"); idx >= 0 {
		path = path[idx+3:]
	}
	for {
		if content, err := os.ReadFile(path); err == nil {
			return content, nil
		}
		idx := strings.Index(path, "/")
		if idx < 0 {
			return nil, fmt.Errorf("no file found for %q", file)
		}
		path = path[idx+1:]
	}
}

var cavefilePath string

// banner heads the help of `zirric` itself. Subcommands keep their own descriptions, so it is shown once rather than above every usage screen.
const banner = `
 ███ █ ██▄ ██▄ █ ▄██
  ▄▀ █ █▄█ █▄█ █ █
 ▄▀  █ █▀▄ █▀▄ █ █
 ███ █ █ █ █ █ █ ▀██
`

// bannerStyle paints the banner yellow. lipgloss reads the color profile from stdout, so a redirected `zirric --help`, a dumb terminal and NO_COLOR each get the plain drawing rather than the escape sequences around it.
var bannerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

// renderBanner paints the drawing a line at a time. Rendering it as one block would pad every line out to the widest, padding the blank ones into runs of spaces, and the art is written to sit exactly as it is.
func renderBanner() string {
	lines := strings.Split(banner, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = bannerStyle.Render(line)
		}
	}
	return strings.Join(lines, "\n")
}

var rootCmd = &cobra.Command{
	Use:  "zirric",
	Long: renderBanner(),
	// A first argument may name a file to run rather than a command, so completion offers those too.
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{strings.TrimPrefix(zirricFileExtension, ".")}, cobra.ShellCompDirectiveFilterFileExt
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cavefilePath, "cavefile", "", "path to the Cavefile, overriding autodetection (must precede the subcommand)")
}

// extractCavefileFlag pulls a leading --cavefile flag out of args into cavefilePath and returns the remainder. Some task commands disable Cobra's own flag parsing, so this must run before Cobra ever sees the args.
// Everything else is passed through exactly as written, so that the flags Cobra handles itself — --version among them — still reach it.
func extractCavefileFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		// The flag only counts before the subcommand, so once one appears the rest is left alone.
		if !strings.HasPrefix(arg, "-") {
			out = append(out, args[i:]...)
			break
		}
		switch {
		case arg == "--cavefile":
			if i+1 < len(args) {
				cavefilePath = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--cavefile="):
			cavefilePath = strings.TrimPrefix(arg, "--cavefile=")
		default:
			out = append(out, arg)
		}
	}
	return out
}

// loadCavefileIfNeeded registers one command per declared task; a missing Cavefile is tolerated, but a name collision or an unsupported type is a hard error.
func loadCavefileIfNeeded(args []string) error {
	if !needsDeclaredTasks(args) {
		return nil
	}
	return registerTaskCommands()
}

// needsDeclaredTasks reports whether the command about to run has to know the project's tasks.
// Everything else is spared reading the Cavefile, which is what lets `zirric cave new` work in a directory that has none.
func needsDeclaredTasks(args []string) bool {
	// The root help is where a project's own tasks are found.
	if len(args) == 0 {
		return true
	}
	target := args[0]
	if target == cobra.ShellCompRequestCmd || target == cobra.ShellCompNoDescRequestCmd {
		// completion requests prefix the real command path with __complete/__completeNoDesc
		if len(args) < 2 {
			return true
		}
		target = args[1]
	}
	if strings.HasPrefix(target, "-") {
		return target == "-h" || target == "--help"
	}
	if target == taskCmd.Name() || slices.Contains(taskCmd.Aliases, target) {
		return true
	}
	return rootCommandFor(target) == nil
}

// rootCommandFor returns the built-in command answering to name, by its own name or an alias.
func rootCommandFor(name string) *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == name || slices.Contains(cmd.Aliases, name) {
			return cmd
		}
	}
	return nil
}

// resolveScriptShorthand turns `zirric main.zirr args...` into `zirric run main.zirr args...`.
// A registered command always wins, which is why this runs once the tasks are registered.
func resolveScriptShorthand(args []string) []string {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return args
	}
	if rootCommandFor(args[0]) != nil {
		return args
	}
	if !namesSomethingToRun(args[0]) {
		return args
	}
	return append([]string{runCmd.Name()}, args...)
}

// namesSomethingToRun reports whether target reads as a program rather than a mistyped command.
// A .zirr file counts whether or not it exists, so a typo names a missing file rather than an unknown command; a directory has to be there, since any word could otherwise be taken for one.
func namesSomethingToRun(target string) bool {
	if filepath.Ext(target) == zirricFileExtension {
		return true
	}
	info, err := os.Stat(target)
	return err == nil && info.IsDir()
}

const zirricFileExtension = ".zirr"
