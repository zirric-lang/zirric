package orchestra_test

import (
	"context"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
)

// runCoProgram runs source against the real standard library, which is how -race reaches the scheduler.
func runCoProgram(t *testing.T, source string) error {
	t.Helper()
	projectFS := memfs.New()
	writeFile(t, projectFS, "main.zirr", source)
	orch := newTestOrchestra(t, projectFS, "project")
	return orch.RunFile(context.Background(), "main.zirr")
}

func TestRoutinesShareTheirProgramsValues(t *testing.T) {
	err := runCoProgram(t, `mod main

import co

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

// Every routine writes the same var and array; one runner at a time is what makes that safe without a lock.
fn run() {
	var total = 0
	const seen = co.scope(fn(s) {
		const counted = co.channel(0)
		const routines = for n <- [1, 2, 3, 4, 5, 6, 7, 8] {
			co.spawn(s, fn() {
				total = total + n
				co.send(counted, n)
				return Ok(n)
			})
		}
		var received = []
		for r <- routines {
			received = append(received, co.receive(counted))
		}
		return received
	})
	return [total, len(seen)]
}

const outcome = run()
assertEqual(outcome[0], 36, "FAIL: every routine should have added its own number")
assertEqual(outcome[1], 8, "FAIL: every routine should have sent exactly once")
`)
	if err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func TestCancellingAScopeStopsItsRoutines(t *testing.T) {
	err := runCoProgram(t, `mod main

import co

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

const outcomes = co.scope(fn(s) {
	const idle = co.channel(0)
	const routines = for n <- [1, 2, 3] {
		co.spawn(s, fn() {
			co.receive(idle)
			return Ok(n)
		})
	}
	co.cancel(s)
	return for r <- routines { co.wait(r) }
})

assertEqual(outcomes[0], Err(co.Cancelled()), "FAIL: a stopped routine reports Cancelled")
assertEqual(outcomes[2], Err(co.Cancelled()), "FAIL: every routine of the scope is stopped")
`)
	if err != nil {
		t.Fatalf("run file: %v", err)
	}
}

func TestAFailureInARoutineEndsTheProgram(t *testing.T) {
	err := runCoProgram(t, `mod main

import co

co.scope(fn(s) {
	co.spawn(s, fn() {
		panic("the routine gave up")
	})
	const idle = co.channel(0)
	return co.receive(idle)
})
`)
	if err == nil {
		t.Fatal("expected the panic inside the routine to end the program")
	}
	if !strings.Contains(err.Error(), "the routine gave up") {
		t.Fatalf("expected the routine's own message, got %v", err)
	}
}

func TestNestedScopesEachWaitForTheirOwn(t *testing.T) {
	err := runCoProgram(t, `mod main

import co

fn assertEqual(actual, expected, label) {
	if actual != expected {
		panic(label)
	}
}

var order = []
co.scope(fn(outer) {
	co.spawn(outer, fn() {
		co.scope(fn(inner) {
			co.spawn(inner, fn() {
				order = append(order, "inner")
				return Ok(void)
			})
			return void
		})
		order = append(order, "after inner scope")
		return Ok(void)
	})
	return void
})
order = append(order, "after outer scope")

assertEqual(order[0], "inner", "FAIL: the inner routine runs first")
assertEqual(order[1], "after inner scope", "FAIL: the inner scope waits for its routine")
assertEqual(order[2], "after outer scope", "FAIL: the outer scope waits for its routine")
`)
	if err != nil {
		t.Fatalf("run file: %v", err)
	}
}
