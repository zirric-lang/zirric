---
title: IO
description: The reader and writer attributes, and the streams that carry bytes.
---

# Module `io`

> Bytes in, bytes out — described as attributes, not classes.

```zirric
import io
```

|            |                                                                                                                                                                                                                        |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `io`                                                                                                                                                                                                                   |
| **Source** | [`io/reader-writer.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/io/reader-writer.zirr), [`io/capabilities.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/io/capabilities.zirr) |

`io` describes streams rather than implementing them. [`Reader`](#reader) and [`Writer`](#writer) are attributes, so any type becomes a stream by carrying one — a file, a socket, or a buffer you wrote yourself. Functions that move bytes take `@io.Reader` or `@io.Writer` and never care which.

[`ReadStream`](#readstream) and [`WriteStream`](#writestream) are simply the ones the host hands out; [`os.stdout()`](../os/index.md#stdout) returns a `WriteStream`, and an open [`fs.File`](../fs/index.md#file) carries both attributes at once.

The `Has…` attributes are the other half: a program declares that its environment provides a writer, and a test passes one that collects into memory instead of reaching for standard output.

## Contents

- **Attributes** — [`Reader`](#reader), [`Writer`](#writer), [`HasStandardWriter`](#hasstandardwriter), [`HasErrorWriter`](#haserrorwriter), [`HasStandardReader`](#hasstandardreader)
- **Data** — [`ReadStream`](#readstream), [`WriteStream`](#writestream)

---

## Attributes

### `Reader` {#reader}

<small>`io/reader-writer.zirr:4`</small>

```zirric
attr Reader {
	// Reads up to the given number of bytes. An empty result means the stream is exhausted.
	read(self: @Reader, length: Int) -> Binary
}
```

Marks a type as a readable stream of bytes.

#### Members

| Member | Signature                                    | Description                                                                           |
| ------ | -------------------------------------------- | ------------------------------------------------------------------------------------- |
| `read` | `read(self: @Reader, length: Int) -> Binary` | Reads up to the given number of bytes. An empty result means the stream is exhausted. |

---

### `Writer` {#writer}

<small>`io/reader-writer.zirr:10`</small>

```zirric
attr Writer {
	// Writes the given buffer and returns the number of bytes written.
	write(self: @Writer, buf: Binary) -> Int
}
```

Marks a type as a writable stream of bytes.

#### Members

| Member  | Signature                                  | Description                                                      |
| ------- | ------------------------------------------ | ---------------------------------------------------------------- |
| `write` | `write(self: @Writer, buf: Binary) -> Int` | Writes the given buffer and returns the number of bytes written. |

---

### `HasStandardWriter` {#hasstandardwriter}

<small>`io/capabilities.zirr:5`</small>

```zirric
attr HasStandardWriter {
	writer(self) -> @Writer
}
```

Provides a standard out writer. In production, this usually points to `os.stdout`.

#### Members

| Member   | Signature                 | Description                        |
| -------- | ------------------------- | ---------------------------------- |
| `writer` | `writer(self) -> @Writer` | The environment's standard output. |

---

### `HasErrorWriter` {#haserrorwriter}

<small>`io/capabilities.zirr:11`</small>

```zirric
attr HasErrorWriter {
	writer(self) -> @Writer
}
```

Provides a standard error writer. In production, this usually points to `os.stderr`.

#### Members

| Member   | Signature                 | Description                       |
| -------- | ------------------------- | --------------------------------- |
| `writer` | `writer(self) -> @Writer` | The environment's standard error. |

---

### `HasStandardReader` {#hasstandardreader}

<small>`io/capabilities.zirr:17`</small>

```zirric
attr HasStandardReader {
	reader(self) -> @Reader
}
```

Provides a standard input writer. In production, this usually points to `os.stdin`.

#### Members

| Member   | Signature                 | Description                       |
| -------- | ------------------------- | --------------------------------- |
| `reader` | `reader(self) -> @Reader` | The environment's standard input. |

---

## Data

### `ReadStream` {#readstream}

<small>`io/reader-writer.zirr:18`</small>

```zirric
@Reader(_readFromStream)
data ReadStream {
	readFrom: fn(Int) -> Binary
}
```

A readable stream backed by the host, such as standard input or an open file. Any type can be a reader by carrying @Reader; this is simply the one the host hands out.

#### Fields

| Field      | Signature                     | Description                                          |
| ---------- | ----------------------------- | ---------------------------------------------------- |
| `readFrom` | `readFrom: fn(Int) -> Binary` | Reads up to the given number of bytes from the host. |

---

### `WriteStream` {#writestream}

<small>`io/reader-writer.zirr:24`</small>

```zirric
@Writer(_writeToStream)
data WriteStream {
	writeTo: fn(Binary) -> Int
}
```

A writable stream backed by the host, such as standard output or an open file.

#### Fields

| Field     | Signature                    | Description                                                    |
| --------- | ---------------------------- | -------------------------------------------------------------- |
| `writeTo` | `writeTo: fn(Binary) -> Int` | Writes the buffer to the host and returns how many bytes went. |

---

## See also

- [`fmt`](../fmt/index.md) — writing text to these streams.
- [`os`](../os/index.md) — the host's standard streams.
- [`fs`](../fs/index.md) — files, which are readers and writers.
- [`bytes`](../bytes/index.md) — the `Binary` values they carry.
