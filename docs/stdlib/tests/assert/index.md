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

```zirric
@Test()
fn testSplitsOnEverySeparator() {
	assert.all([
		assert.equal(["a", "b"], strings.split("a,b", ",")),
		assert.equal([""], strings.split("", ",")),
	])
}
```

See [`tests`](../index.md) for where a test module goes and how `zirric test` finds it.

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

<small>`tests/assert/basic.zirr:8`</small>

```zirric
fn all(res: [@AnyResult]) -> Result
```

Combines several assertions into one: an `Ok` when every one of them is, and otherwise an `Err` naming every failure.
Nothing short-circuits, so a test written with `all` reports every check that failed rather than only the first.

---

### `equal` {#equal}

<small>`tests/assert/basic.zirr:14`</small>

```zirric
fn equal(expect, got) -> Result
```

An `Ok` when the two values are equal, and otherwise an `Err` naming both.
Equality is the language's own `==`, so a `data` value is compared field by field.

---

### `fail` {#fail}

<small>`tests/assert/basic.zirr:81`</small>

```zirric
fn fail(reason) -> Result
```

Always an `Err` with the given reason. For the branch a test should never reach.

---

### `isErr` {#iserr}

<small>`tests/assert/basic.zirr:53`</small>

```zirric
fn isErr(val) -> Result
```

An `Ok` when the value is a failed result, reading it through `@AnyResult` when it is not an `Ok` or an `Err` itself.

---

### `isError` {#iserror}

<small>`tests/assert/basic.zirr:71`</small>

```zirric
fn isError(val) -> Result
```

An `Ok` when the value's type carries `@Error`, whatever else it is.

---

### `isFalse` {#isfalse}

<small>`tests/assert/basic.zirr:28`</small>

```zirric
fn isFalse(val) -> Result
```

An `Ok` when the value is `false`.

---

### `isOk` {#isok}

<small>`tests/assert/basic.zirr:33`</small>

```zirric
fn isOk(val) -> Result
```

An `Ok` when the value is a successful result, reading it through `@AnyResult` when it is not an `Ok` or an `Err` itself.

---

### `isTrue` {#istrue}

<small>`tests/assert/basic.zirr:23`</small>

```zirric
fn isTrue(val) -> Result
```

An `Ok` when the value is `true`.
