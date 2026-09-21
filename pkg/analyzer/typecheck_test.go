package analyzer_test

import (
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/analyzer"
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
)

// analyzeSource analyzes a standalone module and returns the diagnostics it produced.
// The module declares the builtin types it needs, since nothing resolves prelude here.
func analyzeSource(t *testing.T, body string) []analyzer.AnalysisError {
	t.Helper()
	module := parseModule(t, "test", "mod test\nextern type Int {}\nextern type String {}\nextern type Any {}\n"+body)
	errs, _ := analyzer.New(nil).Analyze(module, false)
	return errs
}

// expectError asserts that exactly one diagnostic was produced and that it says what it should.
func expectError(t *testing.T, body string, wantSummary string, wantDetails string) {
	t.Helper()
	errs := analyzeSource(t, body)
	if len(errs) != 1 {
		t.Fatalf("expected exactly one diagnostic, got %d: %v", len(errs), errs)
	}
	if errs[0].Summary != wantSummary {
		t.Errorf("expected summary %q, got %q", wantSummary, errs[0].Summary)
	}
	if !strings.Contains(errs[0].Details, wantDetails) {
		t.Errorf("expected details containing %q, got %q", wantDetails, errs[0].Details)
	}
	if errs[0].Token.Source == nil || errs[0].Token.Source.Line <= 0 {
		t.Errorf("the diagnostic carries no position: %s", errs[0].Error())
	}
}

// expectClean asserts that nothing is reported, which is what matters most: a checker that cries wolf in a language where types are optional is worse than none.
func expectClean(t *testing.T, body string) {
	t.Helper()
	if errs := analyzeSource(t, body); len(errs) > 0 {
		t.Errorf("expected no diagnostics, got: %v", errs)
	}
}

func TestWrongArgumentCountIsReported(t *testing.T) {
	expectError(t, "fn two(a, b) { a }\nfn main() { two(1) }", "wrong number of arguments", "two takes 2, got 1")
	expectError(t, "data P { a, b }\nfn main() { P(1) }", "wrong number of arguments", "P takes 2, got 1")
	expectError(t, "fn none() { 1 }\nfn main() { none(1) }", "wrong number of arguments", "none takes 0, got 1")
}

func TestWrongArgumentTypeIsReported(t *testing.T) {
	expectError(t, "fn takesInt(x: Int) { x }\nfn main() { takesInt(\"hello\") }", "wrong argument type", "declared Int, got String")
	expectError(t, "data P { n: Int }\nfn main() { P(\"hello\") }", "wrong argument type", "declared Int, got String")
}

func TestCallingSomethingThatIsNotAFunctionIsReported(t *testing.T) {
	expectError(t, "fn main() {\n\tconst x = 5\n\tx()\n}", "not callable", "Int cannot be called")
}

func TestUnsupportedOperatorIsReported(t *testing.T) {
	expectError(t, "data P { a }\nfn main() { P(1) + 3 }", "unsupported operator", "+ is not defined for P and Int")
	// A String only ever concatenates, so any other operator on one is a failure at the moment it runs.
	expectError(t, "fn main() { \"a\" - \"b\" }", "unsupported operator", "- is not defined for String and String")
	expectError(t, "data P { a }\nfn main() { P(1) * 2 }", "unsupported operator", "*")
}

func TestUnknownFieldIsReported(t *testing.T) {
	expectError(t, "data P { a }\nfn main() { P(1).nope }", "unknown field", "P has no field nope")
}

func TestWhatIsNotKnownIsNotReported(t *testing.T) {
	// Every one of these would be a false alarm: the checker cannot see what the value is, so it must say nothing.
	expectClean(t, "fn takesAnything(x) { x }\nfn main() { takesAnything(\"hello\") }")
	expectClean(t, "fn takesAny(x: Any) { x }\nfn main() { takesAny(\"hello\") }")
	expectClean(t, "fn main(x) { x + 1 }")
	expectClean(t, "fn main(x) { x() }")
	expectClean(t, "fn main(x) { x.whatever }")
}

