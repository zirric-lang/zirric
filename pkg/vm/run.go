package vm

import (
	"errors"
	"fmt"
	"math/rand"

	"code.knabel.dev/zirric-lang/zirric/pkg/op"
	"code.knabel.dev/zirric-lang/zirric/pkg/runtime"
)

func (vm *VM) Run() error {
	var taskId = TaskId(rand.Uint64())
	vm.taskId = taskId
	err := vm.runTask(taskId)
	if err != nil {
		// A program that stopped early may leave routines parked; one that returned normally already waited for its own.
		if sched := vm.scheduler(); sched != nil {
			sched.Shutdown(err)
		}
	}
	return err
}

// runTask runs the top-level, unbounded dispatch loop.
func (vm *VM) runTask(taskId TaskId) error {
	return vm.runTaskUntil(taskId, -1, -1, -1)
}

// runTaskBounded runs frames until the stack unwinds to (or below) stopIdx.
func (vm *VM) runTaskBounded(taskId TaskId, stopIdx int) error {
	return vm.runTaskUntil(taskId, stopIdx, -1, -1)
}

// resumeBodyUntil resumes the top frame until resumeDepth+endIp (done), or panics iterReturnSignal if the stack unwinds below resumeDepth (early return).
func (vm *VM) resumeBodyUntil(taskId TaskId, resumeDepth int, endIp int) error {
	return vm.runTaskUntil(taskId, -1, resumeDepth, endIp)
}

// runTaskUntil backs runTask/runTaskBounded/resumeBodyUntil with one inline opcode switch, since this loop runs once per instruction for the whole program.
// runTaskUntil runs frames and, if something fails, notes where.
//
// The trace is taken here because the frames are still live at this point: the loop returns rather than unwinding, so the stack is exactly as it was when the failure happened. An error that already carries a trace keeps it.
func (vm *VM) runTaskUntil(taskId TaskId, stopIdx int, resumeDepth int, endIp int) error {
	return vm.fail(vm.runLoop(taskId, stopIdx, resumeDepth, endIp))
}

