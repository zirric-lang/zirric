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
	case "_member":
		return MakeExternFunc(decl, func(_ VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			mod, err := asModule("_member", args[0])
			if err != nil {
				return nil, err
			}
			name, ok := args[1].(String)
			if !ok {
				return nil, fmt.Errorf("_member expects a String name, got %T", args[1])
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
	case "_moduleNamed":
		_, globals := ctx.MainPackageModules()
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			name, ok := args[0].(String)
			if !ok {
				return nil, fmt.Errorf("_moduleNamed expects a String name, got %T", args[0])
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
		return nil, fmt.Errorf("%s expects a Module argument, got %T", fnName, v)
	}
	return mod, nil
}
