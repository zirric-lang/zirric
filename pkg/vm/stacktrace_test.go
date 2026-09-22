package vm_test

import (
	"errors"
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/compiler"
	"code.knabel.dev/zirric-lang/zirric/pkg/vm"
)

// runFailing compiles and runs a program expected to fail, returning the error.
func runFailing(t *testing.T, input string) error {
	t.Helper()
	module, program := prepareSourceFileParsing(t, input)
	comp := compiler.New(newTestModuleResolver(module))
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compile: %s", err)
	}
	err := vm.New(comp.Bytecode()).Run()
	if err == nil {
		t.Fatal("expected the program to fail")
	}
	return err
}

func TestARuntimeFailureCarriesTheFramesThatWereLive(t *testing.T) {
	err := runFailing(t, `
fn inner(d) {
	return d["missing"].nope
}
fn middle(d) {
	return inner(d)
}
fn outer() {
	return middle(["a": 1])
}
outer()
`)

	var runtimeErr *vm.RuntimeError
	if !errors.As(err, &runtimeErr) {
		t.Fatalf("expected a *vm.RuntimeError, got %T: %s", err, err)
	}

	trace := runtimeErr.StackTrace()
	// Innermost first, so the frame that failed is the one read first.
	for _, want := range []string{"inner", "middle", "outer"} {
		if !strings.Contains(trace, want) {
			t.Errorf("expected %q in the trace:\n%s", want, trace)
		}
	}
	if strings.Index(trace, "inner") > strings.Index(trace, "outer") {
		t.Errorf("expected the innermost frame first:\n%s", trace)
	}
}

func TestARuntimeFailureKnowsWhereItHappened(t *testing.T) {
	// The value is only known at runtime, so this is a failure static checks cannot see.
	err := runFailing(t, "fn boom(x) {\n\treturn x[0]\n}\nboom(5)")

	var runtimeErr *vm.RuntimeError
	if !errors.As(err, &runtimeErr) {
		t.Fatalf("expected a *vm.RuntimeError, got %T", err)
	}
	source := runtimeErr.Position()
	if source == nil {
		t.Fatal("expected a position")
	}
	// The failure is the index on line 2, not wherever the call chain started.
	if source.Line != 2 {
		t.Errorf("expected line 2, got %d", source.Line)
	}
	if !strings.Contains(runtimeErr.Error(), ":2:") {
		t.Errorf("the message does not carry the position: %s", runtimeErr.Error())
	}
}

func TestTheInnermostTraceIsKeptAsItUnwinds(t *testing.T) {
	// Re-tracing at every level it unwinds through would replace where it failed with where it was noticed.
	err := runFailing(t, "fn a(x) {\n\treturn x[0]\n}\nfn b(x) {\n\treturn a(x)\n}\nb(5)")

	var runtimeErr *vm.RuntimeError
	if !errors.As(err, &runtimeErr) {
		t.Fatalf("expected a *vm.RuntimeError, got %T", err)
	}
	if source := runtimeErr.Position(); source == nil || source.Line != 2 {
		t.Errorf("expected the failure at line 2, got %v", source)
	}
}

