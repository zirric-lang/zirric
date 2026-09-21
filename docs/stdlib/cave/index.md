---
title: Cave
description: The attributes a Cavefile uses to declare dependencies and package metadata.
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

[`Dependencies`](#dependencies) marks the one `data` declaration whose fields are dependencies. Each field names the import, and the attributes on it say where the package comes from — [`Stdlib`](#stdlib), [`Git`](#git) or [`Local`](#local) — and, optionally, [`Version`](#version).

[`FormattingExcludes`](#formattingexcludes) is unrelated to dependencies and may sit on any declaration; [`zirric fmt`](/tooling/code-formatter) reads it to decide which paths to leave alone.

## Contents

- **Attributes** — [`Dependencies`](#dependencies), [`FormattingExcludes`](#formattingexcludes), [`Package`](#package), [`Version`](#version), [`Stdlib`](#stdlib), [`Git`](#git), [`Local`](#local)
- **Unions** — [`Source`](#source)

---

## Attributes

### `Dependencies` {#dependencies}

<small>`cave/manifest.zirr:4`</small>

```zirric
attr Dependencies {}
```

Marks the current data structure as a dependencies manifest.

---

### `FormattingExcludes` {#formattingexcludes}

<small>`cave/manifest.zirr:8`</small>

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

### `Package` {#package}

<small>`cave/manifest.zirr:17`</small>

```zirric
attr Package {
	// The canonical URL of the package.
	url: String
}
```

Declares the canonical URL of the current package. Placed on the Cavefile's `mod` declaration, e.g. `@cave.Package("https://...") mod mymodule`. Used to derive the package's name and source instead of falling back to the project directory name.

#### Members

| Member | Signature     | Description                       |
| ------ | ------------- | --------------------------------- |
| `url`  | `url: String` | The canonical URL of the package. |

---

### `Version` {#version}

<small>`cave/manifest.zirr:22`</small>

```zirric
attr Version {
	// The version predicate for the Git dependency.
	// Examples:
	//   - "main"
	//   - "~1.2.3"
	//   - "^1.2.3"
	//   - ">=1.2.3"
	//   - ">= 1.0.0"
	//   - "< 1.0.0"
	//   - ">= 1.0.0"
	//   - ">= 1.0.0"
	predicate: String
}
```

Constrains which versions of a Git dependency may be resolved. Omitted, any version is acceptable.

#### Members

| Member      | Signature           | Description                                                                                                                                           |
| ----------- | ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| `predicate` | `predicate: String` | The version predicate for the Git dependency. Examples: - "main" - "~1.2.3" - "^1.2.3" - ">=1.2.3" - ">= 1.0.0" - "< 1.0.0" - ">= 1.0.0" - ">= 1.0.0" |

---

### `Stdlib` {#stdlib}

<small>`cave/manifest.zirr:44`</small>

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

<small>`cave/manifest.zirr:51`</small>

```zirric
attr Git {
	// The URL of the Git repository.
	url: String
}
```

Marks a field as a Git dependency with a URL and predicate. The field name represents the import name of the dependency in this package.

#### Members

| Member | Signature     | Description                    |
| ------ | ------------- | ------------------------------ |
| `url`  | `url: String` | The URL of the Git repository. |

---

### `Local` {#local}

<small>`cave/manifest.zirr:57`</small>

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

<small>`cave/manifest.zirr:37`</small>

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
