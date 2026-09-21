---
title: FS
description: Files, directories and the filesystem value that owns them.
---

# Module `fs`

> A filesystem you pass around rather than reach for.

```zirric
import fs
```

|            |                                                                                                                                                                                                                                                                                               |
| ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `fs`                                                                                                                                                                                                                                                                                          |
| **Source** | [`fs/fs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/fs/fs.zirr), [`fs/operations.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/fs/operations.zirr), [`fs/helpers.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/fs/helpers.zirr) |

A filesystem in Zirric is a value, not a global. [`FileSystem`](#filesystem) carries one function per operation, and every module function here takes it as its first argument — so `fs.readFile(fsys, "a.txt")` and `fsys.readFile("a.txt")` do the same thing, and the first reads better in a chain.

Because it is a value, it can be swapped. [`memory()`](#memory) returns an empty in-memory filesystem that touches no disk and needs no cleaning up; [`os.fs()`](../os/index.md#fs) returns the real one. Code that accepts a `FileSystem` — or requires [`HasFileSystem`](#hasfilesystem) — works with either.

Everything that can fail returns a [`Result`](../prelude/index.md#result) rather than stopping the program.

## Contents

- **Attributes** — [`HasFileSystem`](#hasfilesystem)
- **Data** — [`FileSystem`](#filesystem), [`File`](#file), [`Entry`](#entry)
- **Functions** — [`memory`](#memory), [`readFile`](#readfile), [`writeFile`](#writefile), [`open`](#open), [`create`](#create), [`exists`](#exists), [`remove`](#remove), [`move`](#move), [`list`](#list), [`mkdirAll`](#mkdirall), [`cd`](#cd), [`root`](#root), [`close`](#close), [`withFile`](#withfile), [`readString`](#readstring), [`writeString`](#writestring), [`walk`](#walk), [`glob`](#glob), [`copy`](#copy)

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

#### Members

| Member       | Signature                                        | Description                               |
| ------------ | ------------------------------------------------ | ----------------------------------------- |
| `fileSystem` | `fileSystem(self: @HasFileSystem) -> FileSystem` | The filesystem this environment provides. |

---

## Data

### `FileSystem` {#filesystem}

<small>`fs/fs.zirr:12`</small>

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

A filesystem. Every operation that can fail returns a Result rather than stopping the program. Each field has a matching module function taking the filesystem first, which is usually the nicer way to call it.

#### Fields

| Field       | Signature                                 | Description                    |
| ----------- | ----------------------------------------- | ------------------------------ |
| `readFile`  | `readFile: fn(String) -> Result`          | See [`readFile`](#readfile).   |
| `writeFile` | `writeFile: fn(String, Binary) -> Result` | See [`writeFile`](#writefile). |
| `open`      | `open: fn(String) -> Result`              | See [`open`](#open).           |
| `create`    | `create: fn(String) -> Result`            | See [`create`](#create).       |
| `exists`    | `exists: fn(String) -> Bool`              | See [`exists`](#exists).       |
| `remove`    | `remove: fn(String) -> Result`            | See [`remove`](#remove).       |
| `move`      | `move: fn(String, String) -> Result`      | See [`move`](#move).           |
| `list`      | `list: fn(String) -> Result`              | See [`list`](#list).           |
| `mkdirAll`  | `mkdirAll: fn(String) -> Result`          | See [`mkdirAll`](#mkdirall).   |
| `cd`        | `cd: fn(String) -> Result`                | See [`cd`](#cd).               |
| `root`      | `root: fn() -> String`                    | See [`root`](#root).           |

---

### `File` {#file}

<small>`fs/fs.zirr:29`</small>

```zirric
@io.Reader(_readFrom)
@io.Writer(_writeTo)
data File {
	readFrom: fn(Int) -> Binary
	writeTo: fn(Binary) -> Int
	closeWith: fn() -> Result
}
```

An open file. It is both a reader and a writer, and must be closed.

#### Fields

| Field       | Signature                     | Description                                                  |
| ----------- | ----------------------------- | ------------------------------------------------------------ |
| `readFrom`  | `readFrom: fn(Int) -> Binary` | Reads up to the given number of bytes.                       |
| `writeTo`   | `writeTo: fn(Binary) -> Int`  | Writes the buffer and returns how many bytes went.           |
| `closeWith` | `closeWith: fn() -> Result`   | Closes the file. Prefer [`close`](#close), which calls this. |

---

### `Entry` {#entry}

<small>`fs/fs.zirr:36`</small>

```zirric
data Entry {
	name: String
	isDir: Bool
	size: Int
}
```

One entry of a directory listing.

#### Fields

| Field   | Signature      | Description                                             |
| ------- | -------------- | ------------------------------------------------------- |
| `name`  | `name: String` | The entry's name within its directory, not a full path. |
| `isDir` | `isDir: Bool`  | Whether it is a directory.                              |
| `size`  | `size: Int`    | Size in bytes.                                          |

---

## Functions

### `memory` {#memory}

<small>`fs/fs.zirr:51`</small>

```zirric
extern fn memory() -> FileSystem
```

An empty in-memory filesystem, which touches no disk and so needs no cleaning up.

---

### `readFile` {#readfile}

<small>`fs/operations.zirr:7`</small>

```zirric
fn readFile(fsys: FileSystem, path: String) -> Result
```

Returns the contents of path.

---

### `writeFile` {#writefile}

<small>`fs/operations.zirr:12`</small>

```zirric
fn writeFile(fsys: FileSystem, path: String, content: Binary) -> Result
```

Writes content to path, replacing it if it exists, and returns the number of bytes written.

---

### `open` {#open}

<small>`fs/operations.zirr:17`</small>

```zirric
fn open(fsys: FileSystem, path: String) -> Result
```

Opens path for reading. The file must be closed.

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

### `remove` {#remove}

<small>`fs/operations.zirr:32`</small>

```zirric
fn remove(fsys: FileSystem, path: String) -> Result
```

Removes path.

---

### `move` {#move}

<small>`fs/operations.zirr:37`</small>

```zirric
fn move(fsys: FileSystem, from: String, to: String) -> Result
```

Moves the file at from to to.

---

### `list` {#list}

<small>`fs/operations.zirr:42`</small>

```zirric
fn list(fsys: FileSystem, path: String) -> Result
```

Returns the entries of the directory at path.

---

### `mkdirAll` {#mkdirall}

<small>`fs/operations.zirr:47`</small>

```zirric
fn mkdirAll(fsys: FileSystem, path: String) -> Result
```

Creates path and any missing parent directories.

---

### `cd` {#cd}

<small>`fs/operations.zirr:52`</small>

```zirric
fn cd(fsys: FileSystem, path: String) -> Result
```

Returns a filesystem rooted at path, which is how a confined view of this one is handed out.

---

### `root` {#root}

<small>`fs/operations.zirr:57`</small>

```zirric
fn root(fsys: FileSystem) -> String
```

Returns the path this filesystem is rooted at.

---

### `close` {#close}

<small>`fs/helpers.zirr:8`</small>

```zirric
fn close(file: File) -> Result
```

Releases the file. Writes are not guaranteed to have reached the filesystem until this returns.

---

### `withFile` {#withfile}

<small>`fs/helpers.zirr:14`</small>

```zirric
fn withFile(opened: Result, body: fn(File) -> Any) -> Result
```

Runs body with an open file and closes it afterwards, whether or not body succeeded. This is the safe way to use open and create, which otherwise leak the handle when something goes wrong.

---

### `readString` {#readstring}

<small>`fs/helpers.zirr:33`</small>

```zirric
fn readString(fsys: FileSystem, path: String) -> Result
```

Returns the contents of path decoded as a String.

---

### `writeString` {#writestring}

<small>`fs/helpers.zirr:38`</small>

```zirric
fn writeString(fsys: FileSystem, path: String, content: String) -> Result
```

Writes content to path, encoded as UTF-8.

---

### `walk` {#walk}

<small>`fs/helpers.zirr:43`</small>

```zirric
fn walk(fsys: FileSystem, path: String) -> Result
```

Every file path under path, in no particular order, descending into directories.

---

### `glob` {#glob}

<small>`fs/helpers.zirr:72`</small>

```zirric
fn glob(fsys: FileSystem, base: String, pattern: String) -> Result
```

Every file path under base matching pattern, where * matches within one path element.

---

### `copy` {#copy}

<small>`fs/helpers.zirr:96`</small>

```zirric
fn copy(source: FileSystem, sourcePath: String, target: FileSystem, targetPath: String) -> Result
```

Copies one file between filesystems, which may be the same one.

---

## See also

- [`paths`](../paths/index.md) — building and inspecting the path strings these take.
- [`io`](../io/index.md) — the reader and writer an open [`File`](#file) carries.
- [`scripts`](../scripts/index.md) — the same operations against the host filesystem.
- [`results`](../results/index.md) — working with what these return.
