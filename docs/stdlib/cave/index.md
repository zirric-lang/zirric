---
title: Cave
description: The attributes a Cavefile uses to describe a package and declare its dependencies.
---

# Module `cave`

```zirric
import cave
```

|            |                                                                                                                                                                                                                    |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Module** | `cave`                                                                                                                                                                                                             |
| **Source** | [`cave/manifest.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/cave/manifest.zirr), [`cave/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/cave/module-docs.zirr) |

> The vocabulary of a `Cavefile`.

A `Cavefile` is ordinary Zirric source, and `cave` is the vocabulary it is written in. The package manager reads the file for its types and attributes rather than executing it, so everything here is declaration, not behaviour.

[`Package`](#package) marks the one `data` declaration that is the manifest. The attributes on the declaration describe the package itself; its fields are the dependencies. Each field names the import, and the attributes on it say where the package comes from — [`Stdlib`](#stdlib), [`Git`](#git) or [`Local`](#local) — and, optionally, which [`Version`](#version) is acceptable.

[`Version`](#version), [`LanguageVersion`](#languageversion), [`Description`](#description) and [`Documentation`](#documentation) read the same way on the package and on one of its dependencies.

[`FormattingExcludes`](#formattingexcludes) is unrelated to dependencies and may sit on any declaration; [`zirric fmt`](https://zirric.knabel.dev/tooling/code-formatter) reads it to decide which paths to leave alone.

## Contents

- **Unions** — [`Source`](#source)
- **Attributes** — [`Description`](#description), [`Documentation`](#documentation), [`FormattingExcludes`](#formattingexcludes), [`Git`](#git), [`LanguageVersion`](#languageversion), [`Local`](#local), [`Package`](#package), [`Stdlib`](#stdlib), [`Version`](#version)

---

## Unions

### `Source` {#source}

<small>`cave/manifest.zirr:50`</small>

```zirric
union Source {
	Stdlib
	Git
	Local
}
```

The type of dependency: standard library, Git, or local.

#### Cases

| Case     | Interpretation                                                                                                                                                                 |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `Stdlib` | Marks the current data structure as a standard library dependency.                                                                                                             |
| `Git`    | Declares the Git repository of the package, or marks a field as a Git dependency. On a dependency the field name represents the import name of the dependency in this package. |
| `Local`  | Marks a field as a local dependency with a path.                                                                                                                               |

---

## Attributes

### `Description` {#description}

<small>`cave/manifest.zirr:38`</small>

```zirric
attr Description {
	desc: String
}
```

Describes the package or dependency in one line.

#### Fields

| Field  | Description                                   |
| ------ | --------------------------------------------- |
| `desc` | The description of the package or dependency. |

---

### `Documentation` {#documentation}

<small>`cave/manifest.zirr:44`</small>

```zirric
attr Documentation {
	url: String
}
```

Links to the documentation of the package or dependency.

#### Fields

| Field | Description                   |
| ----- | ----------------------------- |
| `url` | The URL of the documentation. |

---

### `FormattingExcludes` {#formattingexcludes}

<small>`cave/manifest.zirr:9`</small>

```zirric
attr FormattingExcludes {
	patterns: [String]
}
```

Excludes paths from `zirric fmt` and from editor formatting.
Patterns are relative to the package root: `*` matches within one segment, `**` matches any number of segments.

#### Fields

| Field      | Description                             |
| ---------- | --------------------------------------- |
| `patterns` | The path patterns to leave unformatted. |

---

### `Git` {#git}

<small>`cave/manifest.zirr:64`</small>

```zirric
attr Git {
	url: String
}
```

Declares the Git repository of the package, or marks a field as a Git dependency.
On a dependency the field name represents the import name of the dependency in this package.

#### Fields

| Field | Description                    |
| ----- | ------------------------------ |
| `url` | The URL of the Git repository. |

---

### `LanguageVersion` {#languageversion}

<small>`cave/manifest.zirr:28`</small>

```zirric
attr LanguageVersion {
	predicate: String
}
```

Declares which Zirric versions the package or dependency can be built with.

#### Fields

| Field       | Description                                                                                        |
| ----------- | -------------------------------------------------------------------------------------------------- |
| `predicate` | The version predicate for the Zirric toolchain itself. Examples: - "~1.2.3" - "^1.2.3" - ">=1.2.3" |

---

### `Local` {#local}

<small>`cave/manifest.zirr:70`</small>

```zirric
attr Local {
	path: String
}
```

Marks a field as a local dependency with a path.

#### Fields

| Field  | Description                       |
| ------ | --------------------------------- |
| `path` | The local path to the dependency. |

---

### `Package` {#package}

<small>`cave/manifest.zirr:5`</small>

```zirric
attr Package
```

Marks the current data structure as the package manifest.
Its attributes describe the package itself, its fields declare the dependencies.

---

### `Stdlib` {#stdlib}

<small>`cave/manifest.zirr:57`</small>

```zirric
attr Stdlib {
	name: String
}
```

Marks the current data structure as a standard library dependency.

#### Fields

| Field  | Description                                  |
| ------ | -------------------------------------------- |
| `name` | The name of the standard library dependency. |

---

### `Version` {#version}

<small>`cave/manifest.zirr:15`</small>

```zirric
attr Version {
	predicate: String
}
```

Declares the version of the package, or the version predicate a dependency accepts.

#### Fields

| Field       | Description                                                                                                                           |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| `predicate` | The version, or the version predicate for a dependency. Examples: - "main" - "~1.2.3" - "^1.2.3" - ">=1.2.3" - ">= 1.0.0" - "< 1.0.0" |
