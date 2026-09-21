---
title: Options
description: Working with values that may be absent.
---

# Module `options`

> Helpers for `Option`, and for anything shaped like one.

```zirric
import options
```

|            |                                                                                                           |
| ---------- | --------------------------------------------------------------------------------------------------------- |
| **Module** | `options`                                                                                                 |
| **Source** | [`options/options.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/options/options.zirr) |

[`Option`](../prelude/index.md#option) is defined in the prelude; `options` is what you use it with.

Every function accepts a bare value as well as a `Some` or `None`, because [`from`](#from) normalizes first: an existing option passes through, and anything else is lifted into `Some`. That means these helpers can be applied to a value whose origin you do not control, without checking whether it was already optional.

## Contents

- **Functions** — [`from`](#from), [`isSome`](#issome), [`isNone`](#isnone), [`map`](#map), [`flatMap`](#flatmap), [`or`](#or)

---

## Functions

### `from` {#from}

<small>`options/options.zirr:4`</small>

```zirric
fn from(o) -> Option
```

Normalizes o to Option: None stays None, an existing Some stays as it is, and any other value is lifted into Some(o).

---

### `isSome` {#issome}

<small>`options/options.zirr:16`</small>

```zirric
fn isSome(o) -> Bool
```

Returns whether o is present.

---

### `isNone` {#isnone}

<small>`options/options.zirr:21`</small>

```zirric
fn isNone(o) -> Bool
```

Returns whether o is absent.

---

### `map` {#map}

<small>`options/options.zirr:26`</small>

```zirric
fn map(o, transform: fn(Any) -> Any) -> Option
```

Returns a Some with transform applied to o's value, or None unchanged.

---

### `flatMap` {#flatmap}

<small>`options/options.zirr:37`</small>

```zirric
fn flatMap(o, transform: fn(Any) -> @AnyOption) -> Option
```

Returns transform(o's value) normalized to Option, or None unchanged. Use this instead of map when transform can itself be absent.

---

### `or` {#or}

<small>`options/options.zirr:48`</small>

```zirric
fn or(o, default: Any) -> Any
```

Returns o's value, or default if o is absent.

---

## See also

- [`prelude`](../prelude/index.md#option) — the `Option` union and the `@AnyOption` attribute.
- [`results`](../results/index.md) — the same shape of helpers for failure.
