---
title: Math
description: Numeric constants, conversions, rounding and comparisons.
---

# Module `math`

```zirric
import math
```

|            |                                                                                                                                                                                                            |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `math`                                                                                                                                                                                                     |
| **Source** | [`math/math.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/math/math.zirr), [`math/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/math/module-docs.zirr) |

> Arithmetic that the operators do not cover.

`math` operates on [`prelude.Number`](../prelude/index.md#number), the union of [`prelude.Int`](../prelude/index.md#int) and [`prelude.Float`](../prelude/index.md#float). Functions that can only answer in fractions ([`sqrt`](#sqrt), [`floor`](#floor), [`pow`](#pow)) return `Float`; the ones that merely pick or transform a value ([`abs`](#abs), [`min`](#min), [`max`](#max), [`clamp`](#clamp)) return whichever kind they were given.

Conversions are never implicit — use [`toInt`](#toint) and [`toFloat`](#tofloat) when you need to cross between them.

## Contents

- **Functions** — [`abs`](#abs), [`ceil`](#ceil), [`clamp`](#clamp), [`floor`](#floor), [`isInfinite`](#isinfinite), [`isNaN`](#isnan), [`max`](#max), [`min`](#min), [`pow`](#pow), [`round`](#round), [`sign`](#sign), [`sqrt`](#sqrt), [`toFloat`](#tofloat), [`toInt`](#toint), [`trunc`](#trunc)
- **Constants** — [`e`](#e), [`pi`](#pi)

---

## Functions

### `abs` {#abs}

<small>`math/math.zirr:16`</small>

```zirric
extern fn abs(v: Number) -> Number
```

Returns v without its sign, as the same type it was given.

---

### `ceil` {#ceil}

<small>`math/math.zirr:38`</small>

```zirric
extern fn ceil(v: Number) -> Float
```

Returns the least whole number not below v.

---

### `clamp` {#clamp}

<small>`math/math.zirr:53`</small>

```zirric
fn clamp(v: Number, low: Number, high: Number) -> Number
```

Returns v limited to the range from low to high, as the type it was given.

---

### `floor` {#floor}

<small>`math/math.zirr:35`</small>

```zirric
extern fn floor(v: Number) -> Float
```

Returns the greatest whole number not above v.

---

### `isInfinite` {#isinfinite}

<small>`math/math.zirr:29`</small>

```zirric
extern fn isInfinite(v: Number) -> Bool
```

Returns whether v is positive or negative infinity, as produced by dividing a non-zero Float by zero. An Int is never infinite, and NaN is not infinite either.

---

### `isNaN` {#isnan}

<small>`math/math.zirr:26`</small>

```zirric
extern fn isNaN(v: Number) -> Bool
```

Returns whether v is not a number, as produced by an undefined operation such as 0.0 / 0.0 or the square root of a negative. An Int is never NaN.
NaN is equal to nothing, itself included, so this is the only way to detect it.

---

### `max` {#max}

<small>`math/math.zirr:22`</small>

```zirric
extern fn max(a: Number, b: Number) -> Number
```

Returns whichever of a and b is larger, as the type it was given.

---

### `min` {#min}

<small>`math/math.zirr:19`</small>

```zirric
extern fn min(a: Number, b: Number) -> Number
```

Returns whichever of a and b is smaller, as the type it was given.

---

### `pow` {#pow}

<small>`math/math.zirr:50`</small>

```zirric
extern fn pow(base: Number, exponent: Number) -> Float
```

Returns base raised to the power of exponent.

---

### `round` {#round}

<small>`math/math.zirr:41`</small>

```zirric
extern fn round(v: Number) -> Float
```

Returns v rounded to the nearest whole number, halves away from zero.

---

### `sign` {#sign}

<small>`math/math.zirr:32`</small>

```zirric
extern fn sign(v: Number) -> Int
```

Returns -1, 0 or 1 according to whether v is negative, zero or positive.

---

### `sqrt` {#sqrt}

<small>`math/math.zirr:47`</small>

```zirric
extern fn sqrt(v: Number) -> Float
```

Returns the square root of v.

---

### `toFloat` {#tofloat}

<small>`math/math.zirr:10`</small>

```zirric
extern fn toFloat(v: Number) -> Float
```

Returns v as a Float.

---

### `toInt` {#toint}

<small>`math/math.zirr:13`</small>

```zirric
extern fn toInt(v: Number) -> Int
```

Returns v as an Int, discarding any fractional part rather than rounding.

---

### `trunc` {#trunc}

<small>`math/math.zirr:44`</small>

```zirric
extern fn trunc(v: Number) -> Float
```

Returns v with its fractional part discarded, rounding toward zero.

---

## Constants

### `e` {#e}

<small>`math/math.zirr:7`</small>

```zirric
extern const e: Float
```

Euler's number, the base of the natural logarithm.

---

### `pi` {#pi}

<small>`math/math.zirr:4`</small>

```zirric
extern const pi: Float
```

The ratio of a circle's circumference to its diameter.
