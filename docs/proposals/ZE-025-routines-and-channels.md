---
title: "ZE-025 - Routines and Channels"
description: "Lightweight routines, the scopes that own them, and channels between them, as a standard library module."
---

# Routines and Channels

::: callout tip Implemented
This proposal has been accepted and implemented. You can use this feature since Zirric [v0.2.0](/changelog/v0.2.0).
:::

## Introduction

Zirric programs used to run one thing at a time, and a blocking call blocked everything. A TUI could not read a key and redraw a spinner at once; a CLI could not wait on a file, a timer and the user together.

This proposal adds a standard library module, `co`, with lightweight routines, scopes that own them, and channels between them. Channels carry `@Iterable`, so a `for` loop receives from them. There is no new syntax, no new keyword and no new operator.

Two rules govern the design:

- **One routine runs at a time, and it only gives way at a switch point.** Routines are concurrent, not parallel. That keeps shared `var`s and arrays free of data races without locks.
- **Every routine belongs to a scope, and a scope does not end before its routines.** Nothing runs in the background that the code does not show.

## Motivation

The target domain of Zirric is CLI and TUI programs. Almost every interactive one waits on more than one thing:

- keys from standard input, and a timer that redraws;
- a slow job, and a user who presses `q`;
- several files or streams, handled as they become ready.

Without routines, the first blocking read wins. The best a program can do is redraw after each key:

```zirric
mod code.example.spinner

import fmt
import io

fn run(stdin: @io.Reader, out: @io.Writer) {
	var frame = 0
	for {
		const key = io.read(stdin, 1)        // blocks the whole program
		if len(key) == 0 {
			return
		}
		frame = frame + 1
		fmt.fprintln(spinner(frame), out)    // only moves when a key arrives
	}
}
```

With `co`, the key reader runs in its own routine, and the main loop waits on keys and ticks together:

```zirric
mod code.example.spinner

import co
import fmt
import io
import time

fn readKeys(stdin: @io.Reader, keys: co.Channel) {
	for {
		const key = io.read(stdin, 1)
		if len(key) == 0 {
			break
		}
		co.send(keys, key)
	}
}

fn run(stdin: @io.Reader, out: @io.Writer, timer: co.Timer) -> Result {
	return co.scope(fn(s) {
		const keys = co.produce(s, fn(ch) { readKeys(stdin, ch) })
		var frame = 0
		var tick = co.after(timer, time.milliseconds(100))
		for {
			const selected = co.select([keys, tick])
			if selected is co.Closed {
				return Ok(void)          // standard input ended
			}
			if selected.channel == tick {
				frame = frame + 1
				fmt.fprintln(spinner(frame), out)
				tick = co.after(timer, time.milliseconds(100))
			} else if isQuit(selected.value) {
				co.cancel(s)             // the key reader is still waiting on stdin
				return Ok(void)
			}
		}
	})
}
```

Every routine `run` starts is visible inside its `scope` block, and none survives the return.

Producer and consumer pipelines need nothing new at all. A channel is iterable, so the existing loop does the receiving:

```zirric
co.scope(fn(s) {
	for line <- co.produce(s, fn(ch) { readLines(source, ch) }) {
		fmt.fprintln(line, out)
	}
	return Ok(void)
})
```

