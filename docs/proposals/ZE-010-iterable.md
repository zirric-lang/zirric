---
title: "ZE-010 - Iterable"
description: "Define Countable and Iterable attributes and iteration protocols."
---

# Iterable

::: callout warning In Progress
This proposal has been accepted in principle.
It is currently under active development.
Parts might be incomplete or missing in Zirric.
:::

## Introduction

This proposal introduces standard `@Countable` and `@Iterable` attributes for collection-like types, which is used by `for ... <- ...` loops.
Also introduces `Range` and `ClosedRange` data types as examples of countable and iterable.

## Motivation

Zirric needs a consistent way to describe collection behavior. Without a shared protocol, each type defines bespoke iteration helpers, and tooling cannot recognize which values support iteration or have lengths.

Currently only arrays can be used to iterate over.

## Proposed Solution

Define two attributes in the prelude future module:

- `@Countable` for types with a length.
- `@Iterable` for types that can yield values in a `for` loop.

```zirric
@Proposal(ZE_010)
attr Countable {
  length(value: @Countable) -> Int
}

@Proposal(ZE_010)
attr Iterable {
  iterate(value: @Iterable, yield: fn(Any) -> Bool)
}
```

### Range Types

Add `Range` and `ClosedRange` as iterable, countable data types.

```zirric
@Countable(_rangeCount)
@Iterable(_rangeIterate)
data Range {
  start: Int
  end: Int
}

@Countable(_rangeCount)
@Iterable(_rangeIterate)
data ClosedRange {
  start: Int
  end: Int
}
```

## Detailed Design

- `@Countable.length` returns the length of a value.
- `@Iterable.iterate` receives the value and a `yield` function, returning when
  iteration completes or `yield` returns false.
- The `for element <- value` syntax invokes `iterate` under the hood.

The internally used `yield` function has the type:

```zirric
yield: fn(Any) -> Bool
```

When starting the loop, the `iterate` function of the `@Iterable` attribute is extracted.
Then `iterate` is called. The `yield` function passed to `iterate` will execute the body of the loop.
When leaving the loop early via `break` or `return`, `yield` will return `false`, causing `iterate` to stop iteration early.
For arrays a more efficient implementation may be used that does not require function calls per element.

## Changes to the Standard Library

- Add `@Countable`, `@Iterable`, `Range`, and `ClosedRange` in `prelude`.
- `prelude.Dict`, `prelude.String` and `prelude.Array` should all be annotated with `@Countable` and `@Iterable`.

## Alternatives Considered

- A single `Iterable` attribute with optional `length`, which makes length-dependent APIs harder to check.
- Hardcoding iteration support per type, which limits extensibility.

## Acknowledgements

- Inspired by iterator protocols in Go, Swift and Python.
