---
title: Tests.Assert
description: Assertions that produce the Result a test returns.
---

# Module `tests.assert`

```zirric
import tests.assert
```

|            |                                                                                                                                                                                                                                              |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `tests.assert`                                                                                                                                                                                                                               |
| **Source** | [`tests/assert/basic.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/tests/assert/basic.zirr), [`tests/assert/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/tests/assert/module-docs.zirr) |

> Assertions are values, not control flow.

An assertion here returns a [`prelude.Result`](../../prelude/index.md#result) rather than aborting. A test returns one, and that is its outcome — so a failure is an ordinary value you can pass around, and [`all`](#all) combines several into one.

Because they are values, nothing stops after the first failure unless you make it. Return a single assertion for a single check, or gather several with [`all`](#all) to report every one.

## Dependencies

- [`fmt`](../../fmt/index.md)
  - [`bytes`](../../bytes/index.md)
    - [`ranges`](../../ranges/index.md)
  - [`io`](../../io/index.md)
- [`results`](../../results/index.md)

---

## Contents

- **Functions** — [`all`](#all), [`equal`](#equal), [`fail`](#fail), [`isErr`](#iserr), [`isError`](#iserror), [`isFalse`](#isfalse), [`isOk`](#isok), [`isTrue`](#istrue)

---

## Functions

### `all` {#all}

<small>`tests/assert/basic.zirr:6`</small>

```zirric
fn all(res: [@AnyResult]) -> Result
```

---

### `equal` {#equal}

<small>`tests/assert/basic.zirr:10`</small>

```zirric
fn equal(expect, got) -> Result
```

---

### `fail` {#fail}

<small>`tests/assert/basic.zirr:66`</small>

```zirric
fn fail(reason) -> Result
```

---

### `isErr` {#iserr}

<small>`tests/assert/basic.zirr:43`</small>

```zirric
fn isErr(val) -> Result
```

---

### `isError` {#iserror}

<small>`tests/assert/basic.zirr:58`</small>

```zirric
fn isError(val) -> Result
```

---

### `isFalse` {#isfalse}

<small>`tests/assert/basic.zirr:22`</small>

```zirric
fn isFalse(val) -> Result
```

---

### `isOk` {#isok}

<small>`tests/assert/basic.zirr:26`</small>

```zirric
fn isOk(val) -> Result
```

---

### `isTrue` {#istrue}

<small>`tests/assert/basic.zirr:18`</small>

```zirric
fn isTrue(val) -> Result
```
