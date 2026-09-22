---
title: Cave
description: The attributes a Cavefile uses to describe a package and declare its dependencies.
---

# Module `cave`

> The vocabulary of a `Cavefile`.

```zirric
import cave
```

|            |                                                                                                       |
| ---------- | ----------------------------------------------------------------------------------------------------- |
| **Module** | `cave`                                                                                                |
| **Source** | [`cave/manifest.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/cave/manifest.zirr) |

A `Cavefile` is ordinary Zirric source, and `cave` is the vocabulary it is written in. The package manager reads the file for its types and attributes rather than executing it, so everything here is declaration, not behaviour.

[`Package`](#package) marks the one `data` declaration that is the manifest. The attributes on the declaration describe the package itself; its fields are the dependencies. Each field names the import, and the attributes on it say where the package comes from — [`Stdlib`](#stdlib), [`Git`](#git) or [`Local`](#local) — and, optionally, which [`Version`](#version) is acceptable.

[`Version`](#version), [`LanguageVersion`](#languageversion), [`Description`](#description) and [`Documentation`](#documentation) read the same way on the package and on one of its dependencies.

[`FormattingExcludes`](#formattingexcludes) is unrelated to dependencies and may sit on any declaration; [`zirric fmt`](/tooling/code-formatter) reads it to decide which paths to leave alone.

## Contents

- **Attributes** — [`Package`](#package), [`FormattingExcludes`](#formattingexcludes), [`Version`](#version), [`LanguageVersion`](#languageversion), [`Description`](#description), [`Documentation`](#documentation), [`Stdlib`](#stdlib), [`Git`](#git), [`Local`](#local)
- **Unions** — [`Source`](#source)

---

## Attributes

### `Package` {#package}

<small>`cave/manifest.zirr:5`</small>

```zirric
attr Package {}
```

Marks the current data structure as the package manifest. Its attributes describe the package itself, its fields declare the dependencies.

---

### `FormattingExcludes` {#formattingexcludes}

<small>`cave/manifest.zirr:9`</small>

```zirric
attr FormattingExcludes {
	// The path patterns to leave unformatted.
	patterns: [String]
}
```

Excludes paths from `zirric fmt` and from editor formatting. Patterns are relative to the package root: `*` matches within one segment, `**` matches any number of segments.

#### Members

| Member     | Signature            | Description                             |
| ---------- | -------------------- | --------------------------------------- |
| `patterns` | `patterns: [String]` | The path patterns to leave unformatted. |

---

### `Version` {#version}

<small>`cave/manifest.zirr:15`</small>

```zirric
attr Version {
	// The version, or the version predicate for a dependency.
	// Examples:
	//   - "main"
	//   - "~1.2.3"
	//   - "^1.2.3"
	//   - ">=1.2.3"
	//   - ">= 1.0.0"
	//   - "< 1.0.0"
	predicate: String
}
```

On the package, its own version. On a dependency, which versions may be resolved; omitted, any version is acceptable.

#### Members

| Member      | Signature           | Description                                                                                                  |
| ----------- | ------------------- | ------------------------------------------------------------------------------------------------------------ |
| `predicate` | `predicate: String` | The version, or the version predicate for a dependency. Examples: - "main" - "~1.2.3" - "^1.2.3" - ">=1.2.3" |

---

### `LanguageVersion` {#languageversion}

<small>`cave/manifest.zirr:28`</small>

```zirric
attr LanguageVersion {
	// The version predicate for the Zirric toolchain itself.
	predicate: String
}
```

Declares which Zirric versions the package or dependency can be built with. A toolchain that does not satisfy it refuses the project — only `zirric cave describe` still opens it, to report why; a development build that carries no version of its own is exempt.

#### Members

| Member      | Signature           | Description                                            |
| ----------- | ------------------- | ------------------------------------------------------ |
| `predicate` | `predicate: String` | The version predicate for the Zirric toolchain itself. |

---

### `Description` {#description}

<small>`cave/manifest.zirr:38`</small>

```zirric
attr Description {
	// The description of the package or dependency.
	desc: String
}
```

Describes the package or dependency in one line.

#### Members

| Member | Signature      | Description                                   |
| ------ | -------------- | --------------------------------------------- |
| `desc` | `desc: String` | The description of the package or dependency. |

---

### `Documentation` {#documentation}

<small>`cave/manifest.zirr:44`</small>

```zirric
attr Documentation {
	// The URL of the documentation.
	url: String
}
```

Links to the documentation of the package or dependency.

#### Members

| Member | Signature     | Description                   |
| ------ | ------------- | ----------------------------- |
| `url`  | `url: String` | The URL of the documentation. |

---

### `Stdlib` {#stdlib}

<small>`cave/manifest.zirr:57`</small>

```zirric
attr Stdlib {
	// The name of the standard library dependency.
	name: String
}
```

Marks the current data structure as a standard library dependency.

#### Members

| Member | Signature      | Description                                  |
| ------ | -------------- | -------------------------------------------- |
| `name` | `name: String` | The name of the standard library dependency. |

---

### `Git` {#git}

<small>`cave/manifest.zirr:64`</small>

```zirric
attr Git {
	// The URL of the Git repository.
	url: String
}
```

Declares the Git repository of the package, or marks a field as a Git dependency. On a dependency the field name represents the import name of the dependency in this package.

#### Members

| Member | Signature     | Description                    |
| ------ | ------------- | ------------------------------ |
| `url`  | `url: String` | The URL of the Git repository. |

---

### `Local` {#local}

<small>`cave/manifest.zirr:70`</small>

```zirric
attr Local {
	// The local path to the dependency.
	path: String
}
```

Marks a field as a local dependency with a path.

#### Members

| Member | Signature      | Description                       |
| ------ | -------------- | --------------------------------- |
| `path` | `path: String` | The local path to the dependency. |

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

| Case     | Interpretation                                 |
| -------- | ---------------------------------------------- |
| `Stdlib` | A module the toolchain ships.                  |
| `Git`    | A repository, resolved by tag.                 |
| `Local`  | A directory on this machine, by relative path. |

---

## See also

- [Cavefile manifests](/cavefile) — what a whole file looks like.
- [`tasks`](../tasks/index.md) — the other half of a Cavefile.
- [Package Manager](/tooling/package-manager) — how these are resolved.
- [ZE-002 The Cavefile](/proposals/ZE-002-the-cavefile) — the design.
