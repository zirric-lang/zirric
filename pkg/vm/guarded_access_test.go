package vm_test

import "testing"

// preludeShapes declares what `?.`, `??`, `!.` and `!!` are defined against. The VM tests run without the real prelude, so the names those operators resolve — Option, Result, @AnyOption and @AnyResult — are declared here instead, mirroring prelude/option.zirr and prelude/result.zirr.
const preludeShapes = `
attr AnyOption {
	toOption(self) -> Option
}

@AnyOption(fn(o) { o })
union Option {
	@AnyOption(fn(o) { o })
	data Some {
		value
	}
	@AnyOption(fn(o) { o })
	data None
}

attr AnyResult {
	toResult(self) -> Result
}

@AnyResult(fn(r) { r })
union Result {
	@AnyResult(fn(r) { r })
	data Ok {
		value
	}
	@AnyResult(fn(r) { r })
	data Err {
		reason
	}
}
`

func TestOptionalChaining(t *testing.T) {
	const types = preludeShapes + `
	data Address {
		city
	}
	data User {
		name
		address
	}
	fn cityOf(user) {
		const city = user?.address?.city
		switch city {
		case is None:
			"absent"
		case _:
			city.value
		}
	}
	`

	tests := []vmTestCase{
		{
			label:    "reads through every link without an unwrapping step",
			input:    types + `cityOf(Some(User("Ada", Some(Address("London")))))`,
			expected: "London",
		},
		{
			label:    "the first link short-circuits the whole chain",
			input:    types + `cityOf(None())`,
			expected: "absent",
		},
		{
			label:    "a later link short-circuits the whole chain too",
			input:    types + `cityOf(Some(User("Ada", None())))`,
			expected: "absent",
		},
		{
			label: "the result is wrapped in Some",
			input: preludeShapes + `
			data Person {
				name
			}
			const wrapped = Some(Person("Ada"))?.name
			switch wrapped {
			case is Some:
				wrapped.value
			case _:
				"not wrapped"
			}
			`,
			expected: "Ada",
		},
		{
			label: "a field that is already an Option is not wrapped twice",
			input: preludeShapes + `
			data Person {
				nickname
			}
			const nick = Some(Person(Some("Addie")))?.nickname
			nick.value
			`,
			expected: "Addie",
		},
		{
			label: "an index access carries the short-circuit with it",
			input: preludeShapes + `
			data Box {
				items
			}
			fn secondOf(box) {
				const item = box?.items[1]
				switch item {
				case is None:
					0
				case _:
					item.value
				}
			}
			secondOf(Some(Box([10, 20, 30]))) + secondOf(None())
			`,
			expected: 20,
		},
		{
			// A top-level const is initialized lazily, so the read has to happen inside a function that is actually called for the counter to mean anything.
			label: "the index expression is not evaluated once the chain is absent",
			input: preludeShapes + `
			data Box {
				items
			}
			var evaluated = 0
			fn indexOnce() {
				evaluated = evaluated + 1
				return 0
			}
			fn read(box) {
				return box?.items[indexOnce()]
			}
			read(None())
			evaluated
			`,
			expected: 0,
		},
		{
			label: "and is evaluated once the chain is not",
			input: preludeShapes + `
			data Box {
				items
			}
			var evaluated = 0
			fn indexOnce() {
				evaluated = evaluated + 1
				return 0
			}
			fn read(box) {
				return box?.items[indexOnce()]
			}
			read(Some(Box([10])))
			evaluated
			`,
			expected: 1,
		},
		{
			// A union of one's own is read through its own @AnyOption, which is the whole point of going through asOption.
			label: "a shape of one's own decides how it is read",
			input: preludeShapes + `
			@AnyOption(fn(j) { Some(j.held) })
			data Just {
				held
			}
			@AnyOption(fn(n) { None() })
			data Nothing
			data Person {
				name
			}
			fn nameOf(maybe) {
				const name = maybe?.name
				switch name {
				case is None:
					"absent"
				case _:
					name.value
				}
			}
			nameOf(Just(Person("Ada"))) + "/" + nameOf(Nothing())
			`,
			expected: "Ada/absent",
		},
	}

	runVmTests(t, tests)
}

