package vm

import (
	"fmt"
	"math/rand"

	"code.knabel.dev/zirric-lang/zirric/op"
	"code.knabel.dev/zirric-lang/zirric/runtime"
)

func (vm *VM) Run() error {
	var taskId = TaskId(rand.Uint64())
	return vm.runTask(taskId)
}

func (vm *VM) runTask(taskId TaskId) error {
	for vm.currentFrame().ip < len(vm.currentFrame().Instructions()) {
		vm.currentFrame().ip++

		var (
			fr   = vm.currentFrame()
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
				return fmt.Errorf("unexpected type (%T %q)", v, v.Inspect())
			}

		case op.Invert:
			v, ok := vm.pop().(runtime.Bool)
			if !ok {
				return fmt.Errorf("prefix operator ! is only defined on Bool (%T %q)", v, v.Inspect())
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
				return fmt.Errorf("prefix operator - is only defined on Int or Float (%T %q)", v, v.Inspect())
			}
		case op.Add, op.Sub, op.Mul, op.Div,
			op.GreaterThan, op.GreaterThanOrEqual,
			op.LessThan, op.LessThanOrEqual:
			err := vm.numericBinaryOperation(code)
			if err != nil {
				return err
			}
		case op.Mod:
			rhs, ok := vm.pop().(runtime.Int)
			if !ok {
				return fmt.Errorf("operator %% is only defined on Int (%T %q)", rhs, rhs.Inspect())
			}
			lhs, ok := vm.pop().(runtime.Int)
			if !ok {
				return fmt.Errorf("operator %% is only defined on Int (%T %q)", lhs, lhs.Inspect())
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
			length, ok := vm.pop().(runtime.Int)
			if !ok {
				return fmt.Errorf("lenght of an array must be an Int (%T %q)", length, length.Inspect())
			}
			array := make(runtime.Array, length)

			for i := 1; i <= int(length); i++ {
				array[int(length)-i] = vm.pop()
			}

			if err := vm.push(array); err != nil {
				return err
			}
		case op.Dict:
			length, ok := vm.pop().(runtime.Int)
			if !ok {
				return fmt.Errorf("lenght of an array must be an Int (%T %q)", length, length.Inspect())
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

		case op.GetIndex:
			index := vm.pop()
			target := vm.pop()

			switch target := target.(type) {
			case runtime.Array:
				idx, ok := index.(runtime.Int)
				if !ok {
					return fmt.Errorf("array index must be Int (%T %q)", index, index.Inspect())
				}
				pos := int(idx)
				if pos < 0 || pos >= len(target) {
					return fmt.Errorf("array index %d out of bounds", pos)
				}
				if err := vm.push(target[pos]); err != nil {
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
			default:
				return fmt.Errorf("index operator not supported on %T", target)
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
				return fmt.Errorf("len not supported on %T", val)
			}

		case op.ArrayAppend:
			value := vm.pop()
			arrayVal := vm.pop()
			array, ok := arrayVal.(runtime.Array)
			if !ok {
				return fmt.Errorf("array append requires Array (%T)", arrayVal)
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

			global := vm.globals[idx]

			val, err := global.Get(taskId)
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
				return fmt.Errorf("name %q not found in %T %q", name, obj, obj.Inspect())
			}

			if err := vm.push(val); err != nil {
				return err
			}

		case op.MakeAnnotation:
			argCount := int(op.ReadUint16(ins[ip:]))
			fr.ip += 2
			callee := vm.pop()
			at, ok := callee.(*runtime.AnnotationType)
			if !ok {
				return fmt.Errorf("annotation value expects AnnotationType (%T %q)", callee, callee.Inspect())
			}
			if argCount != len(at.FieldSymbols) {
				return fmt.Errorf("wrong number of annotation arguments: want=%d, got=%d", len(at.FieldSymbols), argCount)
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
			case *runtime.AnnotationType:
				if argCount != callee.Arity() {
					return fmt.Errorf("wrong number of arguments: want=%d, got=%d", callee.Arity(), argCount)
				}
				val := vm.pop()
				switch val := val.(type) {
				case *runtime.CompiledFunction:
					if val.Annotations == nil {
						if err := vm.push(runtime.Void{}); err != nil {
							return err
						}
						break
					}
					annoId, ok := val.Annotations[callee.TypeConstantId()]
					if !ok {
						if err := vm.push(runtime.Void{}); err != nil {
							return err
						}
						break
					}
					annoVal, err := vm.globals[annoId].Get(taskId)
					if err != nil {
						return err
					}
					if err := vm.push(annoVal); err != nil {
						return err
					}
				case runtime.ExternFunc:
					if val.Annotations == nil {
						if err := vm.push(runtime.Void{}); err != nil {
							return err
						}
						break
					}
					annoId, ok := val.Annotations[callee.TypeConstantId()]
					if !ok {
						if err := vm.push(runtime.Void{}); err != nil {
							return err
						}
						break
					}
					annoVal, err := vm.globals[annoId].Get(taskId)
					if err != nil {
						return err
					}
					if err := vm.push(annoVal); err != nil {
						return err
					}
				case runtime.SimpleType:
					if val.Annotations == nil {
						if err := vm.push(runtime.Void{}); err != nil {
							return err
						}
						break
					}
					annoId, ok := val.Annotations[callee.TypeConstantId()]
					if !ok {
						if err := vm.push(runtime.Void{}); err != nil {
							return err
						}
						break
					}
					annoVal, err := vm.globals[annoId].Get(taskId)
					if err != nil {
						return err
					}
					if err := vm.push(annoVal); err != nil {
						return err
					}
				case *runtime.AnnotationType:
					if val.Annotations == nil {
						if err := vm.push(runtime.Void{}); err != nil {
							return err
						}
						break
					}
					annoId, ok := val.Annotations[callee.TypeConstantId()]
					if !ok {
						if err := vm.push(runtime.Void{}); err != nil {
							return err
						}
						break
					}
					annoVal, err := vm.globals[annoId].Get(taskId)
					if err != nil {
						return err
					}
					if err := vm.push(annoVal); err != nil {
						return err
					}
				case *runtime.AnnotationValue:
					typeId := val.TypeConstantId()
					typeIdx := int(typeId)
					if typeIdx < 0 || typeIdx >= len(vm.constants) {
						return fmt.Errorf("annotation lookup failed for type id %d", typeId)
					}
					at, ok := vm.constants[typeIdx].(*runtime.AnnotationType)
					if !ok {
						return fmt.Errorf("annotation lookup requires annotation type, got=%T", vm.constants[typeIdx])
					}
					if at.Annotations == nil {
						if err := vm.push(runtime.Void{}); err != nil {
							return err
						}
						break
					}
					annoId, ok := at.Annotations[callee.TypeConstantId()]
					if !ok {
						if err := vm.push(runtime.Void{}); err != nil {
							return err
						}
						break
					}
					annoVal, err := vm.globals[annoId].Get(taskId)
					if err != nil {
						return err
					}
					if err := vm.push(annoVal); err != nil {
						return err
					}
				default:
					typeId := val.TypeConstantId()
					typeIdx := int(typeId)
					if typeIdx < 0 || typeIdx >= len(vm.constants) {
						return fmt.Errorf("annotation lookup failed for type id %d", typeId)
					}
					dt, ok := vm.constants[typeIdx].(*runtime.DataType)
					if !ok {
						return fmt.Errorf("annotation lookup requires data type, got=%T", vm.constants[typeIdx])
					}
					if dt.Annotations == nil {
						if err := vm.push(runtime.Void{}); err != nil {
							return err
						}
						break
					}
					annoId, ok := dt.Annotations[callee.TypeConstantId()]
					if !ok {
						if err := vm.push(runtime.Void{}); err != nil {
							return err
						}
						break
					}
					annoVal, err := vm.globals[annoId].Get(taskId)
					if err != nil {
						return err
					}
					if err := vm.push(annoVal); err != nil {
						return err
					}
				}

			case *runtime.CompiledFunction:
				if argCount != callee.Arity() {
					return fmt.Errorf("wrong number of arguments: want=%d, got=%d", callee.Arity(), argCount)
				}

				closure := runtime.MakeClosure(callee, nil)
				frame := newClosureFrame(closure, vm.sp-argCount)

				vm.pushFrame(frame)
				vm.sp = frame.basep

				for i := 0; i < argCount; i++ {
					frame.locals[argCount-1-i] = vm.stack[vm.sp+i]
				}

			case *runtime.DataType:
				if argCount != callee.Arity() {
					return fmt.Errorf("wrong number of arguments: want=%d, got=%d", callee.Arity(), argCount)
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

		case op.Return:
			ret := vm.pop()
			frame := vm.popFrame()
			vm.sp = frame.basep

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

	return nil
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
	switch rhs := vm.pop().(type) {
	case runtime.Int:
		switch lhs := vm.pop().(type) {
		case runtime.Int:
			return vm.numericBinaryOperationInt(operator, lhs, rhs)
		case runtime.Float:
			return vm.numericBinaryOperationFloat(operator, lhs, runtime.Float(rhs))
		default:
			return fmt.Errorf("unsupported %T", lhs)
		}
	case runtime.Float:
		switch lhs := vm.pop().(type) {
		case runtime.Int:
			return vm.numericBinaryOperationFloat(operator, runtime.Float(lhs), rhs)
		case runtime.Float:
			return vm.numericBinaryOperationFloat(operator, lhs, rhs)
		default:
			return fmt.Errorf("unsupported %T", lhs)
		}
	default:
		return fmt.Errorf("unsupported %T", rhs)
	}
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

	if lhs.TypeConstantId() != rhs.TypeConstantId() {
		return false
	}
	switch lhs := lhs.(type) {
	case runtime.Int:
		rhs, ok := rhs.(runtime.Int)
		if !ok {
			return false
		}
		return lhs == rhs
	case runtime.Float:
		rhs, ok := rhs.(runtime.Float)
		if !ok {
			return false
		}
		return lhs == rhs
	case runtime.Bool:
		rhs, ok := rhs.(runtime.Bool)
		if !ok {
			return false
		}
		return lhs == rhs
	case runtime.Char:
		rhs, ok := rhs.(runtime.Char)
		if !ok {
			return false
		}
		return lhs == rhs
	case runtime.String:
		rhs, ok := rhs.(runtime.String)
		if !ok {
			return false
		}
		return lhs == rhs
	case runtime.Void:
		_, ok := rhs.(runtime.Void)
		return runtime.Bool(ok)
	}
	panic(fmt.Sprintf("unknown type for equality check %T of %q", lhs, lhs.Inspect()))
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

	val := vm.stack[vm.sp-1]
	vm.popFrame()

	return val, nil
}
