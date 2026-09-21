package analyzer

import (
	"context"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

// typeKind names what the checker was able to work out about a value.
//
// kindUnknown is the one that matters: it means "anything", and every check treats it as fitting. Zirric is dynamically typed, so the checker only ever reports what it is certain about, and an unknown is how it stays certain.
type typeKind int

const (
	kindUnknown typeKind = iota
	kindBuiltin
	kindArray
	kindDict
	kindFunc
	kindData
	kindUnion
	kindAttrs
	kindModule
)

// checked is what the checker knows about a value or about a written type hint.
//
// A hint is trusted as the truth about a value. The VM does not enforce hints, so this is a statement about the program's contract rather than about what the VM would do — which is exactly what makes `fn(x: Int)` called with a String worth reporting.
type checked struct {
	kind typeKind
	// name is the declared or builtin name, for reporting and for comparing builtins.
	name string
	// sym is the declaration a named type resolves to, which is how data and union types are compared and how fields are looked up.
	sym *ast.Symbol
	// elem is an array's element type; key and value a dict's.
	elem  *checked
	key   *checked
	value *checked
	// params and result describe anything callable, which includes a data constructor: calling one is the same act as calling a function.
	params []checked
	result *checked
	// arity is how many arguments a call takes, or -1 when that is not known.
	arity int
	// attrs holds the attributes an attribute constraint requires.
	attrs []*ast.Symbol
	// exports is what a module makes available, for checking that a member read off it exists.
	exports *ast.SymbolTable
}

func unknownType() checked {
	return checked{kind: kindUnknown, arity: -1}
}

func builtinType(name string) checked {
	return checked{kind: kindBuiltin, name: name, arity: -1}
}

func (c checked) isKnown() bool {
	return c.kind != kindUnknown
}

// describe names the type the way a person would write it, for error messages.
func (c checked) describe() string {
	switch c.kind {
	case kindUnknown:
		return "Any"
	case kindArray:
		if c.elem != nil && c.elem.isKnown() {
			return "[" + c.elem.describe() + "]"
		}
		return "[Any]"
	case kindDict:
		key, value := "Any", "Any"
		if c.key != nil && c.key.isKnown() {
			key = c.key.describe()
		}
		if c.value != nil && c.value.isKnown() {
			value = c.value.describe()
		}
		return "[" + key + ": " + value + "]"
	case kindFunc:
		if c.name != "" {
			return c.name
		}
		return "a function"
	case kindModule:
		if c.name != "" {
			return "module " + c.name
		}
		return "a module"
	case kindAttrs:
		out := ""
		for i, attr := range c.attrs {
			if i > 0 {
				out += " "
			}
			out += "@" + attr.Name
		}
		return out
	}
	if c.name != "" {
		return c.name
	}
	return "Any"
}

// resolveHint turns a written type hint into what the checker can compare against.
// A name it cannot resolve becomes unknown rather than an error: naming an unknown type is the business of static reference validation, and reporting it twice would help no one.
func (a *Analyzer) resolveHint(hint ast.TypeExpr, symbols *ast.SymbolTable) checked {
	switch hint := hint.(type) {
	case nil:
		return unknownType()
	case ast.TypeExprRef:
		return a.resolveNamedHint(hint.Reference, symbols)
	case ast.TypeExprArray:
		elem := a.resolveHint(hint.Element, symbols)
		return checked{kind: kindArray, elem: &elem, arity: -1}
	case ast.TypeExprDict:
		key := a.resolveHint(hint.Key, symbols)
		value := a.resolveHint(hint.Value, symbols)
		return checked{kind: kindDict, key: &key, value: &value, arity: -1}
	case ast.TypeExprFunc:
		params := make([]checked, len(hint.Parameters))
		for i := range hint.Parameters {
			params[i] = a.resolveHint(hint.Parameters[i].TypeHint, symbols)
		}
		result := a.resolveHint(hint.ReturnType, symbols)
		return checked{kind: kindFunc, params: params, result: &result, arity: len(params)}
	case ast.TypeExprAttrs:
		var attrs []*ast.Symbol
		for _, attr := range hint.Attrs {
			sym := lookupTypeSymbol(attr.Reference, symbols)
			if sym == nil {
				// One unresolvable attribute makes the whole constraint unknowable, since a value might well carry it.
				return unknownType()
			}
			attrs = append(attrs, sym)
		}
		if len(attrs) == 0 {
			return unknownType()
		}
		return checked{kind: kindAttrs, attrs: attrs, arity: -1}
	}
	return unknownType()
}

// resolveHintIn resolves a hint in the first scope that can see the names it uses.
//
// A function's own table does not always chain out to the file that imported prelude, so a hint written inside one can fail to resolve there while resolving perfectly well a level out. Trying the scopes in turn costs nothing and is what makes a hint like Int mean the same thing wherever it is written.
func (a *Analyzer) resolveHintIn(hint ast.TypeExpr, scopes ...*ast.SymbolTable) checked {
	for _, scope := range scopes {
		if scope == nil {
			continue
		}
		if resolved := a.resolveHint(hint, scope); resolved.isKnown() {
			return resolved
		}
	}
	return unknownType()
}

// resolveNamedHint resolves a type named directly, e.g. String, Person or tests.TestCase.
func (a *Analyzer) resolveNamedHint(ref ast.StaticReference, symbols *ast.SymbolTable) checked {
	sym := a.findTypeSymbol(ref, symbols)
	if sym == nil || sym.Decl == nil {
		return unknownType()
	}
	return a.typeOfDeclaration(sym)
}

// findTypeSymbol resolves a written type name, following an import when the name is qualified by one.
//
// A plain `import tests` binds the module under a name but leaves that symbol without a table of the module's members, so `tests.TestCase` cannot be resolved by walking symbols alone and has to be looked up in the module itself.
func (a *Analyzer) findTypeSymbol(ref ast.StaticReference, symbols *ast.SymbolTable) *ast.Symbol {
	if sym := lookupTypeSymbol(ref, symbols); sym != nil && sym.Decl != nil {
		return sym
	}
	if symbols == nil || len(ref) < 2 || a.resolver == nil {
		return nil
	}
	head := symbols.Find(ref[0].Value)
	if head == nil {
		return nil
	}
	imported, ok := head.Original().Decl.(*ast.DeclImport)
	if !ok {
		return nil
	}
	resolved, err := a.resolver.ResolveModule(context.Background(), imported.ModuleName.URI())
	if err != nil || resolved == nil || resolved.Symbols == nil {
		return nil
	}
	sym := resolved.Symbols.FindMember(ref[1].Value)
	for i := 2; i < len(ref); i++ {
		if sym == nil || sym.ChildTable == nil {
			return nil
		}
		sym = sym.ChildTable.FindMember(ref[i].Value)
	}
	if sym == nil {
		return nil
	}
	return sym.Original()
}

// typeOfDeclaration describes the type a declaration stands for when its name is used as a type.
func (a *Analyzer) typeOfDeclaration(sym *ast.Symbol) checked {
	sym = sym.Original()
	if sym == nil || sym.Decl == nil {
		return unknownType()
	}
	switch decl := sym.Decl.(type) {
	case *ast.DeclData:
		return checked{kind: kindData, name: decl.Name.Value, sym: sym, arity: len(decl.Fields)}
	case *ast.DeclUnion:
		return checked{kind: kindUnion, name: decl.Name.Value, sym: sym, arity: -1}
	case *ast.DeclExternType:
		// Any is the escape hatch the language offers, so anything hinted with it fits everywhere.
		if sym.Name == "Any" {
			return unknownType()
		}
		return checked{kind: kindBuiltin, name: sym.Name, sym: sym, arity: -1}
	case *ast.DeclAttr:
		return checked{kind: kindAttrs, name: sym.Name, attrs: []*ast.Symbol{sym}, arity: -1}
	case ast.DeclImportMember:
		// A name imported from another module, which is how prelude's own types arrive, so the declaration itself is one module further on.
		return a.typeOfImportedDeclaration(decl)
	}
	return unknownType()
}

// typeOfImportedDeclaration follows an imported name to the declaration it stands for.
// It only reads what has already been parsed: analyzing the other module from here could recurse back into this one, and a type that cannot be reached is simply unknown.
func (a *Analyzer) typeOfImportedDeclaration(member ast.DeclImportMember) checked {
	if a.resolver == nil {
		return unknownType()
	}
	resolved, err := a.resolver.ResolveModule(context.Background(), member.ModuleName.URI())
	if err != nil || resolved == nil || resolved.Symbols == nil {
		return unknownType()
	}
	target := resolved.Symbols.FindMember(member.Name.Value)
	if target == nil || target.Decl == nil {
		return unknownType()
	}
	if _, again := target.Decl.(ast.DeclImportMember); again {
		// One hop is enough; a chain of re-exports is not worth chasing for a check that may simply say nothing.
		return unknownType()
	}
	return a.typeOfDeclaration(target)
}

// lookupTypeSymbol finds the symbol a type name refers to, following an import to the declaration itself.
func lookupTypeSymbol(ref ast.StaticReference, symbols *ast.SymbolTable) *ast.Symbol {
	if symbols == nil || len(ref) == 0 {
		return nil
	}
	// Read-only: looking a name up the usual way would define phantom symbols and free variables, which a checker must never do.
	sym := symbols.FindRef(ref)
	if sym == nil {
		return nil
	}
	return sym.Original()
}

// unionMembers returns the types a union declares, or nothing when they cannot all be resolved.
// A union that is only partly resolvable is treated as unknown, since a value might be one of the members the checker cannot see.
func (a *Analyzer) unionMembers(c checked, symbols *ast.SymbolTable) []checked {
	if c.kind != kindUnion || c.sym == nil {
		return nil
	}
	decl, ok := c.sym.Decl.(*ast.DeclUnion)
	if !ok {
		return nil
	}
	members := make([]checked, 0, len(decl.Members))
	for _, member := range decl.Members {
		resolved := a.resolveNamedHint(member.Member, memberScope(c.sym, symbols))
		if !resolved.isKnown() {
			return nil
		}
		members = append(members, resolved)
	}
	return members
}

// memberScope is the table a union's member names resolve in, which is the union's own table when it declares its members inline.
func memberScope(sym *ast.Symbol, fallback *ast.SymbolTable) *ast.SymbolTable {
	if sym != nil && sym.ChildTable != nil {
		return sym.ChildTable
	}
	return fallback
}
