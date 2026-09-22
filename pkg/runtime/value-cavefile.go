package runtime

import (
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
)

// cavefileValue builds the reflect.packages.Cavefile describing the manifest a project declared itself with.
// It mirrors what `zirric cavefile` reports, so that a program reads the project the same way the CLI does.
func cavefileValue(caller VMCaller, cave cavefile.Cavefile) (RuntimeValue, error) {
	dependencies := make(Array, 0, len(cave.Dependencies))
	for _, dep := range cave.Dependencies {
		value, err := dependencyValue(caller, dep)
		if err != nil {
			return nil, err
		}
		dependencies = append(dependencies, value)
	}

	tasks := make(Array, 0, len(cave.Tasks))
	for _, task := range cave.Tasks {
		value, err := taskValue(caller, task)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, value)
	}

	excludes := make(Array, 0, len(cave.FormattingExcludes))
	for _, pattern := range cave.FormattingExcludes {
		excludes = append(excludes, String(pattern))
	}

	return MakeDataValueNamed(caller, "packages", "Cavefile", map[string]RuntimeValue{
		"name":               String(cave.Name),
		"source":             String(cave.Source),
		"docs":               String(cave.Docs),
		"dependencies":       dependencies,
		"tasks":              tasks,
		"formattingExcludes": excludes,
	})
}

func dependencyValue(caller VMCaller, dep cavefile.Dependency) (RuntimeValue, error) {
	predicates := make([]string, len(dep.Predicates))
	for i, predicate := range dep.Predicates {
		predicates[i] = predicate.String()
	}
	return MakeDataValueNamed(caller, "packages", "Dependency", map[string]RuntimeValue{
		"name":    String(dep.Name),
		"kind":    String(dependencyKind(dep)),
		"source":  String(dep.Source),
		"module":  String(dep.Module),
		"version": String(strings.Join(predicates, ", ")),
		"docs":    String(dep.Docs),
	})
}

// dependencyKind names how a dependency is reached, deciding it exactly as `zirric cavefile` does.
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

func taskValue(caller VMCaller, task cavefile.Task) (RuntimeValue, error) {
	flags, err := paramValues(caller, task.Flags)
	if err != nil {
		return nil, err
	}
	args, err := paramValues(caller, task.Args)
	if err != nil {
		return nil, err
	}
	kind := "call"
	if task.Kind == cavefile.TaskKindExec {
		kind = "exec"
	}
	return MakeDataValueNamed(caller, "packages", "Task", map[string]RuntimeValue{
		"name":     String(task.Name),
		"kind":     String(kind),
		"declName": String(task.DeclName),
		"aliases":  stringArray(task.Aliases),
		"help":     String(task.Help),
		"exec":     String(task.Exec),
		"docs":     String(task.Docs),
		"flags":    flags,
		"args":     args,
	})
}

func paramValues(caller VMCaller, params []cavefile.TaskParam) (Array, error) {
	result := make(Array, 0, len(params))
	for _, param := range params {
		value, err := MakeDataValueNamed(caller, "packages", "Param", map[string]RuntimeValue{
			"name":     String(param.Name),
			"declName": String(param.DeclName),
			"type":     String(paramTypeName(param.Type)),
			"short":    String(param.Short),
			"aliases":  stringArray(param.Aliases),
			"help":     String(param.Help),
			"docs":     String(param.Docs),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

// paramTypeName names the type a flag or argument takes, or the empty string for one the CLI cannot pass.
func paramTypeName(t cavefile.TaskParamType) string {
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

func stringArray(values []string) Array {
	result := make(Array, 0, len(values))
	for _, value := range values {
		result = append(result, String(value))
	}
	return result
}
