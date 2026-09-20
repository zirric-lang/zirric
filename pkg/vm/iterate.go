package vm

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
)

// iterReturnSignal is panicked by op.Return when a `return` inside a makeIterYieldFunc-resumed loop body pops below the resumeBodyUntil currently resuming it, meaning some frame is genuinely returning rather than just finishing a resumed chunk.
// targetDepth is that frame's homeIdx, identifying which of the nested makeIterYieldFunc/callIterate calls owns it; ret is its real return value. Never escapes the VM package.
type iterReturnSignal struct {
	targetDepth int
	ret         runtime.RuntimeValue
}

// makeIterYieldFunc builds the native `yield` callable: calling it binds the value into frame's local[bindingLocal] and resumes frame's bytecode from bodyStartIp to bodyEndIp.
func (vm *VM) makeIterYieldFunc(taskId TaskId, frame *Frame, bindingLocal, bodyStartIp, bodyEndIp int) *runtime.ExternFunc {
	impl := func(_ runtime.VMCaller, args []runtime.RuntimeValue) (runtime.RuntimeValue, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("yield expects exactly 1 argument, got %d", len(args))
		}
		savedIp := frame.ip
		frame.locals[bindingLocal] = args[0]
		frame.ip = bodyStartIp

		vm.pushFrame(frame)
		resumeDepth := vm.framesIdx

		defer func() {
			r := recover()
			if r == nil {
				return
			}
			sig, ok := r.(iterReturnSignal)
			if ok && sig.targetDepth == frame.homeIdx {
				// This call's own frame is the one genuinely returning, so this is where its real return gets applied.
				vm.framesIdx = frame.homeIdx
				vm.sp = frame.basep
				_ = vm.push(sig.ret) // cannot overflow: sp was just lowered to frame.basep
			}
			// Keep propagating regardless: the generator that invoked this yield was only running on frame's behalf and must unwind too, out to the callIterate that owns the named frame.
			panic(r)
		}()

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

// callIterate reentrantly calls callee (the resolved `iterate` function) with value and yield.
// Its recover is where iterReturnSignal propagation stops, since makeIterYieldFunc applies the returning frame's state but always re-panics, leaving the generator that triggered it still on the Go stack.
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

	guardedYield := runtime.MakeNativeFunc("yield", 1, func(_ runtime.VMCaller, args []runtime.RuntimeValue) (runtime.RuntimeValue, error) {
		if !active {
			return nil, fmt.Errorf("yield called after its for-loop's iterate() call already returned")
		}
		return yieldFn.Impl(vm, args)
	})

	newFrame.locals[0] = value
	newFrame.locals[1] = guardedYield

	startFramesIdx := vm.framesIdx // depth including the calling frame, before pushing callee's frame
	vm.pushFrame(newFrame)

	defer func() {
		r := recover()
		if r == nil {
			return
		}
		sig, ok := r.(iterReturnSignal)
		if !ok || sig.targetDepth > newFrame.homeIdx {
			panic(r)
		}
		// The returning frame is newFrame or something nested in its call chain; its real return was already applied by the makeIterYieldFunc call that owns it, so only propagation needs stopping.
		err = nil
	}()

	if runErr := vm.runTaskBounded(taskId, startFramesIdx); runErr != nil {
		return runErr
	}

	_ = vm.pop()

	return nil
}
