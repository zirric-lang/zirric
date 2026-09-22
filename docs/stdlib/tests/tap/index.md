---
title: Tests.TAP
description: A TAP version 14 reporter for test events.
---

# Module `tests.tap`

```zirric
import tests.tap
```

|            |                                                                                                                                                                                                                                                |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `tests.tap`                                                                                                                                                                                                                                    |
| **Source** | [`tests/tap/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/tests/tap/module-docs.zirr), [`tests/tap/tap-reporter.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/tests/tap/tap-reporter.zirr) |

> Test output anything can read.

[`reporter`](#reporter) turns a [`io.Writer`](../../io/index.md#writer) into the event handler [`tests.runner.exec`](../../tests/runner/index.md#exec) expects, printing [TAP version 14](https://testanything.org) as the run proceeds.

Because it takes a writer rather than reaching for standard output, the same reporter writes to a file, to a buffer, or to the terminal — and a test of the reporter itself can collect what it produced.

## Dependencies

- [`fmt`](../../fmt/index.md)
  - [`bytes`](../../bytes/index.md)
    - [`ranges`](../../ranges/index.md)
  - [`io`](../../io/index.md)
- [`io`](../../io/index.md)
- [`tests`](../../tests/index.md)

---

## Contents

- **Functions** — [`reporter`](#reporter)

---

## Functions

### `reporter` {#reporter}

<small>`tests/tap/tap-reporter.zirr:7`</small>

```zirric
fn reporter(writer: @io.Writer) -> fn(TestEvent)
```
