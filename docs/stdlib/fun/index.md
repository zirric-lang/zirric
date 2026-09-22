---
title: Fun
description: Function combinators and lazy operations over anything iterable.
---

# Module `fun`

```zirric
import fun
```

|            |                                                                                                                                                                                                      |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `fun`                                                                                                                                                                                                |
| **Source** | [`fun/fun.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/fun/fun.zirr), [`fun/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/fun/module-docs.zirr) |

> Composing functions, and streaming through them.

`fun` has two halves. The first composes functions: [`pipe`](#pipe) and [`compose`](#compose) chain them, [`negate`](#negate) and [`flip`](#flip) adjust them, and [`identity`](#identity) and [`constant`](#constant) fill in where an API wants a function and you have a value.

The second is [`map`](#map), [`filter`](#filter), [`flatMap`](#flatmap), [`take`](#take), [`skip`](#skip) and [`zip`](#zip) over anything carrying [`prelude.Iterable`](../prelude/index.md#iterable). These are lazy: they build a new iterable and nothing runs until something consumes it — a `for` loop, [`reduce`](#reduce), or any other consumer. That is what lets [`take`](#take) stop an otherwise unbounded sequence.

For the eager equivalents that always walk the whole collection, see [`arrays`](../arrays/index.md).

## Contents

- **Data** — [`Zipped`](#zipped)
- **Functions** — [`compose`](#compose), [`constant`](#constant), [`filter`](#filter), [`flatMap`](#flatmap), [`flip`](#flip), [`identity`](#identity), [`map`](#map), [`negate`](#negate), [`pipe`](#pipe), [`reduce`](#reduce), [`skip`](#skip), [`take`](#take), [`with`](#with), [`zip`](#zip)

---

## Data

### `Zipped` {#zipped}

<small>`fun/fun.zirr:137`</small>

```zirric
data Zipped {
	first
	second
}
```

A pair of values at the same position in zip's two source sequences.

#### Fields

| Field    | Description |
| -------- | ----------- |
| `first`  |             |
| `second` |             |

---

## Functions

### `compose` {#compose}

<small>`fun/fun.zirr:35`</small>

```zirric
fn compose(f: fn(Any) -> Any, g: fn(Any) -> Any) -> fn(Any) -> Any
```

Combines f and g into a single function that applies g first, then f — compose(f, g)(v) is f(g(v)). The mirror image of pipe, which applies its functions left to right instead.

---

### `constant` {#constant}

<small>`fun/fun.zirr:25`</small>

```zirric
fn constant(value: Any) -> fn(Any) -> Any
```

Returns a function that always returns value, ignoring whatever it's called with.

---

### `filter` {#filter}

<small>`fun/fun.zirr:67`</small>

```zirric
fn filter(v: @Iterable, predicate: fn(Any) -> Bool) -> @Iterable
```

Returns a lazy @Iterable containing only the elements of v for which predicate returns true, in order.

---

### `flatMap` {#flatmap}

<small>`fun/fun.zirr:80`</small>

```zirric
fn flatMap(v: @Iterable, transform: fn(Any) -> @Iterable) -> @Iterable
```

Returns a lazy @Iterable that applies transform to each element of v and flattens each resulting @Iterable into a single sequence, in order.

---

### `flip` {#flip}

<small>`fun/fun.zirr:40`</small>

```zirric
fn flip(f: fn(Any, Any) -> Any) -> fn(Any, Any) -> Any
```

Returns a function that calls f with its two arguments swapped — flip(f)(a, b) is f(b, a).

---

### `identity` {#identity}

<small>`fun/fun.zirr:20`</small>

```zirric
fn identity(v: Any) -> Any
```

Returns v unchanged. Useful wherever an API expects a function but you already have the value.

---

### `map` {#map}

<small>`fun/fun.zirr:56`</small>

```zirric
fn map(v: @Iterable, transform: fn(Any) -> Any) -> @Iterable
```

Returns a lazy @Iterable that applies transform to each element of v, in order.

---

### `negate` {#negate}

<small>`fun/fun.zirr:30`</small>

```zirric
fn negate(predicate: fn(Any) -> Bool) -> fn(Any) -> Bool
```

Returns a predicate that's the boolean negation of predicate — negate(p)(v) is !p(v).

---

### `pipe` {#pipe}

<small>`fun/fun.zirr:9`</small>

```zirric
fn pipe(funs: [fn(Any) -> Any]) -> Any
```

Combines funs into a single function that applies them in order, left to right, passing each result to the next.

---

### `reduce` {#reduce}

<small>`fun/fun.zirr:93`</small>

```zirric
fn reduce(v: @Iterable, initial: Any, combine: fn(Any, Any) -> Any) -> Any
```

Combines v's elements into a single value, starting from initial and applying combine(accumulator, element) left to right. Unlike map/filter/flatMap, this consumes v eagerly.

---

### `skip` {#skip}

<small>`fun/fun.zirr:121`</small>

```zirric
fn skip(v: @Iterable, n: Int) -> @Iterable
```

Returns a lazy @Iterable that omits v's first n elements, yielding the rest unchanged.

---

### `take` {#take}

<small>`fun/fun.zirr:102`</small>

```zirric
fn take(v: @Iterable, n: Int) -> @Iterable
```

Returns a lazy @Iterable containing at most the first n elements of v.

---

### `with` {#with}

<small>`fun/fun.zirr:4`</small>

```zirric
fn with(val, fun: fn(Any))
```

Applies fun to val and returns the result. Useful for starting a pipe()-style chain from a value without naming an intermediate variable.

---

### `zip` {#zip}

<small>`fun/fun.zirr:144`</small>

```zirric
fn zip(a: @Iterable, b: @Iterable) -> @Iterable
```

Returns a lazy @Iterable of Zipped(x, y) for each x in a paired with the element at the same position in b, stopping as soon as either source is exhausted.
b is buffered into memory up front — pairing needs random access into it, which the push-based @Iterable protocol can't give incrementally for two independent sources — so b must be finite, even though a is still walked one element at a time.
