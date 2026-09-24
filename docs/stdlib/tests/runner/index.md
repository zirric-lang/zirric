---
title: Tests.Runner
description: Discovering tests by reflection and running them.
---

# Module `tests.runner`

```zirric
import tests.runner
```

|            |                                                                                                                                                                                                                                                |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `tests.runner`                                                                                                                                                                                                                                 |
| **Source** | [`tests/runner/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/tests/runner/module-docs.zirr), [`tests/runner/runner.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/tests/runner/runner.zirr) |

> Finding the tests, then executing them.

The runner finds tests through [`reflect`](../../reflect/index.md): it walks the modules of the running package, reads the functions carrying [`@tests.Test`](../../tests/index.md#test), and builds a [`tests.TestCase`](../../tests/index.md#testcase) for each.

[`discover`](#discover) and its siblings only find; [`exec`](#exec) runs what it is given and emits a [`tests.TestEvent`](../../tests/index.md#testevent) as each test starts and finishes, which is what a reporter subscribes to. [`runT`](#runt) is the two put together with TAP output, and is what a `Cavefile` test task normally calls.

[`@tests.Only`](../../tests/index.md#only) is applied here rather than at discovery: if any discovered test carries it, the run narrows to those.

[`runT`](#runt) is what `zirric test` does with no `test` task declared: it runs every project module whose name ends in `_t`. See [`tests`](../index.md) for the convention and an example test module.

## Dependencies

- [`os`](../../os/index.md)
  - [`clock`](../../clock/index.md)
    - [`time`](../../time/index.md)
  - [`co`](../../co/index.md)
    - [`time`](../../time/index.md)
  - [`fs`](../../fs/index.md)
    - [`bytes`](../../bytes/index.md)
      - [`ranges`](../../ranges/index.md)
    - [`io`](../../io/index.md)
    - [`paths`](../../paths/index.md)
    - [`results`](../../results/index.md)
  - [`io`](../../io/index.md)
  - [`random`](../../random/index.md)
    - [`arrays`](../../arrays/index.md)
      - [`ranges`](../../ranges/index.md)
- [`reflect`](../../reflect/index.md)
- [`reflect.packages`](../../reflect/packages/index.md)
  - [`arrays`](../../arrays/index.md)
    - [`ranges`](../../ranges/index.md)
- [`strings`](../../strings/index.md)
  - [`ranges`](../../ranges/index.md)
- [`tests`](../../tests/index.md)
- [`tests.tap`](../../tests/tap/index.md)
  - [`fmt`](../../fmt/index.md)
    - [`bytes`](../../bytes/index.md)
      - [`ranges`](../../ranges/index.md)
    - [`io`](../../io/index.md)
  - [`io`](../../io/index.md)
  - [`tests`](../../tests/index.md)

---

## Contents

- **Functions** — [`discover`](#discover), [`discoverAll`](#discoverall), [`discoverWhere`](#discoverwhere), [`exec`](#exec), [`runT`](#runt), [`runWhere`](#runwhere)
- **Constants** — [`ignoreEvents`](#ignoreevents)

---

## Functions

### `discover` {#discover}

<small>`tests/runner/runner.zirr:85`</small>

```zirric
fn discover(module: AnyModule) -> [tests.TestCase]
```

Every @tests.Test function declared in module, as runnable cases.

---

### `discoverAll` {#discoverall}

<small>`tests/runner/runner.zirr:99`</small>

```zirric
fn discoverAll(modules: [AnyModule]) -> [tests.TestCase]
```

Every @tests.Test function declared in any of modules, as runnable cases.

---

### `discoverWhere` {#discoverwhere}

<small>`tests/runner/runner.zirr:111`</small>

```zirric
fn discoverWhere(predicate: fn(String) -> Bool) -> [tests.TestCase]
```

Every @tests.Test function declared in a project module whose name satisfies predicate.
Only the modules the predicate accepts are loaded, so the rest never run.

---

### `exec` {#exec}

<small>`tests/runner/runner.zirr:15`</small>

```zirric
fn exec(discovered: [tests.TestCase], onEvent: fn(tests.TestEvent)) -> tests.TestReport
```

Runs test cases, emitting events, and returns a report.
This is the primitive the entry points below build on: it executes exactly the cases it is given and neither discovers nor reports anything itself.

---

### `runT` {#runt}

<small>`tests/runner/runner.zirr:148`</small>

```zirric
fn runT() -> tests.TestReport
```

Runs every test in the project modules whose name ends with "_t", the convention this repository follows.
Both a `foo/_t` submodule and a sibling module named `foo_t` match.

---

### `runWhere` {#runwhere}

<small>`tests/runner/runner.zirr:138`</small>

```zirric
fn runWhere(predicate: fn(String) -> Bool) -> tests.TestReport
```

Runs every test in the project modules whose name satisfies predicate, reporting TAP to stdout.
Exits with a non-zero status when any test failed, so a task or CI step fails with the suite.

---

## Constants

### `ignoreEvents` {#ignoreevents}

<small>`tests/runner/runner.zirr:11`</small>

```zirric
const ignoreEvents
```

An event callback that discards everything, for runs without output.