The same holds for `for` expressions and the lazy functions in [`fun`](https://zirric.knabel.dev/stdlib/fun/): a `for` expression over a channel collects until it is closed, and `fun.take(ch, 10)` stops after ten.

## Proposed Solution

A new module, `co`:

| Declaration                                        | Meaning                                                                              |
| -------------------------------------------------- | ------------------------------------------------------------------------------------ |
| `extern type Scope`                                | Owns routines. Only `scope` creates one.                                             |
| `scope(body: fn(Scope) -> Any) -> Any`             | Runs `body`, waits for every routine spawned into the scope, returns `body`'s value. |
| `extern type Routine`                              | A running function, returned by `spawn`.                                             |
| `spawn(s: Scope, body: fn() -> Result) -> Routine` | Starts `body` in a new routine owned by `s`. Returns at once.                        |
| `wait(routine: Routine) -> Result`                 | Blocks until the routine ends. Its `Result`, or `Err(Cancelled())`.                  |
| `cancel(s: Scope)`                                 | Stops every routine spawned into `s`.                                                |
| `extern type Channel`                              | A queue between routines. Carries `@Iterable`.                                       |
| `channel(capacity: Int) -> Channel`                | Holds up to `capacity` values. `0`: every send waits for a receiver.                 |
| `send(ch: Channel, value: Any)`                    | Blocks until there is room, then puts `value` in.                                    |
| `receive(ch: Channel) -> Option`                   | Blocks until a value arrives: `Some`. After `close`, once drained: `None()`.         |
| `close(ch: Channel)`                               | No more values will be sent.                                                         |
| `select(channels: [Channel]) -> Selected`          | Blocks until one of the channels can be received from.                               |
| `union Selected { Received, Closed }`              | `data Received { channel, value }`, `data Closed { channel }`.                       |
| `data Timer { after: fn(Duration) -> Channel }`    | The capability to wait for time to pass.                                             |
| `after(timer: Timer, d: Duration) -> Channel`      | Receives one value after `d`, then closes.                                           |
| `sleep(timer: Timer, d: Duration)`                 | Blocks the calling routine for `d`.                                                  |
| `immediateTimer() -> Timer`                        | Every `after` fires at once. For tests.                                              |
| `neverTimer() -> Timer`                            | No `after` ever fires. For tests.                                                    |
| `attr HasTimer`                                    | Provides a `Timer`, like `clock.HasMonotonicClock` provides a clock.                 |
| `data Cancelled`                                   | `@Error`: the routine was stopped by `cancel`.                                       |
| `data TimedOut { limit: Duration }`                | `@Error`: `timeout` gave up.                                                         |

Helpers, written in plain Zirric on top of the above:

| Helper                                                                        | Meaning                                                                       |
| ----------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| `all(bodies: [fn() -> Result]) -> [Result]`                                   | Runs every body at once, returns every result in order.                       |
| `map(items: @Iterable, limit: Int, transform: fn(Any) -> Result) -> [Result]` | `transform` over `items`, at most `limit` at once, in order.                  |
| `first(bodies: [fn() -> Result]) -> Result`                                   | The result of whichever body finishes first; stops the rest.                  |
| `timeout(timer: Timer, limit: Duration, body: fn() -> Result) -> Result`      | `body`'s result, or `Err(TimedOut(limit))`.                                   |
| `produce(s: Scope, body: fn(Channel) -> Any) -> Channel`                      | A generator: `body` sends into the channel, which closes when `body` returns. |
| `merge(s: Scope, channels: [Channel]) -> Channel`                             | All values from all channels in one; closes after the last.                   |

**Helpers that return values open their own scope. Helpers that return a channel take a `Scope`**, because their routines outlive the call.

Two additions outside `co` complete it: `os.timer() -> co.Timer`, the real timer, and `io.read` / `io.write`, which make reading and writing a switch point whatever the reader or writer is (see Scheduling).

### Results

Routines speak `Result`, like every other fallible function in Zirric. A routine's body returns one, `wait` hands it back unchanged, and cancellation is just another error. The usual sugar applies:

```zirric
fn countLines(fsys: fs.FileSystem, path: String) -> Int! {
	const read = fs.readString(fsys, path)
	if read is Err {
		return read
	}
	var lines = 0
	for c <- read.value {
		if c == '\n' {
			lines = lines + 1
		}
	}
	return Ok(lines)
}

fn countAll(fsys: fs.FileSystem, paths: [String]) -> [Result] {
	return co.map(paths, 8, fn(path) { countLines(fsys, path) })
}
```

```zirric
fn confirm(stdin: @io.Reader, timer: co.Timer) -> String {
	return co.timeout(timer, time.seconds(10), fn() { readLine(stdin) }) !! "n"
}
```

### Testing

The `Timer` is passed in like `fs.FileSystem` and `clock.SystemClock`, so timeouts are tested without waiting, and deterministically:

```zirric
@tests.Test()
fn testConfirmDefaultsToNo() -> Result {
	return assert.equal("n", confirm(answering("y"), co.immediateTimer()))
}

@tests.Test()
fn testConfirmReadsAnswer() -> Result {
	return assert.equal("y", confirm(answering("y"), co.neverTimer()))
}
```

The same reader gives different answers under different timers. With `immediateTimer`, the timeout is ready before the spawned body ever runs; with `neverTimer`, it never is.

## Detailed Design

### Scheduling

| Rule                                                          | Consequence                                                             |
| ------------------------------------------------------------- | ----------------------------------------------------------------------- |
| Exactly one routine executes Zirric code at any time.         | No two routines touch a `var` or an array at the same moment. No locks. |
| A routine only gives way at a **switch point**.               | Between two switch points, code runs as if it were alone.               |
| Ready routines run first in, first out.                       | Given the same inputs, a program switches the same way every time.      |
| `spawn` does not run `body` yet.                              | The new routine starts when the spawning one reaches a switch point.    |
| A routine that computes without a switch point keeps running. | CPU-heavy work in one routine stalls the others.                        |

A switch point is where a switch _can_ happen. There are two kinds:

| Kind              | Operations                                                                                                        | Gives way                                        |
| ----------------- | ----------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| The outside world | `io.read`, `io.write`, `fmt.fprint*`, the functions of `fs`, the host streams (`io.ReadStream`, `io.WriteStream`) | Always, even when a test fake answers instantly. |
| Coordination      | `send`, `receive`, `select`, `wait`, `sleep`, iterating a channel, the end of a `scope` body                      | Only when it must wait.                          |

The outside world always gives way because production would wait there, even when a fake does not. Coordination only gives way when there is nothing to do yet: a `send` into a channel with room, or a `select` with a ready channel, carries on. A producer sending into a buffered channel therefore runs until the buffer is full.

**The module decides, not the value behind it.** Reading through `io.read` gives way whether the reader is the terminal, a file or an in-memory reader a test supplies. That is what makes a test representative: the order in which things finish may differ from production, the points where interleaving can happen do not. `io.read` and `io.write` stand to `@io.Reader` and `@io.Writer` as `len` stands to `@Countable`: the function to call, with the attribute behind it.

Calling an attribute directly, `io.Reader(x).read(x, n)`, is ordinary Zirric code. It switches only where its own body does and gives up that guarantee. It is meant for implementing a `@Reader` or `@Writer`, not for using one.

`fmt.fprint*` issues exactly one `io.write` per call. A printed value never interleaves with another routine's output.

A program is free of data races, not of logic races: a check made before a switch point may be stale after it.

### Host calls and host streams

During a call into the host (a system call behind `fs` or a stream), the run lock is released and other routines run. Such a routine is _in the host_: nothing in the program can wake it, and it checks for cancellation itself when the call returns.

A host stream owns its reads. Reading happens on the stream's own goroutine; a routine reading from it waits at an ordinary, cancellable switch point. Bytes that arrive after the waiting routine was cancelled stay in the stream for the next reader. `os.stdin()` returns one process-wide reader, so every call reads from the same queue of bytes. A program without other routines reads exactly as fast as before.

### Scopes

| Rule                                                                                  | Consequence                                                      |
| ------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| `scope` is the only way to get a `Scope`.                                             | Every `spawn` traces back to a visible `scope` call.             |
| When `body` returns, `scope` waits for all its routines, then returns `body`'s value. | Nothing outlives the block. A forgotten `wait` never loses work. |
| A routine may spawn more routines into the scope it runs in, if it holds the `Scope`. | They are waited for too.                                         |
| `spawn` into a finished or cancelled scope panics.                                    | A routine can never escape its scope.                            |
| Stopping a routine cancels every scope it has open.                                   | Nested scopes, like the one inside `all`, never leave orphans.   |

Scopes are ordinary values. A function that starts background work takes a `Scope` parameter, so its signature says so.

### Cancelling

`cancel` needs the `Scope` value, so anyone holding it may call it: the scope's body, a routine in the scope that hit a fatal error, or any routine the scope was passed to.

- `cancel(s)` stops the routines spawned into `s`. It never stops the scope's body, nor the routine that called it; those continue and return normally.
- A routine waiting at a switch point stops there, never halfway between two switch points.
- `wait` on a stopped routine returns `Err(Cancelled())`.
- `cancel` on a cancelled or finished scope does nothing.
- Cancelling does not close channels. Values buffered in them are lost. Receiving from a channel whose only sender was stopped blocks; if nothing else can run, that is a deadlock (below).

Cancellation is always explicit. A scope whose body returns waits; it does not stop anything on its own.

#### Cancelling and the outside world

Cancelling is instant for routines waiting on each other or on a host stream. It cannot undo what already reached the host:

| In flight                     | On cancel                                                                                                                       |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Waiting to read a host stream | Stops at once. No byte is lost; a byte arriving later goes to the next reader.                                                  |
| A write (`io.write`, `fs`)    | Completes. There is no way to un-issue a write, and it may finish after `scope` has returned. Cleaning up is the program's job. |
| An `fs` read                  | The routine stops when the call returns. The disk bounds that delay, not a user: delayed, not stuck.                            |

Reads and writes differ because a read's result can be kept for someone else, while a write's effect is already out.

### Routines

- `spawn` never blocks.
- `wait` may be called more than once and from any routine; it returns the same `Result` each time.
- `spawn` requires a body returning a `Result`. At runtime, `wait` panics on a non-`Result`, as that is a bug in the caller. The static checker does not look inside function-typed hints yet, so it does not report this ahead of time.
- A panic inside a routine ends the program, like a panic anywhere else.
- `os.exit` stops every routine. Otherwise there is nothing left to stop: the entry point cannot finish before its scopes do.
- If every routine is blocked on `send`, `receive`, `select` or `wait`, and no timer or host call is pending, the program panics with a deadlock message naming each blocked routine.

### Channels

- `send` on a closed channel panics. `close` on a closed channel panics. Both are bugs in the caller, not expected failures ([ZE-008](https://zirric.knabel.dev/proposals/ZE-008-error-handling/)).
- `receive` on a closed channel returns the buffered values first, then `None()` forever.
- Channels compare by identity.

`@Iterable` ([ZE-010](https://zirric.knabel.dev/proposals/ZE-010-iterable/)) is implemented by receiving until `None`:

- `for v <- ch { }` receives until the channel is closed and drained.
- `break` or `return` makes `yield` return `false`; the loop stops receiving. The channel stays open.
- A `for` expression over a channel collects every value until close.

### Select

`select` looks at the channels in list order and returns the **first** one that is ready. If none is, it blocks until one is.

- A closed, drained channel is always ready and yields `Closed { channel }`. The caller removes it from the list or stops.
- Order is priority: visible at the call site and deterministic in tests. The cost: an always-ready channel early in the list starves the later ones.
- `select([])` panics.

### Helpers

The helpers need nothing beyond the primitives:

```zirric
fn all(bodies: [fn() -> Result]) -> [Result] {
	return scope(fn(s) {
		const routines = for body <- bodies { spawn(s, body) }
		return for r <- routines { wait(r) }
	})
}

fn map(items: @Iterable, limit: Int, transform: fn(Any) -> Result) -> [Result] {
	if limit < 1 {
		panic("co.map needs a limit of at least 1")
	}
	const slots = channel(limit)   // a buffered channel as a semaphore
	return all(for item <- items {
		fn() {
			send(slots, void)
			const result = transform(item)
			_ = receive(slots)
			return result
		}
	})
}

fn first(bodies: [fn() -> Result]) -> Result {
	if len(bodies) == 0 {
		panic("co.first needs at least one body")
	}
	return scope(fn(s) {
		const results = channel(len(bodies))
		for body <- bodies {
			_ = spawn(s, fn() {
				send(results, body())
				return Ok(void)
			})
		}
		const winner = receive(results) ?? Err(Cancelled())
		cancel(s)
		return winner
	})
}

fn timeout(timer: Timer, limit: Duration, body: fn() -> Result) -> Result {
	return scope(fn(s) {
		const done = channel(1)
		_ = spawn(s, fn() {
			send(done, body())
			return Ok(void)
		})
		const selected = select([done, after(timer, limit)])
		if selected.channel == done {
			return selected.value
		}
		cancel(s)
		return Err(TimedOut(limit))
	})
}

fn produce(s: Scope, body: fn(Channel) -> Any) -> Channel {
	const ch = channel(0)
	_ = spawn(s, fn() {
		body(ch)
		close(ch)
		return Ok(void)
	})
	return ch
}

fn merge(s: Scope, channels: [Channel]) -> Channel {
	return produce(s, fn(out) {
		_ = all(for ch <- channels {
			fn() {
				for v <- ch {
					send(out, v)
				}
				return Ok(void)
			}
		})
	})
}
```

These rely on two properties of closures created inside a `for`: each captures its own element, and a closure inside a `for` expression body can reach the locals of the enclosing function. Both are pinned by regression tests. The second did not hold before this proposal; the compiler fix is listed under Bug Fixes in the changelog.

`produce` uses an unbuffered channel, so the producer moves in lockstep with its consumer. A program that wants a producer to run ahead writes the three lines with `channel` and `spawn` itself. A producer that can fail sends its error as a value.

### Implementation

- Each routine is a goroutine. The VM is split into a shared program (code, globals, scheduler) and a per-routine stack and frames, so starting a routine costs a stack, and routines see each other's values without copying.
- There is no scheduler goroutine. A single run lock is passed on: whoever holds it hands it to the next ready routine when giving way, and whoever wakes a routine from outside hands it over if nobody holds it. A switch therefore only happens inside an operation the language calls a switch point.
- Extern functions declare whether they switch and whether they block. The host is entered with the run lock released and re-entered by reacquiring it.

### Static checks, tooling

Nothing new is needed. `Channel` carries `@Iterable`, so iterating it passes [ZE-022](https://zirric.knabel.dev/proposals/ZE-022-static-checks/); a non-function argument to `spawn` is reported like any other. There is no new grammar, so the formatter, language server and tree-sitter grammar need no work.

### Deliberately not

- **Parallelism.** Routines never run Zirric code at the same time.
- **Syntax.** No `go` keyword, no `<-` send or receive operator, no `select` statement, no `async`/`await`.
- **Unscoped `spawn`.** Every routine has an owner.
- **Cancelling when a scope's body returns.** Cancellation is always a visible call.
- **Sending in `select`, non-blocking `select`.** Only blocking receives. Both can be added later without breaking anything.
- **`@Countable` on channels.** A buffered count is stale the moment it is read.
- **Mutexes, atomics, condition variables.** The scheduling rule makes them unnecessary.
- **A `yield` function.** Switch points are the operations that wait, not a call sprinkled in to be fair.
- **Switching on direct attribute calls.** `io.Reader(x).read(x, n)` stays plain Zirric; `io.read` is the switch point. No magic on attribute calls.
- **Handing standard input to child processes.** A host stream owns its reads and may have one in flight after everything that wanted input is gone. That is invisible today, because nothing else reads the same descriptor. A future process module handing standard input to a child must quiesce the stream or hand the shared reader over.

## Changes to the Standard Library

- New module `co`, as listed above.
- `os.timer() -> co.Timer`.
- `os.stdin()` returns one process-wide reader.
- New `io.read(reader: @Reader, length: Int) -> Binary` and `io.write(writer: @Writer, buf: Binary) -> Int`: the functions to call, as `len` is for `@Countable`. Both are switch points.
- `fmt.fprint*` writes through `io.write`, exactly once per call.
- The functions of `fs` and the host streams are switch points.

## Compatibility

Nothing breaks. Switch points only matter when a program has routines; a program without `co` runs as before. Direct attribute calls such as `io.Reader(x).read(x, n)` keep working; `io.read` is the recommended form.

## Dependencies on Other Proposals

| Proposal                                                                      | Dependency                                                                                 |
| ----------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| [ZE-008](https://zirric.knabel.dev/proposals/ZE-008-error-handling/)          | Routines return results; misuse panics.                                                    |
| [ZE-009](https://zirric.knabel.dev/proposals/ZE-009-option-values/)           | `receive` returns an `Option`.                                                             |
| [ZE-010](https://zirric.knabel.dev/proposals/ZE-010-iterable/)                | Channels are `@Iterable`.                                                                  |
| [ZE-018](https://zirric.knabel.dev/proposals/ZE-018-io-fmt-os/)               | `os` hands out the real `Timer`; `io` gains `read` and `write`; `fmt` writes through them. |
| [ZE-019](https://zirric.knabel.dev/proposals/ZE-019-result-and-option-sugar/) | `!!`, `??` and `!.` work on what `wait`, `receive` and the helpers return.                 |

## Alternatives Considered

### Do nothing

Zirric stays single-threaded and simple. Scripts and batch CLIs, which read input, transform it and write output, never notice.

It loses because it rules out the second half of the target domain. A TUI that cannot redraw while waiting for a key is not a TUI.

### Go syntax: `go`, `<-` and `select`

Familiar and terse. It costs two keywords, an operator and a statement with its own case grammar: a sub-language inside the language. `<-` already appears in `for x <- xs`, so `for x <- <-ch` becomes possible. Every piece is expressible as a function, and a closure is already the unit of "run this later".

### async / await

Splits every function into two colors. In Zirric the split breaks early: a `for` body is a closure passed to `@Iterable.iterate`, so awaiting inside a loop needs a second, async iteration protocol, and the same goes for `fun`, `arrays` and every attribute with function fields. `spawn` and `wait` already are futures, without the coloring.

### Goroutine semantics, directly

The VM is written in Go, and this design uses goroutines for routines. Taking their semantics too, parallel shared memory, would make every captured `var`, array and dict a data race. In the VM that means crashes in Go's runtime rather than Zirric panics; for users it means a `sync` module and bugs the checker cannot report with certainty. Go's random `select` and its deadlock messages in Go stack traces would leak into the language as well. The spec should describe Zirric, not its host.

### Parallel routines, share-nothing

Isolates that copy values across channels (Erlang, Dart) can use several cores without races. They need copying rules for closures and captured `var`s and make scripts heavier. CLI and TUI programs mostly wait on I/O. A good candidate for a later proposal on CPU-bound work, not for the default. Relaxing the one-runner rule later would break programs, so it is kept strict now.

### Unscoped `spawn`

`spawn(body)` without a scope is shorter in scripts. It lets routines outlive their caller, drops results nobody waits for, and hides background work from signatures. `all`, `map`, `first` and `timeout` keep scripts short anyway, since they open their own scope.

### Scopes that cancel when their body returns

Shorter for the key-reader case. But a forgotten `wait` would then silently stop a half-written file. Explicit `cancel` makes the rare case visible instead of the common case dangerous.

### `wait` returning the body's value as is

Any value, no wrapping. It leaves no place for "this routine was stopped", and routines that can fail return results anyway. `Result` fits the sugar Zirric already has.

### Switch points on host streams only

Simpler to implement: only the native functions behind the host streams give way. But a test reading from an in-memory reader then never switches, sees a different interleaving than production, and a fake that answers instantly starves every other routine. `io.read` and `io.write` move the switch point to the module, where tests and production share it.

### Other module names

| Name              | Why not                                                                                                                                                    |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `async`           | Suggests `async`/`await` and non-blocking calls; `co` blocks.                                                                                              |
| `conc`            | Awkward to say.                                                                                                                                            |
| `con`             | `CON` is a reserved file name on Windows, and module paths are directories ([ZE-024](https://zirric.knabel.dev/proposals/ZE-024-qualified-module-names/)). |
| `tasks`, `future` | Taken.                                                                                                                                                     |

## Open Questions

- **Failing fast.** Should there be a variant of `all` that returns one `Result`, cancelling the other bodies on the first `Err` (like Go's `errgroup`)?
- **Manual timers.** `immediateTimer` and `neverTimer` cover both sides of a timeout. A timer a test advances step by step would cover more, at the cost of another API.
- **Shuffled scheduling.** A seeded scheduler for tests, like `random.seeded(n)`, could try other interleavings on purpose.
- **Stopped senders.** Should `cancel` close channels only its routines sent into? It would turn some deadlocks into clean ends, but channels have no owner today.

## Acknowledgements

Prior art: Go's goroutines and channels; CSP (Hoare); Lua coroutines and Python's GIL for the one-runner rule; Trio nurseries, Swift task groups and Kotlin coroutine scopes for structured concurrency.
