---
title: Paths
description: Joining, cleaning, splitting and matching path strings.
---

# Module `paths`

> Path strings, independent of the host.

```zirric
import paths
```

|            |                                                                                                   |
| ---------- | ------------------------------------------------------------------------------------------------- |
| **Module** | `paths`                                                                                           |
| **Source** | [`paths/paths.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/paths/paths.zirr) |

Paths are slash-separated regardless of the host: they describe the [filesystem abstraction](../fs/index.md), not the machine the program runs on. Nothing here touches the disk — these are string operations, and a path that does not exist is manipulated exactly like one that does.

[`match`](#match) is the exception to the "returns a plain value" rule: a malformed pattern is a mistake worth reporting, so it returns a [`Result`](../prelude/index.md#result) rather than silently answering `false`.

## Contents

- **Functions** — [`join`](#join), [`clean`](#clean), [`base`](#base), [`dir`](#dir), [`ext`](#ext), [`stem`](#stem), [`isAbs`](#isabs), [`segments`](#segments), [`match`](#match)

---

## Functions

### `join` {#join}

<small>`paths/paths.zirr:6`</small>

```zirric
extern fn join(parts: [String]) -> String
```

Joins every part into a single path, cleaning the result.

---

### `clean` {#clean}

<small>`paths/paths.zirr:9`</small>

```zirric
extern fn clean(p: String) -> String
```

Returns p with redundant separators and . or .. elements resolved.

---

### `base` {#base}

<small>`paths/paths.zirr:12`</small>

```zirric
extern fn base(p: String) -> String
```

Returns the last element of p.

---

### `dir` {#dir}

<small>`paths/paths.zirr:15`</small>

```zirric
extern fn dir(p: String) -> String
```

Returns p without its last element, i.e. its directory.

---

### `ext` {#ext}

<small>`paths/paths.zirr:18`</small>

```zirric
extern fn ext(p: String) -> String
```

Returns the extension of p's last element, including the leading dot, or "" if it has none.

---

### `stem` {#stem}

<small>`paths/paths.zirr:21`</small>

```zirric
extern fn stem(p: String) -> String
```

Returns the last element of p without its extension.

---

### `isAbs` {#isabs}

<small>`paths/paths.zirr:24`</small>

```zirric
extern fn isAbs(p: String) -> Bool
```

Returns whether p begins at the root.

---

### `segments` {#segments}

<small>`paths/paths.zirr:27`</small>

```zirric
extern fn segments(p: String) -> [String]
```

Returns p's non-empty elements, in order.

---

### `match` {#match}

<small>`paths/paths.zirr:31`</small>

```zirric
extern fn match(pattern: String, p: String) -> Result
```

Returns whether p matches pattern, where * matches any run of non-separator characters, ? matches one, and [abc] matches a character class. Fails with Err when the pattern is malformed, which is why this returns a Result rather than a Bool.

---

## See also

- [`fs`](../fs/index.md) — reading and writing at these paths.
- [`strings`](../strings/index.md) — general text operations.
