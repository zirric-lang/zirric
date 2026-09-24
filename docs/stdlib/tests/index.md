---
title: Tests
description: The attributes and types that describe a test, its outcome and its report.
---

# Module `tests`

```zirric
import tests
```

|            |                                                                                                                                                                                                                  |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `tests`                                                                                                                                                                                                          |
| **Source** | [`tests/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/tests/module-docs.zirr), [`tests/types.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/tests/types.zirr) |

> What a test is, before anything runs it.

A test is a function carrying [`Test`](#test) that returns a [`prelude.Result`](../prelude/index.md#result). Nothing else is required: there is no registry and no base type, so the runner reaches for [`reflect`](../reflect/index.md) and finds tests by reading a module rather than by being told about them.

The other attributes adjust what happens to one. [`Skip`](#skip) keeps it out of the run, [`Todo`](#todo) marks it as expected to be unfinished, [`Only`](#only) narrows the run to itself, and [`Comment`](#comment) attaches a note that reporters print.

This module is only the vocabulary. [`tests.assert`](../tests/assert/index.md) writes the assertions, [`tests.runner`](../tests/runner/index.md) finds and runs them, and [`tests.tap`](../tests/tap/index.md) reports what happened.

## Where tests live

`zirric test` runs every module of the project whose name ends in `_t`. Both shapes count: a `foo/_t/foo_t.zirr` submodule beside the code it tests, and a sibling module named `foo_t`. Nothing has to be registered — the name is the whole convention, and a project has a working `zirric test` before it has written anything about testing. A `Cavefile` that declares a `test` task takes over from the built-in entirely.

```zirric
mod example.strings._t

import tests { Test, Skip, Comment }
import tests.assert
import strings

@Test()
fn testJoinsWithSeparator() {
	assert.equal("a,b", strings.join(["a", "b"], ","))
}

@Test()
fn testReportsEveryFailedCheck() {
	assert.all([
		assert.equal(0, len("")),
		assert.isTrue(strings.hasPrefix("abc", "a")),
	])
}

@Skip()
@Comment("waiting on the parser")
fn testNotYet() {
	assert.fail("unreachable")
}
```

A test takes no arguments and returns a [`prelude.Result`](../prelude/index.md#result) — which the assertions above already are, so returning one is the whole body. `zirric test` exits non-zero when any of them is an `Err`.

## Contents

- **Unions** — [`TestCaseDetails`](#testcasedetails), [`TestEvent`](#testevent)
- **Data** — [`Completed`](#completed), [`Discovered`](#discovered), [`FailureRecord`](#failurerecord), [`Finished`](#finished), [`Skipped`](#skipped), [`Started`](#started), [`TestCase`](#testcase), [`TestCaseDetailsOnly`](#testcasedetailsonly), [`TestCaseDetailsSkip`](#testcasedetailsskip), [`TestCaseDetailsTodo`](#testcasedetailstodo), [`TestReport`](#testreport)
- **Attributes** — [`Comment`](#comment), [`Only`](#only), [`Skip`](#skip), [`Test`](#test), [`Todo`](#todo)

---

## Unions

### `TestCaseDetails` {#testcasedetails}

<small>`tests/types.zirr:47`</small>

```zirric
union TestCaseDetails {
	None
	TestCaseDetailsSkip
	TestCaseDetailsTodo
	TestCaseDetailsOnly
}
```

What [`Skip`](#skip), [`Todo`](#todo) or [`Only`](#only) was written on a test, or `None` when none of them was.

Each member says how to be read as an Option, the way prelude's own Result members do: a union's attributes are not read off a value of one of its members.

#### Cases

| Case                  | Interpretation                    |
| --------------------- | --------------------------------- |
| `None`                | The absent value.                 |
| `TestCaseDetailsSkip` | The test carries [`Skip`](#skip). |
| `TestCaseDetailsTodo` | The test carries [`Todo`](#todo). |
| `TestCaseDetailsOnly` | The test carries [`Only`](#only). |

---

### `TestEvent` {#testevent}

<small>`tests/types.zirr:63`</small>

```zirric
union TestEvent {
	Discovered
	Skipped
	Started
	Finished
	Completed
}
```

What a run emits as it proceeds, in the order a reporter sees it.

#### Cases

| Case         | Interpretation                                                          |
| ------------ | ----------------------------------------------------------------------- |
| `Discovered` | Emitted once, before anything runs, with every case the run will cover. |
| `Skipped`    | A case that will not run.                                               |
| `Started`    | A case that is about to run.                                            |
| `Finished`   | A case that has run, with what it returned.                             |
| `Completed`  | Emitted once, after everything has run.                                 |

---

## Data

### `Completed` {#completed}

<small>`tests/types.zirr:87`</small>

```zirric
data Completed {
	report: TestReport
}
```

Emitted once, after everything has run.

#### Fields

| Field    | Description               |
| -------- | ------------------------- |
| `report` | What the run amounted to. |

---

### `Discovered` {#discovered}

<small>`tests/types.zirr:65`</small>

```zirric
data Discovered {
	cases: [TestCase]
}
```

Emitted once, before anything runs, with every case the run will cover.

#### Fields

| Field   | Description                                           |
| ------- | ----------------------------------------------------- |
| `cases` | Every case the run will cover, skipped ones included. |

---

### `FailureRecord` {#failurerecord}

<small>`tests/types.zirr:106`</small>

```zirric
data FailureRecord {
	test: TestCase
	error: Err
}
```

A failed case together with the error it returned.

#### Fields

| Field   | Description                                                 |
| ------- | ----------------------------------------------------------- |
| `test`  | The case that failed.                                       |
| `error` | The `Err` it returned, whose `reason` says what went wrong. |

---

### `Finished` {#finished}

<small>`tests/types.zirr:80`</small>

```zirric
data Finished {
	test: TestCase
	outcome: Result
}
```

A case that has run, with what it returned.

#### Fields

| Field     | Description                                                                 |
| --------- | --------------------------------------------------------------------------- |
| `test`    | The case that ran.                                                          |
| `outcome` | What it returned: an `Ok`, or an `Err` whose `reason` says what went wrong. |

---

### `Skipped` {#skipped}

<small>`tests/types.zirr:70`</small>

```zirric
data Skipped {
	test: TestCase
}
```

A case that will not run.

#### Fields

| Field  | Description             |
| ------ | ----------------------- |
| `test` | The case being skipped. |

---

### `Started` {#started}

<small>`tests/types.zirr:75`</small>

```zirric
data Started {
	test: TestCase
}
```

A case that is about to run.

#### Fields

| Field  | Description            |
| ------ | ---------------------- |
| `test` | The case about to run. |

---

### `TestCase` {#testcase}

<small>`tests/types.zirr:23`</small>

```zirric
data TestCase {
	name: String
	impl: fn() -> Result
	skipped: Bool
	comment: String
	details: TestCaseDetails
}
```

One test, as the runner will execute it.

#### Fields

| Field     | Description                                                                                |
| --------- | ------------------------------------------------------------------------------------------ |
| `name`    | The fully qualified name, which is the module's name and the function's joined with a dot. |
| `impl`    | The test function itself.                                                                  |
| `skipped` | Whether the runner will report it without running it.                                      |
| `comment` | The [`Comment`](#comment) written on it, or the empty String when none was.                |
| `details` | What [`Skip`](#skip), [`Todo`](#todo) or [`Only`](#only) was written on it, if any.        |

---

### `TestCaseDetailsOnly` {#testcasedetailsonly}

<small>`tests/types.zirr:59`</small>

```zirric
data TestCaseDetailsOnly
```

The test carries [`Only`](#only).

---

### `TestCaseDetailsSkip` {#testcasedetailsskip}

<small>`tests/types.zirr:53`</small>

```zirric
data TestCaseDetailsSkip
```

The test carries [`Skip`](#skip).

---

### `TestCaseDetailsTodo` {#testcasedetailstodo}

<small>`tests/types.zirr:56`</small>

```zirric
data TestCaseDetailsTodo
```

The test carries [`Todo`](#todo).

---

### `TestReport` {#testreport}

<small>`tests/types.zirr:94`</small>

```zirric
data TestReport {
	pass: [TestCase]
	fail: [FailureRecord]
	skip: [TestCase]
	todo: [TestCase]
}
```

What a run amounted to. A case appears in exactly one of the four.

#### Fields

| Field  | Description                                                     |
| ------ | --------------------------------------------------------------- |
| `pass` | The cases that returned an `Ok`.                                |
| `fail` | The cases that returned an `Err`, with the error each returned. |
| `skip` | The cases that carried [`Skip`](#skip) and so were not run.     |
| `todo` | The cases that carried [`Todo`](#todo), whatever they returned. |

---

## Attributes

### `Comment` {#comment}

<small>`tests/types.zirr:17`</small>

```zirric
attr Comment {
	text: String
}
```

Attaches a note to a test that reporters print alongside it.

#### Fields

| Field  | Description        |
| ------ | ------------------ |
| `text` | The note to print. |

---

### `Only` {#only}

<small>`tests/types.zirr:11`</small>

```zirric
attr Only
```

Narrows the run to the tests carrying it. With no such test, every test runs.

---

### `Skip` {#skip}

<small>`tests/types.zirr:8`</small>

```zirric
attr Skip
```

Keeps a test out of the run. It is reported as skipped rather than omitted.

---

### `Test` {#test}

<small>`tests/types.zirr:5`</small>

```zirric
attr Test
```

Marks a function as a test. The function takes no arguments and returns a `Result`.
This is the whole of what makes a test: the runner finds functions carrying it by reading a module through [`reflect`](../reflect/index.md).

---

### `Todo` {#todo}

<small>`tests/types.zirr:14`</small>

```zirric
attr Todo
```

Marks a test as expected to be unfinished. It runs, and its outcome is reported as a todo either way.
