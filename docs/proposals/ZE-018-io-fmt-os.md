---
title: "ZE-018 - I/O, Formatting, and OS"
description: "Introduces io, fmt, and os modules for standard I/O, text formatting, and operating system interaction"
---

# I/O, Formatting, and OS

::: callout tip <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-check-icon lucide-circle-check"><circle cx="12" cy="12" r="10"/><path d="m9 12 2 2 4-4"/></svg> Implemented
This proposal has been accepted and implemented.
You can use this feature in the latest version of Zirric.
:::

## Introduction

This proposal introduces three new standard library modules — `io`, `fmt`, and `os` — that provide Zirric programs with the ability to interact with the outside world. It also proposes adding a `Bytes` type to the prelude and moving the `@Printable` attribute into the prelude alongside `@Numeric`.

Together, these form the minimal foundation needed for real-world programs: reading input, writing output, formatting text, and accessing the operating system environment.

## Motivation

Zirric currently has no mechanism for interacting with the outside world. A program cannot print output, read input, access environment variables, or terminate with an exit code. This makes it impossible to write useful command-line tools or interactive programs.

The proposed module split follows a layered design where abstract interfaces (`io`) are separated from concrete OS bindings (`os`) and formatting utilities (`fmt`). This separation enables testability by construction — since `io.Reader` and `io.Writer` are `data` types, any function that accepts them can be tested with mock implementations without any special tooling or dependency injection framework.

## Proposed Solution

The proposal introduces four additions to the standard library:

1. **`Bytes` and `@Printable` in the prelude** — a fundamental binary data type and string conversion attribute
2. **`io` module** — abstract `Reader` and `Writer` data types for byte-level I/O
3. **`fmt` module** — formatting and printing functions built on `io`
4. **`os` module** — extern functions that provide access to the real operating system

### Module Dependency Graph

```
prelude (Bytes, @Printable)
        ↑
       io (Reader, Writer, CountResult)
      ↑ ↑
    fmt  os
```

- `io` depends only on the prelude.
- `os` depends on `io` (returns `Reader`/`Writer`).
- `fmt` depends on `io` (accepts `Writer`) and `os` (uses `stdout` for convenience functions).

There are no circular dependencies.

## Detailed Design

### Prelude: `Bytes` and `@Printable`

Two declarations are added to the prelude.

**`Bytes`** is the fundamental binary data type, the byte-level counterpart to `String`:

```zirric
extern type Bytes {
    length: Int
}
```

Any module that performs I/O needs `Bytes`. It is placed in the prelude because it is as foundational as `String` or `Int`.

Conversion between `String` and `Bytes` requires an explicit function. Until extern type constructors ([ZE-015](/proposals/ZE-015-extern-type-constructors)) are available, the `io` module provides a bridge function.

**`@Printable`** marks types that have a string representation:

```zirric
attr Printable {
    toString(self: @Printable) -> String
}
```

This lives in the prelude alongside `@Numeric` because converting values to strings is equally fundamental. Built-in types are annotated at declaration:

```zirric
@Printable(fn(self) { return self })
extern type String { length: Int }

@Printable(fn(self) { return _intToString(self) })
extern type Int {}

@Printable(fn(self) { return _floatToString(self) })
extern type Float {}

@Printable(fn(self) { if self { return "true" } else { return "false" } })
extern type Bool { toggle() -> Bool }
```

### The `io` Module

The `io` module defines abstract byte-level I/O interfaces. It contains no extern functions that interact with the OS — all types are `data` declarations that can be freely constructed by anyone.

```zirric
mod io

import future.prelude { Err, AnyResult }

// The result of a read or write operation.
// On success, contains the number of bytes transferred.
// On failure, contains an Err.
@AnyResult()
union CountResult {
    Int
    Err
}

// An abstract byte reader.
// Reads up to buf.length bytes into buf.
// Returns the number of bytes read or an error.
data Reader {
    read(buf: Bytes) -> CountResult
}

// An abstract byte writer.
// Writes the contents of buf.
// Returns the number of bytes written or an error.
data Writer {
    write(buf: Bytes) -> CountResult
}

// Converts a String to its Bytes representation.
// Interim solution until Bytes supports direct construction (ZE-015).
extern fn bytesFromString(str: String) -> Bytes
```

