---
title: "ZE-027 - Processes"
description: "Running other programs and shell scripts through a process runner passed in like a filesystem, as a standard library module."
---

# Processes

::: callout draft Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design. Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

Zirric cannot start another program. A script that builds, tags and pushes a release has to be a shell script, and a TUI that shows `git status` has to shell out through a wrapper written in another language. `tasks.Exec` runs Zirric files, not arbitrary programs.

This proposal adds a standard library module, `proc`, that runs other programs and waits for them. A program is described by a `Command` value and run through a `Runner`, a capability passed in like `fs.FileSystem`. Only `os` hands out the real one; tests use a scripted one. Running a program is a switch point in the sense of [ZE-025](https://zirric.knabel.dev/proposals/ZE-025-routines-and-channels/), so `co.timeout`, `co.map` and `co.cancel` work on processes without anything new. There is no new syntax, no new keyword and no new operator.

Three rules govern the design:

- **Arguments are an array, never a string.** Nothing is parsed, split, quoted or expanded. The shell is a program you run, not a place commands run in.
- **Only `os` starts processes.** Code that runs programs takes a `proc.Runner`, so its signature says so and a test can answer in its place.
- **A non-zero exit is an error.** A failed command returns `Err`, carrying its exit code and output. Ignoring a failure takes code; noticing one does not.

## Motivation

Scripting other programs is the everyday half of the target domain. Release scripts, code generators, project task runners and developer TUIs all glue `git`, compilers and formatters together. Today that glue has to live outside Zirric:

```sh
#!/bin/sh
set -e
if [ -n "$(git status --porcelain)" ]; then
	echo "working tree has uncommitted changes" >&2
	exit 1
fi
git tag "v$1"
git push origin "v$1"
```

It works until it doesn't: a missing `set -e` swallows a failure, an unquoted `$1` splits on a space, and none of it can be tested without a real repository.

With `proc`, the same script is ordinary Zirric. Failures are results, arguments are array elements, and the runner is a parameter:

```zirric
mod code.example.release

import proc
import strings

@Error(fn(err) { return "working tree has uncommitted changes" })
data DirtyTree

// Tags the current commit as v<version> and pushes the tag.
fn tagRelease(runner: proc.Runner, version: String) -> Result {
	const status = proc.output(runner, ["git", "status", "--porcelain"])
	if status is Err {
		return status
	}
	if strings.trim(status.value) != "" {
		return Err(DirtyTree())
	}
	const tagged = proc.call(runner, ["git", "tag", "v" + version])
	if tagged is Err {
		return tagged
	}
	return proc.call(runner, ["git", "push", "origin", "v" + version])
}
```

The entry point is the only place that reaches for the machine:

```zirric
const result = release.tagRelease(os.runner(), os.args()[1])
if result is Err {
	fmt.fprintln(result.reason, os.stderr())
	os.exit(1)
}
```

And the test runs no program at all:

```zirric
mod code.example.release._t

import proc
import strings
import tests
import tests.assert
import code.example.release

@tests.Test()
fn testRefusesDirtyTree() -> Result {
	var ran = []
	const runner = proc.scripted(fn(cmd) {
		ran = append(ran, strings.join(cmd.args, " "))
		return proc.Reply(0, " M Cavefile.zirr\n", "")
	})
	return assert.all([
		assert.isErr(release.tagRelease(runner, "1.2.0")),
		assert.equal(["git status --porcelain"], ran)
	])
}
```

Because running a program is a switch point, the concurrency of ZE-025 applies unchanged. Checking many repositories, four at a time, is one call:

```zirric
fn statuses(runner: proc.Runner, repos: [String]) -> [Result] {
	return co.map(repos, 4, fn(repo) {
		return proc.run(runner, proc.inDir(proc.command(["git", "status", "--porcelain"]), repo))
	})
}
```

A test suite that must not hang a CI job gets a limit the same way:

```zirric
co.timeout(timer, time.minutes(10), fn() { proc.call(runner, ["make", "test"]) })
```

## Proposed Solution

A new module, `proc`:

| Declaration                                                              | Meaning                                                                                              |
| ------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------- |
| `data Command { args, dir, env, stdin, stdout, stderr }`                 | What to run, where, and where its streams go. `args[0]` is the program.                              |
| `command(args: [String]) -> Command`                                     | A command in the current directory, with no environment overrides, input discarded, output captured. |
| `inDir(cmd: Command, dir: String) -> Command`                            | The same command, run in `dir`.                                                                      |
| `withEnv(cmd: Command, key: String, value: String) -> Command`           | The same command, with one environment variable set for the child.                                   |
| `withStdin(cmd: Command, input: Input) -> Command`                       | The same command, reading from `input`.                                                              |
| `withStdout(cmd: Command, output: Output) -> Command`                    | The same command, writing standard output to `output`.                                               |
| `withStderr(cmd: Command, output: Output) -> Command`                    | The same command, writing standard error to `output`.                                                |
| `union Input { Inherit, Discard, FromBytes, FromReader }`                | Where the child's standard input comes from.                                                         |
| `union Output { Inherit, Discard, Capture, ToWriter }`                   | Where the child's standard output or error goes.                                                     |
| `data Finished { stdout: Binary, stderr: Binary }`                       | A successful run. Streams that were not captured are empty.                                          |
| `data Runner { run: fn(Command) -> Result, find: fn(String) -> Option }` | The capability to start programs.                                                                    |
| `run(runner: Runner, cmd: Command) -> Result`                            | Runs `cmd` and waits for it. `Ok(Finished)` on exit code 0, an error otherwise.                      |
| `find(runner: Runner, program: String) -> Option`                        | The path `program` would run from, or `None()`.                                                      |
| `attr HasRunner`                                                         | Provides a `Runner`, like `fs.HasFileSystem` provides a filesystem.                                  |
| `scripted(respond: fn(Command) -> Reply) -> Runner`                      | A runner that starts nothing and answers every command with `respond`. For tests.                    |
| `data Reply { code: Int, stdout: String, stderr: String }`               | What a scripted program "printed" and how it exited.                                                 |
| `forbidden() -> Runner`                                                  | A runner that refuses every command. For tests.                                                      |

The members of `Input` and `Output`:

| Member                                   | In            | Meaning                                                                      |
| ---------------------------------------- | ------------- | ---------------------------------------------------------------------------- |
| `data Inherit`                           | Input, Output | The child uses the program's own stream: the terminal, usually.              |
| `data Discard`                           | Input, Output | Nothing in, nothing kept. The child reads end of input or writes to nowhere. |
| `data FromBytes { content: Binary }`     | Input         | The child reads `content`, then end of input.                                |
| `data FromReader { reader: @io.Reader }` | Input         | The child reads what `reader` yields, until it is exhausted.                 |
| `data Capture`                           | Output        | Collected into `Finished`, or into the error if the run fails.               |
| `data ToWriter { writer: @io.Writer }`   | Output        | Written to `writer` as it arrives.                                           |

Errors, all `@Error`:

| Error                                                                       | When                                                 |
| --------------------------------------------------------------------------- | ---------------------------------------------------- |
| `data NotFound { program: String }`                                         | `args[0]` is not a path and is not on `PATH`.        |
| `data CannotStart { program: String, reason: String }`                      | The program exists but the host refused to start it. |
| `data Exited { args: [String], code: Int, stdout: Binary, stderr: Binary }` | The program ran and exited with a code other than 0. |
| `data Forbidden { args: [String] }`                                         | A `forbidden()` runner was asked to run something.   |

Helpers, written in plain Zirric on top of the above:

| Helper                                                  | Meaning                                                                                               |
| ------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| `call(runner: Runner, args: [String]) -> Result`        | Runs with output shown, as a line in a shell script would. `Ok(void)` or an error.                    |
| `output(runner: Runner, args: [String]) -> String!`     | Runs and returns standard output as a `String`, as `$(…)` would, without trimming.                    |
| `interactive(runner: Runner, args: [String]) -> Result` | Runs with all three streams inherited: editors, pagers, password prompts.                             |
| `shell(script: String, args: [String]) -> [String]`     | `["sh", "-c", script, "sh", …args]`: the arguments to run `script` with `sh`, passing `args` as `$1`… |

Two additions outside `proc` complete it: `os.runner() -> proc.Runner`, the real runner, and a handover for `os.stdin()`, described under Standard input.

### Shell scripts

A script file is a program like any other; the host reads its `#!` line:

```zirric
proc.call(runner, ["./scripts/deploy.sh", "staging"])
```

A snippet of shell is run by running a shell. `shell` builds the arguments and nothing else, so it is pure and its output can be asserted on:

```zirric
proc.call(runner, proc.shell("make && make install", []))
```

Values go in as arguments, never into the script text. The shell sees `"$1"` and receives the value unparsed, so a file name with a space or a `;` stays one argument:

```zirric
proc.call(runner, proc.shell("tar czf \"$1.tar.gz\" \"$1\"", [name]))
```

### Expected failures

Some programs use their exit code as an answer: `grep` exits 1 when nothing matched, `diff` when the files differ. `Exited` carries the code and the captured output, so the caller turns the expected case back into a value:

```zirric
fn contains(runner: proc.Runner, pattern: String, path: String) -> Bool! {
	const result = proc.run(runner, proc.command(["grep", "-q", pattern, path]))
	if result is Ok {
		return Ok(true)
	}
	if result.reason is proc.Exited {
		if result.reason.code == 1 {
			return Ok(false)
		}
	}
	return result
}
```

### Streaming output

`Capture` collects everything until the program exits. `ToWriter` streams instead, so a long build appears as it runs:

```zirric
proc.run(runner, proc.withStdout(proc.command(["make"]), proc.ToWriter(out)))
```

A TUI that must also react to keys needs the output as a channel, to `co.select` on it next to them. That takes a writer of four lines and nothing from `proc`:

```zirric
// Sends every chunk it is given into a channel.
@io.Writer(fn(w, buf) {
	co.send(w.channel, buf)
	return len(buf)
})
data ChannelWriter {
	channel: co.Channel
}
```

The command runs in a routine that closes the channel when it ends; the event loop selects on `[keys, chunks]`, and `co.cancel` on its scope stops the build when the user presses `q`.

### Testing

The `Runner` is passed in like `fs.FileSystem` and `co.Timer`. Two doubles cover the common cases:

- `scripted(respond)` calls `respond` with each `Command` and routes the `Reply` exactly as the real runner routes a real program's output: `Capture` collects it, `ToWriter` writes it, `Inherit` and `Discard` drop it. An exit code other than 0 becomes `Exited`. `find` answers `Some(program)` for every program.
- `forbidden()` answers every `run` with `Err(Forbidden(args))` and every `find` with `None()`. Passing it proves that a function runs nothing.

`respond` sees the whole `Command`, so a test can assert on the directory, the environment and the input as well as the arguments.

## Detailed Design

### Commands

| Rule                                                                                           | Consequence                                                                                                                |
| ---------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| Each element of `args` reaches the child as one argument, byte for byte.                       | No splitting, quoting, globbing, `~` or `$VAR` expansion. Ever.                                                            |
| `args[0]` containing a path separator is a path, relative to the command's directory.          | `./deploy.sh` runs the file next to the command's `dir`.                                                                   |
| Otherwise `args[0]` is looked up on the parent's `PATH` when `run` is called, as `find` does.  | A program installed while the script runs is found. `PATH` overrides in `env` don't change the lookup.                     |
| An empty `args` panics.                                                                        | It is a bug in the caller, not an expected failure ([ZE-008](https://zirric.knabel.dev/proposals/ZE-008-error-handling/)). |
| `dir` is `""` for the program's own directory, `os.cwd()`. A relative `dir` is relative to it. | `inDir(cmd, "build")` means what it reads as.                                                                              |
| The child gets the parent's environment, with each entry of `env` set on top.                  | Overrides are visible at the call site; the rest is inherited.                                                             |

`command(args)` sets `stdin` to `Discard` and both outputs to `Capture`. A program started by accident can neither read the user's keys nor write over a TUI, and the error of a failed run carries what it printed. The helpers state other choices by name: `call` inherits both outputs, `interactive` inherits all three streams.

### Exits and errors

| Outcome                                                 | `run` returns                                                     |
| ------------------------------------------------------- | ----------------------------------------------------------------- |
| Exit code 0                                             | `Ok(Finished(stdout, stderr))`                                    |
| Exit code `n`, not 0                                    | `Err(Exited(args, n, stdout, stderr))`                            |
| Stopped by signal `n` (Unix)                            | `Err(Exited(args, 128 + n, stdout, stderr))`, as shells report it |
| Program not on `PATH`, or no file at the given path     | `Err(NotFound(program))`                                          |
| File exists but cannot be started (permissions, format) | `Err(CannotStart(program, reason))`                               |
| The routine running it was cancelled                    | `Err(co.Cancelled())`                                             |

`Exited` debugs as the command and code followed by the first line of captured standard error: `git push origin v1.2.0 exited with 128: fatal: could not read from remote`. Output that was not captured is empty in `Finished` and in `Exited` alike.

### Streams

| Target       | Behavior                                                                                                         |
| ------------ | ---------------------------------------------------------------------------------------------------------------- |
| `Inherit`    | The child gets the program's own descriptor. For standard input, see below.                                      |
| `Discard`    | The null device.                                                                                                 |
| `FromBytes`  | Written to the child's input, then closed.                                                                       |
| `FromReader` | Read with `io.read` and written to the child's input until the reader returns an empty `Binary`, then closed.    |
| `Capture`    | Read by the host in full, whatever the child does with its other stream. Capturing both outputs cannot deadlock. |
| `ToWriter`   | Written with `io.write` as it arrives, one call per chunk the host read.                                         |

Captured output is held in memory without limit. A program that prints gigabytes should stream with `ToWriter`.

A child writing to an inherited stream and a routine printing with `fmt` may interleave on the terminal. Each `fmt.fprint*` call stays whole (ZE-025), the child's writes are its own.

### Scheduling

`proc.run` is a switch point of the outside-world kind (ZE-025): it always gives way, whatever the runner. As with `io.read`, **the module decides, not the value behind it**, so a test with `scripted` sees the same switch points as production with `os.runner()`.

| Rule                                                                                            | Consequence                                                                         |
| ----------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| While the child runs, the calling routine is in the host and the run lock is released.          | Other routines keep running. A TUI keeps drawing while `make` builds.               |
| Copying for `FromReader` and `ToWriter` happens in routines of a scope that `run` opens itself. | The reader and writer are ordinary Zirric values, called at ordinary switch points. |
| `run` returns once the child has exited and every byte has been copied.                         | Nothing `run` started outlives it, in line with ZE-025's scopes.                    |
| `scripted` routes its `Reply` through the same routines.                                        | A `ToWriter` that blocks, like a channel writer, blocks in tests too.               |

### Standard input

ZE-025 left one thing out on purpose: the process-wide reader behind `os.stdin()` may have a read in flight, and a child given the same descriptor would lose bytes to it. `proc` resolves this with a handover:

| Rule                                                                                                                                | Consequence                                                                 |
| ----------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| The host stream behind `os.stdin()` only reads while a routine is waiting for it, and only once the descriptor reports input ready. | It never holds a read nobody asked for.                                     |
| Before a child with `stdin` set to `Inherit` starts, the stream stops waiting for input.                                            | Nothing typed after the child starts goes to the parent.                    |
| Bytes the stream had already read stay in it for the next Zirric reader.                                                            | Nothing typed before the child started goes to the child.                   |
| While the child runs, a routine reading `os.stdin()` waits at a cancellable switch point.                                           | A key reader in a TUI simply pauses. When the child exits, reading resumes. |
| Only one child inherits standard input at a time. Another `Inherit` waits for it, first come, first served.                         | Two editors never fight over one keyboard.                                  |

`FromReader(os.stdin())` is different on purpose: the child reads a pipe, fed through the Zirric reader, and never sees the terminal. It suits filters like `sort`; it does not suit programs that check whether they talk to a terminal.

### The terminal

`proc` knows nothing about terminal modes. A child that inherits the terminal gets it in whatever state the parent left it, including the raw mode and alternate screen a TUI switches on. A TUI that opens `$EDITOR` must restore the terminal first and re-enter afterwards. That belongs in the TUI library of [ZE-026](https://zirric.knabel.dev/proposals/ZE-026-tui/), as a function that takes a body and runs it with the terminal handed back, so `proc` stays small and a CLI without a TUI pays nothing for it.

### Cancelling

Cancelling a scope stops a routine in `run` like any other (ZE-025). The difference: what it waits on is a process, and a process has to be stopped, not abandoned.

| In flight                         | On cancel                                                                                                                                  |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Waiting to inherit standard input | Stops at once. The program never starts.                                                                                                   |
| Program running                   | Asked to terminate (`SIGTERM`; on Windows, terminated). If it still runs after five seconds, killed. The routine stops once it has exited. |
| Copying into a `ToWriter`         | Stops at the next switch point. What was written stays written.                                                                            |
| Feeding a `FromReader`            | Stops; the child's input is closed.                                                                                                        |

A cancelled run is delayed, not stuck: a scope waits at most the grace period plus the time the host takes to reap the child. `wait` on the routine returns `Err(co.Cancelled())`, never `Exited`, even though the program did exit.

`os.exit` kills every child still running, without a grace period. Children do not outlive the program that started them, just as routines don't.

Only the direct child is stopped. Programs it started itself (a shell script's commands, a build's compilers) are its responsibility, as they are for a shell.

### Implementation

- The real runner uses Go's `os/exec`. A `Command` maps to `exec.Cmd` field by field; `args[0]` is resolved with `exec.LookPath`, which also applies `PATHEXT` on Windows.
- Starting and waiting are host calls: the run lock is released before and reacquired after (ZE-025, Implementation).
- The standard input handover needs the host stream to poll its descriptor before reading, so that it can stop waiting without having consumed a byte. On Unix a terminal, pipe or file descriptor supports this.
- The module decides the switch point: `run` is declared a switching extern, and `scripted` runs through the same Zirric code path around it.

### Static checks, tooling

Nothing new is needed. `Command`, `Input`, `Output` and the errors are ordinary `data` and `union` declarations; a misspelled member or a wrong argument count is reported like any other. There is no new grammar, so the formatter, language server and tree-sitter grammar need no work.

### Deliberately not

- **Command strings.** `proc` never splits `"git commit -m 'x'"`. `shell` exists for the cases where a shell is wanted, and it says so.
- **Choosing a shell per platform.** `shell` runs `sh`. On Windows, run `cmd /c` or `pwsh -Command` explicitly.
- **A process handle.** No `start`, `pid`, `kill` or `wait` in `proc`. `co.spawn`, `co.wait` and `co.cancel` already are the handle, with an owner.
- **Signals.** Beyond termination on cancel, no signal can be sent. A later proposal can add `interrupt` without breaking anything.
- **Pipelines between programs.** `a | b` runs through `shell`, or by capturing `a` and feeding `b` with `FromBytes`. A streaming pipe can follow once real programs ask for it.
- **Replacing the running program** (`exec` in the Unix sense). A Zirric program always gets control back.
- **A pseudo-terminal.** A child that needs a terminal inherits the real one.
- **Configurable grace periods, output limits or environment clearing.** See Open Questions.
- **Terminal modes.** They belong to the TUI library, not to processes.

## Changes to the Standard Library

- New module `proc`, as listed above.
- `os.runner() -> proc.Runner`: the real runner.
- The host stream behind `os.stdin()` reads only on demand and can hand the descriptor to a child.
- ZE-025's "Deliberately not: handing standard input to child processes" is resolved by the handover.

## Compatibility

Nothing breaks. `proc` is a new module, and the change to `os.stdin()` is invisible to programs that start no child: a routine waiting for input receives the same bytes in the same order as before.

## Dependencies on Other Proposals

| Proposal                                                                      | Dependency                                                                                       |
| ----------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| [ZE-008](https://zirric.knabel.dev/proposals/ZE-008-error-handling/)          | Failed runs are `@Error` values; an empty command panics.                                        |
| [ZE-009](https://zirric.knabel.dev/proposals/ZE-009-option-values/)           | `find` returns an `Option`.                                                                      |
| [ZE-018](https://zirric.knabel.dev/proposals/ZE-018-io-fmt-os/)               | `os` hands out the real runner; streams are `@io.Reader` and `@io.Writer`.                       |
| [ZE-019](https://zirric.knabel.dev/proposals/ZE-019-result-and-option-sugar/) | `!.`, `!!` and `??` work on what `run`, `output` and `find` return.                              |
| [ZE-025](https://zirric.knabel.dev/proposals/ZE-025-routines-and-channels/)   | `run` is a switch point; cancellation stops the child; the standard input stream is handed over. |
| [ZE-026](https://zirric.knabel.dev/proposals/ZE-026-tui/)                     | Restoring the terminal before an interactive child runs.                                         |

## Alternatives Considered

### Do nothing

Zirric stays a language for the logic between programs, and shell scripts do the gluing. `tasks.Exec` already runs Zirric files as tasks.

It loses because gluing programs together is what CLI scripts mostly do. A Zirric script that cannot call `git` is replaced by a shell script, and the testability the rest of the standard library is built for never reaches it.

### Command strings

`proc.run(runner, "git tag v" + version)` reads like the shell. It means parsing a string into arguments, which means quoting rules, which means every value spliced in is an injection waiting for a space or a `;`. Arrays have no such rules. `shell` keeps the string form available where the author asks for a shell by name.

### Syntax for commands

Backticks, `$(…)` or a `sh"…"` literal as in zx or Bun Shell. Terse, and a sub-language with its own interpolation and escaping rules inside the language. Everything it offers is a function call here.

### A process handle

Go's `exec.Cmd`, Rust's `Child` and Python's `Popen` return an object to wait on, kill and inspect. In Zirric that would be a second lifecycle next to routines: its own `wait`, its own cancel, and processes that outlive their caller when nobody waits. A routine that runs `proc.run` already is that handle, and it has a scope that owns it.

### Exit codes as values

`run` could return `Ok` for every exit and leave the code in `Finished`, as Python's `subprocess.run` does unless `check=True` is passed. That default is the reason `check=True` exists: a failure is ignored unless someone remembers to look. Here, ignoring a failure is the visible choice, and `Exited` keeps the code and output for the callers that expect it.

### `os.run` directly

Shorter, one import fewer. It puts starting programs behind a global function no test can replace, which is exactly what `os` is kept free of: `os` hands out capabilities, it does not act.

### Inheriting all streams by default

`command` could behave like a shell line and inherit everything. Then any program run by a TUI could read its keys and write over its screen, and a failed run's error would say nothing about why. `call` and `interactive` give the shell behavior by name.

### Other module names

| Name      | Why not                                                                                                   |
| --------- | --------------------------------------------------------------------------------------------------------- |
| `exec`    | Suggests replacing the running program, and reads like `tasks.Exec`, which does something else.           |
| `sh`      | Suggests a shell, which is the one thing `proc` does not use unless asked.                                |
| `cmd`     | Reads like Windows' `cmd.exe`, and is the obvious name for a local `Command`.                             |
| `process` | Fine, but long for a module used on every line of a script. `proc` is to it what `fs` is to "filesystem". |
| `run`     | A verb, and `run.run` reads badly.                                                                        |

## Open Questions

- **Grace period.** Five seconds between terminate and kill is a guess. Should `Command` carry it, or is one fixed value the right amount of choice?
- **Output limits.** Should `Capture` take a limit, failing with an error beyond it, instead of holding anything in memory?
- **A clean environment.** Should there be a way to start a child without the parent's environment, for reproducible builds?
- **Lines.** A helper that turns a program's output into a channel of lines would make TUIs shorter. Does it belong in `proc`, or as a line-splitting writer in `io`?
- **Streaming pipelines.** Is `shell` enough for `a | b`, or do real programs need a pipe between two `Command`s?
- **Process groups.** Should cancelling stop the child's own children too, by starting it in a group of its own? That would stop `Ctrl-C` from reaching it through the terminal.
- **Windows.** The standard input handover assumes a descriptor that can be polled. The Windows console needs a separate check before this proposal leaves Draft.
- **ZE-026.** The terminal handover for interactive children needs a name and a shape in the TUI library.

## Acknowledgements

Prior art: Go's `os/exec` for argument arrays and `LookPath`; Rust's `std::process::Command` for the command-as-value shape; Python's `subprocess` for `check=True` and what happens without it; Trio's `run_process` for cancellation that terminates the child; Deno's permissions for runners a program must be handed; zx and Bun Shell for how short script code wants to be.
