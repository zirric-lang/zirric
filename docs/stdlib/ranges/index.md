---
title: Ranges
description: Comparing, intersecting and merging the three range types.
---

# Module `ranges`

> Set operations over `Range`, `ClosedRange` and `OpenRange`.

```zirric
import ranges
```

|            |                                                                                                       |
| ---------- | ----------------------------------------------------------------------------------------------------- |
| **Module** | `ranges`                                                                                              |
| **Source** | [`ranges/ranges.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/ranges/ranges.zirr) |

The three range types themselves live in [`prelude`](../prelude/index.md) — they are iterable and countable, so `for i <- Range(0, 10)` works without importing anything. `ranges` adds the operations that treat a range as a set of integers.

[`Like`](#like) is the union of all three, so every function here accepts any of them, in any combination. Results come back as [`ClosedRange`](../prelude/index.md#closedrange), the one form that can describe any non-empty span exactly, wrapped in an [`Option`](../prelude/index.md#option) because the answer may be empty.

## Contents

- **Unions** — [`Like`](#like)
- **Functions** — [`contains`](#contains), [`toClosedRange`](#toclosedrange), [`overlap`](#overlap), [`intersect`](#intersect), [`merge`](#merge)

---

## Unions

### `Like` {#like}

<small>`ranges/ranges.zirr:3`</small>

```zirric
union Like {
	Range
	ClosedRange
	OpenRange
}
```

Any of the three range types, so a function can accept whichever form the caller happens to have.

#### Cases

| Case          | Interpretation                                             |
| ------------- | ---------------------------------------------------------- |
| `Range`       | Start inclusive, end exclusive — `Range(1, 4)` is 1, 2, 3. |
| `ClosedRange` | Both ends inclusive — `ClosedRange(1, 4)` is 1, 2, 3, 4.   |
| `OpenRange`   | Both ends exclusive — `OpenRange(1, 4)` is 2, 3.           |

---

## Functions

### `contains` {#contains}

<small>`ranges/ranges.zirr:10`</small>

```zirric
fn contains(r: Like, value: Int) -> Bool
```

Returns whether value lies within r.

---

### `toClosedRange` {#toclosedrange}

<small>`ranges/ranges.zirr:23`</small>

```zirric
fn toClosedRange(r: Like) -> Option
```

Returns r's equivalent closed, inclusive bounds, or None if r contains no integers (e.g. Range(5, 5), or OpenRange(5, 6)).

---

### `overlap` {#overlap}

<small>`ranges/ranges.zirr:44`</small>

```zirric
fn overlap(a: Like, b: Like) -> Bool
```

Returns whether a and b share any values.

---

### `intersect` {#intersect}

<small>`ranges/ranges.zirr:49`</small>

```zirric
fn intersect(a: Like, b: Like) -> Option
```

Returns the values a and b have in common, or None if they don't overlap.

---

### `merge` {#merge}

<small>`ranges/ranges.zirr:74`</small>

```zirric
fn merge(a: Like, b: Like) -> Option
```

Combines a and b into a single range spanning both, if they overlap or are adjacent (e.g. [1,3] and [4,6] merge into [1,6]). Returns None if there's a gap between them, since the result wouldn't be a single contiguous range.

---

## See also

- [`prelude`](../prelude/index.md#range) — the range types and their iteration.
- [`arrays`](../arrays/index.md#range), [`strings`](../strings/index.md#range), [`bytes`](../bytes/index.md#range) — selecting a slice with a range.
