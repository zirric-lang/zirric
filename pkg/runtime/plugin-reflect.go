package runtime

import (
	"fmt"
	"sort"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ ExternPlugin = &ReflectPlugin{}

// ReflectPlugin provides runtime bindings for the reflect module's extern declarations.
// Only enumeration needs Go: a module's exports are otherwise reachable solely through ModuleValue.Lookup, which requires a name the caller already knows.
type ReflectPlugin struct{}

func (*ReflectPlugin) Module() string { return "reflect" }

// Bind implements ExternPlugin.
func (*ReflectPlugin) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "moduleName":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			mod, err := asModule("moduleName", args[0])
			if err != nil {
				return nil, err
			}
			return String(mod.Name()), nil
		})
	case "members":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			mod, err := asModule("members", args[0])
			if err != nil {
				return nil, err
			}
			names := mod.MemberNames()
			result := make(Array, 0, len(names))
			for _, name := range names {
				result = append(result, mod.Lookup(name))
			}
			return result, nil
		})
	case "memberNames":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			mod, err := asModule("memberNames", args[0])
			if err != nil {
				return nil, err
			}
			names := mod.MemberNames()
			result := make(Array, 0, len(names))
			for _, name := range names {
				result = append(result, String(name))
			}
			return result, nil
		})
	case "moduleOf":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			mod, err := asModule("moduleOf", args[0])
			if err != nil {
				return nil, err
			}
			return moduleMetadata(caller, mod)
		})
	case "_docs":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return String(declaredDocs(args[0])), nil
		})
	case "typeName":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			return String(declaredTypeName(args[0])), nil
		})
	case "isDataType":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			_, ok := args[0].(*DataType)
			return Bool(ok), nil
		})
	case "isUnionType":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			_, ok := args[0].(*UnionType)
			return Bool(ok), nil
		})
	case "fieldsOf":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			fielded, ok := args[0].(FieldedType)
			if !ok {
				return make(Array, 0), nil
			}
			return fieldValues(caller, fielded)
		})
	case "unionMembers":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			unionType, ok := args[0].(*UnionType)
			if !ok || caller == nil {
				return make(Array, 0), nil
			}
			result := make(Array, 0, len(unionType.MemberTypeIds))
			for _, memberId := range unionType.MemberTypeIds {
				if member := caller.ResolveType(memberId); member != nil {
					result = append(result, member)
				}
			}
			return result, nil
		})
	case "fieldValues":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			dataValue, ok := args[0].(*DataValue)
			if !ok {
				return make(Array, 0), nil
			}
			values := make(Array, len(dataValue.Values))
			copy(values, dataValue.Values)
			return values, nil
		})
	case "isAttributeType":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			_, ok := args[0].(*AttributeType)
			return Bool(ok), nil
		})
	case "isInstance":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			if caller == nil {
				return Bool(false), nil
			}
			return Bool(caller.IsType(args[0], args[1])), nil
		})
	case "construct":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			dataType, ok := args[0].(*DataType)
			if !ok {
				return ResultErr(caller, String(fmt.Sprintf("construct expects a data type, got %s", args[0].Inspect())))
			}
			values, ok := args[1].(Array)
			if !ok {
				return ResultErr(caller, String(fmt.Sprintf("construct expects an Array of field values, got %T", args[1])))
			}
			if len(values) != len(dataType.FieldSymbols) {
				return ResultErr(caller, String(fmt.Sprintf("%s takes %d fields, got %d", declaredTypeName(dataType), len(dataType.FieldSymbols), len(values))))
			}
			// The array is copied because a DataValue keeps the slice it is given, and the caller's array must not become its storage.
			fields := make([]RuntimeValue, len(values))
			copy(fields, values)
			return ResultOk(caller, MakeDataValue(dataType, fields))
		})
	case "_typeOf":
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			if caller == nil {
				return Void{}, nil
			}
			if resolved := caller.ResolveType(args[0].TypeConstantId()); resolved != nil {
				return resolved, nil
			}
			return Void{}, nil
		})
	case "_member":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			mod, err := asModule("_member", args[0])
			if err != nil {
				return nil, err
			}
			name, ok := args[1].(String)
			if !ok {
				return nil, fmt.Errorf("_member expects a String name, got %s", TypeName(args[1]))
			}
			if value := mod.Lookup(string(name)); value != nil {
				return value, nil
			}
			return Void{}, nil
		})
	}
	return nil
}

var _ ExternPlugin = &ReflectPackagesPlugin{}

