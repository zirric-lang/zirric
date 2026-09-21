---
title: Fmt
description: Turning values into strings and writing them to a stream.
---

# Module `fmt`

> The bridge between a value and its text.

```zirric
import fmt
```

|            |                                                                                               |
| ---------- | --------------------------------------------------------------------------------------------- |
| **Module** | `fmt`                                                                                         |
| **Source** | [`fmt/print.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/fmt/print.zirr) |

`fmt` is how a value becomes text. [`sprint`](#sprint) converts anything at all — it prefers the value's [`Printable`](../prelude/index.md#printable) attribute and falls back to a plain rendering for built-in types — while [`fprint`](#fprint) and [`fprintln`](#fprintln) send that text to a writer.

There is no `print` here on purpose: writing needs somewhere to write to. Pass a writer from [`os`](../os/index.md), or use [`scripts`](../scripts/index.md), which pairs these functions with standard output for you.

## Contents

- **Functions** — [`sprint`](#sprint), [`fprint`](#fprint), [`fprintln`](#fprintln)

---

## Functions

### `sprint` {#sprint}

<small>`fmt/print.zirr:9`</small>

```zirric
extern fn sprint(value: Any) -> String
```

Converts any value to its string representation. Prefers the value's @Printable attribute when it has one, otherwise falls back to a trivial conversion for builtin types (Int, Float, Char, Byte as hex, ...).

---

### `fprint` {#fprint}

<small>`fmt/print.zirr:12`</small>

```zirric
fn fprint(value: @Printable, writer: @io.Writer) -> Int
```

Writes the string representation of a printable value to a writer.

---

### `fprintln` {#fprintln}

<small>`fmt/print.zirr:18`</small>

```zirric
fn fprintln(value: @Printable, writer: @io.Writer) -> Int
```

Writes the string representation of a printable value followed by a newline to a writer.

---

## See also

- [`scripts`](../scripts/index.md) — `println` and friends, for scripts that just want stdout.
- [`io`](../io/index.md) — the `Writer` attribute these functions write through.
- [`strings`](../strings/index.md) — operations on the text once you have it.
