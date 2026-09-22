---
title: Package manifest
description: Cavefile for code.knabel.dev.zirric_lang.zirric
---

# Package `code.knabel.dev.zirric_lang.zirric`

|              |                                                                                          |
| ------------ | ---------------------------------------------------------------------------------------- |
| **Package**  | `code.knabel.dev.zirric_lang.zirric`                                                     |
| **Source**   | [https://code.knabel.dev/zirric-lang/zirric](https://code.knabel.dev/zirric-lang/zirric) |
| **Manifest** | [`Cavefile`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/Cavefile)        |

## Dependencies

What this package imports from outside itself. Each is reached by the name in the first column.

| Dependency                           | Kind | Source                                                                                   | Version  | Description |
| ------------------------------------ | ---- | ---------------------------------------------------------------------------------------- | -------- | ----------- |
| `code.knabel.dev.zirric_lang.zirric` | git  | [https://code.knabel.dev/zirric-lang/zirric](https://code.knabel.dev/zirric-lang/zirric) | `latest` |             |

## Tasks

What this package can be asked to do. Each runs with `zirric task <name>`, or `zirric <name>` when no built-in command claims the name.

[`docs`](#docs), [`manifest`](#manifest)

### `docs` {#docs}

```sh
zirric task docs [--out <String>] [--source <String>]
```

Write one Markdown page per standard library module

Runs [`_tasks/docs/docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/_tasks/docs/docs.zirr).

#### Flags

| Flag       | Type     | Description                                                           |
| ---------- | -------- | --------------------------------------------------------------------- |
| `--out`    | `String` | Directory the pages are written to, relative to the working directory |
| `--source` | `String` | Repository root the Source links point into, branch included          |

### `manifest` {#manifest}

```sh
zirric task manifest [--out <String>] [--source <String>]
```

Write the package's dependencies and tasks as a Markdown page

Runs [`_tasks/manifest/manifest.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/_tasks/manifest/manifest.zirr).

#### Flags

| Flag       | Type     | Description                                                    |
| ---------- | -------- | -------------------------------------------------------------- |
| `--out`    | `String` | File the page is written to, relative to the working directory |
| `--source` | `String` | Repository root the links point into, branch included          |
