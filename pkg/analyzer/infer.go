package analyzer

import (
	"context"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

// infer works out what an expression is, as far as it can be known without running the program.
//
// Anything it cannot see through returns unknown, and every check treats unknown as fitting. That is deliberate: the checker exists to report certainties, and guessing would turn it into a source of false alarms in a language that does not require types at all.
func (a *Analyzer) infer(expr ast.Expr, symbols *ast.SymbolTable) checked {
	// Both forms are matched throughout: the parser builds pointers, but a node reached through EnumerateChildNodes can arrive as either.
	switch expr := expr.(type) {
	case *ast.ExprInt, ast.ExprInt:
		return a.builtinNamed("Int", symbols)
	case *ast.ExprFloat, ast.ExprFloat:
		return a.builtinNamed("Float", symbols)
	case *ast.ExprString, ast.ExprString:
		return a.builtinNamed("String", symbols)
	case *ast.ExprStringInterpolation, ast.ExprStringInterpolation:
		return a.builtinNamed("String", symbols)
	case *ast.ExprBool, ast.ExprBool:
		return a.builtinNamed("Bool", symbols)
	case *ast.ExprChar, ast.ExprChar:
		return a.builtinNamed("Char", symbols)
	case *ast.ExprVoid, ast.ExprVoid:
		return a.builtinNamed("Void", symbols)
	case *ast.ExprArray:
		return a.inferArray(*expr, symbols)
	case ast.ExprArray:
		return a.inferArray(expr, symbols)
	case *ast.ExprDict:
		return a.inferDict(expr.Entries, symbols)
	case ast.ExprDict:
		return a.inferDict(expr.Entries, symbols)
	case *ast.ExprFunc:
		return a.inferFunc(expr, symbols)
	case ast.ExprFunc:
		return a.inferFunc(&expr, symbols)
	case *ast.ExprIdentifier:
		return a.inferIdentifier(expr, symbols)
	case ast.ExprIdentifier:
		return a.inferIdentifier(&expr, symbols)
	case *ast.ExprInvocation:
		return a.inferInvocation(expr, symbols)
	case ast.ExprInvocation:
		return a.inferInvocation(&expr, symbols)
	case *ast.ExprMemberAccess:
		return a.inferMemberAccess(expr, symbols)
	case ast.ExprMemberAccess:
		return a.inferMemberAccess(&expr, symbols)
	case *ast.ExprOperatorBinary:
		return a.inferBinaryOperator(expr, symbols)
	case ast.ExprOperatorBinary:
		return a.inferBinaryOperator(&expr, symbols)
	case *ast.ExprIs:
		return a.builtinNamed("Bool", symbols)
	case ast.ExprIs:
		return a.builtinNamed("Bool", symbols)
	case *ast.ExprIf:
		return a.inferIf(expr, symbols)
	case ast.ExprIf:
		return a.inferIf(&expr, symbols)
	case *ast.ExprIndexAccess:
		return a.inferIndexAccess(expr, symbols)
	case ast.ExprIndexAccess:
		return a.inferIndexAccess(&expr, symbols)
	case *ast.ExprSwitch:
		return a.inferSwitch(expr, symbols)
	case ast.ExprSwitch:
		return a.inferSwitch(&expr, symbols)
	}
	return unknownType()
}

// inferIf describes what an if expression yields, which is knowable only when every branch agrees.
// An if without an else yields nothing when the condition is false, so it says nothing about its type.
func (a *Analyzer) inferIf(expr *ast.ExprIf, symbols *ast.SymbolTable) checked {
	if expr.ElseExpr == nil {
		return unknownType()
	}
	branches := make([]ast.Expr, 0, len(expr.ElseIf)+2)
	branches = append(branches, expr.ThenExpr)
	for _, elseIf := range expr.ElseIf {
		branches = append(branches, elseIf.Then)
	}
	branches = append(branches, expr.ElseExpr)
	return a.inferBranches(branches, symbols)
}

// inferSwitch describes what a switch expression yields.
// Without a default case a value may match nothing, so the result is only knowable when one is present and every case agrees.
func (a *Analyzer) inferSwitch(expr *ast.ExprSwitch, symbols *ast.SymbolTable) checked {
	hasDefault := false
	branches := make([]ast.Expr, 0, len(expr.Cases))
	for _, one := range expr.Cases {
		if one.Kind == ast.SwitchCaseDefault {
			hasDefault = true
		}
		branches = append(branches, one.Body)
	}
	if !hasDefault {
		return unknownType()
	}
	return a.inferBranches(branches, symbols)
}

// inferBranches is the type every branch yields, or unknown when they do not all agree.
func (a *Analyzer) inferBranches(branches []ast.Expr, symbols *ast.SymbolTable) checked {
	var common checked
	for i, branch := range branches {
		if branch == nil {
			return unknownType()
		}
		found := a.infer(branch, symbols)
		if !found.isKnown() {
			return unknownType()
		}
		if i == 0 {
			common = found
			continue
		}
		if !sameType(common, found) {
			return unknownType()
		}
	}
	if len(branches) == 0 {
		return unknownType()
	}
	return common
}

// inferArray describes an array literal, keeping the element type only when every element agrees on it.
// A mixed array is an array of unknown, because Zirric arrays are routinely heterogeneous and guessing a common type would invent errors.
func (a *Analyzer) inferArray(expr ast.ExprArray, symbols *ast.SymbolTable) checked {
	result := checked{kind: kindArray, arity: -1}
	if len(expr.Elements) == 0 {
		return result
	}
	first := a.infer(expr.Elements[0], symbols)
	if !first.isKnown() {
		return result
	}
	for _, element := range expr.Elements[1:] {
		other := a.infer(element, symbols)
		if !other.isKnown() || !sameType(first, other) {
			return result
		}
	}
	result.elem = &first
	return result
}

func (a *Analyzer) inferFunc(expr *ast.ExprFunc, symbols *ast.SymbolTable) checked {
	scope := expr.Symbols
	if scope == nil {
		scope = symbols
	}
	params := make([]checked, len(expr.Parameters))
	for i := range expr.Parameters {
		params[i] = a.resolveHintIn(expr.Parameters[i].TypeHint, scope, symbols)
	}
	result := a.resolveHintIn(expr.ReturnType, scope, symbols)
	return checked{kind: kindFunc, params: params, result: &result, arity: len(params)}
}

// inferIdentifier describes what a name stands for, which resolve_identifiers has already attached to the node.
func (a *Analyzer) inferIdentifier(expr *ast.ExprIdentifier, symbols *ast.SymbolTable) checked {
	sym := expr.Symbol
	if sym == nil && symbols != nil {
		sym = symbols.Find(expr.Name.Value)
	}
	if sym == nil {
		return unknownType()
	}
	return a.typeOfSymbolValue(sym, symbols)
}

// typeOfSymbolValue describes the value a name is bound to, as opposed to the type the name stands for.
// The two differ for a declared type: `Person` names a data type, and the value bound to it is the constructor, so calling it is checked like calling a function.
func (a *Analyzer) typeOfSymbolValue(sym *ast.Symbol, symbols *ast.SymbolTable) checked {
	sym = sym.Original()
	if sym == nil || sym.Decl == nil {
		return unknownType()
	}
	scope := symbols
	if sym.ChildTable != nil {
		scope = sym.ChildTable
	}
	switch decl := sym.Decl.(type) {
	case *ast.DeclParameter:
		return a.resolveHint(decl.TypeHint, symbols)
	case ast.DeclParameter:
		return a.resolveHint(decl.TypeHint, symbols)
	case *ast.DeclConstant:
		if hinted := a.resolveHint(decl.TypeHint, symbols); hinted.isKnown() {
			return hinted
		}
		// A constant cannot be reassigned, so whatever it was given is what it stays.
		return a.inferConstantValue(sym, decl.Value, symbols)
	case *ast.DeclVariable:
		if hinted := a.resolveHint(decl.TypeHint, symbols); hinted.isKnown() {
			return hinted
		}
		// A var can be assigned again, so what it holds is only knowable when everything ever assigned to it agrees.
		return a.inferVariable(sym, decl, symbols)
	case *ast.DeclExternValue:
		return a.resolveHint(decl.TypeHint, symbols)
	case *ast.DeclFunc:
		return a.functionType(decl.Impl, decl.ReturnType, symbols)
	case *ast.DeclExternFunc:
		return a.externFunctionType(decl, symbols)
	case *ast.DeclData:
		// The value bound to a data type's name is its constructor.
		return a.constructorTypeIn(decl, sym, scope, symbols)
	case *ast.DeclAttr:
		// An attribute is called with one argument per field it declares, the same way the VM counts them.
		return checked{kind: kindFunc, name: "@" + decl.Name.Value, arity: len(decl.Fields), result: ptrTo(unknownType())}
	case ast.DeclImportMember:
		// A name imported from another module, which is how prelude's own Some, Ok and Err arrive. Without following it, calling one would say nothing about what it builds.
		return a.valueOfImportedDeclaration(decl)
	case *ast.DeclImport:
		// An imported module is a value, and what can be read off it is exactly what it exports.
		return a.moduleValue(decl)
	}
	return unknownType()
}

// valueOfImportedDeclaration describes the value an imported name is bound to, which for a data type is its constructor.
// Only what has already been parsed is read: analyzing the other module from here could recurse back into this one, and a name that cannot be reached is simply unknown.
func (a *Analyzer) valueOfImportedDeclaration(member ast.DeclImportMember) checked {
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
	return a.typeOfSymbolValue(target, resolved.Symbols)
}

func (a *Analyzer) functionType(impl *ast.ExprFunc, returnType ast.TypeExpr, symbols *ast.SymbolTable) checked {
	if impl == nil {
		return checked{kind: kindFunc, arity: -1, result: ptrTo(unknownType())}
	}
	scope := impl.Symbols
	if scope == nil {
		scope = symbols
	}
	params := make([]checked, len(impl.Parameters))
	for i := range impl.Parameters {
		params[i] = a.resolveHintIn(impl.Parameters[i].TypeHint, scope, symbols)
	}
	hint := returnType
	if hint == nil {
		hint = impl.ReturnType
	}
	result := a.resolveHintIn(hint, scope, symbols)
	return checked{kind: kindFunc, params: params, result: &result, arity: len(params)}
}

func (a *Analyzer) externFunctionType(decl *ast.DeclExternFunc, symbols *ast.SymbolTable) checked {
	params := make([]checked, len(decl.Parameters))
	for i := range decl.Parameters {
		params[i] = a.resolveHint(decl.Parameters[i].TypeHint, symbols)
	}
	result := a.resolveHint(decl.ReturnType, symbols)
	return checked{kind: kindFunc, params: params, result: &result, arity: len(params)}
}

// constructorType describes calling a data type's name, which builds a value of it.
func (a *Analyzer) constructorTypeIn(decl *ast.DeclData, sym *ast.Symbol, scope *ast.SymbolTable, outer *ast.SymbolTable) checked {
	params := make([]checked, len(decl.Fields))
	for i := range decl.Fields {
		params[i] = a.fieldType(decl.Fields[i], scope, outer)
	}
	result := checked{kind: kindData, name: decl.Name.Value, sym: sym, arity: len(decl.Fields)}
	return checked{kind: kindFunc, name: decl.Name.Value, params: params, result: &result, arity: len(decl.Fields)}
}

func (a *Analyzer) inferInvocation(expr *ast.ExprInvocation, symbols *ast.SymbolTable) checked {
	callee := a.infer(expr.Function, symbols)
	if callee.kind == kindFunc && callee.result != nil {
		return *callee.result
	}
	return unknownType()
}

// inferMemberAccess describes reading a field off a value, which is only knowable when the value's type is.
func (a *Analyzer) inferMemberAccess(expr *ast.ExprMemberAccess, symbols *ast.SymbolTable) checked {
	// A guarded read answers with an Option or with whatever the enclosing function returns instead, neither of which is the field's own type.
	if expr.Access() != ast.MemberAccessPlain {
		return unknownType()
	}
	target := a.infer(expr.Target, symbols)
	if target.kind == kindModule {
		// Reading a name off a module gives whatever that module bound to it, which is how a qualified call gets checked like any other.
		if target.exports == nil {
			return unknownType()
		}
		member := target.exports.FindMember(expr.Property.Value)
		if member == nil || member.Decl == nil {
			return unknownType()
		}
		return a.typeOfSymbolValue(member, target.exports)
	}
	if target.kind != kindData || target.sym == nil {
		return unknownType()
	}
	decl, ok := target.sym.Decl.(*ast.DeclData)
	if !ok {
		return unknownType()
	}
	for i := range decl.Fields {
		if decl.Fields[i].Name.Value == expr.Property.Value {
			return a.fieldType(decl.Fields[i], memberScope(target.sym, symbols), symbols)
		}
	}
	return unknownType()
}

func ptrTo(c checked) *checked {
	return &c
}

// sameType reports whether two known types are the same declaration or the same builtin.
func sameType(lhs, rhs checked) bool {
	if lhs.kind != rhs.kind {
		return false
	}
	if lhs.sym != nil && rhs.sym != nil {
		return lhs.sym == rhs.sym
	}
	return lhs.name == rhs.name
}

// inferConstantValue works out what a constant holds, guarding against a declaration that refers to itself.
// A cycle means the program is already broken in a way the checker is not here to report, so it answers unknown and leaves it alone.
func (a *Analyzer) inferConstantValue(sym *ast.Symbol, value ast.Expr, symbols *ast.SymbolTable) checked {
	if value == nil {
		return unknownType()
	}
	if a.inferring == nil {
		a.inferring = map[*ast.Symbol]struct{}{}
	}
	if _, busy := a.inferring[sym]; busy {
		return unknownType()
	}
	a.inferring[sym] = struct{}{}
	defer delete(a.inferring, sym)
	return a.infer(value, symbols)
}

// fieldType describes what a field holds.
//
// A field written with parameters is a method: `greet(name: String) -> String` declares a function, and the hint after the arrow is what it returns rather than what the field is. Reading the hint alone would say the field holds a String and reject the function that has to be passed to build one.
func (a *Analyzer) fieldType(field ast.DeclField, scope *ast.SymbolTable, outer *ast.SymbolTable) checked {
	if len(field.Parameters) == 0 {
		return a.resolveHintIn(field.TypeHint, scope, outer)
	}
	params := make([]checked, len(field.Parameters))
	for i := range field.Parameters {
		params[i] = a.resolveHintIn(field.Parameters[i].TypeHint, scope, outer)
	}
	result := a.resolveHintIn(field.TypeHint, scope, outer)
	return checked{kind: kindFunc, params: params, result: &result, arity: len(params)}
}

// moduleValue describes an imported module, carrying what it exports so that a member read off it can be checked.
func (a *Analyzer) moduleValue(decl *ast.DeclImport) checked {
	if a.resolver == nil {
		return unknownType()
	}
	resolved, err := a.resolver.ResolveModule(context.Background(), decl.ModuleName.URI())
	if err != nil || resolved == nil || resolved.Symbols == nil {
		return unknownType()
	}
	return checked{kind: kindModule, name: string(decl.ModuleName.URI()), exports: resolved.Symbols, arity: -1}
}

// builtinNamed describes a literal by the type that declares it, so that what is written on that declaration — @Iterable, @Countable — can be seen.
// A name that does not resolve still gives the type, just without a declaration to read attributes from, which only makes the checker quieter.
func (a *Analyzer) builtinNamed(name string, symbols *ast.SymbolTable) checked {
	resolved := a.resolveNamedHint(ast.StaticReference{ast.Identifier{Value: name}}, symbols)
	if resolved.kind == kindBuiltin && resolved.name == name {
		return resolved
	}
	return builtinType(name)
}

// inferVariable works out what a var holds, from its initial value and every assignment to it.
//
// One disagreeing assignment makes it unknown, which is the honest answer: the checker does not follow the order statements run in, so it can only speak when the answer is the same whichever ran.
func (a *Analyzer) inferVariable(sym *ast.Symbol, decl *ast.DeclVariable, symbols *ast.SymbolTable) checked {
	values := make([]ast.Expr, 0, 2)
	if decl.Value != nil {
		values = append(values, decl.Value)
	}
	values = append(values, a.variableAssignments[sym]...)
	return a.agreedType(sym, values, symbols)
}

// inferDict describes a dict literal, keeping the key and value types only when every entry agrees on them.
// A mixed dict describes neither, because guessing a common type would invent errors in a language where a dict routinely holds several.
func (a *Analyzer) inferDict(entries []ast.ExprDictEntry, symbols *ast.SymbolTable) checked {
	result := checked{kind: kindDict, arity: -1}
	if len(entries) == 0 {
		return result
	}
	keys := make([]ast.Expr, 0, len(entries))
	values := make([]ast.Expr, 0, len(entries))
	for _, entry := range entries {
		keys = append(keys, entry.Key)
		values = append(values, entry.Value)
	}
	if key := a.inferBranches(keys, symbols); key.isKnown() {
		result.key = &key
	}
	if value := a.inferBranches(values, symbols); value.isKnown() {
		result.value = &value
	}
	return result
}

// inferIndexAccess describes what reading an index yields, following the VM: an array gives its element, a dict its value, and both a Binary and a String give a Byte.
//
// A dict yields void for a key it does not hold, but what is written is what the declaration promises, and it is that promise the checker holds a program to.
func (a *Analyzer) inferIndexAccess(expr *ast.ExprIndexAccess, symbols *ast.SymbolTable) checked {
	target := a.infer(expr.Target, symbols)
	switch {
	case target.kind == kindArray:
		if target.elem != nil && target.elem.isKnown() {
			return *target.elem
		}
	case target.kind == kindDict:
		if target.value != nil && target.value.isKnown() {
			return *target.value
		}
	case isBuiltin(target, "Binary"), isBuiltin(target, "String"):
		// Indexing text gives the byte at that position rather than the character, which is what len counts too.
		return a.builtinNamed("Byte", symbols)
	}
	return unknownType()
}