func (vm *VM) runLoop(taskId TaskId, stopIdx int, resumeDepth int, endIp int) error {
	for {
		fr := vm.currentFrame()

		if stopIdx >= 0 && vm.framesIdx <= stopIdx {
			return nil
		}
		if resumeDepth >= 0 {
			// framesIdx also drops below resumeDepth when a nested `for <-`'s op.CallIterate returns normally after callIterate absorbed an inner signal for a frame at or below this resumeDepth (e.g. two `for <-` loops sharing one frame).
			// Re-raise with the state a matching makeIterYieldFunc already applied: framesIdx is the target depth, and the return value is still on top of the stack.
			if vm.framesIdx < resumeDepth {
				panic(iterReturnSignal{targetDepth: vm.framesIdx, ret: vm.stack[vm.sp-1]})
			}
			if vm.framesIdx == resumeDepth && fr.ip == endIp {
				return nil
			}
		}
		if fr.ip >= len(fr.Instructions()) {
			return nil
		}

		fr.ip++

		var (
			ip   = fr.ip
			ins  = fr.Instructions()
			code = op.Opcode(ins[ip-1])
		)
		switch code {
		case op.Pop:
			vm.pop()

		case op.Const:
			idx := op.ReadUint16(ins[ip:])
			fr.ip += 2

			err := vm.push(vm.constants[idx])
			if err != nil {
				return err
			}
		case op.ConstTrue:
			err := vm.push(runtime.Bool(true))
			if err != nil {
				return err
			}
		case op.ConstFalse:
			err := vm.push(runtime.Bool(false))
			if err != nil {
				return err
			}
		case op.ConstVoid:
			err := vm.push(runtime.Void{})
			if err != nil {
				return err
			}

		case op.Jump:
			pos := int(op.ReadUint16(ins[ip:]))
			fr.ip = pos
		case op.JumpFalse:
			pos := int(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			cond := vm.pop()

			if cond == runtime.Bool(false) {
				fr.ip = pos
			}
		case op.JumpTrue:
			pos := int(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			cond := vm.pop()

			if cond != runtime.Bool(false) {
				fr.ip = pos
			}

		case op.AssertType:
			typeId := runtime.TypeId(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			v := vm.stack[vm.sp-1]
			if v.TypeConstantId() != typeId {
				// An asserted type is checked against a constant id, which is not a name this can report, so it says what it found and leaves the expectation to the line itself.
				return fmt.Errorf("a value of a different type was expected here, got %s %s", typeNameOf(v), v.Inspect())
			}

		case op.IsType:
			constId := int(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			v := vm.pop()
			typeVal := vm.constants[constId]
			var result runtime.Bool
			switch typeVal.(type) {
			case *runtime.UnionType, *runtime.AttributeType, runtime.SimpleType, *runtime.DataType:
				result = runtime.Bool(vm.IsType(v, typeVal))
			default:
				// Not a type value at all, so the constant can only stand for itself.
				result = runtime.Bool(v.TypeConstantId() == runtime.TypeId(constId))
			}
			if err := vm.push(result); err != nil {
				return err
			}

		case op.AsOption:
			unionId := int(op.ReadUint16(ins[ip:]))
			attrId := int(op.ReadUint16(ins[ip+2:]))
			fr.ip += 4
			wrapped, err := vm.asWrapper(taskId, vm.pop(), unionId, attrId, "toOption", "?. and ??", "@AnyOption")
			if err != nil {
				return err
			}
			if err := vm.push(wrapped); err != nil {
				return err
			}

		case op.AsResult:
			unionId := int(op.ReadUint16(ins[ip:]))
			attrId := int(op.ReadUint16(ins[ip+2:]))
			fr.ip += 4
			wrapped, err := vm.asWrapper(taskId, vm.pop(), unionId, attrId, "toResult", "!. and !!", "@AnyResult")
			if err != nil {
				return err
			}
			if err := vm.push(wrapped); err != nil {
				return err
			}

		case op.JumpIsType:
			pos := int(op.ReadUint16(ins[ip:]))
			constId := int(op.ReadUint16(ins[ip+2:]))
			fr.ip += 4
			// The value stays put either way: on the jumping path it is the None a `?.` chain ends as, and on the other it is the target the field is read off.
			if vm.IsType(vm.stack[vm.sp-1], vm.constants[constId]) {
				fr.ip = pos
			}

		case op.ReturnIsType:
			constId := int(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			if !vm.IsType(vm.stack[vm.sp-1], vm.constants[constId]) {
				break
			}
			ret := vm.pop()
			frame := vm.popFrame()
			vm.sp = frame.basep

			// Unwinds exactly as op.Return does, including the signal a `for <-` body's early return has to be raised as.
			if resumeDepth >= 0 && vm.framesIdx < resumeDepth {
				panic(iterReturnSignal{targetDepth: frame.homeIdx, ret: ret})
			}
			if err := vm.push(ret); err != nil {
				return err
			}

		case op.WrapOption:
			optionId := int(op.ReadUint16(ins[ip:]))
			someId := int(op.ReadUint16(ins[ip+2:]))
			fr.ip += 4
			value := vm.pop()
			if vm.IsType(value, vm.constants[optionId]) {
				// Already Some or None, so wrapping again would only bury the value one level deeper.
				if err := vm.push(value); err != nil {
					return err
				}
				break
			}
			some, ok := vm.constants[someId].(*runtime.DataType)
			if !ok {
				return fmt.Errorf("wrapping a value in Some requires a data type constant (%T)", vm.constants[someId])
			}
			if err := vm.push(runtime.MakeDataValue(some, []runtime.RuntimeValue{value})); err != nil {
				return err
			}

		case op.Invert:
			operand := vm.pop()
			v, ok := operand.(runtime.Bool)
			if !ok {
				return errWrongType("prefix operator !", "Bool", operand)
			}
			if err := vm.push(!v); err != nil {
				return err
			}
		case op.Negate:
			v := vm.pop()
			switch v := v.(type) {
			case runtime.Int:
				if err := vm.push(-v); err != nil {
					return err
				}
			case runtime.Float:
				if err := vm.push(-v); err != nil {
					return err
				}
			default:
				return errWrongType("prefix operator -", "Int or Float", v)
			}
		case op.Add, op.Sub, op.Mul, op.Div,
			op.GreaterThan, op.GreaterThanOrEqual,
			op.LessThan, op.LessThanOrEqual:
			err := vm.numericBinaryOperation(code)
			if err != nil {
				return err
			}
		case op.Mod:
			right := vm.pop()
			rhs, ok := right.(runtime.Int)
			if !ok {
				return errWrongType("operator %", "Int", right)
			}
			left := vm.pop()
			lhs, ok := left.(runtime.Int)
			if !ok {
				return errWrongType("operator %", "Int", left)
			}
			err := vm.push(lhs % rhs)
			if err != nil {
				return err
			}
		case op.Equal:
			equal := vm.isEqual()
			if err := vm.push(equal); err != nil {
				return err
			}
		case op.NotEqual:
			equal := vm.isEqual()
			if err := vm.push(!equal); err != nil {
				return err
			}

		case op.Array:
			given := vm.pop()
			length, ok := given.(runtime.Int)
			if !ok {
				return errWrongType("the length of an array", "Int", given)
			}
			array := make(runtime.Array, length)

			for i := 1; i <= int(length); i++ {
				array[int(length)-i] = vm.pop()
			}

			if err := vm.push(array); err != nil {
				return err
			}
		case op.Dict:
			given := vm.pop()
			length, ok := given.(runtime.Int)
			if !ok {
				return errWrongType("the length of a dict", "Int", given)
			}
			dict := make(runtime.Dict)

			for i := 0; i < int(length); i++ {
				value := vm.pop()
				key := vm.pop()
				dict[key] = value
			}

			if err := vm.push(dict); err != nil {
				return err
			}

		case op.Module:
			infoIdx := op.ReadUint16(ins[ip:])
			fr.ip += 2
			info, ok := vm.constants[infoIdx].(*runtime.ModuleInfo)
			if !ok {
				return errWrongType("a module's declaration", "ModuleInfo", vm.constants[infoIdx])
			}
			given := vm.pop()
			length, ok := given.(runtime.Int)
			if !ok {
				return errWrongType("a module's member count", "Int", given)
			}
			exports := make(map[string]runtime.RuntimeValue, int(length))

			for i := 0; i < int(length); i++ {
				value := vm.pop()
				key := vm.pop()
				keyName, ok := key.(runtime.String)
				if !ok {
					return errWrongType("a module's member name", "String", key)
				}
				exports[string(keyName)] = value
			}

			moduleVal := runtime.MakeModuleValue(info, exports)
			if err := vm.push(moduleVal); err != nil {
				return err
			}

		case op.GetIndex:
			index := vm.pop()
			target := vm.pop()

			switch target := target.(type) {
			case runtime.Array:
				idx, ok := index.(runtime.Int)
				if !ok {
					return errWrongType("an array index", "Int", index)
				}
				pos := int(idx)
				if pos < 0 || pos >= len(target) {
					return errIndexOutOfBounds("array", pos, len(target))
				}
				if err := vm.push(target[pos]); err != nil {
					return err
				}
			case runtime.Binary:
				idx, ok := index.(runtime.Int)
				if !ok {
					return errWrongType("a binary index", "Int", index)
				}
				pos := int(idx)
				if pos < 0 || pos >= len(target) {
					return errIndexOutOfBounds("binary", pos, len(target))
				}
				if err := vm.push(runtime.Byte(target[pos])); err != nil {
					return err
				}
			case runtime.Dict:
				val, ok := target[index]
				if !ok {
					if err := vm.push(runtime.Void{}); err != nil {
						return err
					}
					break
				}
				if err := vm.push(val); err != nil {
					return err
				}
			case runtime.String:
				idx, ok := index.(runtime.Int)
				if !ok {
					return errWrongType("a string index", "Int", index)
				}

				pos := int(idx)
				if pos < 0 || pos >= len(target) {
					return errIndexOutOfBounds("string", pos, len(target))
				}

				byte := runtime.Byte(target[pos])

				if err := vm.push(byte); err != nil {
					return err
				}

			default:
				return errNotIndexable(target)
			}

		case op.Len:
			val := vm.pop()
			switch val := val.(type) {
			case runtime.Array:
				if err := vm.push(runtime.Int(len(val))); err != nil {
					return err
				}
			case runtime.Dict:
				if err := vm.push(runtime.Int(len(val))); err != nil {
					return err
				}
			case runtime.String:
				if err := vm.push(runtime.Int(len(val))); err != nil {
					return err
				}
			default:
				return fmt.Errorf("len is not defined for %s", typeNameOf(val))
			}

		case op.ArrayAppend:
			value := vm.pop()
			arrayVal := vm.pop()
			array, ok := arrayVal.(runtime.Array)
			if !ok {
				return errWrongType("append", "Array", arrayVal)
			}
			array = append(array, value)
			if err := vm.push(array); err != nil {
				return err
			}

		case op.SetLocal:
			idx := op.ReadUint16(ins[ip:])
			fr.ip += 2
			val := vm.pop()
			fr.locals[idx] = val

		case op.GetLocal:
			idx := op.ReadUint16(ins[ip:])
			fr.ip += 2

			if err := vm.push(fr.locals[idx]); err != nil {
				return err
			}

		case op.GetGlobal:
			idx := op.ReadUint16(ins[ip:])
			fr.ip += 2

			val, err := vm.getGlobal(int(idx), taskId)
			if err != nil {
				return err
			}

			if err := vm.push(val); err != nil {
				return err
			}

		case op.SetGlobal:
			idx := op.ReadUint16(ins[ip:])
			fr.ip += 2
			val := vm.pop()

			if err := vm.globals[idx].Set(taskId, val); err != nil {
				return err
			}

		case op.GetField:
			nameIdx := op.ReadUint16(ins[ip:])
			fr.ip += 2
			nameConst, ok := vm.constants[nameIdx].(runtime.String)
			if !ok {
				return fmt.Errorf("name lookup requires a String constant (%T %q)", vm.constants[nameIdx], vm.constants[nameIdx].Inspect())
			}
			name := string(nameConst)
			obj := vm.pop()
			val := obj.Lookup(name)
			if val == nil {
				return fmt.Errorf("%s has no member %q", typeNameOf(obj), name)
			}

			if err := vm.push(val); err != nil {
				return err
			}

		case op.SetField:
			// Stack layout: [..., val, obj] — obj is on top, val beneath it.
			nameIdx := op.ReadUint16(ins[ip:])
			fr.ip += 2
			nameConst, ok := vm.constants[nameIdx].(runtime.String)
			if !ok {
				return fmt.Errorf("setfield requires a String constant (%T %q)", vm.constants[nameIdx], vm.constants[nameIdx].Inspect())
			}
			name := string(nameConst)
			obj := vm.pop()
			val := vm.pop()
			dv, ok := obj.(*runtime.DataValue)
			if !ok {
				return errWrongType("assigning to a field", "a data value", obj)
			}
			idx, ok := dv.Fields[name]
			if !ok {
				return fmt.Errorf("field %q not found in data instance", name)
			}
			dv.Values[idx] = val

		case op.SetIndex:
			// Stack layout: [..., val, target, index] — index on top, target beneath, val at bottom.
			index := vm.pop()
			target := vm.pop()
			val := vm.pop()
			switch target := target.(type) {
			case runtime.Array:
				idx, ok := index.(runtime.Int)
				if !ok {
					return errWrongType("an array index", "Int", index)
				}
				pos := int(idx)
				if pos < 0 || pos >= len(target) {
					return fmt.Errorf("array index %d out of bounds", pos)
				}
				target[pos] = val
			case runtime.Dict:
				target[index] = val
			default:
				return fmt.Errorf("assigning to an index is not defined for %s", typeNameOf(target))
			}

		case op.MakeAttribute:
			argCount := int(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			callee := vm.pop()
			at, ok := callee.(*runtime.AttributeType)
			if !ok {
				return errWrongType("reading an attribute", "an attribute type", callee)
			}
			if argCount != len(at.FieldSymbols) {
				return errWrongArgCount("attribute", len(at.FieldSymbols), argCount)
			}
			vals := make([]runtime.RuntimeValue, argCount)
			for i := 0; i < argCount; i++ {
				vals[argCount-1-i] = vm.pop()
			}
			av := at.MakeValue(vals)
			if err := vm.push(av); err != nil {
				return err
			}

		case op.Call:
			argCount := int(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			callee := vm.pop()

			switch callee := callee.(type) {
			case *runtime.AttributeType:
				if argCount != callee.Arity() {
					return errWrongArgCount("", callee.Arity(), argCount)
				}
				val := vm.pop()

				attrs := vm.AttributesOf(val)

				if attrs == nil {
					if err := vm.push(runtime.Void{}); err != nil {
						return err
					}
				} else if annoId, ok := attrs[callee.TypeConstantId()]; !ok {
					if err := vm.push(runtime.Void{}); err != nil {
						return err
					}
				} else {
					annoVal, err := vm.getGlobal(annoId, taskId)
					if err != nil {
						return err
					}
					if err := vm.push(annoVal); err != nil {
						return err
					}
				}

			case *runtime.ExternFunc:
				if argCount != callee.Arity() {
					return errWrongArgCount("", callee.Arity(), argCount)
				}

				args := make([]runtime.RuntimeValue, argCount)
				for i := argCount - 1; i >= 0; i-- {
					args[i] = vm.pop()
				}

				result, err := vm.callExtern(callee, args)
				if err != nil {
					// Already traced to the line it happened on; repeating the wrapper per level only repeats that line.
					var traced *RuntimeError
					if errors.As(err, &traced) {
						return err
					}
					return fmt.Errorf("error calling extern function: %w", err)
				}

				if err := vm.push(result); err != nil {
					return err
				}

			case *runtime.CompiledFunction:
				if argCount != callee.Arity() {
					return errWrongArgCount("", callee.Arity(), argCount)
				}

				closure := runtime.MakeClosure(callee, nil)
				frame := newClosureFrame(closure, vm.sp-argCount)

				vm.pushFrame(frame)
				vm.sp = frame.basep

				for i := 0; i < argCount; i++ {
					frame.locals[i] = vm.stack[vm.sp+i]
				}

			case *runtime.Closure:
				if argCount != callee.Arity() {
					return errWrongArgCount("", callee.Arity(), argCount)
				}

				frame := newClosureFrame(callee, vm.sp-argCount)

				vm.pushFrame(frame)
				vm.sp = frame.basep

				for i := 0; i < argCount; i++ {
					frame.locals[i] = vm.stack[vm.sp+i]
				}

			case *runtime.DataType:
				if argCount != callee.Arity() {
					return errWrongArgCount("", callee.Arity(), argCount)
				}

				vals := make([]runtime.RuntimeValue, argCount)
				for i := 0; i < argCount; i++ {
					vals[argCount-1-i] = vm.pop()
				}

				dv := runtime.MakeDataValue(callee, vals)
				err := vm.push(dv)
				if err != nil {
					return err
				}
			}

		case op.MakeClosure:
			constIdx := int(op.ReadUint16(ins[fr.ip:]))
			fr.ip += 2
			freeCount := int(op.ReadUint16(ins[fr.ip:]))
			fr.ip += 2

			fn, ok := vm.constants[constIdx].(*runtime.CompiledFunction)
			if !ok {
				return fmt.Errorf("MakeClosure: expected *CompiledFunction at constant %d, got %T", constIdx, vm.constants[constIdx])
			}

			free := make([]runtime.RuntimeValue, freeCount)
			for i := freeCount - 1; i >= 0; i-- {
				free[i] = vm.pop()
			}

			closure := runtime.MakeClosure(fn, free)
			if err := vm.push(closure); err != nil {
				return err
			}

		case op.GetFree:
			idx := int(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			if fr.closure == nil {
				return fmt.Errorf("GetFree: no closure in current frame")
			}
			if idx >= len(fr.closure.Free) {
				return fmt.Errorf("GetFree: index %d out of range (%d free values)", idx, len(fr.closure.Free))
			}
			if err := vm.push(fr.closure.Free[idx]); err != nil {
				return err
			}

		case op.GetFreeCell:
			idx := int(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			if fr.closure == nil {
				return fmt.Errorf("GetFreeCell: no closure in current frame")
			}
			if idx >= len(fr.closure.Free) {
				return fmt.Errorf("GetFreeCell: index %d out of range (%d free values)", idx, len(fr.closure.Free))
			}
			cell, ok := fr.closure.Free[idx].(*runtime.UpvalueCell)
			if !ok {
				return fmt.Errorf("GetFreeCell: expected *UpvalueCell at Free[%d], got %T", idx, fr.closure.Free[idx])
			}
			if err := vm.push(cell.Value); err != nil {
				return err
			}

		case op.SetFreeCell:
			idx := int(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			val := vm.pop()
			if fr.closure == nil {
				return fmt.Errorf("SetFreeCell: no closure in current frame")
			}
			if idx >= len(fr.closure.Free) {
				return fmt.Errorf("SetFreeCell: index %d out of range (%d free values)", idx, len(fr.closure.Free))
			}
			cell, ok := fr.closure.Free[idx].(*runtime.UpvalueCell)
			if !ok {
				return fmt.Errorf("SetFreeCell: expected *UpvalueCell at Free[%d], got %T", idx, fr.closure.Free[idx])
			}
			cell.Value = val

		case op.GetLocalCell:
			idx := int(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			if idx >= len(fr.locals) {
				return fmt.Errorf("GetLocalCell: index %d out of range (%d locals)", idx, len(fr.locals))
			}
			cell, ok := fr.locals[idx].(*runtime.UpvalueCell)
			if !ok {
				return fmt.Errorf("GetLocalCell: expected *UpvalueCell at locals[%d], got %T", idx, fr.locals[idx])
			}
			if err := vm.push(cell.Value); err != nil {
				return err
			}

		case op.SetLocalCell:
			idx := int(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			val := vm.pop()
			if idx >= len(fr.locals) {
				return fmt.Errorf("SetLocalCell: index %d out of range (%d locals)", idx, len(fr.locals))
			}
			cell, ok := fr.locals[idx].(*runtime.UpvalueCell)
			if !ok {
				return fmt.Errorf("SetLocalCell: expected *UpvalueCell at locals[%d], got %T", idx, fr.locals[idx])
			}
			cell.Value = val

		case op.WrapLocal:
			idx := int(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			if idx >= len(fr.locals) {
				return fmt.Errorf("WrapLocal: index %d out of range (%d locals)", idx, len(fr.locals))
			}
			fr.locals[idx] = &runtime.UpvalueCell{Value: fr.locals[idx]}

		case op.MakeIterYield:
			bindingLocal := int(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			bodyStartIp := int(op.ReadUint16(ins[fr.ip:]))
			fr.ip += 2
			bodyEndIp := int(op.ReadUint16(ins[fr.ip:]))
			fr.ip += 2

			yieldVal := vm.makeIterYieldFunc(taskId, fr, bindingLocal, bodyStartIp, bodyEndIp)
			if err := vm.push(yieldVal); err != nil {
				return err
			}

		case op.CallIterate:
			argCount := int(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			callee := vm.pop()

			if argCount != 2 {
				return fmt.Errorf("CallIterate: expected 2 arguments, got %d", argCount)
			}

			yield := vm.pop()
			value := vm.pop()
			if err := vm.callIterate(taskId, callee, value, yield); err != nil {
				return err
			}

		case op.Return:
			ret := vm.pop()
			frame := vm.popFrame()
			vm.sp = frame.basep

			// Popping below resumeDepth means frame itself is genuinely returning, not just this resumed chunk finishing, so raise it instead of taking the normal push-and-continue path.
			if resumeDepth >= 0 && vm.framesIdx < resumeDepth {
				panic(iterReturnSignal{targetDepth: frame.homeIdx, ret: ret})
			}

			if err := vm.push(ret); err != nil {
				return err
			}

		default:
			def, err := op.LookupDefinition(byte(code))
			if err != nil {
				return fmt.Errorf("unhandled opcode: %w", err)
			}
			return fmt.Errorf("unknown opcode %q", def.Name)
		}
	}
}

func (vm *VM) push(val runtime.RuntimeValue) error {
	if vm.sp >= stackSize {
		return fmt.Errorf("stack overflow")
	}

	vm.stack[vm.sp] = val
	vm.sp++
	return nil
}

func (vm *VM) pop() runtime.RuntimeValue {
	v := vm.stack[vm.sp-1]
	vm.sp--
	return v
}

func (vm *VM) numericBinaryOperation(operator op.Opcode) error {
	rhs := vm.pop()
	lhs := vm.pop()

	// String concatenation: one side is String, the other any trivially stringifiable value (Int, Float, Char, Byte, Bool, Void) — e.g. "count: " + 5 renders as "count: 5", not the Int reinterpreted as a Unicode codepoint.
	if lhsStr, ok := lhs.(runtime.String); ok {
		if rhsStr, ok := rhs.(runtime.String); ok {
			if operator != op.Add {
				return errUnsupportedOperands(operator, lhsStr, rhsStr)
			}
			return vm.push(lhsStr + rhsStr)
		}
		return vm.concatString(operator, lhsStr, rhs, true)
	}
	if rhsStr, ok := rhs.(runtime.String); ok {
		return vm.concatString(operator, rhsStr, lhs, false)
	}

	// Durations, instants and timestamps carry a unit, so only the combinations that mean something are accepted and everything else is reported rather than silently coerced.
	if handled, err := vm.timeBinaryOperation(operator, lhs, rhs); handled {
		return err
	}

	switch rhs := rhs.(type) {
	case runtime.Int:
		switch lhs := lhs.(type) {
		case runtime.Int:
			return vm.numericBinaryOperationInt(operator, lhs, rhs)
		case runtime.Float:
			return vm.numericBinaryOperationFloat(operator, lhs, runtime.Float(rhs))
		default:
			return errUnsupportedOperands(operator, lhs, rhs)
		}
	case runtime.Float:
		switch lhs := lhs.(type) {
		case runtime.Int:
			return vm.numericBinaryOperationFloat(operator, runtime.Float(lhs), rhs)
		case runtime.Float:
			return vm.numericBinaryOperationFloat(operator, lhs, rhs)
		default:
			return errUnsupportedOperands(operator, lhs, rhs)
		}
	default:
		return errUnsupportedOperands(operator, lhs, rhs)
	}
}

// concatString handles String + other / other + String for op.Add, where other is any trivially stringifiable value (Int, Float, Char, Byte, Bool, Void).
// strFirst reports whether the String operand came first (left) so the result is concatenated in the right order.
func (vm *VM) concatString(operator op.Opcode, str runtime.String, other runtime.RuntimeValue, strFirst bool) error {
	if operator != op.Add {
		if strFirst {
			return errUnsupportedOperands(operator, str, other)
		}
		return errUnsupportedOperands(operator, other, str)
	}
	s, ok := runtime.TrivialString(other)
	if !ok {
		if strFirst {
			return errUnsupportedOperands(operator, str, other)
		}
		return errUnsupportedOperands(operator, other, str)
	}
	if strFirst {
		return vm.push(str + runtime.String(s))
	}
	return vm.push(runtime.String(s) + str)
}

func (vm *VM) numericBinaryOperationInt(operator op.Opcode, lhs, rhs runtime.Int) error {
	switch operator {
	case op.Add:
		return vm.push(lhs + rhs)
	case op.Sub:
		return vm.push(lhs - rhs)
	case op.Mul:
		return vm.push(lhs * rhs)
	case op.Div:
		if rhs == 0 {
			return errDivisionByZero()
		}
		return vm.push(lhs / rhs)
	case op.Mod:
		return vm.push(lhs % rhs)
	case op.LessThan:
		return vm.push(runtime.Bool(lhs < rhs))
	case op.LessThanOrEqual:
		return vm.push(runtime.Bool(lhs <= rhs))
	case op.GreaterThan:
		return vm.push(runtime.Bool(lhs > rhs))
	case op.GreaterThanOrEqual:
		return vm.push(runtime.Bool(lhs >= rhs))
	default:
		return fmt.Errorf("unknown binary operator %x", operator)
	}
}
func (vm *VM) numericBinaryOperationFloat(operator op.Opcode, lhs, rhs runtime.Float) error {
	switch operator {
	case op.Add:
		return vm.push(lhs + rhs)
	case op.Sub:
		return vm.push(lhs - rhs)
	case op.Mul:
		return vm.push(lhs * rhs)
	case op.Div:
		return vm.push(lhs / rhs)
	case op.LessThan:
		return vm.push(runtime.Bool(lhs < rhs))
	case op.LessThanOrEqual:
		return vm.push(runtime.Bool(lhs <= rhs))
	case op.GreaterThan:
		return vm.push(runtime.Bool(lhs > rhs))
	case op.GreaterThanOrEqual:
		return vm.push(runtime.Bool(lhs >= rhs))
	default:
		return fmt.Errorf("unknown binary operator %x", operator)
	}
}
func (vm *VM) isEqual() runtime.Bool {
	rhs := vm.pop()
	lhs := vm.pop()
	return runtime.Bool(valuesEqual(lhs, rhs))
}

// valuesEqual compares two runtime values for deep equality, recursing into Array/Dict elements.
// Unlike isEqual, this is a pure function with no VM stack access, so it can call itself for nested collections.
func valuesEqual(lhs, rhs runtime.RuntimeValue) bool {
	if lhs.TypeConstantId() != rhs.TypeConstantId() {
		return false
	}
	switch lhs := lhs.(type) {
	case runtime.Int:
		rhs, ok := rhs.(runtime.Int)
		return ok && lhs == rhs
	case runtime.Float:
		rhs, ok := rhs.(runtime.Float)
		return ok && lhs == rhs
	case runtime.Bool:
		rhs, ok := rhs.(runtime.Bool)
		return ok && lhs == rhs
	case runtime.Char:
		rhs, ok := rhs.(runtime.Char)
		return ok && lhs == rhs
	case runtime.String:
		rhs, ok := rhs.(runtime.String)
		return ok && lhs == rhs
	case runtime.Byte:
		rhs, ok := rhs.(runtime.Byte)
		return ok && lhs == rhs
	case runtime.Binary:
		rhs, ok := rhs.(runtime.Binary)
		if !ok || len(lhs) != len(rhs) {
			return false
		}
		for i := range lhs {
			if lhs[i] != rhs[i] {
				return false
			}
		}
		return true
	case runtime.Void:
		_, ok := rhs.(runtime.Void)
		return ok
	case runtime.Array:
		rhs, ok := rhs.(runtime.Array)
		if !ok || len(lhs) != len(rhs) {
			return false
		}
		for i := range lhs {
			if !valuesEqual(lhs[i], rhs[i]) {
				return false
			}
		}
		return true
	case runtime.Dict:
		rhs, ok := rhs.(runtime.Dict)
		if !ok || len(lhs) != len(rhs) {
			return false
		}
		for k, v := range lhs {
			rv, found := rhs[k]
			if !found || !valuesEqual(v, rv) {
				return false
			}
		}
		return true
	case runtime.Duration:
		rhs, ok := rhs.(runtime.Duration)
		return ok && lhs == rhs
	case runtime.Instant:
		rhs, ok := rhs.(runtime.Instant)
		return ok && lhs == rhs
	case runtime.Timestamp:
		rhs, ok := rhs.(runtime.Timestamp)
		return ok && lhs == rhs
	case *runtime.DataValue:
		rhs, ok := rhs.(*runtime.DataValue)
		if !ok || len(lhs.Values) != len(rhs.Values) {
			return false
		}
		// The type ids were already compared above, so two values of different data types never reach here.
		for i := range lhs.Values {
			if !valuesEqual(lhs.Values[i], rhs.Values[i]) {
				return false
			}
		}
		return true
	case *runtime.CompiledFunction:
		return lhs == rhs
	case *runtime.Closure:
		return lhs == rhs
	case *runtime.ExternFunc:
		return lhs == rhs
	// Each is one thing rather than a reproducible value, so two are equal only when they are the same one.
	case *runtime.Channel:
		return lhs == rhs
	case *runtime.Routine:
		return lhs == rhs
	case *runtime.RoutineScope:
		return lhs == rhs
	// A type is a value too, now that reflection hands them out, and two of them are equal when they are the same declaration.
	// Each type lives once in the constants table, so identity is that test; a SimpleType is a struct rather than a pointer, so it compares by the type it stands for.
	case *runtime.DataType:
		rhs, ok := rhs.(*runtime.DataType)
		return ok && lhs == rhs
	case *runtime.UnionType:
		rhs, ok := rhs.(*runtime.UnionType)
		return ok && lhs == rhs
	case *runtime.AttributeType:
		rhs, ok := rhs.(*runtime.AttributeType)
		return ok && lhs == rhs
	case runtime.SimpleType:
		rhs, ok := rhs.(runtime.SimpleType)
		return ok && lhs.TypeConstantId() == rhs.TypeConstantId()
	case *runtime.ModuleValue:
		rhs, ok := rhs.(*runtime.ModuleValue)
		return ok && lhs == rhs
	}
	// Anything else compares unequal rather than stopping the program: `==` is written by users, and a kind this function has not been taught is a gap to fill, not a reason to hand someone a crash.
	return false
}

func (vm *VM) initGlobal(owner TaskId, ins op.Instructions, locals int) (runtime.RuntimeValue, error) {
	frame := newGeneralFrame(ins, vm.sp, locals)
	frame.ip = 0
	vm.pushFrame(frame)
	vm.sp = frame.basep

	err := vm.runTask(owner)
	if err != nil {
		return nil, err
	}

	val := vm.pop()
	vm.popFrame()

	return val, nil
}

// timeBinaryOperation implements arithmetic over Duration, Instant and Timestamp, reporting whether it recognized the operand pair at all.
// The accepted combinations are the ones that carry meaning: spans add to spans, a span shifts a point in time, and two points of the same kind differ by a span.
// Everything else — adding a bare number to a duration, or mixing a monotonic reading with a wall-clock one — is an error rather than a silent reinterpretation, which is the whole reason these are distinct types.
func (vm *VM) timeBinaryOperation(operator op.Opcode, lhs, rhs runtime.RuntimeValue) (bool, error) {
	switch lhs := lhs.(type) {
	case runtime.Duration:
		switch rhs := rhs.(type) {
		case runtime.Duration:
			if compared, ok := compareOrdered(int64(lhs), int64(rhs), operator); ok {
				return true, vm.push(compared)
			}
			switch operator {
			case op.Add:
				return true, vm.push(runtime.Duration(lhs + rhs))
			case op.Sub:
				return true, vm.push(runtime.Duration(lhs - rhs))
			}
		case runtime.Int:
			switch operator {
			case op.Mul:
				return true, vm.push(runtime.Duration(int64(lhs) * int64(rhs)))
			case op.Div:
				if rhs == 0 {
					return true, errDivisionByZero()
				}
				return true, vm.push(runtime.Duration(int64(lhs) / int64(rhs)))
			}
		}
	case runtime.Int:
		// Only Int times Duration belongs here; every other Int pairing is ordinary arithmetic and must be left alone.
		scaled, ok := rhs.(runtime.Duration)
		if !ok {
			return false, nil
		}
		if operator == op.Mul {
			return true, vm.push(runtime.Duration(int64(lhs) * int64(scaled)))
		}
	case runtime.Instant:
		switch rhs := rhs.(type) {
		case runtime.Instant:
			if compared, ok := compareOrdered(int64(lhs), int64(rhs), operator); ok {
				return true, vm.push(compared)
			}
			if operator == op.Sub {
				return true, vm.push(runtime.Duration(int64(lhs) - int64(rhs)))
			}
		case runtime.Duration:
			switch operator {
			case op.Add:
				return true, vm.push(runtime.Instant(int64(lhs) + int64(rhs)))
			case op.Sub:
				return true, vm.push(runtime.Instant(int64(lhs) - int64(rhs)))
			}
		}
	case runtime.Timestamp:
		switch rhs := rhs.(type) {
		case runtime.Timestamp:
			if compared, ok := compareOrdered(int64(lhs), int64(rhs), operator); ok {
				return true, vm.push(compared)
			}
			if operator == op.Sub {
				return true, vm.push(runtime.Duration(int64(lhs) - int64(rhs)))
			}
		case runtime.Duration:
			switch operator {
			case op.Add:
				return true, vm.push(runtime.Timestamp(int64(lhs) + int64(rhs)))
			case op.Sub:
				return true, vm.push(runtime.Timestamp(int64(lhs) - int64(rhs)))
			}
		}
	default:
		return false, nil
	}

	return true, errUnsupportedOperands(operator, lhs, rhs)
}

func compareOrdered(lhs, rhs int64, operator op.Opcode) (runtime.RuntimeValue, bool) {
	switch operator {
	case op.LessThan:
		return runtime.Bool(lhs < rhs), true
	case op.LessThanOrEqual:
		return runtime.Bool(lhs <= rhs), true
	case op.GreaterThan:
		return runtime.Bool(lhs > rhs), true
	case op.GreaterThanOrEqual:
		return runtime.Bool(lhs >= rhs), true
	}
	return nil, false
}

// asWrapper reads a value as the prelude union the guarded operators are defined against.
//
// A value that is already a member of that union is itself. Anything else has to say how to become one, through the attribute's conversion — which is a Zirric closure, called reentrantly the way an extern function's own callbacks are. A value that says nothing is reported here rather than left to fail further along, where the message would name a field instead of the thing that is actually missing.
func (vm *VM) asWrapper(taskId TaskId, value runtime.RuntimeValue, unionId int, attrId int, field string, operators string, attribute string) (runtime.RuntimeValue, error) {
	if vm.IsType(value, vm.constants[unionId]) {
		return value, nil
	}

	// Keyed by the attribute type's own id, which is not always where the constant sits — the same lookup op.Call makes to read an attribute off a value.
	attrType, ok := vm.constants[attrId].(*runtime.AttributeType)
	if !ok {
		return nil, fmt.Errorf("reading a value as %s requires an attribute type constant (%T)", attribute, vm.constants[attrId])
	}
	attributeId, carries := vm.AttributesOf(value)[attrType.TypeConstantId()]
	if !carries {
		return nil, fmt.Errorf("%s require a value with %s, got %s %s", operators, attribute, typeNameOf(value), value.Inspect())
	}
	instance, err := vm.getGlobal(attributeId, taskId)
	if err != nil {
		return nil, err
	}
	convert := instance.Lookup(field)
	if convert == nil {
		return nil, fmt.Errorf("the %s of %s declares no %s", attribute, typeNameOf(value), field)
	}
	wrapped, err := vm.CallFunction(convert, value)
	if err != nil {
		return nil, err
	}
	// Checked here so that a conversion answering with something else is named where it went wrong, rather than further along where the message would be about a missing field.
	if !vm.IsType(wrapped, vm.constants[unionId]) {
		return nil, fmt.Errorf("the %s of %s answered with %s %s, which is no %s", attribute, typeNameOf(value), typeNameOf(wrapped), wrapped.Inspect(), unionNameOf(vm.constants[unionId]))
	}
	return wrapped, nil
}

// unionNameOf names the union a constant holds, which TypeName cannot: it answers for the value's own type, and the value here is the type.
func unionNameOf(typeValue runtime.RuntimeValue) string {
	if union, ok := typeValue.(*runtime.UnionType); ok && union.Symbol != nil {
		return union.Symbol.Name
	}
	return typeNameOf(typeValue)
}

// callExtern runs an extern function, giving the other routines a turn first when it waits on the outside world (ZE-025).
// A program that never starts a routine pays one nil comparison.
func (vm *VM) callExtern(callee *runtime.ExternFunc, args []runtime.RuntimeValue) (runtime.RuntimeValue, error) {
	if !callee.Switches {
		return callee.Impl(vm, args)
	}
	sched := vm.scheduler()
	if sched == nil {
		return callee.Impl(vm, args)
	}
	current := sched.Current()
	if current == nil {
		return callee.Impl(vm, args)
	}
	if err := sched.Yield(current); err != nil {
		return nil, err
	}
	if !callee.Blocks {
		return callee.Impl(vm, args)
	}
	// Handed on for the length of the call, so waiting on standard input does not stop the rest.
	sched.BeginHostCall(current)
	result, err := callee.Impl(vm, args)
	if resumeErr := sched.EndHostCall(current); resumeErr != nil {
		return nil, resumeErr
	}
	return result, err
}
