---
title: Results
description: Working with values that may have failed.
---

# Module `results`

```zirric
import results
```

|            |                                                                                                                                                                                                                              |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `results`                                                                                                                                                                                                                    |
| **Source** | [`results/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/results/module-docs.zirr), [`results/results.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/results/results.zirr) |

> Helpers for `Result`, and for anything carrying `@AnyResult`.

[`prelude.Result`](../prelude/index.md#result) is the type; `results` is what you use it with.

Every function takes an [`@prelude.AnyResult`](../prelude/index.md#anyresult) rather than a `Result`, so a library can define its own Ok/Err union and still pass it here — [`from`](#from) converts through the attribute. The pairs are deliberate: [`map`](#map) and [`mapErr`](#maperr) transform a value, [`flatMap`](#flatmap) and [`flatMapErr`](#flatmaperr) transform it into something that can fail again, and [`catch`](#catch) and [`catchTo`](#catchto) turn a failure back into a success.

## Contents

- **Functions** — [`all`](#all), [`catch`](#catch), [`catchTo`](#catchto), [`flatMap`](#flatmap), [`flatMapErr`](#flatmaperr), [`from`](#from), [`isErr`](#iserr), [`isOk`](#isok), [`map`](#map), [`mapErr`](#maperr)

---

## Functions

### `all` {#all}

<small>`results/results.zirr:85`</small>

```zirric
fn all(ress: [@AnyResult]) -> Result
```

Combines ress into a single Result: Ok of every value if all are Ok, or Err of every reason if any are Err.

---

### `catch` {#catch}

<small>`results/results.zirr:63`</small>

```zirric
fn catch(r: @AnyResult, handler: fn(Any) -> Any) -> Result
```

Recovers r's error by calling handler with its reason and wrapping the result as Ok, or returns r unchanged if it's already Ok.

---

### `catchTo` {#catchto}

<small>`results/results.zirr:74`</small>

```zirric
fn catchTo(r: @AnyResult, default: Any) -> Result
```

Recovers r's error by replacing it with Ok(default), or returns r unchanged if it's already Ok.

---

### `flatMap` {#flatmap}

<small>`results/results.zirr:41`</small>

```zirric
fn flatMap(r: @AnyResult, transform: fn(Any) -> @AnyResult) -> Result
```

Returns transform(r's value) normalized to Result, or r's error unchanged. Use this instead of map when transform can itself fail.

---

### `flatMapErr` {#flatmaperr}

<small>`results/results.zirr:52`</small>

```zirric
fn flatMapErr(r: @AnyResult, transform: fn(Any) -> @AnyResult) -> Result
```

Returns transform(r's error reason) normalized to Result, or r's value unchanged. Use this instead of mapErr to recover with a value that might itself be an error.

---

### `from` {#from}

<small>`results/results.zirr:4`</small>

```zirric
fn from(r: @AnyResult) -> Result
```

Normalizes r to Result.

---

### `isErr` {#iserr}

<small>`results/results.zirr:14`</small>

```zirric
fn isErr(r: @AnyResult) -> Bool
```

Returns whether r is Err.

---

### `isOk` {#isok}

<small>`results/results.zirr:9`</small>

```zirric
fn isOk(r: @AnyResult) -> Bool
```

Returns whether r is Ok.

---

### `map` {#map}

<small>`results/results.zirr:19`</small>

```zirric
fn map(r: @AnyResult, transform: fn(Any) -> Any) -> Result
```

Returns a Result with transform applied to r's value, or r's error unchanged.

---

### `mapErr` {#maperr}

<small>`results/results.zirr:30`</small>

```zirric
fn mapErr(r: @AnyResult, transform: fn(Any) -> Any) -> Result
```

Returns a Result with transform applied to r's error reason, or r's value unchanged.