Because `Reader` and `Writer` are `data` types, they are **mockable by construction**:

```zirric
import io { Writer }

const written = []
const mockWriter = Writer(fn(buf) {
    written.push(buf)
    return buf.length
})
```

No test doubles library or dependency injection is needed. Any function that accepts a `Writer` or `Reader` parameter is inherently testable.

### The `fmt` Module

The `fmt` module provides text formatting and printing utilities. It builds on `io` for byte-level output and uses `@Printable` from the prelude.

```zirric
mod fmt

import io { Writer, CountResult, bytesFromString }
import os { stdout }

// Formats a value as a String without performing any I/O.
fn sprint(value: @Printable) -> String {
    return Printable(value).toString(value)
}

// Writes the string representation of a value to a Writer.
fn fprint(value: @Printable, writer: Writer) -> CountResult {
    return writer.write(bytesFromString(sprint(value)))
}

// Writes the string representation of a value followed by a newline to a Writer.
fn fprintln(value: @Printable, writer: Writer) -> CountResult {
    return fprint(sprint(value) + "\n", writer)
}

// Writes the string representation of a value followed by a newline to standard output.
fn println(value: @Printable) -> CountResult {
    return fprintln(value, stdout())
}
```

### The `os` Module

The `os` module provides access to the real operating system. It is the **only module in this proposal** that contains `extern` function declarations with OS side effects.

```zirric
mod os

import io { Reader, Writer }
import future.prelude { Option }

// Returns a Reader for the standard input stream.
extern fn stdin() -> Reader

// Returns a Writer for the standard output stream.
extern fn stdout() -> Writer

// Returns a Writer for the standard error stream.
extern fn stderr() -> Writer

// Terminates the program with the given exit code.
extern fn exit(code: Int) -> Void

// Returns the value of the environment variable with the given key,
// or None if the variable is not set.
extern fn env(key: String) -> Option

// Returns all environment variables as a dictionary.
extern fn envs() -> Dict

// Returns the command-line arguments passed to the program.
extern fn args() -> [String]
```

### Putting It Together

A complete "Hello, World!" program:

```zirric
import fmt { println }

println("Hello, World!")
```

A program that reads environment configuration and writes to stderr:

```zirric
import fmt { fprintln }
import os { env, stderr }

const name = env("USER") ?? "stranger"
fprintln("Hello, " + name + "!", stderr())
```

A testable function that accepts a `Writer`:

```zirric
import fmt { fprint }
import io { Writer }

fn greet(name: String, out: Writer) {
    fprint("Hello, " + name + "!", out)
}

// In production: greet("World", os.stdout())
// In tests:      greet("World", mockWriter)
```

## Changes to the Standard Library

### Prelude Additions

| Declaration | Kind          | Description                                     |
| ----------- | ------------- | ----------------------------------------------- |
| `Bytes`     | `extern type` | Binary data type with `length: Int`             |
| `Printable` | `attr`        | String representation attribute with `toString` |

### New Modules

| Module | Exports                                                    |
| ------ | ---------------------------------------------------------- |
| `io`   | `Reader`, `Writer`, `CountResult`, `bytesFromString`       |
| `fmt`  | `sprint`, `fprint`, `fprintln`, `println`                  |
| `os`   | `stdin`, `stdout`, `stderr`, `exit`, `env`, `envs`, `args` |

### New Types

| Type          | Module  | Kind          | Description                                    |
| ------------- | ------- | ------------- | ---------------------------------------------- |
| `Bytes`       | prelude | `extern type` | Binary data buffer                             |
| `CountResult` | `io`    | `union`       | `Int \| Err` — result of read/write operations |
| `Reader`      | `io`    | `data`        | Abstract byte reader                           |
| `Writer`      | `io`    | `data`        | Abstract byte writer                           |

### New Functions