func TestOperatorsTheVMAcceptsAreNotReported(t *testing.T) {
	expectClean(t, "fn main() { 1 + 2 }")
	expectClean(t, "fn main() { 1.5 * 2 }")
	// Int and Float mix, because the VM promotes one to the other.
	expectClean(t, "fn main() { 1 + 2.5 }")
	// A String concatenates with anything that has an unambiguous textual form.
	expectClean(t, "fn main() { \"count: \" + 5 }")
	expectClean(t, "fn main() { \"a\" + \"b\" }")
	expectClean(t, "fn main() { 1 < 2 }")
}

func TestAUnionFitsWhenAnyMemberDoes(t *testing.T) {
	// The value might be the member that works, so a union is accepted wherever one of its members would be.
	expectClean(t, "data Circle { r }\ndata Square { s }\nunion Shape {\nCircle\nSquare\n}\nfn take(x: Shape) { x }\nfn main() { take(Circle(1)) }")
	// A member the union does not have is still a certainty.
	expectError(t, "data Circle { r }\ndata Square { s }\ndata Other { o }\nunion Shape {\nCircle\nSquare\n}\nfn take(x: Shape) { x }\nfn main() { take(Other(1)) }",
		"wrong argument type", "declared Shape, got Other")
}

func TestArrayElementsAreCheckedWhenTheyAllAgree(t *testing.T) {
	expectError(t, "fn take(xs: [Int]) { xs }\nfn main() { take([\"a\", \"b\"]) }", "wrong argument type", "declared [Int], got [String]")
	// A mixed array says nothing about its elements, so it fits anywhere.
	expectClean(t, "fn take(xs: [Int]) { xs }\nfn main() { take([1, \"a\"]) }")
	expectClean(t, "fn take(xs: [Int]) { xs }\nfn main() { take([1, 2]) }")
	expectClean(t, "fn take(xs: [Int]) { xs }\nfn main() { take([]) }")
}

func TestABranchIsCheckedAsTheTypeItProves(t *testing.T) {
	// A `case is Rect` branch only ever runs for a Rect, so reading a field only Rect has is correct there, however the value was declared.
	expectClean(t, `union Shape {
data Circle { radius }
data Rect { width height }
}
fn main() {
	const shape = Circle(7)
	switch shape {
	case is Circle: shape.radius
	case is Rect: shape.width
	case _: 0
	}
}`)
	// The narrowing is what is checked against, so a field no member has is still reported.
	expectError(t, `union Shape {
data Circle { radius }
data Rect { width height }
}
fn main() {
	const shape = Circle(7)
	switch shape {
	case is Rect: shape.nope
	case _: 0
	}
}`, "unknown field", "Rect has no field nope")
}

func TestUnionMembersMayBeWrittenWithCommas(t *testing.T) {
	// The checker sees the same union however its members were separated.
	expectClean(t, "data Circle { r }\ndata Square { s }\nunion Shape { Circle, Square }\nfn take(x: Shape) { x }\nfn main() { take(Square(1)) }")
}

func TestReturnsAreCheckedAgainstWhatTheFunctionPromises(t *testing.T) {
	expectError(t, "fn wrong() -> Int {\n\treturn \"hello\"\n}", "wrong return type", "wrong returns Int, got String")
	expectError(t, "data P { a }\nfn wrong() -> P {\n\treturn 1\n}", "wrong return type", "wrong returns P, got Int")
}

func TestReturnsThatFitAreNotReported(t *testing.T) {
	expectClean(t, "fn right() -> Int {\n\treturn 1\n}")
	// No promise means nothing to break.
	expectClean(t, "fn unpromised() {\n\treturn \"hello\"\n}")
	// A bare return yields None, which says nothing certain.
	expectClean(t, "fn bare() -> Int {\n\treturn\n}")
	// Anything the checker cannot see through fits.
	expectClean(t, "fn passthrough(x) -> Int {\n\treturn x\n}")
	expectClean(t, "fn anything() -> Any {\n\treturn \"hello\"\n}")
	// A union is satisfied by any of its members.
	expectClean(t, "data Circle { r }\ndata Square { s }\nunion Shape { Circle, Square }\nfn make() -> Shape {\n\treturn Circle(1)\n}")
}

