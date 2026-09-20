package vm

import (
	"fmt"
	"math/rand"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/compiler"
	"code.knabel.dev/zirric-lang/zirric/pkg/op"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
)

const (
	stackSize  = 2048
	globalSize = 65536
	maxFrames  = 1024
)

type Frame struct {
	ins     op.Instructions
	ip      int
	basep   int
	closure *runtime.Closure

	locals []runtime.RuntimeValue

	// homeIdx is the framesIdx this Frame was first pushed at — its real call depth, as opposed to the duplicate depths makeIterYieldFunc re-pushes it at while resuming its `for <-` loop bodies.
	homeIdx int
}

func newClosureFrame(closure *runtime.Closure, basep int) *Frame {
	return &Frame{
		ins:     closure.Fn.Instructions,
		ip:      0,
		basep:   basep,
		closure: closure,
		locals:  make([]runtime.RuntimeValue, closure.Fn.Locals),
		homeIdx: -1,
	}
}
func newGeneralFrame(ins op.Instructions, basep int, locals int) *Frame {
	return &Frame{
		ins:     ins,
		ip:      0,
		basep:   basep,
		locals:  make([]runtime.RuntimeValue, locals),
		homeIdx: -1,
	}
}

func (f *Frame) Instructions() op.Instructions {
	return f.ins
}

type VM struct {
	constants    []runtime.RuntimeValue
	globals      []*Global
	builtinTypes map[runtime.TypeId]runtime.Attributable
	stack        []runtime.RuntimeValue
	sp           int
	frames       []*Frame
	framesIdx    int
	// moduleGlobals maps a module URI to the global slot holding its ModuleValue, so an extern function can reach a module's exports by name.
	moduleGlobals map[registry.LogicalURI]int
	// taskId identifies the running task, so that a global reached reentrantly from an extern function is recognized as recursive rather than waited on as another task's in-progress initialization.
	taskId TaskId
}

func New(bytecode *compiler.Bytecode) *VM {
	frames := make([]*Frame, maxFrames)
	frames[0] = newGeneralFrame(bytecode.Instructions, 0, bytecode.MainLocals)

	vm := &VM{
		stack:         make([]runtime.RuntimeValue, stackSize),
		sp:            0,
		constants:     bytecode.Constants,
		globals:       make([]*Global, len(bytecode.Globals)),
		builtinTypes:  make(map[runtime.TypeId]runtime.Attributable),
		frames:        frames,
		framesIdx:     1,
		moduleGlobals: bytecode.ModuleGlobals,
	}

	// Build a lookup table for builtin types whose TypeConstantId differs
	// from their position in the constants array (e.g. String at index 31
	// but with hardcoded typeIdString=8).
	for i, c := range bytecode.Constants {
		if a, ok := c.(runtime.Attributable); ok {
			tid := a.TypeConstantId()
			if int(tid) != i {
				vm.builtinTypes[tid] = a
			}
		}
	}

	for i := range bytecode.Globals { // TODO: what is going on here?
		scope := bytecode.Globals[i]
		if scope == nil {
			// A slot reserved without a module ever being compiled into it, which reflect.packages does for a package module that fails to compile on its own.
			id := i
			vm.globals[i] = MakeGlobal(func(TaskId) (runtime.RuntimeValue, error) {
				return nil, fmt.Errorf("global %d was reserved but never compiled", id)
			})
			continue
		}
		ins := scope.Instructions
		locals := scope.LocalsCount()
		vm.globals[i] = MakeGlobal(func(ti TaskId) (runtime.RuntimeValue, error) {
			return vm.initGlobal(ti, ins, locals)
		})
	}

	return vm
}

func (vm *VM) LastPoppedStackElem() runtime.RuntimeValue {
	return vm.stack[vm.sp]
}

// ExtendGlobals appends new global slots to a running VM.
func (vm *VM) ExtendGlobals(newGlobals []*compiler.CompilationScope) {
	for _, scope := range newGlobals {
		if scope == nil {
			vm.globals = append(vm.globals, MakeGlobal(func(TaskId) (runtime.RuntimeValue, error) {
				return nil, nil
			}))
			continue
		}
		ins := scope.Instructions
		locals := scope.LocalsCount()
		vm.globals = append(vm.globals, MakeGlobal(func(ti TaskId) (runtime.RuntimeValue, error) {
			return vm.initGlobal(ti, ins, locals)
		}))
	}
}

// ExtendConstants appends new constants to a running VM.
func (vm *VM) ExtendConstants(newConstants []runtime.RuntimeValue) {
	vm.constants = append(vm.constants, newConstants...)
}

