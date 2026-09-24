package runtime

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ ExternPlugin = &CoPlugin{}

// CoPlugin provides runtime bindings for the co module's extern declarations.
// It only translates between Zirric values and the scheduler.
type CoPlugin struct{}

func (*CoPlugin) Module() string { return "co" }

// Bind implements ExternPlugin.
func (*CoPlugin) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "scope":
		return MakeExternFunc(decl, coScope)
	case "spawn":
		return MakeExternFunc(decl, coSpawn)
	case "wait":
		return MakeExternFunc(decl, coWait)
	case "cancel":
		return MakeExternFunc(decl, coCancel)
	case "channel":
		return MakeExternFunc(decl, coChannel)
	case "send":
		return MakeExternFunc(decl, coSend)
	case "receive":
		return MakeExternFunc(decl, coReceive)
	case "close":
		return MakeExternFunc(decl, coClose)
	case "select":
		return MakeExternFunc(decl, coSelect)
	}
	return nil
}

// current returns the scheduler and the routine asking, always the one holding the run lock.
func current(caller VMCaller) (*Scheduler, *Routine, error) {
	if caller == nil {
		return nil, nil, fmt.Errorf("co can only be used while a VM is running")
	}
	sched := caller.Routines()
	routine := sched.Current()
	if routine == nil {
		return nil, nil, fmt.Errorf("co was called from outside any routine")
	}
	return sched, routine, nil
}

func coScope(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
	sched, routine, err := current(caller)
	if err != nil {
		return nil, err
	}
	scope := sched.OpenScope(routine)
	value, bodyErr := caller.CallFunction(args[0], scope)
	// Closed either way: leaving routines behind is what a scope exists to prevent.
	closeErr := sched.CloseScope(scope)
	if bodyErr != nil {
		return nil, bodyErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return value, nil
}

func coSpawn(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
	sched, _, err := current(caller)
	if err != nil {
		return nil, err
	}
	scope, ok := args[0].(*RoutineScope)
	if !ok {
		return nil, fmt.Errorf("co.spawn expects a Scope, got %s", TypeName(args[0]))
	}
	switch body := args[1].(type) {
	case *CompiledFunction, *Closure:
		return sched.Spawn(scope, body, caller.Fork())
	default:
		return nil, fmt.Errorf("co.spawn expects a function, got %s", TypeName(args[1]))
	}
}

func coWait(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
	sched, routine, err := current(caller)
	if err != nil {
		return nil, err
	}
	target, ok := args[0].(*Routine)
	if !ok {
		return nil, fmt.Errorf("co.wait expects a Routine, got %s", TypeName(args[0]))
	}
	result, err := sched.Wait(routine, target)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return cancelledResult(caller)
	}
	// A body that answered with something else is a bug in whoever spawned it, so it stops the program.
	if !isResultValue(caller, result) {
		return nil, fmt.Errorf("co.wait: the routine returned %s, not a Result", TypeName(result))
	}
	return result, nil
}

// cancelledResult builds Err(Cancelled()), how a stopped routine is reported.
func cancelledResult(caller VMCaller) (RuntimeValue, error) {
	member, err := caller.ResolveModuleMember("co", "Cancelled")
	if err != nil {
		return nil, err
	}
	dataType, ok := member.(*DataType)
	if !ok {
		return nil, fmt.Errorf("co.Cancelled is %s, not a data type", TypeName(member))
	}
	return ResultErr(caller, MakeDataValue(dataType, nil))
}

// isResultValue reports whether value is a prelude Result, what a routine body must answer with.
func isResultValue(caller VMCaller, value RuntimeValue) bool {
	resultType, err := caller.ResolveModuleMember("prelude", "Result")
	if err != nil {
		return true
	}
	return caller.IsType(value, resultType)
}

func coCancel(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
	sched, _, err := current(caller)
	if err != nil {
		return nil, err
	}
	scope, ok := args[0].(*RoutineScope)
	if !ok {
		return nil, fmt.Errorf("co.cancel expects a Scope, got %s", TypeName(args[0]))
	}
	sched.Cancel(scope)
	return Void{}, nil
}

func coChannel(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
	sched, _, err := current(caller)
	if err != nil {
		return nil, err
	}
	capacity, ok := args[0].(Int)
	if !ok {
		return nil, fmt.Errorf("co.channel expects an Int capacity, got %s", TypeName(args[0]))
	}
	if capacity < 0 {
		return nil, fmt.Errorf("co.channel expects a capacity of zero or more, got %d", capacity)
	}
	return sched.NewChannel(int(capacity)), nil
}

func coSend(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
	_, routine, err := current(caller)
	if err != nil {
		return nil, err
	}
	ch, ok := args[0].(*Channel)
	if !ok {
		return nil, fmt.Errorf("co.send expects a Channel, got %s", TypeName(args[0]))
	}
	if err := ch.Send(routine, args[1]); err != nil {
		return nil, err
	}
	return Void{}, nil
}

func coReceive(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
	_, routine, err := current(caller)
	if err != nil {
		return nil, err
	}
	ch, ok := args[0].(*Channel)
	if !ok {
		return nil, fmt.Errorf("co.receive expects a Channel, got %s", TypeName(args[0]))
	}
	value, open, err := ch.Receive(routine)
	if err != nil {
		return nil, err
	}
	if !open {
		return OptionNone(caller)
	}
	return OptionSome(caller, value)
}

func coClose(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
	if _, _, err := current(caller); err != nil {
		return nil, err
	}
	ch, ok := args[0].(*Channel)
	if !ok {
		return nil, fmt.Errorf("co.close expects a Channel, got %s", TypeName(args[0]))
	}
	if err := ch.Close(); err != nil {
		return nil, err
	}
	return Void{}, nil
}

func coSelect(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
	sched, routine, err := current(caller)
	if err != nil {
		return nil, err
	}
	list, ok := args[0].(Array)
	if !ok {
		return nil, fmt.Errorf("co.select expects an array of channels, got %s", TypeName(args[0]))
	}
	channels := make([]*Channel, len(list))
	for i, entry := range list {
		ch, ok := entry.(*Channel)
		if !ok {
			return nil, fmt.Errorf("co.select expects an array of channels, got %s at index %d", TypeName(entry), i)
		}
		channels[i] = ch
	}
	ch, value, open, err := sched.Select(routine, channels)
	if err != nil {
		return nil, err
	}
	if !open {
		return MakeDataValueNamed(caller, "co", "Closed", map[string]RuntimeValue{"channel": ch})
	}
	return MakeDataValueNamed(caller, "co", "Received", map[string]RuntimeValue{"channel": ch, "value": value})
}
