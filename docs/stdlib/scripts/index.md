---
title: Scripts
description: Printing and file helpers wired to the host, for short programs.
---

# Module `scripts`

```zirric
import scripts
```

|            |                                                                                                                                                                                                                                                                                                                                                       |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `scripts`                                                                                                                                                                                                                                                                                                                                             |
| **Source** | [`scripts/fmt-scripts.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/scripts/fmt-scripts.zirr), [`scripts/fs-scripts.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/scripts/fs-scripts.zirr), [`scripts/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/scripts/module-docs.zirr) |

> The convenient, already-wired versions.

`scripts` is [`fmt`](../fmt/index.md) and [`fs`](../fs/index.md) with the host already filled in. [`println`](#println) is `fmt.fprintln(value, os.stdout())`; [`readFile`](#readfile) is `fs.readFile(os.fs(), path)`.

That convenience is also its limit. A function calling [`scripts.println`](#println) writes to real standard output and a function calling [`scripts.readFile`](#readfile) touches the real disk, neither of which a test can redirect. For anything beyond a script, take a [`io.Writer`](../io/index.md#writer) or a [`fs.FileSystem`](../fs/index.md#filesystem) as a parameter instead.

## Dependencies

- [`fmt`](../fmt/index.md)
  - [`bytes`](../bytes/index.md)
    - [`ranges`](../ranges/index.md)
  - [`io`](../io/index.md)
- [`fs`](../fs/index.md)
  - [`bytes`](../bytes/index.md)
  - [`io`](../io/index.md)
  - [`paths`](../paths/index.md)
  - [`results`](../results/index.md)
- [`os`](../os/index.md)
  - [`clock`](../clock/index.md)
    - [`time`](../time/index.md)
  - [`fs`](../fs/index.md)
  - [`io`](../io/index.md)

---

## Contents

- **Functions** — [`eprint`](#eprint), [`eprintln`](#eprintln), [`exists`](#exists), [`glob`](#glob), [`list`](#list), [`mkdirAll`](#mkdirall), [`move`](#move), [`print`](#print), [`println`](#println), [`readFile`](#readfile), [`readString`](#readstring), [`remove`](#remove), [`writeFile`](#writefile), [`writeString`](#writestring)

---

## Functions

### `eprint` {#eprint}

<small>`scripts/fmt-scripts.zirr:22`</small>

```zirric
fn eprint(value: @Printable)
```

Prints the string representation of a printable value followed by a newline to stderr.

---

### `eprintln` {#eprintln}

<small>`scripts/fmt-scripts.zirr:17`</small>

```zirric
fn eprintln(value: @Printable)
```

Prints the string representation of a printable value followed by a newline to stderr.

---

### `exists` {#exists}

<small>`scripts/fs-scripts.zirr:27`</small>

```zirric
fn exists(path: String) -> Bool
```

Returns whether path exists on the host's filesystem.

---

### `glob` {#glob}

<small>`scripts/fs-scripts.zirr:52`</small>

```zirric
fn glob(base: String, pattern: String) -> Result
```

Every file path under base on the host's filesystem matching pattern.

---

### `list` {#list}

<small>`scripts/fs-scripts.zirr:32`</small>

```zirric
fn list(path: String) -> Result
```

Lists the entries of a directory on the host's filesystem.

---

### `mkdirAll` {#mkdirall}

<small>`scripts/fs-scripts.zirr:47`</small>

```zirric
fn mkdirAll(path: String) -> Result
```

Creates a directory and any missing parents on the host's filesystem.

---

### `move` {#move}

<small>`scripts/fs-scripts.zirr:42`</small>

```zirric
fn move(from: String, to: String) -> Result
```

Moves a file on the host's filesystem.

---

### `print` {#print}

<small>`scripts/fmt-scripts.zirr:12`</small>

```zirric
fn print(value: @Printable)
```

Prints the string representation of a printable value followed by a newline to stdout.

---

### `println` {#println}

<small>`scripts/fmt-scripts.zirr:7`</small>

```zirric
fn println(value: @Printable)
```

Prints the string representation of a printable value followed by a newline to stdout.

---

### `readFile` {#readfile}

<small>`scripts/fs-scripts.zirr:7`</small>

```zirric
fn readFile(path: String) -> Result
```

Reads path from the host's filesystem.

---

### `readString` {#readstring}

<small>`scripts/fs-scripts.zirr:17`</small>

```zirric
fn readString(path: String) -> Result
```

Reads path from the host's filesystem, decoded as a String.

---

### `remove` {#remove}

<small>`scripts/fs-scripts.zirr:37`</small>

```zirric
fn remove(path: String) -> Result
```

Removes path from the host's filesystem.

---

### `writeFile` {#writefile}

<small>`scripts/fs-scripts.zirr:12`</small>

```zirric
fn writeFile(path: String, content: Binary) -> Result
```

Writes content to path on the host's filesystem.

---

### `writeString` {#writestring}

<small>`scripts/fs-scripts.zirr:22`</small>

```zirric
fn writeString(path: String, content: String) -> Result
```

Writes content to path on the host's filesystem, encoded as UTF-8.
