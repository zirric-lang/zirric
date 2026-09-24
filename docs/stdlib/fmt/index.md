---
title: Fmt
description: Turning values into strings and writing them to a stream.
---

# Module `fmt`

```zirric
import fmt
```

|            |                                                                                                                                                                                                                                                                                                           |
| ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `fmt`                                                                                                                                                                                                                                                                                                     |
| **Source** | [`fmt/io-fmt.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/fmt/io-fmt.zirr), [`fmt/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/fmt/module-docs.zirr), [`fmt/print.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/fmt/print.zirr) |

> The bridge between a value and its text.

`fmt` is how a value becomes text. [`sprint`](#sprint) converts anything at all — it prefers the value's [`prelude.Printable`](../prelude/index.md#printable) attribute and falls back to a plain rendering for built-in types — while [`fprint`](#fprint) and [`fprintln`](#fprintln) send that text to a writer.

There is no `print` here on purpose: writing needs somewhere to write to. Pass a writer from [`os`](../os/index.md), or use [`scripts`](https://code.knabel.dev/zirric-lang/scripts), a separate package pairing these functions with standard output for you.

## Dependencies

- [`bytes`](../bytes/index.md)
  - [`ranges`](../ranges/index.md)
- [`io`](../io/index.md)

---

## Contents

- **Functions** — [`fprint`](#fprint), [`fprintln`](#fprintln), [`sprint`](#sprint)

---

## Functions

### `fprint` {#fprint}

<small>`fmt/print.zirr:10`</small>

```zirric
fn fprint(value: @Printable, writer: @io.Writer) -> Int
```

Writes the string representation of a printable value to a writer.

---

### `fprintln` {#fprintln}

<small>`fmt/print.zirr:16`</small>

```zirric
fn fprintln(value: @Printable, writer: @io.Writer) -> Int
```

Writes the string representation of a printable value followed by a newline to a writer.

---

### `sprint` {#sprint}

<small>`fmt/print.zirr:7`</small>

```zirric
extern fn sprint(value: Any) -> String
```

Converts any value to its string representation. Prefers the value's @Printable attribute when it has one, otherwise falls back to a trivial conversion for builtin types (Int, Float, Char, Byte as hex, ...).
