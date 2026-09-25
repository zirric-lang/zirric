package runtime

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ ExternPlugin = &FmtPlugin{}

// FmtPlugin provides runtime bindings for the fmt module's extern declarations.
type FmtPlugin struct{}

func (*FmtPlugin) Module() string { return "fmt" }

// Bind implements ExternPlugin.
func (*FmtPlugin) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "sprint":
		printableAttrId := attributeIdOf(ctx.ResolveModuleSymbol("prelude", "Printable"))
		return MakeExternFunc(decl, func(caller VMCaller, args []RuntimeValue) (RuntimeValue, error) {
			str, err := SprintValue(caller, printableAttrId, args[0])
			if err != nil {
				return nil, err
			}
			return String(str), nil
		})
	}
	return nil
}

// attributeIdOf is the TypeId an attribute declaration was compiled to, or nil when the declaration could not be resolved at all.
func attributeIdOf(symbol *ast.Symbol) *TypeId {
	if symbol == nil || symbol.ConstantId == nil {
		return nil
	}
	id := TypeId(*symbol.ConstantId)
	return &id
}

// SprintValue converts value to a displayable string, which is what fmt.sprint answers with and what a `\( … )` interpolation splices into its literal.
// printableAttrId is the TypeId of prelude's @Printable, or nil where it could not be resolved, in which case only the fallbacks below apply.
//
// It prefers the value's @Printable attribute when it has one (covering String itself, which declares @Printable in prelude/shim.zirr, and any user-defined @Printable type), then falls back to a fixed set of trivial conversions for builtin types that aren't @Printable-annotated (Int, Float, Char, Byte, Bool, Void), and finally to Inspect() as a last resort so it never errors out for a value with no other printable representation.
func SprintValue(caller VMCaller, printableAttrId *TypeId, value RuntimeValue) (string, error) {
	if printableAttrId != nil && caller != nil {
		if attrs := caller.AttributesOf(value); attrs != nil {
			if globalId, ok := attrs[*printableAttrId]; ok {
				attrValue, err := caller.ResolveGlobal(globalId)
				if err != nil {
					return "", err
				}
				toStringFn := attrValue.Lookup("toString")
				if toStringFn == nil {
					return "", fmt.Errorf("@Printable attribute value has no toString field")
				}
				result, err := caller.CallFunction(toStringFn, value)
				if err != nil {
					return "", err
				}
				str, ok := result.(String)
				if !ok {
					return "", fmt.Errorf("@Printable toString must return a String, got %s", TypeName(result))
				}
				return string(str), nil
			}
		}
	}

	if s, ok := TrivialString(value); ok {
		return s, nil
	}

	return value.Inspect(), nil
}
