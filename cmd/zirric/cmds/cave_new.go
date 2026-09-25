package cmds

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"code.knabel.dev/zirric-lang/zirric/pkg/orchestra"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/toolchain"
	"github.com/go-git/go-billy/v5"
	git "github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
)

type caveNewOptions struct {
	modulePath       string
	version          string
	languageVersion  string
	gitURL           string
	description      string
	documentationURL string
}

var caveNewOpts caveNewOptions

func init() {
	caveCmd.AddCommand(caveNewCmd)

	flags := caveNewCmd.Flags()
	flags.StringVar(&caveNewOpts.modulePath, "mod", "", "the package's module path; derived from the Git remote when there is one")
	flags.StringVar(&caveNewOpts.version, "version", "", "the package's own version")
	flags.StringVar(&caveNewOpts.languageVersion, "language-version", defaultLanguageVersion(), "which Zirric versions can build this package")
	flags.StringVar(&caveNewOpts.gitURL, "git-url", "", "the repository the package is published at; read from the Git remote when there is one")
	flags.StringVar(&caveNewOpts.description, "description", "", "one line about the package")
	flags.StringVar(&caveNewOpts.documentationURL, "documentation-url", "", "where the package's documentation lives")
}

var caveNewCmd = &cobra.Command{
	Use:     "new",
	Aliases: []string{"n"},
	Short:   "Create a new Cavefile",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFS, err := cwdFS()
		if err != nil {
			return err
		}
		opts, err := resolveCaveNewOptions(caveNewOpts, projectFS.Root())
		if err != nil {
			return err
		}
		return runCaveNew(projectFS, opts, cmd.OutOrStdout())
	},
}

// defaultLanguageVersion asks for this Zirric or newer, and for nothing at all when the build stamped no version.
func defaultLanguageVersion() string {
	running := toolchain.Version()
	if running == nil {
		return ""
	}
	return ">=" + running.String()
}

// resolveCaveNewOptions fills in what a Git checkout already says: the repository it is published at, and the package name derived from it.
func resolveCaveNewOptions(opts caveNewOptions, dir string) (caveNewOptions, error) {
	if opts.gitURL == "" {
		opts.gitURL = gitRemoteURL(dir)
	}
	opts.gitURL = normalizeGitURL(opts.gitURL)

	if opts.modulePath == "" && opts.gitURL != "" {
		opts.modulePath = string(registry.CanonicalizeModuleSource(opts.gitURL))
	}
	if opts.modulePath == "" {
		return opts, errors.New("nothing to name the package after: pass --mod, or --git-url to derive it from")
	}
	if !registry.IsModulePath(opts.modulePath) {
		return opts, fmt.Errorf("%q is not a module path: it is dot-separated identifiers, as in code.knabel.dev.example.myapp", opts.modulePath)
	}
	return opts, nil
}

// gitRemoteURL reports the URL of the remote around dir, preferring "origin".
func gitRemoteURL(dir string) string {
	repo, err := git.PlainOpenWithOptions(dir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return ""
	}
	remotes, err := repo.Remotes()
	if err != nil || len(remotes) == 0 {
		return ""
	}
	// Sorted so that several remotes and no origin still pick the same one every time.
	sort.Slice(remotes, func(i, j int) bool { return remotes[i].Config().Name < remotes[j].Config().Name })
	for _, remote := range remotes {
		if remote.Config().Name != git.DefaultRemoteName {
			continue
		}
		if urls := remote.Config().URLs; len(urls) > 0 {
			return urls[0]
		}
	}
	for _, remote := range remotes {
		if urls := remote.Config().URLs; len(urls) > 0 {
			return urls[0]
		}
	}
	return ""
}

// normalizeGitURL rewrites the forms Git accepts into the one a Cavefile records: https, without a trailing .git.
func normalizeGitURL(url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return ""
	}
	url = strings.TrimSuffix(url, ".git")
	switch {
	case strings.HasPrefix(url, "ssh://"):
		url = "https://" + stripUserInfo(strings.TrimPrefix(url, "ssh://"))
	case strings.HasPrefix(url, "git://"):
		url = "https://" + strings.TrimPrefix(url, "git://")
	case !strings.Contains(url, "://"):
		// The scp-like form, git@host:owner/repo, whose colon separates a path rather than a port.
		if host, path, ok := strings.Cut(url, ":"); ok {
			url = "https://" + stripUserInfo(host) + "/" + path
		}
	}
	return strings.TrimSuffix(url, "/")
}

