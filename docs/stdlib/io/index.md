---
title: IO
description: The reader and writer attributes, and the streams that carry bytes.
---

# Module `io`

```zirric
import io
```

|            |                                                                                                                                                                                                                                                                                                                                 |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `io`                                                                                                                                                                                                                                                                                                                            |
| **Source** | [`io/capabilities.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/io/capabilities.zirr), [`io/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/io/module-docs.zirr), [`io/reader-writer.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/io/reader-writer.zirr) |

> Bytes in, bytes out — described as attributes, not classes.

`io` describes streams rather than implementing them. [`Reader`](#reader) and [`Writer`](#writer) are attributes, so any type becomes a stream by carrying one — a file, a socket, or a buffer you wrote yourself. Functions that move bytes take [`@io.Reader`](#reader) or [`@io.Writer`](#writer) and never care which.

[`ReadStream`](#readstream) and [`WriteStream`](#writestream) are simply the ones the host hands out; `os.stdout()` returns a [`WriteStream`](#writestream), and an open [`fs.File`](../fs/index.md#file) carries both attributes at once.

[`read`](#read) and [`write`](#write) are how bytes actually move: `io.read(stdin, 1)` rather than `io.Reader(stdin).read(stdin, 1)`, the way `len` stands in for `@Countable`. They are also the switch points routines give way at ([ZE-025](https://zirric.knabel.dev/proposals/ZE-025-routines-and-channels/)), and that holds whatever reader or writer they are handed — so a fake that answers instantly interleaves exactly where the host's own stream would, and cannot starve the other routines. Calling the attribute directly still works and is plain Zirric code, switching only where its own body does.

The `Has…` attributes are the other half: a program declares that its environment provides a writer, and a test passes one that collects into memory instead of reaching for standard output.

## Contents

- **Data** — [`ReadStream`](#readstream), [`WriteStream`](#writestream)
- **Attributes** — [`HasErrorWriter`](#haserrorwriter), [`HasStandardReader`](#hasstandardreader), [`HasStandardWriter`](#hasstandardwriter), [`Reader`](#reader), [`Writer`](#writer)
- **Functions** — [`read`](#read), [`write`](#write)

---

## Data

### `ReadStream` {#readstream}

<small>`io/reader-writer.zirr:35`</small>

```zirric
data ReadStream {
	readFrom: fn(Int) -> Binary
}
```

A readable stream backed by the host, such as standard input or an open file.
Any type can be a reader by carrying @Reader; this is simply the one the host hands out.

#### Fields

| Field      | Description                                          |
| ---------- | ---------------------------------------------------- |
| `readFrom` | Reads up to the given number of bytes from the host. |

---

### `WriteStream` {#writestream}

<small>`io/reader-writer.zirr:42`</small>

```zirric
data WriteStream {
	writeTo: fn(Binary) -> Int
}
```

A writable stream backed by the host, such as standard output or an open file.

#### Fields

| Field     | Description                                                            |
| --------- | ---------------------------------------------------------------------- |
| `writeTo` | Writes the buffer to the host and returns the number of bytes written. |

---

## Attributes

### `HasErrorWriter` {#haserrorwriter}

<small>`io/capabilities.zirr:12`</small>

```zirric
attr HasErrorWriter {
	writer(self) -> @Writer
}
```

Provides a standard error writer.
In production, this usually points to [`os.stderr`](../os/index.md#stderr).

#### Fields

| Field    | Description                                            |
| -------- | ------------------------------------------------------ |
| `writer` | Returns the standard error writer this value provides. |

---

### `HasStandardReader` {#hasstandardreader}

<small>`io/capabilities.zirr:19`</small>

```zirric
attr HasStandardReader {
	reader(self) -> @Reader
}
```

Provides a standard input writer.
In production, this usually points to [`os.stdin`](../os/index.md#stdin).

#### Fields

| Field    | Description                                            |
| -------- | ------------------------------------------------------ |
| `reader` | Returns the standard input reader this value provides. |

---

### `HasStandardWriter` {#hasstandardwriter}

<small>`io/capabilities.zirr:5`</small>

```zirric
attr HasStandardWriter {
	writer(self) -> @Writer
}
```

Provides a standard out writer.
In production, this usually points to [`os.stdout`](../os/index.md#stdout).

#### Fields

| Field    | Description                                             |
| -------- | ------------------------------------------------------- |
| `writer` | Returns the standard output writer this value provides. |

---

### `Reader` {#reader}

<small>`io/reader-writer.zirr:4`</small>

```zirric
attr Reader {
	read(self: @Reader, length: Int) -> Binary
}
```

Marks a type as a readable stream of bytes.

#### Fields

| Field  | Description                                                                           |
| ------ | ------------------------------------------------------------------------------------- |
| `read` | Reads up to the given number of bytes. An empty result means the stream is exhausted. |

---

### `Writer` {#writer}

<small>`io/reader-writer.zirr:10`</small>

```zirric
attr Writer {
	write(self: @Writer, buf: Binary) -> Int
}
```

Marks a type as a writable stream of bytes.

#### Fields

| Field   | Description                                                      |
| ------- | ---------------------------------------------------------------- |
| `write` | Writes the given buffer and returns the number of bytes written. |

---

## Functions

### `read` {#read}

<small>`io/reader-writer.zirr:20`</small>

```zirric
fn read(reader: @Reader, length: Int) -> Binary
```

Reads up to length bytes from any reader. An empty result means the stream is exhausted.
This is the way to read: it gives the other routines a turn first, so a reader a test supplies interleaves exactly where the host's own would.

---

### `write` {#write}

<small>`io/reader-writer.zirr:27`</small>

```zirric
fn write(writer: @Writer, buf: Binary) -> Int
```

Writes buf to any writer and returns the number of bytes written.
This is the way to write, for the same reason [`read`](#read) is the way to read.
