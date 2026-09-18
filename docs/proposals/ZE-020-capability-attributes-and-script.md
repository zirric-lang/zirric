---
title: ZE-020 - Capability Attributes and the script Module
status: Draft
---

# ZE-020 - Capability Attributes and the `script` Module

::: callout draft Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design. Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

This proposal introduces a family of `@HasX` protocol attributes for declaring ambient-capability requirements structurally, and a new `script` module that provides zero-ceremony access to the real operating system for single-file, test-free scripts.

## Motivation

[ZE-018](./ZE-018-io-fmt-os.md) established `io.Reader` / `io.Writer` as mockable-by-construction capabilities, but left two problems open:

1. **Fan-in.** A function or component that needs several capabilities at once (a clock, a writer, a shell) has no way to accept "an environment that provides these" without either taking several positional parameters or accepting one concrete, project-specific bundle type that every caller must match exactly.
2. **The `println` exception.** `fmt.println` quietly depends on `os` internally to default to `stdout`, which contradicts `fmt`'s stated dependency on `io` alone and makes it the one function in the module that isn't actually mockable. This pattern will only get worse as more OS-backed modules (`clock`, `proc`) grow their own convenience wrappers.

## Proposed Solution

**Capability attributes** (`@HasClock`, `@HasWriter`, `@HasReader`, `@HasShell`, ...) are protocol attributes, exactly like `@Countable` or `@Iterable`, that describe what a value _provides_ rather than what it _is_. Any type — a single capability, or a larger environment bundle — can conform, and multi-attribute type hints (`@HasClock @HasShell`) let a function request precisely the capabilities it needs, no more.

The **`script` module** is the single, clearly-named home for functions that default to the real OS instead of requiring an explicit capability argument. It exists so `os.println`-style ergonomics don't have to be smuggled into otherwise-pure modules.

## Detailed Design

### Capability attributes

Each capability-providing module defines its own `@HasX` attribute alongside its abstract type:

```zirric
mod clock

attr HasClock {
    now(self: @HasClock) -> DateTime
}

// A bare Clock trivially satisfies its own capability attribute.
@HasClock(fn(self) { return self.now() })
data Clock { now() -> DateTime }
```

```zirric
mod io

attr HasWriter {
    write(self: @HasWriter, buf: Bytes) -> CountResult
}

attr HasReader {
    read(self: @HasReader, buf: Bytes) -> CountResult
}

@HasWriter(fn(self, buf) { return self.write(buf) })
data Writer { write(buf: Bytes) -> CountResult }

@HasReader(fn(self, buf) { return self.read(buf) })
data Reader { read(buf: Bytes) -> CountResult }

// Role-specific capabilities. An attribute has exactly one implementation
// per conforming type, so a single generic @HasWriter cannot distinguish
// "give me the standard output channel" from "give me the diagnostic
// channel" on the same Env — these give each role its own, differently
// named method so both can be implemented on the same type at once.
attr HasStandardWriter {
    writeStdout(self: @HasStandardWriter, buf: Bytes) -> CountResult
}

attr HasErrorWriter {
    writeStderr(self: @HasErrorWriter, buf: Bytes) -> CountResult
}

attr HasStandardReader {
    readStdin(self: @HasStandardReader, buf: Bytes) -> CountResult
}
```

```zirric
mod proc

data ProcResult { exitCode: Int, stdout: Bytes, stderr: Bytes }

attr HasShell {
    run(self: @HasShell, cmd: [String]) -> ProcResult
}

@HasShell(fn(self, cmd) { return self.run(cmd) })
data Shell { run(cmd: [String]) -> ProcResult }
```

Application code composes an environment `data` type and conforms it to whichever attributes its own functions need. A single `Env` can hold distinct writers for distinct roles, since each role has its own attribute and method name:

```zirric
@HasClock(fn(self) { return self.realClock.now() })
@HasStandardWriter(fn(self, buf) { return self.stdout.write(buf) })
@HasErrorWriter(fn(self, buf) { return self.stderr.write(buf) })
data Env {
    realClock: clock.Clock
    stdout: io.Writer
    stderr: io.Writer
    shell: proc.Shell
}
```

A function then requests exactly what it uses — including which _role_, not just which capability:

```zirric
fn syncConfig(env: @HasClock @HasStandardWriter) {
    fmt.fprintln("synced at " + fmt.sprint(env.now()), env)
}
```

`syncConfig` accepts a bare `Clock`+`Writer` pair, a full `Env`, or a `TestEnv` with mock implementations — anything conforming to both attributes. A `Cmd` or function that only ever needs `@HasErrorWriter` cannot reach `Env`'s standard-output channel by accident, even though both live on the same `Env` value — the two are genuinely distinct capabilities, not one capability wired to two different things depending on context.

### Naming convention

Capability attributes are prefixed `Has` to avoid colliding with the concrete `data`/`extern type` they describe (`@HasWriter` vs. `io.Writer` live in the same declaration namespace but never collide), and to read correctly at the call site: "this environment `Has`-a-Clock." Every stdlib module that exposes an abstract capability type should export a matching `@HasX` attribute in the same module.