func stripUserInfo(host string) string {
	if _, rest, ok := strings.Cut(host, "@"); ok {
		return rest
	}
	return host
}

func runCaveNew(projectFS billy.Filesystem, opts caveNewOptions, out io.Writer) error {
	if _, err := projectFS.Stat(orchestra.DefaultCavefileName); err == nil {
		return fmt.Errorf("%s already exists", orchestra.DefaultCavefileName)
	}
	if err := writeProjectFile(projectFS, orchestra.DefaultCavefileName, []byte(caveNewContents(opts))); err != nil {
		return err
	}
	return writeCaveNewWelcome(out)
}

// docsBaseURL is the documentation site the welcome points at.
const docsBaseURL = "https://zirric.knabel.dev"

// The welcome is a signpost rather than a tutorial, so every entry is one line and the prose lives behind the links.
var (
	caveNewNextSteps = []struct{ command, purpose string }{
		{"zirric cave install", "install dependencies"},
		{"zirric ci github", "add a GitHub Actions workflow"},
		{"zirric ci forgejo", "add a Forgejo Actions workflow"},
		{"zirric test", "run the tests"},
	}
	caveNewLinks = []struct{ label, url string }{
		{"Getting started", docsBaseURL + "/guides/getting-started"},
		{"CLI reference", docsBaseURL + "/tooling/zirric-cli"},
	}
)

var (
	welcomeHeadingStyle = lipgloss.NewStyle().Bold(true)
	welcomeDimStyle     = lipgloss.NewStyle().Faint(true)
)

func writeCaveNewWelcome(out io.Writer) error {
	var b strings.Builder

	fmt.Fprintf(&b, "%s\n", renderBannerFor(out))
	fmt.Fprintf(&b, "created %s — welcome to Zirric.\n\n", orchestra.DefaultCavefileName)

	fmt.Fprintf(&b, "%s\n", paint(out, welcomeHeadingStyle, "Next"))
	width := 0
	for _, step := range caveNewNextSteps {
		width = max(width, len(step.command))
	}
	for _, step := range caveNewNextSteps {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, step.command, paint(out, welcomeDimStyle, step.purpose))
	}

	fmt.Fprintf(&b, "\n%s\n", paint(out, welcomeHeadingStyle, "Docs"))
	width = 0
	for _, link := range caveNewLinks {
		width = max(width, len(link.label))
	}
	for _, link := range caveNewLinks {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, link.label, paint(out, welcomeDimStyle, link.url))
	}

	_, err := io.WriteString(out, b.String())
	return err
}

// caveNewContents writes the manifest, leaving out every attribute nothing was said about.
func caveNewContents(opts caveNewOptions) string {
	var b strings.Builder
	fmt.Fprintf(&b, "mod %s\n\nimport cave\n\n@cave.Package()\n", opts.modulePath)
	for _, attr := range []struct{ name, value string }{
		{"Git", opts.gitURL},
		{"Version", opts.version},
		{"LanguageVersion", opts.languageVersion},
		{"Description", opts.description},
		{"Documentation", opts.documentationURL},
	} {
		if attr.value != "" {
			// Zirric string literals take the escapes Go's do, so quoting them the Go way is exact.
			fmt.Fprintf(&b, "@cave.%s(%s)\n", attr.name, strconv.Quote(attr.value))
		}
	}
	fmt.Fprintf(&b, "data %s {\n}\n", packageDataName(opts.modulePath))
	return b.String()
}

// packageDataName turns the module path's last segment into a type name: my_app becomes MyApp.
func packageDataName(modulePath string) string {
	segments := strings.Split(modulePath, ".")
	last := segments[len(segments)-1]

	var name strings.Builder
	for _, word := range strings.Split(last, "_") {
		if word == "" {
			continue
		}
		name.WriteString(strings.ToUpper(word[:1]))
		name.WriteString(word[1:])
	}
	if name.Len() == 0 {
		return "Package"
	}
	return name.String()
}
