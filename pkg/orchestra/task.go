package orchestra

import (
	"context"
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
	"code.knabel.dev/zirric-lang/zirric/pkg/vm"
)

const futureTasksModuleURI = "future.tasks"

// FindTask looks up a declared task by name or alias.
func (o *Orchestra) FindTask(name string) (cavefile.Task, bool) {
	for _, task := range o.cave.Tasks {
		if task.Name == name {
			return task, true
		}
		for _, alias := range task.Aliases {
			if alias == name {
				return task, true
			}
		}
	}
	return cavefile.Task{}, false
}

// RunTask executes a declared task by name or alias. values feeds a TaskKindCall task's fields; execArgs feeds a TaskKindExec task's argv.
func (o *Orchestra) RunTask(ctx context.Context, name string, values map[string]runtime.RuntimeValue, execArgs []string) error {
	task, ok := o.FindTask(name)
	if !ok {
		return fmt.Errorf("task %q not found", name)
	}
	switch task.Kind {
	case cavefile.TaskKindExec:
		return o.RunFileWithArgs(ctx, task.Exec, execArgs)
	case cavefile.TaskKindCall:
		return o.runCallTask(ctx, task, values)
	default:
		return fmt.Errorf("task %q is not runnable yet", name)
	}
}

func (o *Orchestra) runCallTask(ctx context.Context, task cavefile.Task, values map[string]runtime.RuntimeValue) error {
	machine, dt, fn, err := o.resolveCallFunction(ctx, task)
	if err != nil {
		return err
	}

	dataValue, err := makeTaskDataValue(dt, values)
	if err != nil {
		return fmt.Errorf("task %q: %w", task.Name, err)
	}

	_, err = machine.CallFunction(fn, dataValue)
	return err
}

// resolveCallFunction compiles the Cavefile for real and retrieves the compiled function behind a TaskKindCall task's @tasks.Call attribute.
func (o *Orchestra) resolveCallFunction(ctx context.Context, task cavefile.Task) (*vm.VM, *runtime.DataType, runtime.RuntimeValue, error) {
	bytecode, module, resolver, err := o.compileCavefileForTasks(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	machine := vm.New(bytecode)
	if err := machine.Run(); err != nil {
		return nil, nil, nil, err
	}

	sym, ok := module.Symbols.Symbols[task.DeclName]
	if !ok || sym.ConstantId == nil {
		return nil, nil, nil, fmt.Errorf("task %q: declaration %q not found", task.Name, task.DeclName)
	}
	dt, ok := bytecode.Constants[*sym.ConstantId].(*runtime.DataType)
	if !ok {
		return nil, nil, nil, fmt.Errorf("task %q: declaration %q is not a data type", task.Name, task.DeclName)
	}

	futureTasksMod, err := resolver.ResolveModule(ctx, futureTasksModuleURI)
	if err != nil {
		return nil, nil, nil, err
	}
	callSym, ok := futureTasksMod.Symbols.Symbols["Call"]
	if !ok || callSym.ConstantId == nil {
		return nil, nil, nil, fmt.Errorf("future.tasks: Call attribute not found")
	}

	globalId, ok := dt.Attributes[runtime.TypeId(*callSym.ConstantId)]
	if !ok {
		return nil, nil, nil, fmt.Errorf("task %q: missing @tasks.Call attribute", task.Name)
	}

	attrValue, err := machine.ResolveGlobal(globalId)
	if err != nil {
		return nil, nil, nil, err
	}
	av, ok := attrValue.(*runtime.AttributeValue)
	if !ok || len(av.Values) == 0 {
		return nil, nil, nil, fmt.Errorf("task %q: @tasks.Call attribute has no function", task.Name)
	}

	return machine, dt, av.Values[0], nil
}

// makeTaskDataValue builds the data value for a TaskKindCall task's function, ordering values per dt.FieldSymbols.
func makeTaskDataValue(dt *runtime.DataType, values map[string]runtime.RuntimeValue) (*runtime.DataValue, error) {
	vals := make([]runtime.RuntimeValue, len(dt.FieldSymbols))
	for i, fsym := range dt.FieldSymbols {
		name := fsym.Decl.DeclName().String()
		v, ok := values[name]
		if !ok {
			return nil, fmt.Errorf("missing value for field %q", name)
		}
		vals[i] = v
	}
	return runtime.MakeDataValue(dt, vals), nil
}
