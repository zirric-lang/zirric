package analyzer

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

// fits reports whether a value of type value may be used where target is expected.
//
// It answers "not certainly wrong" rather than "provably right": anything unknown on either side fits, and so does anything the checker has not been taught. Only a combination it is sure about can be rejected, because a false alarm in a dynamically typed language costs more than a missed one.
func (a *Analyzer) fits(value checked, target checked, symbols *ast.SymbolTable) bool {
	if !value.isKnown() || !target.isKnown() {
		return true
	}

	// A union fits when any member does, in either direction: the value might be the member that works, and the target accepts any of its members.
	if target.kind == kindUnion {
		members := a.unionMembers(target, symbols)
		if len(members) == 0 {
			return true
		}
		for _, member := range members {
			if a.fits(value, member, symbols) {
				return true
			}
		}
		return false
	}
	if value.kind == kindUnion {
		members := a.unionMembers(value, symbols)
		if len(members) == 0 {
			return true
		}
		for _, member := range members {
			if a.fits(member, target, symbols) {
				return true
			}
		}
		return false
	}

	if target.kind == kindAttrs {
		return a.carriesAttributes(value, target.attrs, symbols)
	}
	// A value that is only known as "something carrying these attributes" could be anything that carries them.
	if value.kind == kindAttrs {
		return true
	}

	// An array, dict, function or module is a value of the builtin type that names its kind, so `[1, 2]` fits a parameter declared Array just as it fits one declared [Int].
	if builtinNameForKind(value.kind) != "" && isBuiltin(target, builtinNameForKind(value.kind)) {
		return true
	}
	if builtinNameForKind(target.kind) != "" && isBuiltin(value, builtinNameForKind(target.kind)) {
		return true
	}
	// Nothing is ever checked against one particular module, only against being a module at all.
	if target.kind == kindModule {
		return true
	}

	if value.kind != target.kind {
		return false
	}

	switch target.kind {
	case kindBuiltin:
		return builtinFits(value.name, target.name)
	case kindData:
		return value.sym == nil || target.sym == nil || value.sym == target.sym
	case kindArray:
		return a.elementFits(value.elem, target.elem, symbols)
	case kindDict:
		return a.elementFits(value.key, target.key, symbols) && a.elementFits(value.value, target.value, symbols)
	case kindFunc:
		// Only the shape is compared: a function of the wrong arity can never be called correctly, whereas its parameter types are the callee's business.
		if value.arity < 0 || target.arity < 0 {
			return true
		}
		return value.arity == target.arity
	}
	return true
}

func (a *Analyzer) elementFits(value *checked, target *checked, symbols *ast.SymbolTable) bool {
	if value == nil || target == nil {
		return true
	}
	return a.fits(*value, *target, symbols)
}

// builtinFits compares two builtin type names.
// Int is accepted where Float is expected because the VM promotes one to the other in arithmetic, so refusing it would report a mismatch the language does not have.
func builtinFits(value, target string) bool {
	if value == target {
		return true
	}
	return value == "Int" && target == "Float"
}

// carriesAttributes reports whether a value's type is declared with every attribute a constraint requires.
// It is only ever sure for a data or union type, whose attributes are written on the declaration; anything else is left alone.
func (a *Analyzer) carriesAttributes(value checked, required []*ast.Symbol, symbols *ast.SymbolTable) bool {
	declared := a.declaredAttributes(value)
	if declared == nil {
		return true
	}
	for _, want := range required {
		if !containsSymbol(declared, want, symbols) {
			return false
		}
	}
	return true
}

// declaredAttributes returns the attribute chain written on a type, or nil when the checker cannot see one.
func (a *Analyzer) declaredAttributes(value checked) ast.AttributeChain {
	if value.sym == nil || value.sym.Decl == nil {
		return nil
	}
	switch decl := value.sym.Decl.(type) {
	case *ast.DeclData:
		return decl.Attributes
	case *ast.DeclUnion:
		return decl.Attributes
	case *ast.DeclExternType:
		return decl.Attributes
	}
	return nil
}

// containsSymbol reports whether a chain names the attribute want.
// An entry that cannot be resolved counts as a match, since it might well be the one being looked for.
func containsSymbol(chain ast.AttributeChain, want *ast.Symbol, symbols *ast.SymbolTable) bool {
	for _, instance := range chain {
		if instance == nil {
			continue
		}
		sym := lookupTypeSymbol(instance.Reference, symbols)
		if sym == nil {
			return true
		}
		if sameAttribute(sym, want, instance.Reference) {
			return true
		}
	}
	return false
}

// sameAttribute reports whether two references name the same attribute.
//
// The chain written on a type belongs to the module that declared it, while the constraint being checked was resolved where it was written, so the two can be different symbols for the same attribute — prelude's own @Countable against an importer's view of it. Identity settles the easy case and the written name settles the rest.
// Matching by name can at worst accept a value that carries a different attribute of the same name, which costs a missed report; getting this wrong the other way would reject working code.
func sameAttribute(found *ast.Symbol, want *ast.Symbol, reference ast.StaticReference) bool {
	if want == nil {
		return true
	}
	if found.Original() == want.Original() {
		return true
	}
	if found.Name != "" && found.Name == want.Name {
		return true
	}
	return reference.Name().Value == want.Name
}

// builtinNameForKind is the builtin type whose values are of that kind, or empty for a kind that has none.
// Writing `Array` and writing `[Int]` describe the same values, so the checker has to see them as the same thing.
func builtinNameForKind(kind typeKind) string {
	switch kind {
	case kindArray:
		return "Array"
	case kindDict:
		return "Dict"
	case kindFunc:
		return "Func"
	case kindModule:
		return "AnyModule"
	}
	return ""
}