func TestANestedFunctionIsCheckedAgainstItsOwnPromise(t *testing.T) {
	// The inner promise applies inside it, and the outer one comes back afterwards.
	expectError(t, "fn outer() -> Int {\n\tconst inner = fn() -> String {\n\t\treturn 1\n\t}\n\treturn 2\n}", "wrong return type", "returns String, got Int")
	expectClean(t, "fn outer() -> Int {\n\tconst inner = fn() -> String {\n\t\treturn \"s\"\n\t}\n\treturn 2\n}")
}

func TestAReturnMustBeAMemberOfTheUnionPromised(t *testing.T) {
	// Some belongs to Option, so promising a Result and returning one is a mistake however tolerant the VM is.
	expectError(t, `data Ok { value }
data Err { reason }
union Result { Ok, Err }
data Some { value }
fn wrong(xyz) -> Result {
	return Some(xyz)
}`, "wrong return type", "wrong returns Result, got Some")

	// A genuine member is fine, and so is a union whose members overlap.
	expectClean(t, `data Ok { value }
data Err { reason }
union Result { Ok, Err }
fn right(xyz) -> Result {
	return Ok(xyz)
}`)
}

func TestFieldAccessIsCheckedOnAnythingWithADeclaredType(t *testing.T) {
	expectError(t, "data Person { name }\nfn read(p: Person) {\n\treturn p.nope\n}", "unknown field", "Person has no field nope")
	expectError(t, "data Person { name }\nfn read() {\n\tconst p: Person = Person(\"a\")\n\treturn p.nope\n}", "unknown field", "Person has no field nope")
	expectClean(t, "data Person { name }\nfn read(p: Person) {\n\treturn p.name\n}")
	// Without a declared type there is nothing to check against.
	expectClean(t, "data Person { name }\nfn read(p) {\n\treturn p.nope\n}")
}

func TestUndefinedNamesAreReported(t *testing.T) {
	// A typo is a name that stands for nothing, wherever it is written.
	expectError(t, "fn main() { nosuchthing() }", "undefined", "nosuchthing is not declared anywhere in scope")
	expectError(t, "fn main() {\n\tconst x = nosuchthing\n}", "undefined", "nosuchthing")
	expectError(t, "fn main() {\n\tconst x = 1 + nosuchthing\n}", "undefined", "nosuchthing")
	expectError(t, "data P { a }\nfn main() { P(nosuchthing) }", "undefined", "nosuchthing")
	expectError(t, "fn main() { nosuchthing.field }", "undefined", "nosuchthing")
}

