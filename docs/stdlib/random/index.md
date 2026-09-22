---
title: Random
description: Seeded randomness, and the operations every source shares.
---

# Module `random`

```zirric
import random
```

|            |                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `random`                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| **Source** | [`random/helpers.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/random/helpers.zirr), [`random/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/random/module-docs.zirr), [`random/operations.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/random/operations.zirr), [`random/random.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/random/random.zirr) |

> Randomness as a value, so a test can pin it down.

A [`Source`](#source) is a value carrying four functions, and every function here takes one as its first argument. Which source you pass decides where the bits come from; the code using it does not change.

`seeded(n)` is the only source built here, and it produces the same sequence every run. The two drawing on the host live in [`os`](../os/index.md), which is where everything reaching the machine enters: `os.fastRandom()` is seeded from the clock, and `os.strongRandom()` is the host's cryptographic generator.

[`HasFastRandom`](#hasfastrandom) and [`HasStrongRandom`](#hasstrongrandom) are separate attributes so that code needing secrecy has to say so, and cannot be handed a seeded generator by mistake.

## Dependencies

- [`arrays`](../arrays/index.md)
  - [`ranges`](../ranges/index.md)

---

## Contents

- **Data** — [`Source`](#source)
- **Attributes** — [`HasFastRandom`](#hasfastrandom), [`HasStrongRandom`](#hasstrongrandom)
- **Functions** — [`bool`](#bool), [`bytes`](#bytes), [`choice`](#choice), [`float`](#float), [`floatBetween`](#floatbetween), [`int`](#int), [`intBetween`](#intbetween), [`seeded`](#seeded), [`shuffle`](#shuffle)

---

## Data

### `Source` {#source}

<small>`random/random.zirr:17`</small>

```zirric
data Source {
	int: fn(Int) -> Int
	float: fn() -> Float
	bytes: fn(Int) -> Binary
	bool: fn() -> Bool
}
```

A source of randomness. The constructors differ only in where the bits come from, so code written against a Source works with any of them.
Each field has a matching module function taking the source first, which is usually the nicer way to call it.

#### Fields

| Field   | Description |
| ------- | ----------- |
| `int`   |             |
| `float` |             |
| `bytes` |             |
| `bool`  |             |

---

## Attributes

### `HasFastRandom` {#hasfastrandom}

<small>`random/random.zirr:5`</small>

```zirric
attr HasFastRandom {
	random(self: @HasFastRandom) -> Source
}
```

Provides a fast source of randomness, for simulations, sampling and anything else where predictability is not a hazard.
A seeded source satisfies this, which is what lets a test pin down behaviour that depends on randomness.

#### Fields

| Field    | Description |
| -------- | ----------- |
| `random` |             |

---

### `HasStrongRandom` {#hasstrongrandom}

<small>`random/random.zirr:11`</small>

```zirric
attr HasStrongRandom {
	random(self: @HasStrongRandom) -> Source
}
```

Provides a cryptographically strong source of randomness, for tokens, keys and anything an adversary should not predict.
Kept apart from HasFastRandom so that code needing secrecy says so, and cannot be handed a seeded generator by mistake.

#### Fields

| Field    | Description |
| -------- | ----------- |
| `random` |             |

---

## Functions

### `bool` {#bool}

<small>`random/operations.zirr:21`</small>

```zirric
fn bool(source: Source) -> Bool
```

Returns true or false with equal likelihood.

---

### `bytes` {#bytes}

<small>`random/operations.zirr:16`</small>

```zirric
fn bytes(source: Source, count: Int) -> Binary
```

Returns count random bytes.

---

### `choice` {#choice}

<small>`random/helpers.zirr:16`</small>

```zirric
fn choice(source: Source, items: [Any]) -> Option
```

Returns one element of items, or None if items is empty.

---

### `float` {#float}

<small>`random/operations.zirr:11`</small>

```zirric
fn float(source: Source) -> Float
```

Returns a number from 0 up to but excluding 1.

---

### `floatBetween` {#floatbetween}

<small>`random/helpers.zirr:11`</small>

```zirric
fn floatBetween(source: Source, low: Float, high: Float) -> Float
```

Returns a number from low up to but excluding high.

---

### `int` {#int}

<small>`random/operations.zirr:6`</small>

```zirric
fn int(source: Source, bound: Int) -> Int
```

Returns a whole number from 0 up to but excluding bound, which must be above zero.

---

### `intBetween` {#intbetween}

<small>`random/helpers.zirr:6`</small>

```zirric
fn intBetween(source: Source, low: Int, high: Int) -> Int
```

Returns a whole number from low up to but excluding high.

---

### `seeded` {#seeded}

<small>`random/random.zirr:27`</small>

```zirric
extern fn seeded(seed: Int) -> Source
```

A generator seeded with the given value, producing the same sequence every run.
This is what makes code using randomness testable: pass a seeded source and the outcome is fixed.
It is the only source this module builds; the two that draw on the host are [`os.fastRandom`](../os/index.md#fastrandom) and [`os.strongRandom`](../os/index.md#strongrandom).

---

### `shuffle` {#shuffle}

<small>`random/helpers.zirr:24`</small>

```zirric
fn shuffle(source: Source, items: [Any]) -> [Any]
```

Returns items in a new order, leaving the original untouched.