`HasWriter`/`HasReader` are deliberately kept as generic, role-agnostic capabilities — "provides _some_ writer/reader, whichever it is" — for code that genuinely doesn't care which channel it's given. They are not meant to be collapsed into a single `@HasIO`: a function or `Cmd` declaring `@HasIO` would gain access to every I/O role at once (standard output, error output, standard input, and anything added later), defeating the least-authority point of having capability attributes in the first place — the same reason a single, coarse `@IO` capability was rejected in favor of `@HasClock`/`@HasShell` earlier. Role-specific capabilities (`HasStandardWriter`, `HasErrorWriter`, `HasStandardReader`) exist alongside the generic ones for exactly the cases where the role matters — most notably `script.println` vs. `script.eprintln` below.

### The `script` module

`script` depends on `os` directly and provides the convenience layer that `fmt` and future OS-backed modules should _not_ provide themselves:

```zirric
mod script

import fmt { fprintln }
import os { stdout, stderr }
import io { CountResult }

// Writes to standard output. Reaches into the real OS by design —
// importing `script` instead of `fmt` is the signal that this code
// is not meant to be tested or reused as a library.
fn println(value: @Printable) -> CountResult {
    return fprintln(value, stdout())
}

// Writes to standard error — for diagnostics, warnings, and errors
// that should not pollute piped stdout output.
fn eprintln(value: @Printable) -> CountResult {
    return fprintln(value, stderr())
}

// Convenience read of the real system clock.
fn now() -> DateTime {
    return os.systemClock().now()
}
```

`println` and `eprintln` themselves are plain functions calling `os.stdout()`/`os.stderr()` directly — there's no ambient `env` to declare a capability on here. The role-specific attributes from above matter one layer up: any `Cmd`, `Sub`, or application function that takes an `Env`-like value and needs to write diagnostics without also being able to write program output declares `@HasErrorWriter` specifically, not the generic `@HasWriter`:

```zirric
fn logWarning(msg: String, env: @HasErrorWriter) {
    fmt.fprintln("warning: " + msg, env)
}
```

With this in place, `fmt` drops its dependency on `os` entirely and goes back to depending only on `io`, matching its documented design.

## Changes to the Standard Library

| Declaration         | Module   | Kind   | Description                                                         |
| ------------------- | -------- | ------ | ------------------------------------------------------------------- |
| `HasClock`          | `clock`  | `attr` | Capability requirement for reading time                             |
| `HasWriter`         | `io`     | `attr` | Generic, role-agnostic byte-write capability                        |
| `HasReader`         | `io`     | `attr` | Generic, role-agnostic byte-read capability                         |
| `HasStandardWriter` | `io`     | `attr` | Capability requirement for the standard output channel specifically |
| `HasErrorWriter`    | `io`     | `attr` | Capability requirement for the error output channel specifically    |
| `HasStandardReader` | `io`     | `attr` | Capability requirement for the standard input channel specifically  |
| `HasShell`          | `proc`   | `attr` | Capability requirement for running processes                        |
| `script`            | —        | `mod`  | New module: OS-defaulting convenience layer                         |
| `script.println`    | `script` | `fn`   | Print `@Printable` value + newline to stdout                        |
| `script.eprintln`   | `script` | `fn`   | Print `@Printable` value + newline to stderr                        |
| `script.now`        | `script` | `fn`   | Read the real system clock                                          |

**Breaking change:** `fmt.println` is removed; `fmt` no longer imports `os`. Existing callers migrate to `script.println`.

## Dependencies on Other Proposals

| Proposal                                     | Dependency                                                                               |
| -------------------------------------------- | ---------------------------------------------------------------------------------------- |
| [ZE-018 I/O, Fmt, OS](./ZE-018-io-fmt-os.md) | Extends the `io`/`os` split; removes `fmt`'s `os` dependency                             |
| [ZE-017 Type Hints](./ZE-017-type-hints.md)  | Multi-attribute hints (`@HasX @HasY`) are the mechanism capability composition relies on |

## Best Practices

- **Prefer the narrowest `@HasX` combination a function actually uses.** A function that only writes should take `@HasWriter`, not a whole `Env` type, even if every caller happens to have one on hand.
- **Bundle at the composition root, unbundle at every reusable boundary.** A project's `Env` shape is free to vary; library and component code should never require "an `Env`," only the capability attributes it needs.
- **Reserve `script` for entry points that own no tests.** Anything meant to be reused, composed, or independently tested should import `fmt` / `io` / `clock` directly and accept capabilities as parameters.
- **Route diagnostics through `eprintln`, program output through `println`.** This keeps piped/redirected `stdout` output clean even in quick scripts.
- **Give one `Env` value both a real and a test conformance**, mirroring `Reader`/`Writer`'s `mockWriter` pattern, rather than declaring two differently-shaped bundle types for the same set of capabilities.
- **Reach for a role-specific attribute whenever "which one" matters.** Use `@HasStandardWriter`/`@HasErrorWriter`/`@HasStandardReader` for anything that must be pinned to one specific channel; reserve the generic `@HasWriter`/`@HasReader` for code that is genuinely indifferent to which writer or reader it receives. Never collapse these into one `@HasIO` — that reintroduces the coarse-authority problem capability attributes exist to avoid.
