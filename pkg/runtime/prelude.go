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

var _ ExternPlugin = &Prelude{}

type Prelude struct{}

// Bind implements runtime.ExternPlugin.
func (*Prelude) Bind(module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "Array",
		"Bool",
		"Char",
		"Dict",
		"Float",
		"Func",
		"Int",
		"Module",
		"String",
		"Void":
		return SimpleType{Decl: decl}
	case "Any":
		return MakeAnyType(decl)
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
