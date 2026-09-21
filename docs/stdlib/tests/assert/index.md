---
title: Tests.Assert
description: Assertions that produce the Result a test returns.
---

# Module `tests.assert`

> Assertions are values, not control flow.

```zirric
import tests.assert
```

|            |                                                                                                                 |
| ---------- | --------------------------------------------------------------------------------------------------------------- |
| **Module** | `tests.assert`                                                                                                  |
| **Source** | [`tests/assert/basic.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/tests/assert/basic.zirr) |

An assertion here returns a [`Result`](../../prelude/index.md#result) rather than aborting. A test returns one, and that is its outcome — so a failure is an ordinary value you can pass around, and [`all`](#all) combines several into one.

Because they are values, nothing stops after the first failure unless you make it. Return a single assertion for a single check, or gather several with `all` to report every one.

## Contents

- **Functions** — [`all`](#all), [`equal`](#equal), [`isTrue`](#istrue), [`isFalse`](#isfalse), [`isOk`](#isok), [`isErr`](#iserr), [`isError`](#iserror), [`fail`](#fail)

---

## Functions

### `all` {#all}

<small>`tests/assert/basic.zirr:6`</small>

```zirric
fn all(res: [@AnyResult]) -> Result
```

Combines several assertions into one, which is `Ok` only if every one of them is. Every failure is reported, not just the first.

---

### `equal` {#equal}

<small>`tests/assert/basic.zirr:10`</small>

```zirric
fn equal(expect, got) -> Result
```

Passes when the two values are equal, and otherwise fails with both of them rendered.

---

### `isTrue` {#istrue}

<small>`tests/assert/basic.zirr:18`</small>

```zirric
fn isTrue(val) -> Result
```

Passes when the value is `true`.

---

### `isFalse` {#isfalse}

<small>`tests/assert/basic.zirr:22`</small>

```zirric
fn isFalse(val) -> Result
```

Passes when the value is `false`.

---

### `isOk` {#isok}

<small>`tests/assert/basic.zirr:26`</small>

```zirric
fn isOk(val) -> Result
```

Passes when the value is an `Ok`.

---

### `isErr` {#iserr}

<small>`tests/assert/basic.zirr:43`</small>

```zirric
fn isErr(val) -> Result
```

Passes when the value is an `Err`.

---

### `isError` {#iserror}

<small>`tests/assert/basic.zirr:58`</small>

```zirric
fn isError(val) -> Result
```

Passes when the value carries the [`Error`](../../prelude/index.md#error) attribute.

---

### `fail` {#fail}

<small>`tests/assert/basic.zirr:66`</small>

```zirric
fn fail(reason) -> Result
```

Always fails, with the given reason. Use it for a branch that should not have been reached.

---

## See also

- [`tests`](../index.md) — the attributes that mark a function as a test.
- [`results`](../../results/index.md) — combining and transforming what these return.
