---
title: Random
description: Fast, seeded and cryptographic sources of randomness.
---

# Module `random`

> Randomness as a value, so a test can pin it down.

```zirric
import random
```

|            |                                                                                                                                                                                                                                                                                                                               |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `random`                                                                                                                                                                                                                                                                                                                      |
| **Source** | [`random/random.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/random/random.zirr), [`random/operations.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/random/operations.zirr), [`random/helpers.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/random/helpers.zirr) |

A [`Source`](#source) is a value carrying four functions, and every function here takes one as its first argument. Which source you pass decides where the bits come from; the code using it does not change.

The three constructors differ only in that: [`fast()`](#fast) is seeded from the clock, [`seeded(n)`](#seeded) produces the same sequence every run, and [`strong()`](#strong) uses the operating system's cryptographic generator.

[`HasFastRandom`](#hasfastrandom) and [`HasStrongRandom`](#hasstrongrandom) are separate attributes so that code needing secrecy has to say so, and cannot be handed a seeded generator by mistake.

## Contents

- **Attributes** — [`HasFastRandom`](#hasfastrandom), [`HasStrongRandom`](#hasstrongrandom)
- **Data** — [`Source`](#source)
- **Functions** — [`fast`](#fast), [`seeded`](#seeded), [`strong`](#strong), [`int`](#int), [`float`](#float), [`bytes`](#bytes), [`bool`](#bool), [`intBetween`](#intbetween), [`floatBetween`](#floatbetween), [`choice`](#choice), [`shuffle`](#shuffle)

---

## Attributes

### `HasFastRandom` {#hasfastrandom}

<small>`random/random.zirr:5`</small>

```zirric
attr HasFastRandom {
	random(self: @HasFastRandom) -> Source
}
```

Provides a fast source of randomness, for simulations, sampling and anything else where predictability is not a hazard. A seeded source satisfies this, which is what lets a test pin down behaviour that depends on randomness.

#### Members

| Member   | Signature                                | Description                                                |
| -------- | ---------------------------------------- | ---------------------------------------------------------- |
| `random` | `random(self: @HasFastRandom) -> Source` | A fast source. May be seeded, so never use it for secrets. |

---

### `HasStrongRandom` {#hasstrongrandom}

<small>`random/random.zirr:11`</small>

```zirric
attr HasStrongRandom {
	random(self: @HasStrongRandom) -> Source
}
```

Provides a cryptographically strong source of randomness, for tokens, keys and anything an adversary should not predict. Kept apart from HasFastRandom so that code needing secrecy says so, and cannot be handed a seeded generator by mistake.

#### Members

| Member   | Signature                                  | Description                        |
| -------- | ------------------------------------------ | ---------------------------------- |
| `random` | `random(self: @HasStrongRandom) -> Source` | A cryptographically strong source. |

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

A source of randomness. The three constructors below differ only in where the bits come from, so code written against a Source works with any of them. Each field has a matching module function taking the source first, which is usually the nicer way to call it.

#### Fields

| Field   | Signature                  | Description            |
| ------- | -------------------------- | ---------------------- |
| `int`   | `int: fn(Int) -> Int`      | See [`int`](#int).     |
| `float` | `float: fn() -> Float`     | See [`float`](#float). |
| `bytes` | `bytes: fn(Int) -> Binary` | See [`bytes`](#bytes). |
| `bool`  | `bool: fn() -> Bool`       | See [`bool`](#bool).   |

---

## Functions

### `fast` {#fast}

<small>`random/random.zirr:25`</small>

```zirric
extern fn fast() -> Source
```

A fast generator seeded from the clock. Suitable for simulations and sampling, never for secrets.

---

### `seeded` {#seeded}

<small>`random/random.zirr:29`</small>

```zirric
extern fn seeded(seed: Int) -> Source
```

A fast generator seeded with the given value, producing the same sequence every run. This is what makes code using randomness testable: pass a seeded source and the outcome is fixed.

---

### `strong` {#strong}

<small>`random/random.zirr:32`</small>

```zirric
extern fn strong() -> Source
```

The operating system's cryptographic generator. Slower, and the only one to use for tokens, keys or anything an adversary should not predict.

---

### `int` {#int}

<small>`random/operations.zirr:6`</small>

```zirric
fn int(source: Source, bound: Int) -> Int
```

Returns a whole number from 0 up to but excluding bound, which must be above zero.

---

### `float` {#float}

<small>`random/operations.zirr:11`</small>

```zirric
fn float(source: Source) -> Float
```

Returns a number from 0 up to but excluding 1.

---

### `bytes` {#bytes}

<small>`random/operations.zirr:16`</small>

```zirric
fn bytes(source: Source, count: Int) -> Binary
```

Returns count random bytes.

---

### `bool` {#bool}

<small>`random/operations.zirr:21`</small>

```zirric
fn bool(source: Source) -> Bool
```

Returns true or false with equal likelihood.

---

### `intBetween` {#intbetween}

<small>`random/helpers.zirr:6`</small>

```zirric
fn intBetween(source: Source, low: Int, high: Int) -> Int
```

Returns a whole number from low up to but excluding high.

---

### `floatBetween` {#floatbetween}

<small>`random/helpers.zirr:11`</small>

```zirric
fn floatBetween(source: Source, low: Float, high: Float) -> Float
```

Returns a number from low up to but excluding high.

---

### `choice` {#choice}

<small>`random/helpers.zirr:16`</small>

```zirric
fn choice(source: Source, items: [Any]) -> Option
```

Returns one element of items, or None if items is empty.

---

### `shuffle` {#shuffle}

<small>`random/helpers.zirr:24`</small>

```zirric
fn shuffle(source: Source, items: [Any]) -> [Any]
```

Returns items in a new order, leaving the original untouched.

---

## See also

- [`math`](../math/index.md) — computing numbers rather than drawing them.
- [`arrays`](../arrays/index.md) — the arrays [`choice`](#choice) and [`shuffle`](#shuffle) work over.