func TestResultUnwrapping(t *testing.T) {
	const types = preludeShapes + `
	data Config {
		host
	}
	fn hostOf(result) {
		const host = result!.host
		return "host is " + host
	}
	`

	tests := []vmTestCase{
		{
			label:    "reads the field of a success without an unwrapping step",
			input:    types + `hostOf(Ok(Config("example.com")))`,
			expected: "host is example.com",
		},
		{
			label: "returns the error from the enclosing function instead",
			input: types + `
			const propagated = hostOf(Err("missing file"))
			switch propagated {
			case is Err:
				propagated.reason
			case _:
				"not propagated"
			}
			`,
			expected: "missing file",
		},
		{
			label: "the rest of the function does not run",
			input: types + `
			var reached = 0
			fn readTwice(result) {
				const first = result!.host
				reached = reached + 1
				return first
			}
			const ignored = readTwice(Err("boom"))
			reached
			`,
			expected: 0,
		},
		{
			label: "a shape of one's own decides how it is read",
			input: preludeShapes + `
			data Config {
				host
			}
			@AnyResult(fn(l) { Ok(l.held) })
			data Loaded {
				held
			}
			@AnyResult(fn(f) { Err(f.why) })
			data Failed {
				why
			}
			fn hostOf(outcome) {
				return Ok(outcome!.host)
			}
			switch hostOf(Failed("nope")) {
			case is Err:
				hostOf(Loaded(Config("custom"))).value + "/" + hostOf(Failed("nope")).reason
			case _:
				"not propagated"
			}
			`,
			expected: "custom/nope",
		},
	}

	runVmTests(t, tests)
}

// TestResultUnwrappingEscapesIterable pins the awkward case: a `for <-` body runs as a resumed chunk of its frame, driven by the iterable, so returning out of it has to unwind past the iterable rather than just ending the chunk.
func TestResultUnwrappingEscapesIterable(t *testing.T) {
	const types = preludeShapes + iterablePreamble + `
	@Iterable(_listIterate)
	data List {
		values
		count
	}

	fn _listIterate(list, yield) {
		var i = 0
		for {
			if i >= list.count {
				return
			}
			if !yield(list.values[i]) {
				break
			}
			i = i + 1
		}
	}

	data Holder {
		amount
	}

	fn sum(list) {
		var total = 0
		for item <- list {
			total = total + item!.amount
		}
		return Ok(total)
	}
	`

	tests := []vmTestCase{
		{
			label:    "adds every success",
			input:    types + `sum(List([Ok(Holder(1)), Ok(Holder(2)), Ok(Holder(3))], 3)).value`,
			expected: 6,
		},
		{
			label: "an error in the body returns from the function around the loop",
			input: types + `
			const propagated = sum(List([Ok(Holder(1)), Err("bad"), Ok(Holder(3))], 3))
			switch propagated {
			case is Err:
				propagated.reason
			case _:
				"not propagated"
			}
			`,
			expected: "bad",
		},
	}

	runVmTests(t, tests)
}

