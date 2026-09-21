---
title: Tests.Runner
description: Discovering tests by reflection and running them.
---

# Module `tests.runner`

> Finding the tests, then executing them.

```zirric
import tests.runner
// or: import runner = tests.runner
```

|            |                                                                                                                   |
| ---------- | ----------------------------------------------------------------------------------------------------------------- |
| **Module** | `tests.runner`                                                                                                    |
| **Source** | [`tests/runner/runner.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/tests/runner/runner.zirr) |

The runner finds tests by [reflection](../../reflect/index.md): it walks the modules of the running package, reads the functions carrying [`@tests.Test`](../index.md#test), and builds a [`TestCase`](../index.md#testcase) for each.

[`discover`](#discover) and its siblings only find; [`exec`](#exec) runs what it is given and emits a [`TestEvent`](../index.md#testevent) as each test starts and finishes, which is what a reporter subscribes to. [`runT`](#runt) is the two put together with TAP output, and is what a `Cavefile` test task normally calls.

[`@tests.Only`](../index.md#only) is applied here rather than at discovery: if any discovered test carries it, the run narrows to those.

## Contents

- **Constants** — [`ignoreEvents`](#ignoreevents)
- **Functions** — [`exec`](#exec), [`discover`](#discover), [`discoverAll`](#discoverall), [`discoverWhere`](#discoverwhere), [`runWhere`](#runwhere), [`runT`](#runt)

---

## Constants

### `ignoreEvents` {#ignoreevents}

<small>`tests/runner/runner.zirr:11`</small>

```zirric
const ignoreEvents = fn(event: tests.TestEvent) {}
```

An event callback that discards everything, for runs without output.

---

## Functions

### `exec` {#exec}

<small>`tests/runner/runner.zirr:15`</small>

```zirric
fn exec(discovered: [tests.TestCase], onEvent: fn(tests.TestEvent)) -> tests.TestReport
```

Runs test cases, emitting events, and returns a report. This is the primitive the entry points below build on: it executes exactly the cases it is given and neither discovers nor reports anything itself.

---

### `discover` {#discover}

<small>`tests/runner/runner.zirr:85`</small>

```zirric
fn discover(module: Module) -> [tests.TestCase]
```

Every @tests.Test function declared in module, as runnable cases.

---

### `discoverAll` {#discoverall}

<small>`tests/runner/runner.zirr:99`</small>

```zirric
fn discoverAll(modules: [Module]) -> [tests.TestCase]
```

Every @tests.Test function declared in any of modules, as runnable cases.

---

### `discoverWhere` {#discoverwhere}

<small>`tests/runner/runner.zirr:111`</small>

```zirric
fn discoverWhere(predicate: fn(String) -> Bool) -> [tests.TestCase]
```

Every @tests.Test function declared in a project module whose name satisfies predicate. Only the modules the predicate accepts are loaded, so the rest never run.

---

### `runWhere` {#runwhere}

<small>`tests/runner/runner.zirr:138`</small>

```zirric
fn runWhere(predicate: fn(String) -> Bool) -> tests.TestReport
```

Runs every test in the project modules whose name satisfies predicate, reporting TAP to stdout. Exits with a non-zero status when any test failed, so a task or CI step fails with the suite.

---

### `runT` {#runt}

<small>`tests/runner/runner.zirr:148`</small>

```zirric
fn runT() -> tests.TestReport
```

Runs every test in the project modules whose name ends with "_t", the convention this repository follows. Both a `foo/_t` submodule and a sibling module named `foo_t` match.

---

## See also

- [`tests`](../index.md) — the attributes and types being discovered.
- [`tests.tap`](../tap/index.md) — the reporter `runT` uses.
- [`reflect.packages`](../../reflect/packages/index.md) — the module discovery underneath.
