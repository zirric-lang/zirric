package vm

import (
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
	ins   op.Instructions
	ip    int
	basep int

	locals []runtime.RuntimeValue
}

func newClosureFrame(closure *runtime.Closure, basep int) *Frame {
	return &Frame{
		ins:    closure.Fn.Instructions,
		ip:     0,
		basep:  basep,
		locals: make([]runtime.RuntimeValue, closure.Fn.Locals),
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
	frames[0] = newGeneralFrame(bytecode.Instructions, 0, 0)

	vm := &VM{
		stack:     make([]runtime.RuntimeValue, stackSize),
		sp:        0,
		constants: bytecode.Constants,
		globals:   make([]*Global, len(bytecode.Globals)),
		frames:    frames,
		framesIdx: 1,
	}

	for i := range bytecode.Globals {
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
