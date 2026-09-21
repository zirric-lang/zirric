---
title: Bytes
description: Building, slicing and searching Binary values.
---

# Module `bytes`

> Raw byte sequences and the conversions into them.

```zirric
import bytes
```

|            |                                                                                                   |
| ---------- | ------------------------------------------------------------------------------------------------- |
| **Module** | `bytes`                                                                                           |
| **Source** | [`bytes/bytes.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/bytes/bytes.zirr) |

`bytes` works on [`Binary`](../prelude/index.md#binary), the type that carries raw bytes. Most functions accept [`Like`](#like), so a single [`Byte`](../prelude/index.md#byte) can be passed anywhere a `Binary` is expected.

Text crosses the boundary explicitly: [`fromString`](#fromstring) and [`toString`](#tostring) convert through UTF-8, and [`toHex`](#tohex) and [`fromHex`](#fromhex) through hexadecimal. Indexing a `Binary` yields a `Byte` and iterating one yields bytes, never characters — for characters, convert to `String` and use [`strings`](../strings/index.md).

## Contents

- **Unions** — [`Like`](#like)
- **Functions** — [`from`](#from), [`fromString`](#fromstring), [`fromChar`](#fromchar), [`toString`](#tostring), [`toHex`](#tohex), [`fromHex`](#fromhex), [`slice`](#slice), [`range`](#range), [`firstIndexOf`](#firstindexof), [`lastIndexOf`](#lastindexof), [`contains`](#contains), [`hasPrefix`](#hasprefix), [`hasSuffix`](#hassuffix), [`concat`](#concat), [`repeat`](#repeat)

---

## Unions

### `Like` {#like}

<small>`bytes/bytes.zirr:6`</small>

```zirric
union Like {
	Byte
	Binary
}
```

A single byte or a sequence of bytes.

#### Cases

| Case     | Interpretation      |
| -------- | ------------------- |
| `Byte`   | One byte.           |
| `Binary` | Any number of them. |

---

## Functions

### `from` {#from}

<small>`bytes/bytes.zirr:12`</small>

```zirric
extern fn from(v: Like) -> Binary
```

Normalizes a Byte or Binary to Binary.

---

### `fromString` {#fromstring}

<small>`bytes/bytes.zirr:15`</small>

```zirric
extern fn fromString(str: String) -> Binary
```

Returns the UTF-8 byte representation of str.

---

### `fromChar` {#fromchar}

<small>`bytes/bytes.zirr:18`</small>

```zirric
extern fn fromChar(c: Char) -> Binary
```

Returns the UTF-8 byte representation of c. A Char can be 1-4 bytes.

---

### `toString` {#tostring}

<small>`bytes/bytes.zirr:21`</small>

```zirric
extern fn toString(v: Like) -> String
```

Reinterprets v's bytes as a String.

---

### `toHex` {#tohex}

<small>`bytes/bytes.zirr:24`</small>

```zirric
extern fn toHex(v: Like) -> String
```

Returns v's hex representation, two lowercase digits per byte.

---

### `fromHex` {#fromhex}

<small>`bytes/bytes.zirr:27`</small>

```zirric
extern fn fromHex(str: String) -> Binary
```

Decodes a hex string (two digits per byte, either case) into Binary.

---

### `slice` {#slice}

<small>`bytes/bytes.zirr:30`</small>

```zirric
extern fn slice(v: Like, start: Int, end: Int) -> Binary
```

Returns the bytes of v from start (inclusive) to end (exclusive).

---

### `range` {#range}

<small>`bytes/bytes.zirr:33`</small>

```zirric
fn range(v: Like, r: ranges.Like) -> Binary
```

Returns the bytes of v selected by r.

---

### `firstIndexOf` {#firstindexof}

<small>`bytes/bytes.zirr:48`</small>

```zirric
fn firstIndexOf(v: Like, needle: Like) -> Option
```

Returns the index of needle's first occurrence in v, or None if absent.

---

### `lastIndexOf` {#lastindexof}

<small>`bytes/bytes.zirr:58`</small>

```zirric
fn lastIndexOf(v: Like, needle: Like) -> Option
```

Returns the index of needle's last occurrence in v, or None if absent.

---

### `contains` {#contains}

<small>`bytes/bytes.zirr:68`</small>

```zirric
fn contains(v: Like, needle: Like) -> Bool
```

Returns whether needle occurs anywhere in v.

---

### `hasPrefix` {#hasprefix}

<small>`bytes/bytes.zirr:73`</small>

```zirric
fn hasPrefix(v: Like, prefix: Like) -> Bool
```

Returns whether v starts with prefix.

---

### `hasSuffix` {#hassuffix}

<small>`bytes/bytes.zirr:83`</small>

```zirric
fn hasSuffix(v: Like, suffix: Like) -> Bool
```

Returns whether v ends with suffix.

---

### `concat` {#concat}

<small>`bytes/bytes.zirr:93`</small>

```zirric
fn concat(parts: [Like]) -> Binary
```

Concatenates every part into a single Binary, in order.

---

### `repeat` {#repeat}

<small>`bytes/bytes.zirr:102`</small>

```zirric
fn repeat(v: Like, n: Int) -> Binary
```

Returns v repeated n times.

---

## See also

- [`strings`](../strings/index.md) — the same operations over text.
- [`io`](../io/index.md) — readers and writers, which move `Binary` around.
