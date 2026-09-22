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

## Contents

- **Unions** — [`TestCaseDetails`](#testcasedetails), [`TestEvent`](#testevent)
- **Data** — [`Completed`](#completed), [`Discovered`](#discovered), [`FailureRecord`](#failurerecord), [`Finished`](#finished), [`Skipped`](#skipped), [`Started`](#started), [`TestCase`](#testcase), [`TestCaseDetailsOnly`](#testcasedetailsonly), [`TestCaseDetailsSkip`](#testcasedetailsskip), [`TestCaseDetailsTodo`](#testcasedetailstodo), [`TestReport`](#testreport)
- **Attributes** — [`Comment`](#comment), [`Only`](#only), [`Skip`](#skip), [`Test`](#test), [`Todo`](#todo)

---

## Unions

### `TestCaseDetails` {#testcasedetails}

<small>`tests/types.zirr:28`</small>

```zirric
union TestCaseDetails {
	None
	TestCaseDetailsSkip
	TestCaseDetailsTodo
	TestCaseDetailsOnly
}
```

Each member says how to be read as an Option, the way prelude's own Result members do: a union's attributes are not read off a value of one of its members.

#### Cases

| Case                  | Interpretation    |
| --------------------- | ----------------- |
| `None`                | The absent value. |
| `TestCaseDetailsSkip` |                   |
| `TestCaseDetailsTodo` |                   |
| `TestCaseDetailsOnly` |                   |

---

### `TestEvent` {#testevent}

<small>`tests/types.zirr:39`</small>

```zirric
union TestEvent {
	Discovered
	Skipped
	Started
	Finished
	Completed
}
```

#### Cases

| Case         | Interpretation |
| ------------ | -------------- |
| `Discovered` |                |
| `Skipped`    |                |
| `Started`    |                |
| `Finished`   |                |
| `Completed`  |                |

---

## Data

### `Completed` {#completed}

<small>`tests/types.zirr:44`</small>

```zirric
data Completed {
	report: TestReport
}
```

#### Fields

| Field    | Description |
| -------- | ----------- |
| `report` |             |

---

### `Discovered` {#discovered}

<small>`tests/types.zirr:40`</small>

```zirric
data Discovered {
	cases: [TestCase]
}
```

#### Fields

| Field   | Description |
| ------- | ----------- |
| `cases` |             |

---

### `FailureRecord` {#failurerecord}

<small>`tests/types.zirr:54`</small>

```zirric
data FailureRecord {
	test: TestCase
	error: Err
}
```

#### Fields

| Field   | Description |
| ------- | ----------- |
| `test`  |             |
| `error` |             |

---

### `Finished` {#finished}

<small>`tests/types.zirr:43`</small>

```zirric
data Finished {
	test: TestCase
	outcome: Result
}
```

#### Fields

| Field     | Description |
| --------- | ----------- |
| `test`    |             |
| `outcome` |             |

---

### `Skipped` {#skipped}

<small>`tests/types.zirr:41`</small>

```zirric
data Skipped {
	test: TestCase
}
```

#### Fields

| Field  | Description |
| ------ | ----------- |
| `test` |             |

---

### `Started` {#started}

<small>`tests/types.zirr:42`</small>

```zirric
data Started {
	test: TestCase
}
```

#### Fields

| Field  | Description |
| ------ | ----------- |
| `test` |             |

---

### `TestCase` {#testcase}

<small>`tests/types.zirr:11`</small>

```zirric
data TestCase {
	name: String
	impl: fn() -> Result
	skipped: Bool
	comment: String
	details: TestCaseDetails
}
```

#### Fields

| Field     | Description |
| --------- | ----------- |
| `name`    |             |
| `impl`    |             |
| `skipped` |             |
| `comment` |             |
| `details` |             |

---

### `TestCaseDetailsOnly` {#testcasedetailsonly}

<small>`tests/types.zirr:36`</small>

```zirric
data TestCaseDetailsOnly
```

---

### `TestCaseDetailsSkip` {#testcasedetailsskip}

<small>`tests/types.zirr:32`</small>

```zirric
data TestCaseDetailsSkip
```

---

### `TestCaseDetailsTodo` {#testcasedetailstodo}

<small>`tests/types.zirr:34`</small>

```zirric
data TestCaseDetailsTodo
```

---

### `TestReport` {#testreport}

<small>`tests/types.zirr:47`</small>

```zirric
data TestReport {
	pass: [TestCase]
	fail: [FailureRecord]
	skip: [TestCase]
	todo: [TestCase]
}
```

#### Fields

| Field  | Description |
| ------ | ----------- |
| `pass` |             |
| `fail` |             |
| `skip` |             |
| `todo` |             |

---

## Attributes

### `Comment` {#comment}

<small>`tests/types.zirr:7`</small>

```zirric
attr Comment {
	text: String
}
```

#### Fields

| Field  | Description |
| ------ | ----------- |
| `text` |             |

---

### `Only` {#only}

<small>`tests/types.zirr:5`</small>

```zirric
attr Only
```

---

### `Skip` {#skip}

<small>`tests/types.zirr:4`</small>

```zirric
attr Skip
```

---

### `Test` {#test}

<small>`tests/types.zirr:3`</small>

```zirric
attr Test
```

---

### `Todo` {#todo}

<small>`tests/types.zirr:6`</small>

```zirric
attr Todo
```
