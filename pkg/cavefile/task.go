package cavefile

import (
	"sort"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
)

type TaskKind int

const (
	TaskKindExec TaskKind = iota
	TaskKindCall
)

type Task struct {
	Name    string
	Aliases []string
	Help    string
	Kind    TaskKind

	// Docs is the comment written above the declaration that declares the task.
	// @tasks.Help is what the CLI prints; this is what the declaration says about itself.
	Docs string

	// DeclName is the data declaration's own name, independent of Name (which @tasks.Name may override).
	DeclName string

	// Exec is the file to run; set only when Kind is TaskKindExec.
	Exec string

	Flags []TaskParam
	Args  []TaskParam
}

type TaskParam struct {
	Name    string
	Aliases []string
	Short   string
	Help    string
	Type    TaskParamType

	// Docs is the comment written above the field that declares the flag or argument.
	Docs string

	// DeclName is the field's own declared name, independent of Name (which @tasks.Name may override).
	DeclName string
}

type TaskParamType int

const (
	// TaskParamTypeUnsupported is the zero value: no type hint, or one outside Bool/String/Int.
	TaskParamTypeUnsupported TaskParamType = iota
	TaskParamTypeBool
	TaskParamTypeString
	TaskParamTypeInt
)

// extractTasks finds every data declaration carrying @tasks.Exec or @tasks.Call and converts it into a Task.
func extractTasks(cavefileMod *ast.ContextModule, tasksMod *ast.ContextModule, aliasMap map[string]registry.LogicalURI) []Task {
	var tasks []Task
	for _, sym := range cavefileMod.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		data, ok := sym.Decl.(*ast.DeclData)
		if !ok {
			continue
		}
		task, ok := dataToTask(data, tasksMod, aliasMap)
		if !ok {
			continue
		}
		tasks = append(tasks, task)
	}
	// Declarations come out of a map, so they are ordered here rather than left to differ between runs.
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Name < tasks[j].Name })
	return tasks
}

func dataToTask(data *ast.DeclData, tasksMod *ast.ContextModule, aliasMap map[string]registry.LogicalURI) (Task, bool) {
	task := Task{Name: strings.ToLower(data.Name.Value), DeclName: data.Name.Value, Docs: ast.DocsOf(data)}
	found := false
	for _, attr := range data.Attributes {
		switch {
		case isFutureCaveAttr(attr, "Exec", tasksMod, aliasMap):
			task.Kind = TaskKindExec
			task.Exec = firstStringArg(attr)
			found = true
		case isFutureCaveAttr(attr, "Call", tasksMod, aliasMap):
			task.Kind = TaskKindCall
			found = true
		case isFutureCaveAttr(attr, "Name", tasksMod, aliasMap):
			task.Name = firstStringArg(attr)
		case isFutureCaveAttr(attr, "Alias", tasksMod, aliasMap):
			task.Aliases = append(task.Aliases, stringArrayArgs(attr)...)
		case isFutureCaveAttr(attr, "Help", tasksMod, aliasMap):
			task.Help = firstStringArg(attr)
		}
	}
	if !found {
		return Task{}, false
	}
	task.Flags, task.Args = fieldsToTaskParams(data.Fields, tasksMod, aliasMap)
	return task, true
}

func fieldsToTaskParams(fields []ast.DeclField, tasksMod *ast.ContextModule, aliasMap map[string]registry.LogicalURI) (flags []TaskParam, args []TaskParam) {
	for _, field := range fields {
		param := TaskParam{Name: field.Name.Value, DeclName: field.Name.Value, Type: taskParamType(field.TypeHint), Docs: ast.DocsOf(field)}
		isFlag := false
		isArg := false
		for _, attr := range field.Attributes {
			switch {
			case isFutureCaveAttr(attr, "Flag", tasksMod, aliasMap):
				isFlag = true
			case isFutureCaveAttr(attr, "Arg", tasksMod, aliasMap):
				isArg = true
			case isFutureCaveAttr(attr, "Name", tasksMod, aliasMap):
				param.Name = firstStringArg(attr)
			case isFutureCaveAttr(attr, "Alias", tasksMod, aliasMap):
				param.Aliases = append(param.Aliases, stringArrayArgs(attr)...)
			case isFutureCaveAttr(attr, "Short", tasksMod, aliasMap):
				param.Short = firstCharArg(attr)
			case isFutureCaveAttr(attr, "Help", tasksMod, aliasMap):
				param.Help = firstStringArg(attr)
			}
		}
		switch {
		case isFlag:
			flags = append(flags, param)
		case isArg:
			args = append(args, param)
		}
	}
	return flags, args
}

func taskParamType(hint ast.TypeExpr) TaskParamType {
	if hint == nil {
		return TaskParamTypeUnsupported
	}
	switch hint.TypeExpression() {
	case "Bool":
		return TaskParamTypeBool
	case "String":
		return TaskParamTypeString
	case "Int":
		return TaskParamTypeInt
	default:
		return TaskParamTypeUnsupported
	}
}

func stringArrayArgs(attr *ast.DeclAttrInstance) []string {
	for _, arg := range attr.Arguments {
		arr, ok := arg.(*ast.ExprArray)
		if !ok {
			continue
		}
		var out []string
		for _, el := range arr.Elements {
			if s, ok := el.(*ast.ExprString); ok {
				out = append(out, s.Literal)
			}
		}
		return out
	}
	return nil
}

func firstCharArg(attr *ast.DeclAttrInstance) string {
	for _, arg := range attr.Arguments {
		if c, ok := arg.(*ast.ExprChar); ok {
			return string(c.Literal)
		}
	}
	return ""
}