func TestEveryUndefinedNameIsReportedAtOnce(t *testing.T) {
	// Reporting only the first would mean compiling once per typo.
	errs := analyzeSource(t, "fn main() {\n\tconst a = firstTypo\n\tconst b = secondTypo\n\tconst c = thirdTypo\n}")
	if len(errs) != 3 {
		t.Fatalf("expected 3 diagnostics, got %d: %v", len(errs), errs)
	}
	for _, want := range []string{"firstTypo", "secondTypo", "thirdTypo"} {
		found := false
		for _, err := range errs {
			if strings.Contains(err.Details, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("expected %q to be reported, got %v", want, errs)
		}
	}
}

func TestNamesThatExistAreNotReported(t *testing.T) {
	expectClean(t, "fn greet() { 1 }\nfn main() { greet() }")
	expectClean(t, "fn main() {\n\tconst x = 1\n\tconst y = x\n}")
	expectClean(t, "fn main(param) {\n\tconst y = param\n}")
	// A name declared later in the module is still declared.
	expectClean(t, "fn main() { later() }\nfn later() { 1 }")
}

func TestUnknownModuleMembersAreReported(t *testing.T) {
	// A qualified name is a name too: reading something a module does not offer finds nothing at runtime.
	errs := analyzeModuleWithImport(t, "fn main() { other.nosuchfn() }")
	if len(errs) == 0 {
		t.Fatal("expected a diagnostic for the unknown module member")
	}
	if errs[0].Summary != "unknown module member" {
		t.Fatalf("expected an unknown module member error, got %q: %s", errs[0].Summary, errs[0].Error())
	}
	if !strings.Contains(errs[0].Details, "nosuchfn") {
		t.Errorf("the error does not name the member: %s", errs[0].Error())
	}
}

func TestKnownModuleMembersAreNotReported(t *testing.T) {
	if errs := analyzeModuleWithImport(t, "fn main() { other.greet() }"); len(errs) > 0 {
		t.Errorf("expected no diagnostics, got %v", errs)
	}
}

// analyzeModuleWithImport analyzes a module that imports another one declaring a single public greet.
func analyzeModuleWithImport(t *testing.T, body string) []analyzer.AnalysisError {
	t.Helper()
	other := parseModule(t, "other", "mod other\nfn greet() { 1 }\nfn _hidden() { 2 }\ndata Thing { name }")
	module := parseModule(t, "test", "mod test\nimport other\n"+body)
	a := analyzer.New(mapResolver{modules: map[registry.LogicalURI]*ast.ContextModule{
		other.Name: other,
	}})
	a.Analyze(other, false)
	errs, _ := a.Analyze(module, false)
	return errs
}

func TestQualifiedTypeHintsAreResolvedThroughImports(t *testing.T) {
	// `import other` binds the module under a name but records none of its members on that symbol, so a hint qualified by it has to be looked up in the module itself.
	errs := analyzeModuleWithImport(t, "fn read(c: other.Thing) {\n\tc.nope\n}")
	if len(errs) == 0 {
		t.Fatal("expected a diagnostic for the unknown field")
	}
	if errs[0].Summary != "unknown field" {
		t.Fatalf("expected an unknown field error, got %q: %s", errs[0].Summary, errs[0].Error())
	}
	if !strings.Contains(errs[0].Details, "Thing has no field nope") {
		t.Errorf("unexpected details: %s", errs[0].Details)
	}
}

func TestQualifiedTypeHintsAcceptTheirOwnFields(t *testing.T) {
	if errs := analyzeModuleWithImport(t, "fn read(c: other.Thing) {\n\tc.name\n}"); len(errs) > 0 {
		t.Errorf("expected no diagnostics, got %v", errs)
	}
}

func TestCallsThroughAModuleAreChecked(t *testing.T) {
	// Reading a name off a module gives what the module bound to it, so a qualified call is checked like any other.
	errs := analyzeModuleWithImport(t, "fn main() { other.greet(1) }")
	if len(errs) == 0 {
		t.Fatal("expected a diagnostic for the wrong argument count")
	}
	if errs[0].Summary != "wrong number of arguments" {
		t.Fatalf("expected a wrong argument count error, got %q: %s", errs[0].Summary, errs[0].Error())
	}
}

func TestACompositeLiteralFitsItsBuiltinType(t *testing.T) {
	// Writing Array and writing [Int] describe the same values, and a parameter declared either way accepts a literal.
	expectClean(t, "extern type Array {}\nfn take(xs: Array) { xs }\nfn main() { take([1, 2]) }")
	expectClean(t, "extern type Dict {}\nfn take(d: Dict) { d }\nfn main() { take([\"a\": 1]) }")
	expectClean(t, "extern type Func {}\nfn take(f: Func) { f }\nfn main() { take(fn(x) { x }) }")
}

func TestADiagnosticOnADictLiteralHasAPosition(t *testing.T) {
	// ExprDict used to discard the token the parser gave it, leaving anything reported about a dict literal with nowhere to point.
	// expectError asserts a position on every diagnostic, so this fails outright if that returns.
	expectError(t, "fn takeInt(x: Int) { x }\nfn main() {\n\ttakeInt([:])\n}", "wrong argument type", "declared Int")
}

func TestADeclaredHintIsCheckedAgainstItsValue(t *testing.T) {
	expectError(t, "fn main() {\n\tconst x: Int = \"hello\"\n}", "wrong type", "x is declared Int, got String")
	expectError(t, "fn main() {\n\tvar x: Int = \"hello\"\n}", "wrong type", "x is declared Int, got String")
	expectClean(t, "fn main() {\n\tconst x: Int = 1\n}")
	// No hint, or a value the checker cannot see through, says nothing.
	expectClean(t, "fn main() {\n\tconst x = \"hello\"\n}")
	expectClean(t, "fn main(y) {\n\tconst x: Int = y\n}")
}

func TestAssignmentIsCheckedAgainstTheDeclaredType(t *testing.T) {
	expectError(t, "fn main() {\n\tvar x: Int = 1\n\tx = \"hello\"\n}", "wrong type", "x is declared Int, got String")
	expectClean(t, "fn main() {\n\tvar x: Int = 1\n\tx = 2\n}")
	// Without a declared type there is nothing to break.
	expectClean(t, "fn main() {\n\tvar x = 1\n\tx = \"hello\"\n}")
}

func TestUnaryOperatorsAreChecked(t *testing.T) {
	expectError(t, "fn main() { -\"hello\" }", "unsupported operator", "- is not defined for String")
	expectError(t, "fn main() { !5 }", "unsupported operator", "! is not defined for Int")
	expectClean(t, "fn main() { -5 }")
	expectClean(t, "fn main() { -1.5 }")
	expectClean(t, "fn main() { !true }")
	expectClean(t, "fn main(x) { -x }")
}

func TestIndexingSomethingThatCannotBeIndexedIsReported(t *testing.T) {
	expectError(t, "fn main() {\n\tconst x = 5\n\tx[0]\n}", "not indexable", "Int cannot be indexed")
	expectClean(t, "fn main() {\n\tconst xs = [1, 2]\n\txs[0]\n}")
	expectClean(t, "fn main() {\n\tconst d = [\"a\": 1]\n\td[\"a\"]\n}")
	expectClean(t, "fn main(x) { x[0] }")
}

// analyzeRaw analyzes a module exactly as written, for tests that need to declare their own builtins.
func analyzeRaw(t *testing.T, source string) []analyzer.AnalysisError {
	t.Helper()
	module := parseModule(t, "test", source)
	errs, _ := analyzer.New(nil).Analyze(module, false)
	return errs
}

func TestIteratingSomethingThatIsNotIterableIsReported(t *testing.T) {
	// The VM requires @Iterable at runtime, so a type that does not carry it can never be iterated.
	errs := analyzeRaw(t, "mod test\nattr Iterable {}\nattr Numeric {}\n@Numeric()\nextern type Int {}\nfn main() {\n\tfor x <- 5 {\n\t\tx\n\t}\n}")
	if len(errs) == 0 {
		t.Fatal("expected a diagnostic for iterating an Int")
	}
	if errs[0].Summary != "not iterable" {
		t.Fatalf("expected a not iterable error, got %q: %s", errs[0].Summary, errs[0].Error())
	}
}

func TestIteratingSomethingIterableIsNotReported(t *testing.T) {
	if errs := analyzeRaw(t, "mod test\nattr Iterable {}\n@Iterable()\nextern type Int {}\nfn main() {\n\tfor x <- 5 {\n\t\tx\n\t}\n}"); len(errs) > 0 {
		t.Errorf("expected no diagnostics, got %v", errs)
	}
	// An array is iterable by definition, and an unknown value says nothing.
	if errs := analyzeRaw(t, "mod test\nattr Iterable {}\nfn main() {\n\tfor x <- [1, 2] {\n\t\tx\n\t}\n}"); len(errs) > 0 {
		t.Errorf("expected no diagnostics for an array, got %v", errs)
	}
	if errs := analyzeRaw(t, "mod test\nattr Iterable {}\nfn main(xs) {\n\tfor x <- xs {\n\t\tx\n\t}\n}"); len(errs) > 0 {
		t.Errorf("expected no diagnostics for an unknown value, got %v", errs)
	}
}

func TestAVarNarrowsWhenEveryAssignmentAgrees(t *testing.T) {
	// Its declaration and every assignment to it say Int, so it is one.
	expectError(t, "fn main() {\n\tvar x = 5\n\tx()\n}", "not callable", "Int cannot be called")
	expectError(t, "fn takeString(s: String) { s }\nfn main() {\n\tvar x = 5\n\tx = 7\n\ttakeString(x)\n}", "wrong argument type", "declared String, got Int")
}

func TestAVarStaysUnknownWhenAssignmentsDisagree(t *testing.T) {
	// The checker does not follow the order statements run in, so it can only speak when the answer is the same whichever ran.
	expectClean(t, "fn takeString(s: String) { s }\nfn main() {\n\tvar x = 5\n\tx = \"a string\"\n\ttakeString(x)\n}")
	expectClean(t, "fn main() {\n\tvar x = 5\n\tx = \"a string\"\n\tx()\n}")
	// An assignment further down still counts, even though it is read before then.
	expectClean(t, "fn takeInt(n: Int) { n }\nfn main() {\n\ttakeIntLater()\n}\nfn takeIntLater() {\n\tvar x = 5\n\ttakeInt(x)\n\tx = \"a string\"\n}")
}

func TestAnIfExpressionYieldsWhatItsBranchesAgreeOn(t *testing.T) {
	expectError(t, "fn takeInt(n: Int) { n }\nfn main(c) {\n\tconst v = if c { \"a\" } else { \"b\" }\n\ttakeInt(v)\n}", "wrong argument type", "declared Int, got String")
	expectClean(t, "fn takeInt(n: Int) { n }\nfn main(c) {\n\tconst v = if c { 1 } else { 2 }\n\ttakeInt(v)\n}")
	// Branches that disagree say nothing.
	expectClean(t, "fn takeInt(n: Int) { n }\nfn main(c) {\n\tconst v = if c { \"a\" } else { 2 }\n\ttakeInt(v)\n}")
}

func TestASwitchExpressionYieldsWhatItsCasesAgreeOn(t *testing.T) {
	expectError(t, "fn takeInt(n: Int) { n }\nfn main(c) {\n\tconst v = switch c {\n\tcase is String: \"a\"\n\tcase _: \"b\"\n\t}\n\ttakeInt(v)\n}", "wrong argument type", "declared Int, got String")
	// Without a default a value may match nothing, so the result is not knowable.
	expectClean(t, "fn takeInt(n: Int) { n }\nfn main(c) {\n\tconst v = switch c {\n\tcase is String: \"a\"\n\t}\n\ttakeInt(v)\n}")
}

func TestIndexingYieldsTheElementType(t *testing.T) {
	// An array literal whose elements agree describes them, so reading one back is known.
	expectError(t, "fn takeString(s: String) { s }\nfn main() {\n\tconst xs = [1, 2]\n\ttakeString(xs[0])\n}", "wrong argument type", "declared String, got Int")
	expectClean(t, "fn takeInt(n: Int) { n }\nfn main() {\n\tconst xs = [1, 2]\n\ttakeInt(xs[0])\n}")
	// A mixed array describes nothing, so reading one says nothing.
	expectClean(t, "fn takeInt(n: Int) { n }\nfn main() {\n\tconst xs = [1, \"a\"]\n\ttakeInt(xs[0])\n}")
}

func TestIndexingADictYieldsItsValueType(t *testing.T) {
	expectError(t, "fn takeInt(n: Int) { n }\nfn main(d: [String: String]) {\n\ttakeInt(d[\"k\"])\n}", "wrong argument type", "declared Int, got String")
	expectClean(t, "fn takeInt(n: Int) { n }\nfn main(d: [String: Int]) {\n\ttakeInt(d[\"k\"])\n}")
	// A dict literal whose values agree describes them too.
	expectError(t, "fn takeInt(n: Int) { n }\nfn main() {\n\tconst d = [\"a\": \"x\", \"b\": \"y\"]\n\ttakeInt(d[\"a\"])\n}", "wrong argument type", "got String")
	expectClean(t, "fn takeInt(n: Int) { n }\nfn main() {\n\tconst d = [\"a\": 1, \"b\": \"mixed\"]\n\ttakeInt(d[\"a\"])\n}")
}

func TestIndexingTextYieldsAByte(t *testing.T) {
	// The VM gives the byte at that position rather than the character, for both a String and a Binary.
	expectError(t, "extern type Byte {}\nfn takeInt(n: Int) { n }\nfn main(s: String) {\n\ttakeInt(s[0])\n}", "wrong argument type", "declared Int, got Byte")
	expectClean(t, "extern type Byte {}\nfn takeByte(b: Byte) { b }\nfn main(s: String) {\n\ttakeByte(s[0])\n}")
}
