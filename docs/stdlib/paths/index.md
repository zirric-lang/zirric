---
title: Paths
description: Joining, cleaning, splitting and matching path strings.
---

# Module `paths`

```zirric
import paths
```

|            |                                                                                                                                                                                                                  |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `paths`                                                                                                                                                                                                          |
| **Source** | [`paths/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/paths/module-docs.zirr), [`paths/paths.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/paths/paths.zirr) |

> Path strings, independent of the host.

Paths are slash-separated regardless of the host: they describe the filesystem [`fs`](../fs/index.md) presents, not the machine the program runs on. Nothing here touches the disk — these are string operations, and a path that does not exist is manipulated exactly like one that does.

[`match`](#match) is the exception to the "returns a plain value" rule: a malformed pattern is a mistake worth reporting, so it returns a [`prelude.Result`](../prelude/index.md#result) rather than silently answering `false`.

## Contents

- **Functions** — [`base`](#base), [`clean`](#clean), [`dir`](#dir), [`ext`](#ext), [`isAbs`](#isabs), [`join`](#join), [`match`](#match), [`segments`](#segments), [`stem`](#stem)

---

## Functions

### `base` {#base}

<small>`paths/paths.zirr:12`</small>

```zirric
extern fn base(p: String) -> String
```

Returns the last element of p.

---

### `clean` {#clean}

<small>`paths/paths.zirr:9`</small>

```zirric
extern fn clean(p: String) -> String
```

Returns p with redundant separators and . or .. elements resolved.

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

### `isAbs` {#isabs}

<small>`paths/paths.zirr:24`</small>

```zirric
extern fn isAbs(p: String) -> Bool
```

Returns whether p begins at the root.

---

### `join` {#join}

<small>`paths/paths.zirr:6`</small>

```zirric
extern fn join(parts: [String]) -> String
```

Joins every part into a single path, cleaning the result.

---

### `match` {#match}

<small>`paths/paths.zirr:31`</small>

```zirric
extern fn match(pattern: String, p: String) -> Result
```

Returns whether p matches pattern, where * matches any run of non-separator characters, ? matches one, and [abc] matches a character class.
Fails with Err when the pattern is malformed, which is why this returns a Result rather than a Bool.

---

### `segments` {#segments}

<small>`paths/paths.zirr:27`</small>

```zirric
extern fn segments(p: String) -> [String]
```

Returns p's non-empty elements, in order.

---

### `stem` {#stem}

<small>`paths/paths.zirr:21`</small>

```zirric
extern fn stem(p: String) -> String
```

Returns the last element of p without its extension.
