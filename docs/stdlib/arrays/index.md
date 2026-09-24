---
title: Arrays
description: Slicing, searching and transforming the built-in Array type.
---

# Module `arrays`

```zirric
import arrays
```

|            |                                                                                                                                                                                                                        |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `arrays`                                                                                                                                                                                                               |
| **Source** | [`arrays/arrays.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/arrays/arrays.zirr), [`arrays/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/arrays/module-docs.zirr) |

> Eager operations over `Array`.

`arrays` operates on the built-in [`prelude.Array`](../prelude/index.md#array) type. Every function returns a new array rather than changing the one passed in.

These are the eager versions: each one walks the whole array and builds its result immediately. For the same operations applied lazily to anything iterable, see [`fun`](../fun/index.md).

## Dependencies

- [`ranges`](../ranges/index.md)

---

## Contents

- **Functions** — [`concat`](#concat), [`contains`](#contains), [`filter`](#filter), [`first`](#first), [`firstIndexOf`](#firstindexof), [`flatMap`](#flatmap), [`isEmpty`](#isempty), [`last`](#last), [`lastIndexOf`](#lastindexof), [`map`](#map), [`range`](#range), [`reduce`](#reduce), [`repeat`](#repeat), [`reverse`](#reverse), [`slice`](#slice)

---

## Functions

### `concat` {#concat}

<small>`arrays/arrays.zirr:30`</small>

```zirric
fn concat(parts: [[Any]]) -> [Any]
```

Concatenates every array in parts into a single array, in order. Unlike append(), which adds each argument as one new element, this flattens one level — concat([[1, 2], [3]]) is [1, 2, 3], not [[1, 2], [3]].

---

### `contains` {#contains}

<small>`arrays/arrays.zirr:86`</small>

```zirric
fn contains(v: [Any], value: Any) -> Bool
```

Returns whether value occurs anywhere in v.

---

### `filter` {#filter}

<small>`arrays/arrays.zirr:135`</small>

```zirric
fn filter(v: [Any], predicate: fn(Any) -> Bool) -> [Any]
```

Returns a new array containing only the elements of v for which predicate returns true, in order.

---

### `first` {#first}

<small>`arrays/arrays.zirr:70`</small>

```zirric
fn first(v: [Any]) -> Option
```

Returns v's first element, or None if v is empty.

---

### `firstIndexOf` {#firstindexof}

<small>`arrays/arrays.zirr:91`</small>

```zirric
fn firstIndexOf(v: [Any], value: Any) -> Option
```

Returns the index of value's first occurrence in v, or None if absent.

---

### `flatMap` {#flatmap}

<small>`arrays/arrays.zirr:124`</small>

```zirric
fn flatMap(v: [Any], transform: fn(Any) -> [Any]) -> [Any]
```

Returns a new array with transform applied to each element of v, flattening each resulting array one level into the result.

---

### `isEmpty` {#isempty}

<small>`arrays/arrays.zirr:65`</small>

```zirric
fn isEmpty(v: [Any]) -> Bool
```

Returns whether v has no elements.

---

### `last` {#last}

<small>`arrays/arrays.zirr:78`</small>

```zirric
fn last(v: [Any]) -> Option
```

Returns v's last element, or None if v is empty.

---

### `lastIndexOf` {#lastindexof}

<small>`arrays/arrays.zirr:103`</small>

```zirric
fn lastIndexOf(v: [Any], value: Any) -> Option
```

Returns the index of value's last occurrence in v, or None if absent.

---

### `map` {#map}

<small>`arrays/arrays.zirr:115`</small>

```zirric
fn map(v: [Any], transform: fn(Any) -> Any) -> [Any]
```

Returns a new array with transform applied to each element of v, in order.

---

### `range` {#range}

<small>`arrays/arrays.zirr:18`</small>

```zirric
fn range(v: [Any], r: ranges.Like) -> [Any]
```

Returns the elements of v selected by r.

---

### `reduce` {#reduce}

<small>`arrays/arrays.zirr:146`</small>

```zirric
fn reduce(v: [Any], initial: Any, combine: fn(Any, Any) -> Any) -> Any
```

Combines v's elements into a single value, starting from initial and applying combine(accumulator, element) left to right.

---

### `repeat` {#repeat}

<small>`arrays/arrays.zirr:41`</small>

```zirric
fn repeat(v: [Any], n: Int) -> [Any]
```

Returns v repeated n times, concatenated.

---

### `reverse` {#reverse}

<small>`arrays/arrays.zirr:54`</small>

```zirric
fn reverse(v: [Any]) -> [Any]
```

Returns v with its elements in reverse order.

---

### `slice` {#slice}

<small>`arrays/arrays.zirr:6`</small>

```zirric
fn slice(v: [Any], start: Int, end: Int) -> [Any]
```

Returns the elements of v from start (inclusive) to end (exclusive).
