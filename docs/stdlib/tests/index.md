---
title: Tests
description: The attributes and types that describe a test, its outcome and its report.
---

# Module `tests`

> What a test is, before anything runs it.

```zirric
import tests
```

|            |                                                                                                   |
| ---------- | ------------------------------------------------------------------------------------------------- |
| **Module** | `tests`                                                                                           |
| **Source** | [`tests/types.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/tests/types.zirr) |

A test is a function carrying [`Test`](#test) that returns a [`Result`](../prelude/index.md#result). Nothing else is required: there is no registry and no base type, so the runner finds tests by [reflecting](../reflect/index.md) over a module rather than by being told about them.

The other attributes adjust what happens to one. [`Skip`](#skip) keeps it out of the run, [`Todo`](#todo) marks it as expected to be unfinished, [`Only`](#only) narrows the run to itself, and [`Comment`](#comment) attaches a note that reporters print.

This module is only the vocabulary. [`tests.assert`](./assert/index.md) writes the assertions, [`tests.runner`](./runner/index.md) finds and runs them, and [`tests.tap`](./tap/index.md) reports what happened.

## Contents

- **Attributes** — [`Test`](#test), [`Skip`](#skip), [`Only`](#only), [`Todo`](#todo), [`Comment`](#comment)
- **Unions** — [`TestCaseDetails`](#testcasedetails), [`TestEvent`](#testevent)
- **Data** — [`TestCase`](#testcase), [`TestReport`](#testreport), [`FailureRecord`](#failurerecord)

---

## Attributes

### `Test` {#test}

<small>`tests/types.zirr:3`</small>

```zirric
attr Test {}
```

Marks a function as a test. The function takes no arguments and returns a [`Result`](../prelude/index.md#result); `Ok` passed, `Err` failed.

---

### `Skip` {#skip}

<small>`tests/types.zirr:4`</small>

```zirric
attr Skip {}
```

Keeps a test out of the run. It is still discovered and still reported, as skipped.

---

### `Only` {#only}

<small>`tests/types.zirr:5`</small>

```zirric
attr Only {}
```

Narrows the run to the tests carrying it. Applied after discovery, so one `@Only` anywhere silences every test without it.

---

### `Todo` {#todo}

<small>`tests/types.zirr:6`</small>

```zirric
attr Todo {}
```

Marks a test as known to be unfinished. It runs, but its failure is reported as expected rather than as a problem.

---

### `Comment` {#comment}

<small>`tests/types.zirr:7`</small>

```zirric
attr Comment {
	text: String
}
```

Attaches a note to a test, which reporters print alongside its result.

#### Members

| Member | Signature      | Description                             |
| ------ | -------------- | --------------------------------------- |
| `text` | `text: String` | The note to print alongside the result. |

---

## Unions

### `TestCaseDetails` {#testcasedetails}

<small>`tests/types.zirr:20`</small>

```zirric
@AnyOption()
union TestCaseDetails {
	None

	data TestCaseDetailsSkip {}
	data TestCaseDetailsTodo {}
	data TestCaseDetailsOnly {}
}
```

Which of the marker attributes a test carried, if any. Declared as an option type, so `None` means an ordinary test.

#### Cases

| Case                  | Interpretation                                        |
| --------------------- | ----------------------------------------------------- |
| `None`                | An ordinary test, with none of the marker attributes. |
| `TestCaseDetailsSkip` | The test carries [`@Skip`](#skip).                    |
| `TestCaseDetailsTodo` | The test carries [`@Todo`](#todo).                    |
| `TestCaseDetailsOnly` | The test carries [`@Only`](#only).                    |

---

### `TestEvent` {#testevent}

<small>`tests/types.zirr:28`</small>

```zirric
union TestEvent {
	data Discovered { cases: [TestCase] }
	data Skipped { test: TestCase }
	data Started { test: TestCase }
	data Finished { test: TestCase, outcome: Result }
	data Completed { report: TestReport }
}
```

What the runner reports as it proceeds. A reporter is a function taking one of these.

#### Cases

| Case         | Interpretation                                       |
| ------------ | ---------------------------------------------------- |
| `Discovered` | Emitted once, with every case the run will consider. |
| `Skipped`    | A case was passed over rather than run.              |
| `Started`    | A case is about to run.                              |
| `Finished`   | A case finished, with the `Result` it returned.      |
| `Completed`  | The run is over, with the full report.               |

---

## Data

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

One discovered test: the function to call, the name it was found under, and what the attributes on it said.

#### Fields

| Field     | Signature                  | Description                                     |
| --------- | -------------------------- | ----------------------------------------------- |
| `name`    | `name: String`             | The name the test was discovered under.         |
| `impl`    | `impl: fn() -> Result`     | The function to call.                           |
| `skipped` | `skipped: Bool`            | Whether it will be passed over rather than run. |
| `comment` | `comment: String`          | The note from [`@Comment`](#comment), or empty. |
| `details` | `details: TestCaseDetails` | Which marker attribute it carried, if any.      |

---

### `TestReport` {#testreport}

<small>`tests/types.zirr:36`</small>

```zirric
data TestReport {
	pass: [TestCase]
	fail: [FailureRecord]
	skip: [TestCase]
	todo: [TestCase]
}
```

The outcome of a whole run, with the cases grouped by what happened to them.

#### Fields

| Field  | Signature               | Description                                 |
| ------ | ----------------------- | ------------------------------------------- |
| `pass` | `pass: [TestCase]`      | Cases that returned `Ok`.                   |
| `fail` | `fail: [FailureRecord]` | Cases that returned `Err`, with the reason. |
| `skip` | `skip: [TestCase]`      | Cases that were not run.                    |
| `todo` | `todo: [TestCase]`      | Cases marked [`@Todo`](#todo).              |

---

### `FailureRecord` {#failurerecord}

<small>`tests/types.zirr:43`</small>

```zirric
data FailureRecord { test: TestCase, error: Err }
```

A failed test together with the `Err` it returned.

#### Fields

| Field  | Signature                      | Description           |
| ------ | ------------------------------ | --------------------- |
| `test` | `test: TestCase, error: Err }` | The case that failed. |

---

## See also

- [`tests.assert`](./assert/index.md) — producing the `Result` a test returns.
- [`tests.runner`](./runner/index.md) — discovery and execution.
- [`tests.tap`](./tap/index.md) — TAP output.
