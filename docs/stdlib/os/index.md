---
title: OS
description: Standard streams, environment, arguments, filesystem and clocks.
---

# Module `os`

> The only module that talks to the machine.

```zirric
import os
```

|            |                                                                                           |
| ---------- | ----------------------------------------------------------------------------------------- |
| **Module** | `os`                                                                                      |
| **Source** | [`os/shim.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/os/shim.zirr) |

`os` is the seam between a program and its host. Everything the outside world provides — the standard streams, the environment, the command line, the real filesystem, the real clocks — enters here and nowhere else.

That is what makes the rest testable. A function that takes a [`fs.FileSystem`](../fs/index.md#filesystem) or a [`clock.SystemClock`](../clock/index.md#systemclock) can be handed [`fs.memory()`](../fs/index.md#memory) or [`clock.fixed`](../clock/index.md#fixed) in a test; a function that calls `os.fs()` itself cannot. Take the capability as a parameter and let the entry point reach for `os`.

## Contents

- **Functions** — [`stdout`](#stdout), [`stdin`](#stdin), [`stderr`](#stderr), [`exit`](#exit), [`env`](#env), [`args`](#args), [`fs`](#fs), [`systemClock`](#systemclock), [`monotonicClock`](#monotonicclock)

---

## Functions

### `stdout` {#stdout}

<small>`os/shim.zirr:8`</small>

```zirric
extern fn stdout() -> @io.Writer
```

Returns a Writer for the standard output stream.

---

### `stdin` {#stdin}

<small>`os/shim.zirr:11`</small>

```zirric
extern fn stdin() -> @io.Reader
```

Returns a Reader for the standard input stream.

---

### `stderr` {#stderr}

<small>`os/shim.zirr:14`</small>

```zirric
extern fn stderr() -> @io.Writer
```

Returns a Writer for the standard error stream.

---

### `exit` {#exit}

<small>`os/shim.zirr:17`</small>

```zirric
extern fn exit(code: Int) -> Void
```

Terminates the process with the given exit code.

---

### `env` {#env}

<small>`os/shim.zirr:20`</small>

```zirric
extern fn env(key: String) -> String
```

Returns the value of the environment variable with the given key.

---

### `args` {#args}

<small>`os/shim.zirr:23`</small>

```zirric
extern fn args() -> [String]
```

Returns the command line arguments as an array of strings.

---

### `fs` {#fs}

<small>`os/shim.zirr:26`</small>

```zirric
extern fn fs() -> fs.FileSystem
```

Returns the host's own filesystem, rooted so that absolute paths resolve as written.

---

### `systemClock` {#systemclock}

<small>`os/shim.zirr:32`</small>

```zirric
fn systemClock() -> clock.SystemClock
```

The host's wall clock, which reports civil time and can jump when the machine is corrected.

---

### `monotonicClock` {#monotonicclock}

<small>`os/shim.zirr:37`</small>

```zirric
fn monotonicClock() -> clock.MonotonicClock
```

The host's monotonic clock, which only moves forward and is the one to measure elapsed time with.

---

## See also

- [`fs`](../fs/index.md) — the filesystem `fs()` returns.
- [`clock`](../clock/index.md) — the clocks `systemClock()` and `monotonicClock()` return.
- [`io`](../io/index.md) — the reader and writer the standard streams carry.
- [`scripts`](../scripts/index.md) — the same things, already wired to the host.