func TestAFailureNamesTheValueItActuallyGot(t *testing.T) {
	// A failed type assertion leaves the zero value of the type that was wanted, so a message written from it named a value that was never there — `!5` reported a Bool of "false".
	for _, tt := range []struct {
		name  string
		input string
		want  string
	}{
		{"invert", "fn bad(x) {\n\treturn !x\n}\nbad(5)", "got Int 5"},
		{"negate", "fn bad(x) {\n\treturn -x\n}\nbad(\"hello\")", `got String "hello"`},
		{"modulo", "fn bad(x, y) {\n\treturn x % y\n}\nbad(\"a\", 2)", `got String "a"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := runFailing(t, tt.input)
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected %q in the message, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestAFailureNamesTypesTheWayTheLanguageDoes(t *testing.T) {
	// A reader knows what an Int is; runtime.Int is an implementation detail they never wrote.
	err := runFailing(t, "fn bad(x) {\n\treturn x[0]\n}\nbad(5)")
	if strings.Contains(err.Error(), "runtime.") {
		t.Errorf("the message exposes a Go type: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "Int") {
		t.Errorf("expected the message to name the type, got %q", err.Error())
	}
}

func TestAnOutOfBoundsIndexSaysWhatTheBoundWas(t *testing.T) {
	err := runFailing(t, "fn bad(xs) {\n\treturn xs[9]\n}\nbad([1, 2])")
	if !strings.Contains(err.Error(), "length is 2") {
		t.Errorf("expected the length in the message, got %q", err.Error())
	}
}

func TestATraceCarriesNoPositionlessFrames(t *testing.T) {
	// A module's own token names a file but no line, and a frame built from it used to appear as a name with a file where a position belongs.
	err := runFailing(t, "fn bad(x) {\n\treturn x[0]\n}\nbad(5)")
	var runtimeErr *vm.RuntimeError
	if !errors.As(err, &runtimeErr) {
		t.Fatalf("expected a *vm.RuntimeError, got %T", err)
	}
	for _, frame := range runtimeErr.Frames {
		if frame.Source != nil && frame.Source.Line <= 0 {
			t.Errorf("a frame carries a source with no line: %+v", frame)
		}
	}
	if strings.Contains(runtimeErr.Error(), ".zirr: ") {
		t.Errorf("the message has a file where a position belongs: %s", runtimeErr.Error())
	}
}

func TestAnOperatorIsNamedTheWayItIsWritten(t *testing.T) {
	// A message saying "operator 16" asks a reader to know the bytecode; one saying "operator -" asks them to look at their own line.
	err := runFailing(t, "fn combine(a, b) {\n\treturn a - b\n}\ncombine(\"a\", 5)")
	if !strings.Contains(err.Error(), "operator - is not defined for String and Int") {
		t.Errorf("unexpected message: %s", err.Error())
	}
	if strings.Contains(err.Error(), "runtime.") {
		t.Errorf("the message exposes an implementation detail: %s", err.Error())
	}
}

func TestReadingAMissingMemberNamesTheType(t *testing.T) {
	err := runFailing(t, "fn get(o) {\n\treturn o.nope\n}\nget(5)")
	if !strings.Contains(err.Error(), `Int has no member "nope"`) {
		t.Errorf("unexpected message: %s", err.Error())
	}
}

func TestNoUserFacingMessageNamesAGoType(t *testing.T) {
	for _, input := range []string{
		"fn f(a, b) {\n\treturn a - b\n}\nf(\"a\", 5)",
		"fn f(o) {\n\treturn o.nope\n}\nf(5)",
		"fn f(x) {\n\treturn x[0]\n}\nf(5)",
		"fn f(x) {\n\treturn !x\n}\nf(5)",
	} {
		err := runFailing(t, input)
		if strings.Contains(err.Error(), "runtime.") {
			t.Errorf("%q exposes a Go type: %s", input, err.Error())
		}
	}
}

// TestAFailureAfterACallReturnsBlamesTheCaller is a regression test: the frame of a call that has already returned stays in the frames array, and the trace used to start on it — so a failure in the caller, on the line right after the call, was reported inside the function that had just finished.
func TestAFailureAfterACallReturnsBlamesTheCaller(t *testing.T) {
	err := runFailing(t, `
data Person {
	name
}
fn identity(x) {
	return x
}
fn read(p) {
	return identity(p).missing
}
read(Person("a"))
`)

	var runtimeErr *vm.RuntimeError
	if !errors.As(err, &runtimeErr) {
		t.Fatalf("expected a *vm.RuntimeError, got %T: %s", err, err)
	}

	source := runtimeErr.Position()
	if source == nil {
		t.Fatal("expected a position")
	}
	// Line 9 is the field read in read(), not line 6 where identity returns.
	if source.Line != 9 {
		t.Errorf("expected the failure on line 9, where the field is read, got line %d", source.Line)
	}
	trace := runtimeErr.StackTrace()
	if strings.Contains(trace, "identity") {
		t.Errorf("identity had already returned, so it must not be in the trace:\n%s", trace)
	}
	if !strings.Contains(trace, "read") {
		t.Errorf("expected the frame that actually failed in the trace:\n%s", trace)
	}
}
