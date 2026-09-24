---
title: FS
description: Files, directories and the filesystem value that owns them.
---

# Module `fs`

```zirric
import fs
```

|            |                                                                                                                                                                                                                                                                                                                                                                                                        |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Module** | `fs`                                                                                                                                                                                                                                                                                                                                                                                                   |
| **Source** | [`fs/fs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/fs/fs.zirr), [`fs/helpers.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/fs/helpers.zirr), [`fs/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/fs/module-docs.zirr), [`fs/operations.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/fs/operations.zirr) |

> A filesystem you pass around rather than reach for.

A filesystem in Zirric is a value, not a global. [`FileSystem`](#filesystem) carries one function per operation, and every module function here takes it as its first argument — so `fs.readFile(fsys, "a.txt")` and `fsys.readFile("a.txt")` do the same thing, and the first reads better in a chain.

Because it is a value, it can be swapped. `memory()` returns an empty in-memory filesystem that touches no disk and needs no cleaning up; `os.fs()` returns the real one. Code that accepts a [`FileSystem`](#filesystem) — or requires [`HasFileSystem`](#hasfilesystem) — works with either.

Everything that can fail returns a [`prelude.Result`](../prelude/index.md#result) rather than stopping the program.

## Dependencies

- [`bytes`](../bytes/index.md)
  - [`ranges`](../ranges/index.md)
- [`io`](../io/index.md)
- [`paths`](../paths/index.md)
- [`results`](../results/index.md)

---

## Contents

- **Data** — [`Entry`](#entry), [`File`](#file), [`FileSystem`](#filesystem)
- **Attributes** — [`HasFileSystem`](#hasfilesystem)
- **Functions** — [`cd`](#cd), [`close`](#close), [`copy`](#copy), [`create`](#create), [`exists`](#exists), [`glob`](#glob), [`list`](#list), [`memory`](#memory), [`mkdirAll`](#mkdirall), [`move`](#move), [`open`](#open), [`readFile`](#readfile), [`readString`](#readstring), [`remove`](#remove), [`root`](#root), [`walk`](#walk), [`withFile`](#withfile), [`writeFile`](#writefile), [`writeString`](#writestring)

---

## Data

### `Entry` {#entry}

<small>`fs/fs.zirr:51`</small>

```zirric
data Entry {
	name: String
	isDir: Bool
	size: Int
}
```

One entry of a directory listing.

#### Fields

| Field   | Description                                                         |
| ------- | ------------------------------------------------------------------- |
| `name`  | The entry's own name, without the directory it was listed from.     |
| `isDir` | Whether it is a directory rather than a file.                       |
| `size`  | The file's size in bytes, which carries no meaning for a directory. |

---

### `File` {#file}

<small>`fs/fs.zirr:41`</small>

```zirric
data File {
	readFrom: fn(Int) -> Binary
	writeTo: fn(Binary) -> Int
	closeWith: fn() -> Result
}
```

An open file. It is both a reader and a writer, and must be closed.

#### Fields

| Field       | Description                                                                         |
| ----------- | ----------------------------------------------------------------------------------- |
| `readFrom`  | Reads up to the given number of bytes. An empty result means the file is exhausted. |
| `writeTo`   | Writes the buffer and returns the number of bytes written.                          |
| `closeWith` | Closes the file, releasing what the host holds for it.                              |

---

### `FileSystem` {#filesystem}

<small>`fs/fs.zirr:13`</small>

```zirric
data FileSystem {
	readFile: fn(String) -> Result
	writeFile: fn(String, Binary) -> Result
	open: fn(String) -> Result
	create: fn(String) -> Result
	exists: fn(String) -> Bool
	remove: fn(String) -> Result
	move: fn(String, String) -> Result
	list: fn(String) -> Result
	mkdirAll: fn(String) -> Result
	cd: fn(String) -> Result
	root: fn() -> String
}
```

A filesystem. Every operation that can fail returns a Result rather than stopping the program.
Each field has a matching module function taking the filesystem first, which is usually the nicer way to call it.

#### Fields

| Field       | Description                                                                                    |
| ----------- | ---------------------------------------------------------------------------------------------- |
| `readFile`  | Returns the contents of a path.                                                                |
| `writeFile` | Writes content to a path, replacing it if it exists, and returns the number of bytes written.  |
| `open`      | Opens a path for reading. The file must be closed.                                             |
| `create`    | Creates a path for writing, replacing it if it exists. The file must be closed.                |
| `exists`    | Returns whether a path exists.                                                                 |
| `remove`    | Removes a path.                                                                                |
| `move`      | Moves the file at the first path to the second.                                                |
| `list`      | Returns the entries of the directory at a path, as Entry values.                               |
| `mkdirAll`  | Creates a path and any missing parent directories.                                             |
| `cd`        | Returns a filesystem rooted at a path, which is how a confined view of this one is handed out. |
| `root`      | Returns the path this filesystem is rooted at.                                                 |

---

## Attributes

### `HasFileSystem` {#hasfilesystem}

<small>`fs/fs.zirr:6`</small>

```zirric
attr HasFileSystem {
	fileSystem(self: @HasFileSystem) -> FileSystem
}
```

Provides a filesystem, so that code can require one instead of reaching for os directly.

#### Fields

| Field        | Description                                 |
| ------------ | ------------------------------------------- |
| `fileSystem` | Returns the filesystem this value provides. |

---

## Functions

### `cd` {#cd}

<small>`fs/operations.zirr:52`</small>

```zirric
fn cd(fsys: FileSystem, path: String) -> Result
```

Returns a filesystem rooted at path, which is how a confined view of this one is handed out.

---

### `close` {#close}

<small>`fs/helpers.zirr:8`</small>

```zirric
fn close(file: File) -> Result
```

Releases the file. Writes are not guaranteed to have reached the filesystem until this returns.

---

### `copy` {#copy}

<small>`fs/helpers.zirr:96`</small>

```zirric
fn copy(source: FileSystem, sourcePath: String, target: FileSystem, targetPath: String) -> Result
```

Copies one file between filesystems, which may be the same one.

---

### `create` {#create}

<small>`fs/operations.zirr:22`</small>

```zirric
fn create(fsys: FileSystem, path: String) -> Result
```

Creates path for writing, replacing it if it exists. The file must be closed.

---

### `exists` {#exists}

<small>`fs/operations.zirr:27`</small>

```zirric
fn exists(fsys: FileSystem, path: String) -> Bool
```

Returns whether path exists.

---

### `glob` {#glob}

<small>`fs/helpers.zirr:72`</small>

```zirric
fn glob(fsys: FileSystem, base: String, pattern: String) -> Result
```

Every file path under base matching pattern, where * matches within one path element.

---

### `list` {#list}

<small>`fs/operations.zirr:42`</small>

```zirric
fn list(fsys: FileSystem, path: String) -> Result
```

Returns the entries of the directory at path.

---

### `memory` {#memory}

<small>`fs/fs.zirr:69`</small>

```zirric
extern fn memory() -> FileSystem
```

An empty in-memory filesystem, which touches no disk and so needs no cleaning up.

---

### `mkdirAll` {#mkdirall}

<small>`fs/operations.zirr:47`</small>

```zirric
fn mkdirAll(fsys: FileSystem, path: String) -> Result
```

Creates path and any missing parent directories.

---

### `move` {#move}

<small>`fs/operations.zirr:37`</small>

```zirric
fn move(fsys: FileSystem, from: String, to: String) -> Result
```

Moves the file at from to to.

---

### `open` {#open}

<small>`fs/operations.zirr:17`</small>

```zirric
fn open(fsys: FileSystem, path: String) -> Result
```

Opens path for reading. The file must be closed.

---

### `readFile` {#readfile}

<small>`fs/operations.zirr:7`</small>

```zirric
fn readFile(fsys: FileSystem, path: String) -> Result
```

Returns the contents of path.

---

### `readString` {#readstring}

<small>`fs/helpers.zirr:33`</small>

```zirric
fn readString(fsys: FileSystem, path: String) -> Result
```

Returns the contents of path decoded as a String.

---

### `remove` {#remove}

<small>`fs/operations.zirr:32`</small>

```zirric
fn remove(fsys: FileSystem, path: String) -> Result
```

Removes path.

---

### `root` {#root}

<small>`fs/operations.zirr:57`</small>

```zirric
fn root(fsys: FileSystem) -> String
```

Returns the path this filesystem is rooted at.

---

### `walk` {#walk}

<small>`fs/helpers.zirr:43`</small>

```zirric
fn walk(fsys: FileSystem, path: String) -> Result
```

Every file path under path, in no particular order, descending into directories.

---

### `withFile` {#withfile}

<small>`fs/helpers.zirr:14`</small>

```zirric
fn withFile(opened: Result, body: fn(File) -> Any) -> Result
```

Runs body with an open file and closes it afterwards, whether or not body succeeded.
This is the safe way to use open and create, which otherwise leak the handle when something goes wrong.

---

### `writeFile` {#writefile}

<small>`fs/operations.zirr:12`</small>

```zirric
fn writeFile(fsys: FileSystem, path: String, content: Binary) -> Result
```

Writes content to path, replacing it if it exists, and returns the number of bytes written.

---

### `writeString` {#writestring}

<small>`fs/helpers.zirr:38`</small>

```zirric
fn writeString(fsys: FileSystem, path: String, content: String) -> Result
```

Writes content to path, encoded as UTF-8.
