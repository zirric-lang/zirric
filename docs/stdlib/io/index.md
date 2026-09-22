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

[`ReadStream`](#readstream) and [`WriteStream`](#writestream) are simply the ones the host hands out; `os.os.stdout` returns a [`WriteStream`](#writestream), and an open [`fs.File`](../fs/index.md#file) carries both attributes at once.

The `Has…` attributes are the other half: a program declares that its environment provides a writer, and a test passes one that collects into memory instead of reaching for standard output.

## Contents

- **Data** — [`ReadStream`](#readstream), [`WriteStream`](#writestream)
- **Attributes** — [`HasErrorWriter`](#haserrorwriter), [`HasStandardReader`](#hasstandardreader), [`HasStandardWriter`](#hasstandardwriter), [`Reader`](#reader), [`Writer`](#writer)

---

## Data

### `ReadStream` {#readstream}

<small>`io/reader-writer.zirr:18`</small>

```zirric
data ReadStream {
	readFrom: fn(Int) -> Binary
}
```

A readable stream backed by the host, such as standard input or an open file.
Any type can be a reader by carrying @Reader; this is simply the one the host hands out.

#### Fields

| Field      | Description |
| ---------- | ----------- |
| `readFrom` |             |

---

### `WriteStream` {#writestream}

<small>`io/reader-writer.zirr:24`</small>

```zirric
data WriteStream {
	writeTo: fn(Binary) -> Int
}
```

A writable stream backed by the host, such as standard output or an open file.

#### Fields

| Field     | Description |
| --------- | ----------- |
| `writeTo` |             |

---

## Attributes

### `HasErrorWriter` {#haserrorwriter}

<small>`io/capabilities.zirr:11`</small>

```zirric
attr HasErrorWriter {
	writer(self) -> @Writer
}
```

Provides a standard error writer.
In production, this usually points to [`os.stderr`](../os/index.md#stderr).

#### Fields

| Field    | Description |
| -------- | ----------- |
| `writer` |             |

---

### `HasStandardReader` {#hasstandardreader}

<small>`io/capabilities.zirr:17`</small>

```zirric
attr HasStandardReader {
	reader(self) -> @Reader
}
```

Provides a standard input writer.
In production, this usually points to [`os.stdin`](../os/index.md#stdin).

#### Fields

| Field    | Description |
| -------- | ----------- |
| `reader` |             |

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

| Field    | Description |
| -------- | ----------- |
| `writer` |             |

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
