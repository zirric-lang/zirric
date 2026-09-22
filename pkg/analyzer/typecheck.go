package analyzer

import (
	"reflect"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// checkTypes walks a module reporting the mistakes it is certain about.
//
// It reports two kinds of thing. Some are certain to stop the VM: calling a value that is not callable, calling with the wrong number of arguments, combining values with an operator that has no meaning for them. Others break a contract the program itself declared — passing a String where a parameter says Int — which the VM tolerates today, since it does not enforce hints, but which is a mistake in the program either way.
//
// Everything it cannot see through is treated as fitting, so what it reports is a certainty rather than a suspicion.
func (a *Analyzer) checkTypes(module *ast.ContextModule) []AnalysisError {
	if module == nil || module.Symbols == nil {
		return nil
	}
	// Gathered first: what a var holds depends on every assignment to it, including ones further down than where it is read.
	a.collectVariableAssignments(module)
	defer func() { a.variableAssignments = nil }()

	c := &typeChecker{analyzer: a, seen: map[ast.Node]struct{}{}, narrowed: map[string]checked{}, reported: map[string]struct{}{}}
	c.walk(module, module.Symbols)
	return c.errs
}

// narrowedCase is one branch of a switch, along with the type the branch proves its subject to be.
type narrowedCase struct {
	typeRef ast.TypeExpr
	isType  bool
	body    ast.Node
	extra   []ast.Node
}

type typeChecker struct {
	analyzer *Analyzer
	// narrowed records what a `case is T` branch proves about the value being switched on, by the name it was switched on under.
	// Without it, a branch that can only run for one member of a union would be checked against whatever the subject was declared as, and reading a field only that member has would look like a mistake.
	narrowed map[string]checked
	// reported remembers what has already been said, so one problem is reported once.
	reported map[string]struct{}
	// returns is what the function being walked promises to return, or nil outside any function that declared one.
	returns *checked
	// returnsName is that function's name, for a message that says which promise was broken.
	returnsName string
	errs        []AnalysisError
	// seen records the nodes already checked.
	//
	// EnumerateChildNodes is not consistent about how deep it goes: a context module yields only what it directly holds, while a statement or an operator yields its whole subtree. Walking recursively is the only way to reach everything, and remembering what has been checked is what stops the deeper nodes from being reported once per ancestor.
	seen map[ast.Node]struct{}
}

func (c *typeChecker) report(err *AnalysisError) {
	if err == nil {
		return
	}
	// A node held by value cannot be remembered as visited, so the same problem can be reached more than once. Reporting it once is what a reader wants either way.
	key := err.Error()
	if _, already := c.reported[key]; already {
		return
	}
	c.reported[key] = struct{}{}
	c.errs = append(c.errs, *err)
}

// walk visits a node and everything below it.
//
// The scope it carries is the module's. That is enough, because resolve_identifiers has already attached to each name the symbol it resolved to, so inference does not depend on knowing which inner scope a node sits in; the table is used to resolve type hints, which name module-level or prelude types.
func (c *typeChecker) walk(node ast.Node, symbols *ast.SymbolTable) {
	if node == nil {
		return
	}
	if !c.firstVisit(node) {
		return
	}
	switch n := node.(type) {
	case *ast.ExprFunc:
		c.walkFunc(n, symbols)
		return
	case ast.ExprFunc:
		c.walkFunc(&n, symbols)
		return
	case *ast.StmtReturn:
		c.checkReturn(n.Expr, n.Token, symbols)
	case *ast.ExprSwitch:
		c.walkSwitch(n.Value, exprSwitchCases(n.Cases), symbols)
		return
	case ast.ExprSwitch:
		c.walkSwitch(n.Value, exprSwitchCases(n.Cases), symbols)
		return
	case *ast.StmtSwitch:
		c.walkSwitch(n.Value, stmtSwitchCases(n.Cases), symbols)
		return
	case ast.StmtSwitch:
		c.walkSwitch(n.Value, stmtSwitchCases(n.Cases), symbols)
		return
	case *ast.SourceFile:
		// A file has its own table, and that is where its imports live — prelude among them — so names resolve from here rather than from the module.
		scope := symbols
		if n.Symbols != nil {
			scope = n.Symbols
		}
		n.EnumerateChildNodes(func(child ast.Node) {
			c.walk(child, scope)
		})
		return
	case *ast.ExprIdentifier:
		c.checkIdentifier(n, symbols)
	case *ast.DeclConstant:
		c.checkDeclaredValue(n.TypeHint, n.Value, n.Name.Value, symbols)
	case *ast.DeclVariable:
		c.checkDeclaredValue(n.TypeHint, n.Value, n.Name.Value, symbols)
	case *ast.StmtAssign:
		c.checkAssignment(n, symbols)
	case *ast.ExprOperatorUnary:
		c.checkUnaryOperator(n, symbols)
	case ast.ExprOperatorUnary:
		c.checkUnaryOperator(&n, symbols)
	case *ast.ExprIndexAccess:
		c.checkIndexAccess(n, symbols)
	case ast.ExprIndexAccess:
		c.checkIndexAccess(&n, symbols)
	case *ast.StmtFor:
		c.checkIterated(n.CollectionExpr, symbols)
	case ast.StmtFor:
		c.checkIterated(n.CollectionExpr, symbols)
	case *ast.ExprFor:
		c.checkIterated(n.CollectionExpr, symbols)
	case ast.ExprFor:
		c.checkIterated(n.CollectionExpr, symbols)
	case *ast.ExprInvocation:
		c.checkInvocation(n, symbols)
	case ast.ExprInvocation:
		c.checkInvocation(&n, symbols)
	case *ast.ExprOperatorBinary:
		c.checkBinaryOperator(n, symbols)
	case ast.ExprOperatorBinary:
		c.checkBinaryOperator(&n, symbols)
	case *ast.ExprMemberAccess:
		c.checkMemberAccess(n, symbols)
	case ast.ExprMemberAccess:
		c.checkMemberAccess(&n, symbols)
	}

	node.EnumerateChildNodes(func(child ast.Node) {
		c.walk(child, symbols)
	})
}

// firstVisit reports whether this node has not been walked yet, and records it.
// Only a node held by pointer can be remembered: the value forms are structs holding slices, which a map key cannot be. Those are walked again rather than risking a panic, and the errors they produce are deduplicated by position before being reported.
func (c *typeChecker) firstVisit(node ast.Node) bool {
	if !isPointerNode(node) {
		return true
	}
	if _, visited := c.seen[node]; visited {
		return false
	}
	c.seen[node] = struct{}{}
	return true
}

func isPointerNode(node ast.Node) bool {
	return reflect.ValueOf(node).Kind() == reflect.Pointer
}

// checkInvocation reports a call that cannot succeed: one on a value that is not callable, one with the wrong number of arguments, or one whose arguments do not fit the parameters.
func (c *typeChecker) checkInvocation(expr *ast.ExprInvocation, symbols *ast.SymbolTable) {
	callee := c.inferHere(expr.Function, symbols)
	if !callee.isKnown() {
		return
	}
	position := exprToken(expr.Function)

	if callee.kind != kindFunc {
		c.report(errNotCallable(position, callee.describe()))
		return
	}
	if callee.arity >= 0 && callee.arity != len(expr.Arguments) {
		c.report(errWrongArgumentCount(position, calleeName(expr.Function, callee), callee.arity, len(expr.Arguments)))
		return
	}
	for i, argument := range expr.Arguments {
		if i >= len(callee.params) {
			break
		}
		param := callee.params[i]
		if !param.isKnown() {
			continue
		}
		value := c.inferHere(argument, symbols)
		if !c.analyzer.fits(value, param, symbols) {
			c.report(errArgumentMismatch(exprToken(argument), i+1, calleeName(expr.Function, callee), value.describe(), param.describe()))
		}
	}
}

// checkBinaryOperator reports a combination the VM has no meaning for, which is a failure at the moment it runs.
func (c *typeChecker) checkBinaryOperator(expr *ast.ExprOperatorBinary, symbols *ast.SymbolTable) {
	if !isArithmeticOperator(expr.Operator) {
		return
	}
	lhs := c.inferHere(expr.Left, symbols)
	rhs := c.inferHere(expr.Right, symbols)
	if !lhs.isKnown() || !rhs.isKnown() {
		return
	}
	if arithmeticApplies(expr.Operator, lhs, rhs) {
		return
	}
	c.report(errUnsupportedOperator(exprToken(expr.Left), operatorText(expr.Operator), lhs.describe(), rhs.describe()))
}

// checkMemberAccess reports reading a field a data type does not declare.
func (c *typeChecker) checkMemberAccess(expr *ast.ExprMemberAccess, symbols *ast.SymbolTable) {
	target := c.inferHere(expr.Target, symbols)
	if target.kind == kindModule {
		c.checkModuleMember(expr, target)
		return
	}
	if expr.Access() != ast.MemberAccessPlain {
		// A guarded read takes the value out of its wrapper before reading the field, so the field belongs to what the Option or Result holds — which only a hint written as `T?` or `T!` records.
		held, ok := heldByGuard(target)
		if !ok {
			return
		}
		target = held
	}
	if target.kind != kindData || target.sym == nil {
		return
	}
	decl, ok := target.sym.Decl.(*ast.DeclData)
	if !ok {
		return
	}
	for i := range decl.Fields {
		if decl.Fields[i].Name.Value == expr.Property.Value {
			return
		}
	}
	c.report(errUnknownField(expr.Token, target.describe(), expr.Property.Value))
}

// heldByGuard is what a `?.` or `!.` reads the field off: what the wrapper holds, which is recorded only by a hint written as `T?` or `T!`.
// Anything else — a value that merely happens to be a Some, a union of one's own, a type the checker cannot see into — says nothing about what comes out of it, and nothing is checked.
func heldByGuard(target checked) (checked, bool) {
	if target.kind == kindUnion && target.elem != nil {
		return *target.elem, true
	}
	return checked{}, false
}

// calleeName names what is being called, for a message that says which call is wrong.
func calleeName(fn ast.Expr, callee checked) string {
	switch fn := fn.(type) {
	case *ast.ExprIdentifier:
		return fn.Name.Value
	case ast.ExprIdentifier:
		return fn.Name.Value
	case *ast.ExprMemberAccess:
		return fn.Property.Value
	case ast.ExprMemberAccess:
		return fn.Property.Value
	}
	if callee.name != "" {
		return callee.name
	}
	return "this function"
}

func exprToken(expr ast.Expr) token.Token {
	if expr == nil {
		return token.Token{}
	}
	return expr.TokenLiteral()
}

func exprSwitchCases(cases []ast.ExprSwitchCase) []narrowedCase {
	out := make([]narrowedCase, len(cases))
	for i := range cases {
		out[i] = narrowedCase{
			typeRef: cases[i].TypeRef,
			isType:  cases[i].Kind == ast.SwitchCaseIsType,
			body:    cases[i].Body,
			extra:   []ast.Node{cases[i].Pattern},
		}
	}
	return out
}

func stmtSwitchCases(cases []ast.StmtSwitchCase) []narrowedCase {
	out := make([]narrowedCase, len(cases))
	for i := range cases {
		extra := make([]ast.Node, 0, len(cases[i].Body)+1)
		extra = append(extra, cases[i].Pattern)
		for _, stmt := range cases[i].Body {
			extra = append(extra, stmt)
		}
		out[i] = narrowedCase{
			typeRef: cases[i].TypeRef,
			isType:  cases[i].Kind == ast.SwitchCaseIsType,
			extra:   extra,
		}
	}
	return out
}

// walkSwitch checks a switch, treating the value as the type each branch proves it to be.
// Only a switch on a plain name can be narrowed, since that is the only subject a branch body refers to by the same name.
func (c *typeChecker) walkSwitch(value ast.Expr, cases []narrowedCase, symbols *ast.SymbolTable) {
	c.walk(value, symbols)

	subject := switchSubjectName(value)
	for _, one := range cases {
		restore, restoring := c.narrow(subject, one, symbols)
		if one.body != nil {
			c.walk(one.body, symbols)
		}
		for _, node := range one.extra {
			c.walk(node, symbols)
		}
		if restoring {
			c.restore(subject, restore)
		}
	}
}

// narrow records what this branch proves about the subject, returning what was there before.
func (c *typeChecker) narrow(subject string, one narrowedCase, symbols *ast.SymbolTable) (previous *checked, narrowing bool) {
	if subject == "" || !one.isType {
		return nil, false
	}
	if existing, ok := c.narrowed[subject]; ok {
		previous = &existing
	}
	// A type the checker cannot resolve leaves the subject unknown for this branch, which is the safe answer rather than the declared one.
	c.narrowed[subject] = c.analyzer.resolveHint(one.typeRef, symbols)
	return previous, true
}

func (c *typeChecker) restore(subject string, previous *checked) {
	if previous == nil {
		delete(c.narrowed, subject)
		return
	}
	c.narrowed[subject] = *previous
}

// switchSubjectName is the name a switch is on, or empty when it is on something that has none.
func switchSubjectName(value ast.Expr) string {
	switch value := value.(type) {
	case *ast.ExprIdentifier:
		return value.Name.Value
	case ast.ExprIdentifier:
		return value.Name.Value
	}
	return ""
}

// inferHere infers an expression, preferring what the branch being checked proves about a plain name.
func (c *typeChecker) inferHere(expr ast.Expr, symbols *ast.SymbolTable) checked {
	if name := switchSubjectName(expr); name != "" {
		if narrowed, ok := c.narrowed[name]; ok {
			return narrowed
		}
	}
	return c.analyzer.infer(expr, symbols)
}

// walkFunc walks a function body, remembering what it promises to return.
// A nested function replaces that promise for its own body and puts back what was there, so a return is always checked against the function it belongs to.
func (c *typeChecker) walkFunc(fn *ast.ExprFunc, symbols *ast.SymbolTable) {
	scope := fn.Symbols
	if scope == nil {
		scope = symbols
	}

	previous, previousName := c.returns, c.returnsName
	if declared := c.analyzer.resolveHintIn(fn.ReturnType, scope, symbols); declared.isKnown() {
		c.returns = &declared
	} else {
		c.returns = nil
	}
	c.returnsName = fn.Name

	for i := range fn.Parameters {
		c.walk(&fn.Parameters[i], symbols)
	}
	for _, stmt := range fn.Impl {
		c.walk(stmt, symbols)
	}

	c.returns, c.returnsName = previous, previousName
}

// checkReturn reports a returned value that cannot be what the function says it returns.
func (c *typeChecker) checkReturn(value ast.Expr, tok token.Token, symbols *ast.SymbolTable) {
	if c.returns == nil || !c.returns.isKnown() {
		return
	}
	// A bare return yields None, which the checker has nothing certain to say about.
	if value == nil {
		return
	}
	found := c.inferHere(value, symbols)
	if !found.isKnown() {
		return
	}
	if c.analyzer.fits(found, *c.returns, symbols) {
		return
	}
	c.report(errReturnMismatch(tok, c.returnsName, found.describe(), c.returns.describe()))
}

// checkIdentifier reports a name that stands for nothing, which is what a typo looks like.
//
// The compiler already refuses to compile one, but only the first it meets and only once a program is built. Reporting it here means every one is found at once and an editor can show it while it is being typed.
func (c *typeChecker) checkIdentifier(expr *ast.ExprIdentifier, symbols *ast.SymbolTable) {
	symbol := expr.Symbol
	if symbol == nil && symbols != nil {
		symbol = symbols.Find(expr.Name.Value)
	}
	if symbol != nil && symbol.Decl != nil {
		return
	}
	c.report(errUndefinedName(expr.Name.TokenLiteral(), expr.Name.Value))
}

// checkModuleMember reports a name read off a module that the module does not export, which is what a typo in a qualified name looks like.
func (c *typeChecker) checkModuleMember(expr *ast.ExprMemberAccess, module checked) {
	if module.exports == nil {
		return
	}
	found := module.exports.FindMember(expr.Property.Value)
	if found == nil || found.Decl == nil {
		c.report(errUnknownModuleMember(expr.Token, module.name, expr.Property.Value, "does not declare"))
		return
	}
	// A module value holds only what the module exports, so reading anything else finds nothing at runtime.
	if found.Decl.ExportScope() != ast.ExportScopePublic {
		c.report(errUnknownModuleMember(expr.Token, module.name, expr.Property.Value, "does not export"))
	}
}

// checkDeclaredValue reports a declaration whose value cannot be what its hint says it is.
func (c *typeChecker) checkDeclaredValue(hint ast.TypeExpr, value ast.Expr, name string, symbols *ast.SymbolTable) {
	if hint == nil || value == nil {
		return
	}
	declared := c.analyzer.resolveHint(hint, symbols)
	if !declared.isKnown() {
		return
	}
	found := c.inferHere(value, symbols)
	if !found.isKnown() || c.analyzer.fits(found, declared, symbols) {
		return
	}
	c.report(errDeclaredValueMismatch(exprToken(value), name, found.describe(), declared.describe()))
}

// checkAssignment reports a value that cannot be what the thing being assigned to was declared to hold.
// Only a plain assignment is checked: a compound one combines the old value with the new, and the operator check already covers that.
func (c *typeChecker) checkAssignment(stmt *ast.StmtAssign, symbols *ast.SymbolTable) {
	if stmt.Op != "" || stmt.Target == nil || stmt.Value == nil {
		return
	}
	declared := c.inferHere(stmt.Target, symbols)
	if !declared.isKnown() {
		return
	}
	found := c.inferHere(stmt.Value, symbols)
	if !found.isKnown() || c.analyzer.fits(found, declared, symbols) {
		return
	}
	c.report(errDeclaredValueMismatch(exprToken(stmt.Value), assignedName(stmt.Target), found.describe(), declared.describe()))
}

// checkUnaryOperator reports an operand the operator has no meaning for, which is a failure at the moment it runs.
func (c *typeChecker) checkUnaryOperator(expr *ast.ExprOperatorUnary, symbols *ast.SymbolTable) {
	operand := c.inferHere(expr.Expr, symbols)
	if mightBeAnything(operand) {
		return
	}
	operator := token.Token(expr.Operator).Type
	switch operator {
	case token.MINUS:
		if isNumeric(operand) {
			return
		}
	case token.BANG:
		if isBuiltin(operand, "Bool") {
			return
		}
	default:
		return
	}
	c.report(errUnsupportedUnaryOperator(exprToken(expr.Expr), string(operator), operand.describe()))
}

// checkIndexAccess reports indexing something the VM cannot index.
func (c *typeChecker) checkIndexAccess(expr *ast.ExprIndexAccess, symbols *ast.SymbolTable) {
	target := c.inferHere(expr.Target, symbols)
	if mightBeAnything(target) || isIndexable(target) {
		return
	}
	c.report(errNotIndexable(exprToken(expr.Target), target.describe()))
}

// isIndexable lists what the VM accepts an index on, which is Array, Binary, Dict and String and nothing else.
func isIndexable(c checked) bool {
	switch c.kind {
	case kindArray, kindDict:
		return true
	}
	return isBuiltin(c, "Array") || isBuiltin(c, "Binary") || isBuiltin(c, "Dict") || isBuiltin(c, "String")
}

// checkIterated reports iterating over a value that cannot be iterated.
// An array is iterable by definition, and everything else has to say so by carrying @Iterable, which is exactly the check the VM makes at runtime.
func (c *typeChecker) checkIterated(collection ast.Expr, symbols *ast.SymbolTable) {
	if collection == nil {
		return
	}
	value := c.inferHere(collection, symbols)
	if mightBeAnything(value) || value.kind == kindArray || isBuiltin(value, "Array") {
		return
	}
	iterable := c.analyzer.findTypeSymbol(ast.StaticReference{ast.Identifier{Value: "Iterable"}}, symbols)
	if iterable == nil {
		// Without the attribute itself there is nothing to compare against, and guessing would reject working code.
		return
	}
	if c.analyzer.carriesAttributes(value, []*ast.Symbol{iterable}, symbols) {
		return
	}
	c.report(errNotIterable(exprToken(collection), value.describe()))
}

// assignedName names what is being assigned to, for a message that says which one is wrong.
func assignedName(target ast.Expr) string {
	switch target := target.(type) {
	case *ast.ExprIdentifier:
		return target.Name.Value
	case *ast.ExprMemberAccess:
		return target.Property.Value
	}
	return "this"
}

// collectVariableAssignments records every expression assigned to a var anywhere in the module.
//
// It is gathered before checking because an assignment further down decides what a var holds just as much as its declaration does, and the checker reads a var long before it reaches the end of the function.
func (a *Analyzer) collectVariableAssignments(module *ast.ContextModule) {
	found := map[*ast.Symbol][]ast.Expr{}

	seen := map[ast.Node]struct{}{}
	var gather func(node ast.Node)
	gather = func(node ast.Node) {
		if node == nil {
			return
		}
		if isPointerNode(node) {
			if _, already := seen[node]; already {
				return
			}
			seen[node] = struct{}{}
		}
		if assign, ok := node.(*ast.StmtAssign); ok {
			// A compound assignment combines the old value with the new, which is the operator check's business rather than this one's.
			if assign.Op == "" && assign.Value != nil {
				if sym := assignedSymbol(assign.Target); sym != nil {
					found[sym] = append(found[sym], assign.Value)
				}
			}
		}
		node.EnumerateChildNodes(gather)
	}
	gather(module)

	a.variableAssignments = found
}

// agreedType is the type every one of these expressions yields, or unknown when they disagree or when there are none.
// A guard against a variable whose own value mentions it keeps a self-referential declaration from looping.
func (a *Analyzer) agreedType(sym *ast.Symbol, values []ast.Expr, symbols *ast.SymbolTable) checked {
	if len(values) == 0 {
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

	var common checked
	for i, value := range values {
		found := a.infer(value, symbols)
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
	return common
}

// assignedSymbol is the symbol an assignment target names, when it names one at all.
func assignedSymbol(target ast.Expr) *ast.Symbol {
	ident, ok := target.(*ast.ExprIdentifier)
	if !ok || ident.Symbol == nil {
		return nil
	}
	return ident.Symbol.Original()
}