// AttributesOf returns the attribute map that applies to v.
// Some values (e.g. a *CompiledFunction) carry their own, per-declaration attributes directly and implement Attributable themselves; others (e.g. a DataValue whose DataType attributes it mirrors) don't need the fallback either, but values with no direct Attributable implementation fall back to the attributes declared on their type, looked up via builtinTypes or the constants table.
// Only the type's own declared attributes are checked — union attributes do NOT propagate to member types.
func (vm *VM) AttributesOf(v runtime.RuntimeValue) map[runtime.TypeId]int {
	if a, ok := v.(runtime.Attributable); ok {
		return a.TypeAttributes()
	}
	tid := v.TypeConstantId()
	if a, ok := vm.builtinTypes[tid]; ok {
		return a.TypeAttributes()
	}
	idx := int(tid)
	if idx < 0 || idx >= len(vm.constants) {
		return nil
	}
	if a, ok := vm.constants[idx].(runtime.Attributable); ok {
		return a.TypeAttributes()
	}
	return nil
}

// hasAttribute reports whether v carries the given attribute — either as its own instance-level attribute (e.g. a specific function's own @Attr) or via its declared type's attributes (e.g. a data/union value).
func (vm *VM) hasAttribute(v runtime.RuntimeValue, attrConstId runtime.TypeId) bool {
	_, found := vm.AttributesOf(v)[attrConstId]
	return found
}

// CallFunction calls a CompiledFunction or Closure in the VM with the given arguments and returns its result.
// Safe to call reentrantly — e.g. from an extern function's Impl, itself invoked mid-execution of an already-running outer Run() — since it stops once execution has unwound back to the frame depth it started at, rather than running until the whole call stack (the entire rest of the outer program, for a reentrant call) is exhausted.
func (vm *VM) CallFunction(fn runtime.RuntimeValue, args ...runtime.RuntimeValue) (runtime.RuntimeValue, error) {
	var closure *runtime.Closure
	switch callee := fn.(type) {
	case *runtime.CompiledFunction:
		closure = runtime.MakeClosure(callee, nil)
	case *runtime.Closure:
		closure = callee
	default:
		return nil, fmt.Errorf("CallFunction: expected *runtime.CompiledFunction or *runtime.Closure, got %T", fn)
	}
	if len(args) != closure.Arity() {
		return nil, fmt.Errorf("CallFunction: wrong number of arguments: want=%d, got=%d", closure.Arity(), len(args))
	}

	stopIdx := vm.framesIdx
	frame := newClosureFrame(closure, vm.sp)
	for i, arg := range args {
		frame.locals[i] = arg
	}
	vm.pushFrame(frame)
	if err := vm.runTaskBounded(TaskId(rand.Uint64()), stopIdx); err != nil {
		return nil, err
	}
	return vm.pop(), nil
}

// ResolveGlobal forces evaluation of the global at index id, for use outside normal bytecode execution.
func (vm *VM) ResolveGlobal(id int) (runtime.RuntimeValue, error) {
	if id < 0 || id >= len(vm.globals) {
		return nil, fmt.Errorf("ResolveGlobal: index %d out of range (have %d globals)", id, len(vm.globals))
	}
	owner := vm.taskId
	if owner == 0 {
		owner = TaskId(rand.Uint64())
	}
	return vm.globals[id].Get(owner)
}

// ResolveModuleMember implements runtime.VMCaller.
// The name is matched against each compiled module's URI by suffix, the same way plugin binding resolves a module, so that "prelude" finds it whether it was compiled as "prelude" or as "<project>.prelude".
func (vm *VM) ResolveModuleMember(moduleName string, memberName string) (runtime.RuntimeValue, error) {
	id, ok := vm.moduleGlobalId(moduleName)
	if !ok {
		return nil, fmt.Errorf("ResolveModuleMember: module %q is not part of this program", moduleName)
	}
	value, err := vm.ResolveGlobal(id)
	if err != nil {
		return nil, err
	}
	module, ok := value.(*runtime.ModuleValue)
	if !ok {
		return nil, fmt.Errorf("ResolveModuleMember: global for %q holds %T, not a module", moduleName, value)
	}
	member := module.Lookup(memberName)
	if member == nil {
		return nil, fmt.Errorf("ResolveModuleMember: module %q has no public member %q", moduleName, memberName)
	}
	return member, nil
}

func (vm *VM) moduleGlobalId(moduleName string) (int, bool) {
	if id, ok := vm.moduleGlobals[registry.LogicalURI(moduleName)]; ok {
		return id, true
	}
	suffix := "." + moduleName
	for uri, id := range vm.moduleGlobals {
		if strings.HasSuffix(string(uri), suffix) {
			return id, true
		}
	}
	return 0, false
}

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.framesIdx-1]
}

func (vm *VM) pushFrame(f *Frame) {
	if f.homeIdx < 0 {
		f.homeIdx = vm.framesIdx
	}
	vm.frames[vm.framesIdx] = f
	vm.framesIdx++
}

func (vm *VM) popFrame() *Frame {
	vm.framesIdx--
	return vm.frames[vm.framesIdx]
}
