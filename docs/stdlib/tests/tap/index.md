---
title: Tests.TAP
description: A TAP version 14 reporter for test events.
---

# Module `tests.tap`

> Test output anything can read.

```zirric
import tests.tap
```

|            |                                                                                                                         |
| ---------- | ----------------------------------------------------------------------------------------------------------------------- |
| **Module** | `tests.tap`                                                                                                             |
| **Source** | [`tests/tap/tap-reporter.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/tests/tap/tap-reporter.zirr) |

[`reporter`](#reporter) turns a [`Writer`](../../io/index.md#writer) into the event handler [`exec`](../runner/index.md#exec) expects, printing [TAP version 14](https://testanything.org) as the run proceeds.

Because it takes a writer rather than reaching for standard output, the same reporter writes to a file, to a buffer, or to the terminal — and a test of the reporter itself can collect what it produced.

## Contents

- **Functions** — [`reporter`](#reporter)

---

## Functions

### `reporter` {#reporter}

<small>`tests/tap/tap-reporter.zirr:7`</small>

```zirric
fn reporter(writer: @io.Writer) -> fn(TestEvent)
```

Builds the event handler [`exec`](../runner/index.md#exec) expects, writing TAP version 14 to the given writer as the run proceeds.

---

## See also

- [`tests.runner`](../runner/index.md) — the events this consumes.
- [`io`](../../io/index.md) — the writer it takes.
