package cmds

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

var cavefileOutputFormat string

func init() {
	rootCmd.AddCommand(cavefileCmd)
	skipCavefileFetchForCmds["cavefile"] = false
	cavefileCmd.Flags().StringVarP(&cavefileOutputFormat, "output", "o", "yaml", `output format: "yaml" or "json"`)
}

var cavefileCmd = &cobra.Command{
	Use:   "cavefile",
	Short: "Describes the current Cavefile",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFS, err := cwdFS()
		if err != nil {
			return err
		}
		orch, err := newOrchestra(projectFS, currentDirPackageName())
		if err != nil {
			return err
		}
		return printCavefile(cmd.OutOrStdout(), orch.Cavefile(), cavefileOutputFormat)
	},
}

type cavefileDoc struct {
	Name         string          `yaml:"name" json:"name"`
	Source       string          `yaml:"source,omitempty" json:"source,omitempty"`
	Dependencies []dependencyDoc `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Tasks        []taskDoc       `yaml:"tasks,omitempty" json:"tasks,omitempty"`
}

type dependencyDoc struct {
	Name    string `yaml:"name" json:"name"`
	Kind    string `yaml:"kind,omitempty" json:"kind,omitempty"`
	Source  string `yaml:"source,omitempty" json:"source,omitempty"`
	Module  string `yaml:"module,omitempty" json:"module,omitempty"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

// dependencyKind classifies how a dependency resolves, mirroring the distinction
// pkg/cavefile/parse.go draws between @cave.Stdlib, @cave.Local, and @cave.Git.
func dependencyKind(dep cavefile.Dependency) string {
	switch {
	case dep.Module != "":
		return "stdlib"
	case strings.HasPrefix(dep.Source, "file://"):
		return "local"
	case dep.Source != "":
		return "git"
	default:
		return ""
	}
}

func dependencyVersion(dep cavefile.Dependency) string {
	parts := make([]string, len(dep.Predicates))
	for i, p := range dep.Predicates {
		parts[i] = p.String()
	}
	return strings.Join(parts, ", ")
}

type taskDoc struct {
	Name    string         `yaml:"name" json:"name"`
	Kind    string         `yaml:"kind" json:"kind"`
	Aliases []string       `yaml:"aliases,omitempty" json:"aliases,omitempty"`
	Help    string         `yaml:"help,omitempty" json:"help,omitempty"`
	Exec    string         `yaml:"exec,omitempty" json:"exec,omitempty"`
	Flags   []taskParamDoc `yaml:"flags,omitempty" json:"flags,omitempty"`
	Args    []taskParamDoc `yaml:"args,omitempty" json:"args,omitempty"`
}

type taskParamDoc struct {
	Name    string   `yaml:"name" json:"name"`
	Type    string   `yaml:"type,omitempty" json:"type,omitempty"`
	Short   string   `yaml:"short,omitempty" json:"short,omitempty"`
	Aliases []string `yaml:"aliases,omitempty" json:"aliases,omitempty"`
	Help    string   `yaml:"help,omitempty" json:"help,omitempty"`
}

func taskKind(kind cavefile.TaskKind) string {
	switch kind {
	case cavefile.TaskKindExec:
		return "exec"
	case cavefile.TaskKindCall:
		return "call"
	default:
		return ""
	}
}

func taskParamType(t cavefile.TaskParamType) string {
	switch t {
	case cavefile.TaskParamTypeBool:
		return "Bool"
	case cavefile.TaskParamTypeString:
		return "String"
	case cavefile.TaskParamTypeInt:
		return "Int"
	default:
		return ""
	}
}

func taskParamDocs(params []cavefile.TaskParam) []taskParamDoc {
	docs := make([]taskParamDoc, len(params))
	for i, p := range params {
		docs[i] = taskParamDoc{
			Name:    p.Name,
			Type:    taskParamType(p.Type),
			Short:   p.Short,
			Aliases: p.Aliases,
			Help:    p.Help,
		}
	}
	return docs
}

func printCavefile(w io.Writer, cave cavefile.Cavefile, format string) error {
	deps := make([]dependencyDoc, len(cave.Dependencies))
	for i, dep := range cave.Dependencies {
		deps[i] = dependencyDoc{
			Name:    dep.Name,
			Kind:    dependencyKind(dep),
			Source:  dep.Source,
			Module:  string(dep.Module),
			Version: dependencyVersion(dep),
		}
	}

	tasks := make([]taskDoc, len(cave.Tasks))
	for i, task := range cave.Tasks {
		tasks[i] = taskDoc{
			Name:    task.Name,
			Kind:    taskKind(task.Kind),
			Aliases: task.Aliases,
			Help:    task.Help,
			Exec:    task.Exec,
			Flags:   taskParamDocs(task.Flags),
			Args:    taskParamDocs(task.Args),
		}
	}

	doc := cavefileDoc{Name: cave.Name, Source: cave.Source, Dependencies: deps, Tasks: tasks}

	switch format {
	case "yaml":
		out, err := yaml.Marshal(doc)
		if err != nil {
			return err
		}
		_, err = w.Write(out)
		return err
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(doc)
	default:
		return fmt.Errorf(`unsupported --output %q: want "yaml" or "json"`, format)
	}
}
