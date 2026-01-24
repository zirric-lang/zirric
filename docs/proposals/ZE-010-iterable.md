---
title: "ZE-010 - Iterable"
description: "Define Countable and Iterable annotations and iteration protocols."
---

# Iterable

::: callout draft <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-search-slash-icon lucide-search-slash"><path d="m13.5 8.5-5 5"/><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg> Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design.
Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

This proposal introduces standard `@Countable` and `@Iterable` annotations for
collection-like types, which is used by `for ... <- ...` loops.
Also introduces `Range` and `ClosedRange` data types as examples of countable and iterable.

## Motivation

Zirric needs a consistent way to describe collection behavior. Without a shared
protocol, each type defines bespoke iteration helpers, and tooling cannot
recognize which values support iteration or have lengths.

Currently only arrays can be used to iterate over.

## Proposed Solution

Define two annotations in the prelude future module:

- `@Countable` for types with a length.
- `@Iterable` for types that can yield values in a `for` loop.

```zirric
@Proposal(ZE_010)
annotation Countable {
  @Returns(Int)
  length(@Has(Countable) value)
}

@Proposal(ZE_010)
annotation Iterable {
  iterate(@Has(Iterable) value, @Func yield)
}
```

### Range Types

Add `Range` and `ClosedRange` as iterable, countable data types.

```zirric
@Countable(_rangeCount)
@Iterable(_rangeIterate)
data Range { @Int start, @Int end }

@Countable(_rangeCount)
@Iterable(_rangeIterate)
data ClosedRange { @Int start, @Int end }
```

## Detailed Design

- `@Countable.length` returns the length of a value.
- `@Iterable.iterate` receives the value and a `yield` function, returning when
  iteration completes or `yield` returns false.
- The `for element <- value` syntax invokes `iterate` under the hood.

The internally used `yield` function has the signature:

```zirric
@Func
@Returns(Bool)
yield(element)
```

When starting the loop, the `iterate` function of the `@Iterable` annotation is extracted.
Then `iterate` is called. The `yield` function passed to `iterate` will execute the body of the loop.
When leaving the loop early via `break` or `return`, `yield` will return `false`, causing `iterate` to stop iteration early.
For arrays a more efficient implementation may be used that does not require function calls per element.

## Changes to the Standard Library

- Add `@Countable`, `@Iterable`, `Range`, and `ClosedRange` in
  `stdlib/prelude`.
- `prelude.Dict`, `prelude.String` and `prelude.Array` should all be annotated with `@Countable` and `@Iterable`.

## Alternatives Considered

- A single `Iterable` annotation with optional `length`, which makes
  length-dependent APIs harder to check.
- Hardcoding iteration support per type, which limits extensibility.

## Acknowledgements

- Inspired by iterator protocols in Swift and Python.
