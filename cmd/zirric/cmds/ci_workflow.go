package cmds

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/go-git/go-billy/v5"
	"github.com/spf13/cobra"
)

// ciForge describes what differs between the forges: where the workflow is read from, and how an action outside the repository is named.
// Forgejo resolves a fully qualified URL in `uses`, which is how it reaches an action hosted outside itself; GitHub accepts only owner/repo, so it needs the mirror.
type ciForge struct {
	name               string
	aliases            []string
	short              string
	workflowPath       string
	defaultSetupAction string
}

var (
	forgejoForge = ciForge{
		name:               "forgejo",
		aliases:            []string{"fj", "gitea"},
		short:              "Create a Forgejo Actions workflow",
		workflowPath:       ".forgejo/workflows/zirric.yaml",
		defaultSetupAction: "https://code.knabel.dev/zirric-lang/setup-action@v1",
	}
	githubForge = ciForge{
		name:               "github",
		aliases:            []string{"gh"},
		short:              "Create a GitHub Actions workflow",
		workflowPath:       ".github/workflows/zirric.yaml",
		defaultSetupAction: "zirric-lang/setup-action@v1",
	}
)

type ciWorkflowOptions struct {
	setupAction string
	branch      string
	runsOn      string
	force       bool
}

func init() {
	ciCmd.AddCommand(
		newCIWorkflowCommand(forgejoForge),
		newCIWorkflowCommand(githubForge),
	)
}

// newCIWorkflowCommand builds the subcommand for one forge, with flags of its own so that the two never share parsed state.
func newCIWorkflowCommand(forge ciForge) *cobra.Command {
	var opts ciWorkflowOptions

	cmd := &cobra.Command{
		Use:     forge.name,
		Aliases: forge.aliases,
		Short:   forge.short,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectFS, err := cwdFS()
			if err != nil {
				return err
			}
			return runCIWorkflow(projectFS, forge, opts, cmd.OutOrStdout())
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.setupAction, "action", forge.defaultSetupAction, "the action that installs Zirric")
	flags.StringVar(&opts.branch, "branch", "main", "the branch pushes and pull requests are built for")
	flags.StringVar(&opts.runsOn, "runs-on", "ubuntu-latest", "the runner label the job asks for")
	flags.BoolVar(&opts.force, "force", false, "overwrite an existing workflow")

	return cmd
}

func runCIWorkflow(projectFS billy.Filesystem, forge ciForge, opts ciWorkflowOptions, out io.Writer) error {
	if _, err := projectFS.Stat(forge.workflowPath); err == nil && !opts.force {
		return fmt.Errorf("%s already exists; pass --force to overwrite it", forge.workflowPath)
	}

	contents, err := ciWorkflowContents(opts)
	if err != nil {
		return err
	}
	if err := writeProjectFile(projectFS, forge.workflowPath, []byte(contents)); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "created %s\n", forge.workflowPath)
	return err
}

// checkoutAction is pinned rather than made a flag: it is the one step that has nothing to do with Zirric, and a project that wants something else can edit the file it was handed.
const checkoutAction = "actions/checkout@v7"

var ciWorkflowTemplate = template.Must(template.New("workflow").Parse(`name: Zirric

on:
  push:
    branches: [{{.Branch}}]
  pull_request:
    branches: [{{.Branch}}]

jobs:
  test:
    runs-on: {{.RunsOn}}
    steps:
      - uses: {{.CheckoutAction}}

      - name: Setup Zirric
        uses: {{.SetupAction}}

      - name: Check formatting
        run: zirric fmt --check --diff

      - name: Run tests
        run: zirric test
`))

func ciWorkflowContents(opts ciWorkflowOptions) (string, error) {
	var b strings.Builder
	err := ciWorkflowTemplate.Execute(&b, struct {
		Branch         string
		RunsOn         string
		CheckoutAction string
		SetupAction    string
	}{
		Branch:         opts.branch,
		RunsOn:         opts.runsOn,
		CheckoutAction: checkoutAction,
		SetupAction:    opts.setupAction,
	})
	return b.String(), err
}
