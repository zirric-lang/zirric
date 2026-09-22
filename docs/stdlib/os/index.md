---
title: OS
description: Standard streams, environment, arguments, filesystem and clocks.
---

# Module `os`

```zirric
import os
```

|            |                                                                                                                                                                                                    |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `os`                                                                                                                                                                                               |
| **Source** | [`os/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/os/module-docs.zirr), [`os/shim.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/os/shim.zirr) |

> The only module that talks to the machine.

`os` is the seam between a program and its host. Everything the outside world provides — the standard streams, the environment, the command line, the real filesystem, the real clocks — enters here and nowhere else.

That is what makes the rest testable. A function that takes a [`fs.FileSystem`](../fs/index.md#filesystem) or a [`clock.SystemClock`](../clock/index.md#systemclock) can be handed `fs.fs.memory` or [`clock.fixed`](../clock/index.md#fixed) in a test; a function that calls `os.fs()` itself cannot. Take the capability as a parameter and let the entry point reach for `os`.

## Dependencies

- [`clock`](../clock/index.md)
  - [`time`](../time/index.md)
- [`fs`](../fs/index.md)
  - [`bytes`](../bytes/index.md)
    - [`ranges`](../ranges/index.md)
  - [`io`](../io/index.md)
  - [`paths`](../paths/index.md)
  - [`results`](../results/index.md)
- [`io`](../io/index.md)

---

## Contents

- **Functions** — [`args`](#args), [`cwd`](#cwd), [`env`](#env), [`exit`](#exit), [`fs`](#fs), [`monotonicClock`](#monotonicclock), [`stderr`](#stderr), [`stdin`](#stdin), [`stdout`](#stdout), [`systemClock`](#systemclock)

---

## Functions

### `args` {#args}

<small>`os/shim.zirr:23`</small>

```zirric
extern fn args() -> [String]
```

Returns the command line arguments as an array of strings.

---

### `cwd` {#cwd}

<small>`os/shim.zirr:30`</small>

```zirric
extern fn cwd() -> String
```

Returns the directory the process is running in, as an absolute path.
The filesystem os.fs returns is rooted at the host's own root, so this is what a path written relative to where the program was started has to be joined onto.

---

### `env` {#env}

<small>`os/shim.zirr:20`</small>

```zirric
extern fn env(key: String) -> String
```

Returns the value of the environment variable with the given key.

---

### `exit` {#exit}

<small>`os/shim.zirr:17`</small>

```zirric
extern fn exit(code: Int) -> Void
```

Terminates the process with the given exit code.

---

### `fs` {#fs}

<small>`os/shim.zirr:26`</small>

```zirric
extern fn fs() -> fs.FileSystem
```

Returns the host's own filesystem, rooted so that absolute paths resolve as written.

---

### `monotonicClock` {#monotonicclock}

<small>`os/shim.zirr:41`</small>

```zirric
fn monotonicClock() -> clock.MonotonicClock
```

The host's monotonic clock, which only moves forward and is the one to measure elapsed time with.

---

### `stderr` {#stderr}

<small>`os/shim.zirr:14`</small>

```zirric
extern fn stderr() -> @io.Writer
```

Returns a Writer for the standard error stream.

---

### `stdin` {#stdin}

<small>`os/shim.zirr:11`</small>

```zirric
extern fn stdin() -> @io.Reader
```

Returns a Reader for the standard input stream.

---

### `stdout` {#stdout}

<small>`os/shim.zirr:8`</small>

```zirric
extern fn stdout() -> @io.Writer
```

Returns a Writer for the standard output stream.

---

### `systemClock` {#systemclock}

<small>`os/shim.zirr:36`</small>

```zirric
fn systemClock() -> clock.SystemClock
```

The host's wall clock, which reports civil time and can jump when the machine is corrected.
