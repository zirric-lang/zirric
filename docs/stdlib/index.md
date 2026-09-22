---
title: Standard Library
description: Every module Zirric ships, grouped by what it is for.
---

# Standard Library

Zirric's standard library is written in Zirric. Every page here mirrors the module's own sources — the signatures, doc comments and line references come from the `.zirr` files the compiler reads.

[`prelude`](./prelude/index.md) is in scope everywhere and needs no import. Everything else is imported by name:

```zirric
import strings
import tests.runner
import pkgs = reflect.packages
```

::: callout warning Experimental
[ZE-020 Standard Library](/proposals/ZE-020-standard-library) landed in [v0.1.0](/changelog/v0.1.0). These modules are usable today, but names and signatures can still change.
:::

## Core

The types every program is built from, and the two unions the rest of the library returns.

- [`prelude`](./prelude/index.md) — built-in types, the attributes that describe them, `Option` and `Result`.
- [`options`](./options/index.md) — working with values that may be absent.
- [`results`](./results/index.md) — working with values that may have failed.
- [`errors`](./errors/index.md) — combining, annotating and unwrapping error values.

## Data

- [`arrays`](./arrays/index.md) — slicing, searching and transforming `Array`.
- [`dicts`](./dicts/index.md) — mapping, filtering and reducing `Dict`.
- [`strings`](./strings/index.md) — text operations that count characters, not bytes.
- [`bytes`](./bytes/index.md) — raw `Binary` values and the conversions into them.
- [`ranges`](./ranges/index.md) — set operations over the three range types.
- [`math`](./math/index.md) — numeric constants, conversions and rounding.
- [`fun`](./fun/index.md) — function combinators, and lazy operations over anything iterable.

## The outside world

- [`os`](./os/index.md) — the only module that talks to the machine.
- [`io`](./io/index.md) — the reader and writer attributes, and the streams that carry bytes.
- [`fmt`](./fmt/index.md) — turning values into strings and writing them out.
- [`fs`](./fs/index.md) — files and directories, as a value you pass around.
- [`paths`](./paths/index.md) — joining, cleaning and matching path strings.
- [`clock`](./clock/index.md) — wall-clock and monotonic time, as values a test can replace.
- [`time`](./time/index.md) — durations, instants and timestamps.
- [`random`](./random/index.md) — seeded randomness, and the operations every source shares.
- [`scripts`](https://code.knabel.dev/zirric-lang/scripts) — the printing and file helpers, already wired to the host. A package of its own: depend on it to use it.

## Serialization

- [`coding`](./coding/index.md) — between your own types and a native value tree.
- [`json`](./json/index.md) — reading and writing JSON.
- [`yaml`](./yaml/index.md) — reading and writing YAML.

## Reflection

- [`reflect`](./reflect/index.md) — inspecting modules, types, fields and values at runtime.
- [`reflect.packages`](./reflect/packages/index.md) — discovering the modules of the running package.

## Testing

- [`tests`](./tests/index.md) — the attributes and types that describe a test.
- [`tests.assert`](./tests/assert/index.md) — assertions that produce the `Result` a test returns.
- [`tests.runner`](./tests/runner/index.md) — discovery and execution.
- [`tests.tap`](./tests/tap/index.md) — TAP output.

## Project manifests

Both of these are the vocabulary a `Cavefile` is written in rather than modules you import into a program. See [Cavefile manifests](/cavefile) for a whole file.

- [`cave`](./cave/index.md) — dependencies and package metadata.
- [`tasks`](./tasks/index.md) — turning a declaration into a CLI subcommand.

## Future

Implementations of proposals that have not landed. They may change or disappear.

- [`future`](./future/index.md) — what the module is and how to treat it.
- [`future.prelude`](./future/prelude/index.md) — proposed prelude additions.
- [`future.reflect`](./future/reflect/index.md) — the original reflection sketch, superseded by [`reflect`](./reflect/index.md).
