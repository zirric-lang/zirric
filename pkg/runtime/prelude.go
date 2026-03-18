package runtime

import (
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
	typeIdVoid
)

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
	"Void":   typeIdVoid,
}

var _ ExternPlugin = &Prelude{}

type Prelude struct{}

// Bind implements runtime.ExternPlugin.
func (*Prelude) Bind(module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
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
	case "Void":
		return MakeBuiltinSimpleType(decl, typeIdVoid)
	case "Any":
		return MakeAnyType(decl)
	case "void":
		return Void{}
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
func (p *Prelude) Void() Void                                  { return Void{} }
