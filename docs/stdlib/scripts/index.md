---
title: Scripts
description: Printing and file helpers wired to the host, for short programs.
---

# Module `scripts`

> The convenient, already-wired versions.

```zirric
import scripts
```

|            |                                                                                                                                                                                                                                    |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `scripts`                                                                                                                                                                                                                          |
| **Source** | [`scripts/fmt-scripts.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/scripts/fmt-scripts.zirr), [`scripts/fs-scripts.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/scripts/fs-scripts.zirr) |

`scripts` is [`fmt`](../fmt/index.md) and [`fs`](../fs/index.md) with the host already filled in. [`println`](#println) is `fmt.fprintln(value, os.stdout())`; [`readFile`](#readfile) is `fs.readFile(os.fs(), path)`.

That convenience is also its limit. A function calling `scripts.println` writes to real standard output and a function calling `scripts.readFile` touches the real disk, neither of which a test can redirect. For anything beyond a script, take a [`Writer`](../io/index.md#writer) or a [`FileSystem`](../fs/index.md#filesystem) as a parameter instead.

## Contents

- **Functions** — [`println`](#println), [`print`](#print), [`eprintln`](#eprintln), [`eprint`](#eprint), [`readFile`](#readfile), [`writeFile`](#writefile), [`readString`](#readstring), [`writeString`](#writestring), [`exists`](#exists), [`list`](#list), [`remove`](#remove), [`move`](#move), [`mkdirAll`](#mkdirall), [`glob`](#glob)

---

## Functions

### `println` {#println}

<small>`scripts/fmt-scripts.zirr:7`</small>

```zirric
fn println(value: @Printable)
```

Prints the string representation of a printable value followed by a newline to stdout.

---

### `print` {#print}

<small>`scripts/fmt-scripts.zirr:12`</small>

```zirric
fn print(value: @Printable)
```

Prints the string representation of a printable value followed by a newline to stdout.

---

### `eprintln` {#eprintln}

<small>`scripts/fmt-scripts.zirr:17`</small>

```zirric
fn eprintln(value: @Printable)
```

Prints the string representation of a printable value followed by a newline to stderr.

---

### `eprint` {#eprint}

<small>`scripts/fmt-scripts.zirr:22`</small>

```zirric
fn eprint(value: @Printable)
```

Prints the string representation of a printable value followed by a newline to stderr.

---

### `readFile` {#readfile}

<small>`scripts/fs-scripts.zirr:7`</small>

```zirric
fn readFile(path: String) -> Result
```

Reads path from the host's filesystem.

---

### `writeFile` {#writefile}

<small>`scripts/fs-scripts.zirr:12`</small>

```zirric
fn writeFile(path: String, content: Binary) -> Result
```

Writes content to path on the host's filesystem.

---

### `readString` {#readstring}

<small>`scripts/fs-scripts.zirr:17`</small>

```zirric
fn readString(path: String) -> Result
```

Reads path from the host's filesystem, decoded as a String.

---

### `writeString` {#writestring}

<small>`scripts/fs-scripts.zirr:22`</small>

```zirric
fn writeString(path: String, content: String) -> Result
```

Writes content to path on the host's filesystem, encoded as UTF-8.

---

### `exists` {#exists}

<small>`scripts/fs-scripts.zirr:27`</small>

```zirric
fn exists(path: String) -> Bool
```

Returns whether path exists on the host's filesystem.

---

### `list` {#list}

<small>`scripts/fs-scripts.zirr:32`</small>

```zirric
fn list(path: String) -> Result
```

Lists the entries of a directory on the host's filesystem.

---

### `remove` {#remove}

<small>`scripts/fs-scripts.zirr:37`</small>

```zirric
fn remove(path: String) -> Result
```

Removes path from the host's filesystem.

---

### `move` {#move}

<small>`scripts/fs-scripts.zirr:42`</small>

```zirric
fn move(from: String, to: String) -> Result
```

Moves a file on the host's filesystem.

---

### `mkdirAll` {#mkdirall}

<small>`scripts/fs-scripts.zirr:47`</small>

```zirric
fn mkdirAll(path: String) -> Result
```

Creates a directory and any missing parents on the host's filesystem.

---

### `glob` {#glob}

<small>`scripts/fs-scripts.zirr:52`</small>

```zirric
fn glob(base: String, pattern: String) -> Result
```

Every file path under base on the host's filesystem matching pattern.

---

## See also

- [`fmt`](../fmt/index.md), [`io`](../io/index.md) — the printing underneath.
- [`fs`](../fs/index.md), [`paths`](../paths/index.md) — the file operations underneath.
- [`os`](../os/index.md) — the host values being filled in.