| Function          | Module | Signature                             | Description                               |
| ----------------- | ------ | ------------------------------------- | ----------------------------------------- |
| `bytesFromString` | `io`   | `(String) -> Bytes`                   | String-to-Bytes conversion (interim)      |
| `sprint`          | `fmt`  | `(@Printable) -> String`              | Format value as String                    |
| `fprint`          | `fmt`  | `(@Printable, Writer) -> CountResult` | Write formatted value to Writer           |
| `fprintln`        | `fmt`  | `(@Printable, Writer) -> CountResult` | Write formatted value + newline to Writer |
| `println`         | `fmt`  | `(@Printable) -> CountResult`         | Write formatted value + newline to stdout |
| `stdin`           | `os`   | `() -> Reader`                        | Standard input stream                     |
| `stdout`          | `os`   | `() -> Writer`                        | Standard output stream                    |
| `stderr`          | `os`   | `() -> Writer`                        | Standard error stream                     |
| `exit`            | `os`   | `(Int) -> Void`                       | Terminate with exit code                  |
| `env`             | `os`   | `(String) -> Option`                  | Get environment variable                  |
| `envs`            | `os`   | `() -> Dict`                          | Get all environment variables             |
| `args`            | `os`   | `() -> [String]`                      | Get command-line arguments                |

### Breaking Changes

None. This proposal only adds new modules and types.

## Dependencies on Other Proposals

| Proposal                                                                      | Dependency                                                                     |
| ----------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| [ZE-008 Error Handling](/proposals/ZE-008-error-handling)                     | `CountResult` uses `Err` and `@AnyResult()` for result sugar                   |
| [ZE-009 Option Values](/proposals/ZE-009-option-values)                       | `os.env()` returns `Option` for missing variables                              |
| [ZE-015 Extern Type Constructors](/proposals/ZE-015-extern-type-constructors) | Once implemented, `bytesFromString` can be replaced with a `Bytes` constructor |

## Alternatives Considered

### Single Module

All I/O, formatting, and OS functionality could live in a single `os` or `io` module. This was rejected because it conflates abstract interfaces with concrete OS bindings, making it harder to write testable code and harder to reason about which functions have side effects.

### Capability Bundle (`data OS`)

An alternative design bundles all OS capabilities into a single `data` type:

```zirric
data OS {
    stdin: Reader
    stdout: Writer
    stderr: Writer
    exit(code: Int)
    envs() -> Dict
    args() -> [String]
}

const host = OS(...)
```

This enables single-point mockability for testing. However, it was deferred because:

1. **Too coarse** — functions that only need `stdout` would receive `exit`, `envs`, and `args` too, violating the principle of least authority.
2. **Premature** — it cements a dependency injection pattern before real usage reveals what groupings are natural.
3. **Unnecessary** — since `Reader` and `Writer` are already `data` types, individual function parameters are already testable.
4. **Additive** — a capability bundle can be introduced later as a non-breaking addition once `fs` and other modules establish clearer grouping needs.

### `@Printable` in `fmt` Instead of Prelude

`@Printable` could live in the `fmt` module rather than the prelude. This would give `fmt` ownership of its primary attribute, but creates a circular dependency: built-in types (`String`, `Int`, etc.) need `@Printable` annotations, and those types are declared in the prelude. Without retroactive attribute application, `@Printable` must be in scope when types are declared. Placing it in the prelude — alongside the analogous `@Numeric` attribute — is the pragmatic choice.

### Separate `proc` Module for Process Operations

Process-related functions (`exit`, `args`, `envs`, `env`) could be split into a `proc` module separate from `os`. This was rejected because the total API surface is small (~7 functions) and a split would add a module without meaningful conceptual benefit. If the OS surface grows significantly in the future (e.g., signals, child processes), splitting can be reconsidered.

## Acknowledgements

The module split follows patterns established by Go's standard library (`io`, `fmt`, `os`) adapted to Zirric's `data`-type-based interface model. The `Reader`/`Writer` abstraction takes inspiration from Go's `io.Reader` and `io.Writer` interfaces, with the key difference that Zirric's `data` types are inherently mockable without requiring interface satisfaction.
