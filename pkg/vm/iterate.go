package vm

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
)

// iterReturnSignal is panicked when a resumed loop body hits `return`; op.CallIterate recovers it. Never escapes the VM package.
type iterReturnSignal struct{}

// makeIterYieldFunc builds the native `yield` callable: calling it binds the value into frame's local[bindingLocal] and resumes frame's bytecode from bodyStartIp to bodyEndIp.
func (vm *VM) makeIterYieldFunc(taskId TaskId, frame *Frame, bindingLocal, bodyStartIp, bodyEndIp int) *runtime.ExternFunc {
	impl := func(args []runtime.RuntimeValue) (runtime.RuntimeValue, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("yield expects exactly 1 argument, got %d", len(args))
		}

		savedIp := frame.ip
		frame.locals[bindingLocal] = args[0]
		frame.ip = bodyStartIp

		vm.pushFrame(frame)
		resumeDepth := vm.framesIdx

		if err := vm.resumeBodyUntil(taskId, resumeDepth, bodyEndIp); err != nil {
			return nil, err
		}

		// Normal completion: pop the duplicate frame and restore the real ip.
		vm.popFrame()
		frame.ip = savedIp

		return vm.pop(), nil
	}

	return runtime.MakeNativeFunc("yield", 1, impl)
}

// callIterate reentrantly calls callee (the resolved `iterate` function) with value and yield, so an iterReturnSignal from deep in the resumed body can be caught here and turned into the enclosing frame's real return.
func (vm *VM) callIterate(taskId TaskId, callee, value, yield runtime.RuntimeValue) (err error) {
	var newFrame *Frame
	switch c := callee.(type) {
	case *runtime.CompiledFunction:
		newFrame = newClosureFrame(runtime.MakeClosure(c, nil), vm.sp)
	case *runtime.Closure:
		newFrame = newClosureFrame(c, vm.sp)
	default:
		return fmt.Errorf("@Iterable.iterate must be a function, got %T", callee)
	}

	if newFrame.closure.Arity() != 2 {
		return fmt.Errorf("@Iterable.iterate must take 2 parameters (value, yield), got %d", newFrame.closure.Arity())
	}

	yieldFn, ok := yield.(*runtime.ExternFunc)
	if !ok {
		return fmt.Errorf("internal: yield value is %T, not *runtime.ExternFunc", yield)
	}

	// Guards against a misbehaving `iterate` calling a stored `yield` after returning.
	active := true
	defer func() { active = false }()

	guardedYield := runtime.MakeNativeFunc("yield", 1, func(args []runtime.RuntimeValue) (runtime.RuntimeValue, error) {
		if !active {
			return nil, fmt.Errorf("yield called after its for-loop's iterate() call already returned")
		}
		return yieldFn.Impl(args)
	})

	newFrame.locals[0] = value
	newFrame.locals[1] = guardedYield

	startFramesIdx := vm.framesIdx // depth including the calling frame, before pushing callee's frame
	vm.pushFrame(newFrame)

	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(iterReturnSignal); ok {
				// op.Return already fixed up vm.sp/pushed the value; just discard the still-active frames.
				vm.framesIdx = startFramesIdx - 1
				err = nil
				return
			}
			panic(r)
		}
	}()

	if runErr := vm.runTaskBounded(taskId, startFramesIdx); runErr != nil {
		return runErr
	}

	_ = vm.pop()

	return nil
}