func TestFallbackOperators(t *testing.T) {
	const types = preludeShapes + `
	var evaluated = 0
	fn _fallback() {
		evaluated = evaluated + 1
		return "fallback"
	}
	`

	tests := []vmTestCase{
		{
			label:    "?? takes the value out of a present option",
			input:    types + `Some("Ada") ?? "Unknown"`,
			expected: "Ada",
		},
		{
			label:    "?? takes the fallback when absent",
			input:    types + `None() ?? "Unknown"`,
			expected: "Unknown",
		},
		{
			label:    "!! takes the value out of a success",
			input:    types + `Ok("example.com") !! "default"`,
			expected: "example.com",
		},
		{
			label:    "!! takes the fallback on an error",
			input:    types + `Err("missing file") !! "default"`,
			expected: "default",
		},
		{
			label: "?? reads a shape of one's own through its @AnyOption",
			input: preludeShapes + `
			@AnyOption(fn(j) { Some(j.held) })
			data Just {
				held
			}
			@AnyOption(fn(n) { None() })
			data Nothing
			(Just("Ada") ?? "Unknown") + "/" + (Nothing() ?? "Unknown")
			`,
			expected: "Ada/Unknown",
		},
		{
			// A top-level const is initialized lazily, so the fallback has to sit inside a function that is actually called for the counter to mean anything.
			label:    "the fallback is not evaluated when it is not needed",
			input:    types + "fn run(o) { return o ?? _fallback() }\nrun(Some(1))\nevaluated",
			expected: 0,
		},
		{
			label:    "the fallback is evaluated when it is",
			input:    types + "fn run(o) { return o ?? _fallback() }\nrun(None())\nevaluated",
			expected: 1,
		},
		{
			label:    "?? is right associative, so the last fallback wins",
			input:    types + `None() ?? None() ?? "last"`,
			expected: "last",
		},
		{
			label:    "?? binds looser than arithmetic and tighter than comparison",
			input:    types + `(None() ?? 1 + 1) == 2`,
			expected: true,
		},
		{
			label: "a chain reads as one expression with its fallback",
			input: preludeShapes + `
			data User {
				name
			}
			Some(User("Ada"))?.name ?? "Unknown"
			`,
			expected: "Ada",
		},
		{
			label: "and falls back when the chain is absent",
			input: preludeShapes + `
			data User {
				name
			}
			None()?.name ?? "Unknown"
			`,
			expected: "Unknown",
		},
	}

	runVmTests(t, tests)
}

// TestGuardedAccessNeedsTheAttribute pins that a value standing for neither an Option nor a Result is reported rather than quietly passed through, the way `for <-` reports a value with no @Iterable.
// The message carries no position here because a source file's top-level statements are compiled into the main frame, which this harness leaves without debug entries; in a module they run inside __init__ and are located normally.
func TestGuardedAccessNeedsTheAttribute(t *testing.T) {
	tests := []vmTestCase{
		{
			label: "?? on a value that is no option at all",
			input: preludeShapes + `"plain" ?? "fallback"`,
			err:   `?. and ?? require a value with @AnyOption, got String "plain"`,
		},
		{
			label: "!! on a value that is no result at all",
			input: preludeShapes + `"plain" !! "fallback"`,
			err:   `!. and !! require a value with @AnyResult, got String "plain"`,
		},
		{
			// Named where the conversion went wrong, rather than further along where the message would be about a missing field.
			label: "a conversion that answers with something else",
			input: preludeShapes + `
			@AnyOption(fn(b) { 42 })
			data Broken {
				name
			}
			Broken("x")?.name
			`,
			err: `the @AnyOption of Broken answered with Int 42, which is no Option`,
		},
	}

	runVmTests(t, tests)
}

func TestTypeShorthands(t *testing.T) {
	const types = preludeShapes + `
	data User {
		name
	}
	fn findUser(present) -> User? {
		if present {
			return Some(User("Ada"))
		} else {
			return None()
		}
	}
	fn readConfig(ok) -> String! {
		if ok {
			return Ok("example.com")
		} else {
			return Err("missing")
		}
	}
	`

	tests := []vmTestCase{
		{
			label:    "T? accepts a present value",
			input:    types + `findUser(true) is User?`,
			expected: true,
		},
		{
			label:    "T? accepts an absent one",
			input:    types + `findUser(false) is User?`,
			expected: true,
		},
		{
			label:    "T? rejects a value that is no Option at all",
			input:    types + `7 is User?`,
			expected: false,
		},
		{
			label:    "T! accepts a success and an error alike",
			input:    types + `readConfig(true) is String! && readConfig(false) is String!`,
			expected: true,
		},
		{
			label: "a type shorthand matches in a switch too",
			input: types + `
			switch findUser(false) {
			case is User?:
				"an option"
			case _:
				"something else"
			}
			`,
			expected: "an option",
		},
	}

	runVmTests(t, tests)
}
