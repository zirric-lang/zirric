package runtime

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

// Declares some TypeId constants for the prelude data types.
// These are not guaranteed to be constant over versions and are not safe to serialize.
// They only offer fast creation for literals without excessive lookups.
// May change in the future.
// TODO: Either use an alternative or find out how to prefill these.
const (
	typeIdArray TypeId = iota
	typeIdBool
	typeIdChar
	typeIdDict
	typeIdFloat
	typeIdFunc
	typeIdInt
	typeIdModule
	typeIdString
	typeIdBytes
	typeIdVoid
)

// allBuiltinTypeIds lists every hardcoded builtin TypeId in declaration order.
// Its length is the number of reserved builtin TypeIds; the analyzer uses
// len(allBuiltinTypeIds) to ensure user-defined ConstantIds never collide with them.
var allBuiltinTypeIds = []TypeId{
	typeIdArray,
	typeIdBool,
	typeIdChar,
	typeIdDict,
	typeIdFloat,
	typeIdFunc,
	typeIdInt,
	typeIdModule,
	typeIdString,
	typeIdBytes,
	typeIdVoid,
}

// NumBuiltinTypeIds is the number of reserved builtin TypeIds.
// The analyzer reserves this many ConstantId slots so user-defined types
// never receive a ConstantId that collides with a hardcoded builtin TypeId.
var NumBuiltinTypeIds = len(allBuiltinTypeIds)

// BuiltinTypeIds maps prelude type names to their hardcoded TypeIds.
// Used by the compiler to create SimpleType values that match the runtime
// TypeConstantId of builtin literal values (Array, Dict, Int, etc.).
var BuiltinTypeIds = map[string]TypeId{
	"Array":  typeIdArray,
	"Bool":   typeIdBool,
	"Char":   typeIdChar,
	"Dict":   typeIdDict,
	"Float":  typeIdFloat,
	"Func":   typeIdFunc,
	"Int":    typeIdInt,
	"Module": typeIdModule,
	"String": typeIdString,
	"Bytes":  typeIdBytes,
	"Void":   typeIdVoid,
}

var _ ExternPlugin = &Prelude{}

type Prelude struct{}

func (*Prelude) Module() string { return "prelude" }

// Bind implements runtime.ExternPlugin.
func (*Prelude) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "Array":
		return MakeBuiltinSimpleType(decl, typeIdArray)
	case "Bool":
		return MakeBuiltinSimpleType(decl, typeIdBool)
	case "Char":
		return MakeBuiltinSimpleType(decl, typeIdChar)
	case "Dict":
		return MakeBuiltinSimpleType(decl, typeIdDict)
	case "Float":
		return MakeBuiltinSimpleType(decl, typeIdFloat)
	case "Func":
		return MakeBuiltinSimpleType(decl, typeIdFunc)
	case "Int":
		return MakeBuiltinSimpleType(decl, typeIdInt)
	case "Module":
		return MakeBuiltinSimpleType(decl, typeIdModule)
	case "String":
		return MakeBuiltinSimpleType(decl, typeIdString)
	case "Bytes":
		return MakeBuiltinSimpleType(decl, typeIdBytes)
	case "Void":
		return MakeBuiltinSimpleType(decl, typeIdVoid)
	case "Any":
		return MakeAnyType(decl)
	case "void":
		return Void{}

	case "bytesFromString": // TODO: remove this once there are extern constructors
		return MakeExternFunc(decl, func(args []RuntimeValue) (RuntimeValue, error) {
			if len(args) != 1 {
				panic("expected exactly 1 argument")
			}

			switch v := args[0].(type) {
			case String:
				return Bytes(v), nil
			case Bytes:
				return v, nil
			case Char:
				return Bytes([]byte(string(v))), nil
			default:
				return nil, fmt.Errorf("expected argument of type String, got %T", args[0])
			}
		})
	case "panic":
		return MakeExternFunc(decl, func(args []RuntimeValue) (RuntimeValue, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("panic expects exactly 1 argument, got %d", len(args))
			}
			msg, ok := args[0].(String)
			if !ok {
				return nil, fmt.Errorf("panic expects a String argument, got %T", args[0])
			}
			return nil, fmt.Errorf("panic: %s", string(msg))
		})
	}
	return nil
}

func (p *Prelude) Bool(val bool) Bool                          { return Bool(val) }
func (p *Prelude) Array(val []RuntimeValue) Array              { return Array(val) }
func (p *Prelude) Char(val rune) Char                          { return Char(val) }
func (p *Prelude) Dict(val map[RuntimeValue]RuntimeValue) Dict { return Dict(val) }
func (p *Prelude) Float(val float64) Float                     { return Float(val) }
func (p *Prelude) Int(val int64) Int                           { return Int(val) }
func (p *Prelude) String(val string) String                    { return String(val) }
func (p *Prelude) Bytes(val []byte) Bytes                      { return Bytes(val) }
func (p *Prelude) Void() Void                                  { return Void{} }
