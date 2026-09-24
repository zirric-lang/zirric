---
title: Ranges
description: Comparing, intersecting and merging the three range types.
---

# Module `ranges`

```zirric
import ranges
```

|            |                                                                                                                                                                                                                        |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `ranges`                                                                                                                                                                                                               |
| **Source** | [`ranges/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/ranges/module-docs.zirr), [`ranges/ranges.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/ranges/ranges.zirr) |

> Set operations over `Range`, `ClosedRange` and `OpenRange`.

The three range types themselves live in [`prelude`](../prelude/index.md) — they are iterable and countable, so `for i <- Range(0, 10)` works without importing anything. `ranges` adds the operations that treat a range as a set of integers.

[`Like`](#like) is the union of all three, so every function here accepts any of them, in any combination. Results come back as [`prelude.ClosedRange`](../prelude/index.md#closedrange), the one form that can describe any non-empty span exactly, wrapped in an [`prelude.Option`](../prelude/index.md#option) because the answer may be empty.

## Contents

- **Unions** — [`Like`](#like)
- **Functions** — [`contains`](#contains), [`intersect`](#intersect), [`merge`](#merge), [`overlap`](#overlap), [`toClosedRange`](#toclosedrange)

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

#### Cases

| Case          | Interpretation                                                                                                                                                                                                  |
| ------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Range`       | The integers from start up to but excluding end, written `[start, end)`. `Range(0, 3)` iterates 0, 1, 2. This is the form to reach for by default: it counts `end - start` values, so it pairs with a length.   |
| `ClosedRange` | The integers from start to end, both included, written `[start, end]`. `ClosedRange(0, 3)` iterates 0, 1, 2, 3. It is the only form that can name any non-empty span exactly, which is why `ranges` returns it. |
| `OpenRange`   | The integers strictly between start and end, both excluded, written `(start, end)`. `OpenRange(0, 3)` iterates 1, 2.                                                                                            |

---

## Functions

### `contains` {#contains}

<small>`ranges/ranges.zirr:10`</small>

```zirric
fn contains(r: Like, value: Int) -> Bool
```

Returns whether value lies within r.

---

### `intersect` {#intersect}

<small>`ranges/ranges.zirr:48`</small>

```zirric
fn intersect(a: Like, b: Like) -> Option
```

Returns the values a and b have in common, or None if they don't overlap.

---

### `merge` {#merge}

<small>`ranges/ranges.zirr:71`</small>

```zirric
fn merge(a: Like, b: Like) -> Option
```

Combines a and b into a single range spanning both, if they overlap or are adjacent (e.g. [1,3] and [4,6] merge into [1,6]). Returns None if there's a gap between them, since the result wouldn't be a single contiguous range.

---

### `overlap` {#overlap}

<small>`ranges/ranges.zirr:43`</small>

```zirric
fn overlap(a: Like, b: Like) -> Bool
```

Returns whether a and b share any values.

---

### `toClosedRange` {#toclosedrange}

<small>`ranges/ranges.zirr:22`</small>

```zirric
fn toClosedRange(r: Like) -> Option
```

Returns r's equivalent closed, inclusive bounds, or None if r contains no integers (e.g. Range(5, 5), or OpenRange(5, 6)).
