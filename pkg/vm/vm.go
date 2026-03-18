package vm

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/compiler"
	"code.knabel.dev/zirric-lang/zirric/pkg/op"
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
}

func newClosureFrame(closure *runtime.Closure, basep int) *Frame {
	return &Frame{
		ins:     closure.Fn.Instructions,
		ip:      0,
		basep:   basep,
		closure: closure,
		locals:  make([]runtime.RuntimeValue, closure.Fn.Locals),
	}
}
func newGeneralFrame(ins op.Instructions, basep int, locals int) *Frame {
	return &Frame{
		ins:    ins,
		ip:     0,
		basep:  basep,
		locals: make([]runtime.RuntimeValue, locals),
	}
}

func (f *Frame) Instructions() op.Instructions {
	return f.ins
}

type VM struct {
	constants []runtime.RuntimeValue
	globals   []*Global
	stack     []runtime.RuntimeValue
	sp        int
	frames    []*Frame
	framesIdx int
}

func New(bytecode *compiler.Bytecode) *VM {
	frames := make([]*Frame, maxFrames)
	frames[0] = newGeneralFrame(bytecode.Instructions, 0, bytecode.MainLocals)

	vm := &VM{
		stack:     make([]runtime.RuntimeValue, stackSize),
		sp:        0,
		constants: bytecode.Constants,
		globals:   make([]*Global, len(bytecode.Globals)),
		frames:    frames,
		framesIdx: 1,
	}

	for i := range bytecode.Globals { // TODO: what is going on here?
		ins := bytecode.Globals[i].Instructions
		locals := bytecode.Globals[i].LocalsCount()
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

// hasAttribute checks whether a type (identified by typeId) has the given attribute.
// Only the type's own declared attributes are checked — union attributes do NOT
// propagate to member types.
func (vm *VM) hasAttribute(typeId runtime.TypeId, attrConstId runtime.TypeId) bool {
	idx := int(typeId)
	if idx >= 0 && idx < len(vm.constants) {
		if a, ok := vm.constants[idx].(runtime.Attributable); ok {
			if _, found := a.TypeAttributes()[attrConstId]; found {
				return true
			}
		}
	}
	return false
}

// CallFunction calls a zero-argument CompiledFunction in the VM and returns its result.
func (vm *VM) CallFunction(fn runtime.RuntimeValue) (runtime.RuntimeValue, error) {
	compiledFn, ok := fn.(*runtime.CompiledFunction)
	if !ok {
		return nil, fmt.Errorf("CallFunction: expected *runtime.CompiledFunction, got %T", fn)
	}
	closure := runtime.MakeClosure(compiledFn, nil)
	frame := newClosureFrame(closure, vm.sp)
	vm.pushFrame(frame)
	if err := vm.Run(); err != nil {
		return nil, err
	}
	return vm.pop(), nil
}

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.framesIdx-1]
}

func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.framesIdx] = f
	vm.framesIdx++
}

func (vm *VM) popFrame() *Frame {
	vm.framesIdx--
	return vm.frames[vm.framesIdx]
}
