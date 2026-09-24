package runtime

import "fmt"

// TypeName names a value's type the way the language spells it.
//
// Messages are read by people writing Zirric, and runtime.String is an implementation detail they never wrote. Every message that names what it was given goes through here, so they all say the same thing.
func TypeName(value RuntimeValue) string {
	switch value := value.(type) {
	case nil:
		return "nothing"
	case Int:
		return "Int"
	case Float:
		return "Float"
	case String:
		return "String"
	case Bool:
		return "Bool"
	case Char:
		return "Char"
	case Byte:
		return "Byte"
	case Binary:
		return "Binary"
	case Void:
		return "Void"
	case Array:
		return "Array"
	case Dict:
		return "Dict"
	case Duration:
		return "Duration"
	case Instant:
		return "Instant"
	case Timestamp:
		return "Timestamp"
	case *Channel:
		return "Channel"
	case *RoutineScope:
		return "Scope"
	case *Routine:
		return "Routine"
	case *DataValue:
		return value.TypeName
	case *ModuleValue:
		return "AnyModule"
	case *CompiledFunction, *Closure, *ExternFunc:
		return "Func"
	case *DataType, *UnionType, *AttributeType, SimpleType:
		return "a type"
	}
	return fmt.Sprintf("%T", value)
}