// ReflectPackagesPlugin provides runtime bindings for the reflect.packages module.
// It is separate from ReflectPlugin because binding these forces every module of the project package to compile, and that cost should fall only on programs that import reflect.packages, not on every user of reflect.
type ReflectPackagesPlugin struct{}

func (*ReflectPackagesPlugin) Module() string { return "packages" }

// Bind implements ExternPlugin.
// The project package is enumerated here, while binding, rather than when a program calls one of these functions: every module it finds is given a global slot, and the VM is built from the slots that exist once compilation is over. Asking for it later reserves slots the running VM has no room for, and reaching one fails with an index past the end of the globals.
func (*ReflectPackagesPlugin) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "_mainPackageName":
		name, _ := ctx.MainPackageModules()
		return MakeExternFunc(decl, func(_ VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			return String(name), nil
		})
	case "_mainPackageModuleNames":
		_, globals := ctx.MainPackageModules()
		names := sortedKeys(globals)
		return MakeExternFunc(decl, func(_ VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			result := make(Array, 0, len(names))
			for _, name := range names {
				result = append(result, String(name))
			}
			return result, nil
		})
	case "_cavefile":
		cave, ok := ctx.MainPackageCavefile()
		return MakeExternFunc(decl, func(caller VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			if !ok || caller == nil {
				return Void{}, nil
			}
			return cavefileValue(caller, cave)
		})
	case "_moduleNamed":
		_, globals := ctx.MainPackageModules()
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			name, ok := args[0].(String)
			if !ok {
				return nil, fmt.Errorf("_moduleNamed expects a String name, got %s", TypeName(args[0]))
			}
			globalId, found := globals[string(name)]
			if !found || caller == nil {
				return Void{}, nil
			}
			// Resolving the global is what actually initializes the module, so a package's modules stay uncompiled-but-listed until something asks for one by name.
			value, err := caller.ResolveGlobal(globalId)
			if err != nil {
				return nil, err
			}
			return value, nil
		})
	}
	return nil
}

// declaredDocs returns the comment written above the declaration a value came from, or the empty string when there is none to read.
// A const is not here: what a module exports for one is the value itself, which carries nothing of the declaration, so its documentation is only reachable through the module's declarations.
func declaredDocs(value RuntimeValue) string {
	var symbol *ast.Symbol
	switch value := value.(type) {
	case *ModuleValue:
		if value.Info() == nil {
			return ""
		}
		return value.Info().Docs
	case *DataType:
		symbol = value.Symbol
	case *UnionType:
		symbol = value.Symbol
	case *AttributeType:
		symbol = value.Symbol
	case SimpleType:
		symbol = value.Decl
	case *CompiledFunction:
		symbol = value.Symbol
	case CompiledFunction:
		symbol = value.Symbol
	case *Closure:
		symbol = value.Fn.Symbol
	case *ExternFunc:
		symbol = value.symbol
	default:
		return ""
	}
	if symbol == nil || symbol.Decl == nil {
		return ""
	}
	return ast.DocsOf(symbol.Decl)
}

// declaredTypeName returns the name a type was declared under, or the empty string for anything that is not a type.
func declaredTypeName(value RuntimeValue) string {
	var symbol *ast.Symbol
	switch value := value.(type) {
	case *DataType:
		symbol = value.Symbol
	case *UnionType:
		symbol = value.Symbol
	case *AttributeType:
		symbol = value.Symbol
	case SimpleType:
		symbol = value.Decl
	default:
		return ""
	}
	if symbol == nil || symbol.Decl == nil {
		return ""
	}
	return symbol.Decl.DeclName().Value
}

// moduleMetadata builds the reflect.Module describing what a module declared, which is everything about it that does not require reaching into the values it exports.
func moduleMetadata(caller VMCaller, mod *ModuleValue) (RuntimeValue, error) {
	declarations, err := declarationValues(caller, mod)
	if err != nil {
		return nil, err
	}
	info := mod.Info()
	if info == nil {
		info = &ModuleInfo{}
	}
	return MakeDataValueNamed(caller, "reflect", "Module", map[string]RuntimeValue{
		"name":         String(info.Name),
		"docs":         String(info.Docs),
		"imports":      stringArray(info.Imports),
		"sources":      stringArray(info.Sources),
		"declarations": declarations,
	})
}

