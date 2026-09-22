---
title: Options
description: Working with values that may be absent.
---

# Module `options`

```zirric
import options
```

|            |                                                                                                                                                                                                                              |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `options`                                                                                                                                                                                                                    |
| **Source** | [`options/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/options/module-docs.zirr), [`options/options.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/options/options.zirr) |

> Helpers for `Option`, and for anything shaped like one.

[`prelude.Option`](../prelude/index.md#option) is the type; `options` is what you use it with.

Every function accepts a bare value as well as a `Some` or `None`, because [`from`](#from) normalizes first: an existing option passes through, and anything else is lifted into `Some`. That means these helpers can be applied to a value whose origin you do not control, without checking whether it was already optional.

## Contents

- **Functions** — [`flatMap`](#flatmap), [`from`](#from), [`isNone`](#isnone), [`isSome`](#issome), [`map`](#map), [`or`](#or)

---

## Functions

### `flatMap` {#flatmap}

<small>`options/options.zirr:37`</small>

```zirric
fn flatMap(o, transform: fn(Any) -> @AnyOption) -> Option
```

Returns transform(o's value) normalized to Option, or None unchanged. Use this instead of map when transform can itself be absent.

---

### `from` {#from}

<small>`options/options.zirr:4`</small>

```zirric
fn from(o) -> Option
```

Normalizes o to Option: None stays None, an existing Some stays as it is, and any other value is lifted into Some(o).

---

### `isNone` {#isnone}

<small>`options/options.zirr:21`</small>

```zirric
fn isNone(o) -> Bool
```

Returns whether o is absent.

---

### `isSome` {#issome}

<small>`options/options.zirr:16`</small>

```zirric
fn isSome(o) -> Bool
```

Returns whether o is present.

---

### `map` {#map}

<small>`options/options.zirr:26`</small>

```zirric
fn map(o, transform: fn(Any) -> Any) -> Option
```

Returns a Some with transform applied to o's value, or None unchanged.

---

### `or` {#or}

<small>`options/options.zirr:48`</small>

```zirric
fn or(o, default: Any) -> Any
```

Returns o's value, or default if o is absent.
