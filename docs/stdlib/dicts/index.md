---
title: Dicts
description: Mapping, filtering and reducing the built-in Dict type.
---

# Module `dicts`

> Operations over `Dict`, by key, by value, or by pair.

```zirric
import dicts
```

|            |                                                                                                   |
| ---------- | ------------------------------------------------------------------------------------------------- |
| **Module** | `dicts`                                                                                           |
| **Source** | [`dicts/dicts.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/dicts/dicts.zirr) |

`dicts` operates on the built-in [`Dict`](../prelude/index.md#dict) type and returns new dictionaries rather than changing the one passed in.

Iterating a `Dict` yields [`Pair`](../prelude/index.md#pair) values, which is why [`mapPairs`](#mappairs) exists alongside [`mapKeys`](#mapkeys) and [`mapValues`](#mapvalues): it is the one that can change both halves of an entry at once.

## Contents

- **Functions** — [`isEmpty`](#isempty), [`hasKey`](#haskey), [`map`](#map), [`mapKeys`](#mapkeys), [`mapValues`](#mapvalues), [`mapPairs`](#mappairs), [`filter`](#filter), [`reduce`](#reduce)

---

## Functions

### `isEmpty` {#isempty}

<small>`dicts/dicts.zirr:4`</small>

```zirric
fn isEmpty(d: Dict) -> Bool
```

Returns whether d has no entries.

---

### `hasKey` {#haskey}

<small>`dicts/dicts.zirr:9`</small>

```zirric
fn hasKey(d: Dict, key: Any) -> Bool
```

Returns whether d has an entry for key. Indexing alone cannot tell, since a missing key and a void value both read as void.

---

### `map` {#map}

<small>`dicts/dicts.zirr:19`</small>

```zirric
fn map(d: Dict, transform: fn(Any, Any) -> Any) -> Dict
```

Returns a new dict with transform applied to each value, keeping the same keys. transform receives both the key and its value.

---

### `mapKeys` {#mapkeys}

<small>`dicts/dicts.zirr:28`</small>

```zirric
fn mapKeys(d: Dict, transform: fn(Any) -> Any) -> Dict
```

Returns a new dict with transform applied to each key, keeping the same values.

---

### `mapValues` {#mapvalues}

<small>`dicts/dicts.zirr:37`</small>

```zirric
fn mapValues(d: Dict, transform: fn(Any) -> Any) -> Dict
```

Returns a new dict with transform applied to each value, keeping the same keys.

---

### `mapPairs` {#mappairs}

<small>`dicts/dicts.zirr:46`</small>

```zirric
fn mapPairs(d: Dict, transform: fn(Pair) -> Pair) -> Dict
```

Returns a new dict with transform applied to each key/value Pair, replacing both.

---

### `filter` {#filter}

<small>`dicts/dicts.zirr:56`</small>

```zirric
fn filter(d: Dict, predicate: fn(Any, Any) -> Bool) -> Dict
```

Returns a new dict containing only the entries of d for which predicate returns true. predicate receives both the key and its value.

---

### `reduce` {#reduce}

<small>`dicts/dicts.zirr:67`</small>

```zirric
fn reduce(d: Dict, initial: Any, combine: fn(Any, Any, Any) -> Any) -> Any
```

Combines d's entries into a single value, starting from initial and applying combine(accumulator, key, value) for each entry, in no particular order.

---

## See also

- [`arrays`](../arrays/index.md) — the same shape of helpers for `Array`.
- [`fun`](../fun/index.md) — lazy versions that work on anything iterable.