// declarationValues builds one reflect.Declaration per public declaration, each carrying the attributes written on that declaration rather than on the value it produced.
func declarationValues(caller VMCaller, mod *ModuleValue) (RuntimeValue, error) {
	info := mod.Info()
	if info == nil || caller == nil {
		return make(Array, 0), nil
	}
	result := make(Array, 0, len(info.Declarations))
	for _, declared := range info.Declarations {
		kind, err := MakeDataValueNamed(caller, "reflect", declared.KindTypeName(), nil)
		if err != nil {
			return nil, err
		}
		path, line := declared.Source()
		source, err := MakeDataValueNamed(caller, "reflect", "Source", map[string]RuntimeValue{
			"path": String(path),
			"line": Int(line),
		})
		if err != nil {
			return nil, err
		}
		value, err := MakeDataValueNamed(caller, "reflect", "Declaration", map[string]RuntimeValue{
			"name":      String(declared.Name),
			"docs":      String(declared.Docs()),
			"kind":      kind,
			"signature": String(declared.Signature()),
			"source":    source,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, WithAttributes(value, declared.Attributes))
	}
	return result, nil
}

// fieldValues builds one reflect.Field per declared field, each carrying that field's own attributes rather than the type's.
func fieldValues(caller VMCaller, fielded FieldedType) (RuntimeValue, error) {
	fields := fielded.Fields()
	result := make(Array, 0, len(fields))
	for i, fieldSymbol := range fields {
		declaredType, err := makeTypeRefValue(caller, fielded.FieldTypeAt(i))
		if err != nil {
			return nil, err
		}
		field, err := MakeDataValueNamed(caller, "reflect", "Field", map[string]RuntimeValue{
			"name":         String(fieldSymbol.Name),
			"docs":         String(fielded.FieldDocsAt(i)),
			"declaredType": declaredType,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, WithAttributes(field, fielded.FieldAttributesAt(i)))
	}
	return result, nil
}

// makeTypeRefValue builds the reflect.TypeRef describing a written type hint.
func makeTypeRefValue(caller VMCaller, ref TypeRef) (RuntimeValue, error) {
	switch ref.Kind {
	case TypeRefNamed:
		resolved, err := resolvedTypeOption(caller, ref.Type)
		if err != nil {
			return nil, err
		}
		return MakeDataValueNamed(caller, "reflect", "NamedType", map[string]RuntimeValue{
			"name":     String(ref.Name),
			"resolved": resolved,
		})
	case TypeRefArray:
		element, err := makeTypeRefValue(caller, derefTypeRef(ref.Element))
		if err != nil {
			return nil, err
		}
		return MakeDataValueNamed(caller, "reflect", "ArrayType", map[string]RuntimeValue{"element": element})
	case TypeRefDict:
		key, err := makeTypeRefValue(caller, derefTypeRef(ref.Key))
		if err != nil {
			return nil, err
		}
		value, err := makeTypeRefValue(caller, derefTypeRef(ref.Value))
		if err != nil {
			return nil, err
		}
		return MakeDataValueNamed(caller, "reflect", "DictType", map[string]RuntimeValue{"key": key, "value": value})
	case TypeRefFunc:
		parameters, err := makeTypeRefValues(caller, ref.Parameters)
		if err != nil {
			return nil, err
		}
		returns, err := makeTypeRefValue(caller, derefTypeRef(ref.Returns))
		if err != nil {
			return nil, err
		}
		return MakeDataValueNamed(caller, "reflect", "FuncType", map[string]RuntimeValue{
			"parameters": parameters,
			"returns":    returns,
		})
	case TypeRefAttrs:
		attributes, err := makeTypeRefValues(caller, ref.Attributes)
		if err != nil {
			return nil, err
		}
		return MakeDataValueNamed(caller, "reflect", "AttrsType", map[string]RuntimeValue{"attributes": attributes})
	}
	return MakeDataValueNamed(caller, "reflect", "UnknownType", nil)
}

func makeTypeRefValues(caller VMCaller, refs []TypeRef) (Array, error) {
	result := make(Array, 0, len(refs))
	for _, ref := range refs {
		value, err := makeTypeRefValue(caller, ref)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

// resolvedTypeOption wraps the type a name resolved to, or None when the name resolved to nothing that exists at runtime.
func resolvedTypeOption(caller VMCaller, typeId *TypeId) (RuntimeValue, error) {
	if typeId != nil && caller != nil {
		if resolved := caller.ResolveType(*typeId); resolved != nil {
			return OptionSome(caller, resolved)
		}
	}
	return OptionNone(caller)
}

func derefTypeRef(ref *TypeRef) TypeRef {
	if ref == nil {
		return TypeRef{}
	}
	return *ref
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func asModule(fnName string, v RuntimeValue) (*ModuleValue, error) {
	mod, ok := v.(*ModuleValue)
	if !ok {
		return nil, fmt.Errorf("%s expects a Module argument, got %s", fnName, TypeName(v))
	}
	return mod, nil
}
