---
title: Reflect.Packages
description: Discovering the modules that make up the running package.
---

# Module `reflect.packages`

```zirric
import reflect.packages
```

|            |                                                                                                                                                                                                                                                                                                                                                                                                   |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `reflect.packages`                                                                                                                                                                                                                                                                                                                                                                                |
| **Source** | [`reflect/packages/cavefile.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/reflect/packages/cavefile.zirr), [`reflect/packages/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/reflect/packages/module-docs.zirr), [`reflect/packages/packages.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/reflect/packages/packages.zirr) |

> Finding modules by name, or by asking.

Where [`reflect`](../../reflect/index.md) inspects a module you already hold, `reflect.packages` finds the modules in the first place. `mainPackage()` is the package being run, and [`modulesWhere`](#moduleswhere) selects from it by name.

This is how test discovery works: [`tests.runner`](../../tests/runner/index.md) asks for every module whose name matches a convention, then reflects over each one to find the functions carrying [`@tests.Test`](../../tests/index.md#test).

## Dependencies

- [`arrays`](../../arrays/index.md)
  - [`ranges`](../../ranges/index.md)

---

## Contents

- **Data** — [`Cavefile`](#cavefile), [`Dependency`](#dependency), [`Package`](#package), [`Param`](#param), [`Task`](#task)
- **Functions** — [`mainPackage`](#mainpackage), [`module`](#module), [`modulesExcept`](#modulesexcept), [`modulesWhere`](#moduleswhere)

---

## Data

### `Cavefile` {#cavefile}

<small>`reflect/packages/cavefile.zirr:5`</small>

```zirric
data Cavefile {
	name: String
	source: String
	docs: String
	dependencies: [Dependency]
	tasks: [Task]
	formattingExcludes: [String]
}
```

The manifest a package declares itself with, as `zirric cave describe` reports it.
It describes the project rather than its code: what it is called, what it depends on, and what it can be asked to do.

#### Fields

| Field                | Description                                                                                          |
| -------------------- | ---------------------------------------------------------------------------------------------------- |
| `name`               | The canonical name of the package, e.g. "code.knabel.dev.zirric_lang.zirric".                        |
| `source`             | Where the package comes from, as a URL or a file:// path, or the empty String when it declares none. |
| `docs`               | The comment written above the Cavefile's own `mod` declaration.                                      |
| `dependencies`       | Everything the package depends on, in the order the manifest declares it.                            |
| `tasks`              | Everything the package can be asked to do, in the order the manifest declares it.                    |
| `formattingExcludes` | The path patterns left unformatted, from @cave.FormattingExcludes.                                   |

---

### `Dependency` {#dependency}

<small>`reflect/packages/cavefile.zirr:21`</small>

```zirric
data Dependency {
	name: String
	kind: String
	source: String
	module: String
	version: String
	docs: String
}
```

One dependency of a package.

#### Fields

| Field     | Description                                                                             |
| --------- | --------------------------------------------------------------------------------------- |
| `name`    | The name it is imported under in this package.                                          |
| `kind`    | How it is reached: "stdlib", "git" or "local".                                          |
| `source`  | Where it comes from, as a URL or a file:// path.                                        |
| `module`  | The module a @cave.Stdlib dependency binds to, or the empty String for the other kinds. |
| `version` | The version predicates it was declared with, joined as they were written.               |
| `docs`    | The comment written above the field that declares it.                                   |

---

### `Package` {#package}

<small>`reflect/packages/packages.zirr:6`</small>

```zirric
data Package {
	name: String
	moduleNames: [String]
	cavefile: Option
}
```

A package and the modules it declares.

#### Fields

| Field         | Description                                                                                      |
| ------------- | ------------------------------------------------------------------------------------------------ |
| `name`        |                                                                                                  |
| `moduleNames` |                                                                                                  |
| `cavefile`    | The manifest the package declares itself with, or None for a project running without a Cavefile. |

---

### `Param` {#param}

<small>`reflect/packages/cavefile.zirr:59`</small>

```zirric
data Param {
	name: String
	declName: String
	type: String
	short: String
	aliases: [String]
	help: String
	docs: String
}
```

One flag or positional argument of a task.

#### Fields

| Field      | Description                                                                                    |
| ---------- | ---------------------------------------------------------------------------------------------- |
| `name`     | The name it is written under on the command line.                                              |
| `declName` | The name of the field it was declared as.                                                      |
| `type`     | The type it takes: "Bool", "String" or "Int", or the empty String for one the CLI cannot pass. |
| `short`    | The single-character form of a flag, from @tasks.Short, or the empty String.                   |
| `aliases`  | The other names it answers to.                                                                 |
| `help`     | The one-line help the CLI prints, from @tasks.Help.                                            |
| `docs`     | The comment written above the field that declares it.                                          |

---

### `Task` {#task}

<small>`reflect/packages/cavefile.zirr:37`</small>

```zirric
data Task {
	name: String
	kind: String
	declName: String
	aliases: [String]
	help: String
	exec: String
	docs: String
	flags: [Param]
	args: [Param]
}
```

One task a package can be asked to run.

#### Fields

| Field      | Description                                                                                                    |
| ---------- | -------------------------------------------------------------------------------------------------------------- |
| `name`     | The name it is run under, which @tasks.Name may have changed from the declaration's own.                       |
| `kind`     | How it runs: "exec" for a file, "call" for a function.                                                         |
| `declName` | The name of the declaration it was written as.                                                                 |
| `aliases`  | The other names it answers to.                                                                                 |
| `help`     | The one-line help the CLI prints, from @tasks.Help.                                                            |
| `exec`     | The file an "exec" task runs, or the empty String for a "call" task.                                           |
| `docs`     | The comment written above the declaration, which is what it says about itself rather than what the CLI prints. |
| `flags`    | The flags it takes.                                                                                            |
| `args`     | The positional arguments it takes, in order.                                                                   |

---

## Functions

### `mainPackage` {#mainpackage}

<small>`reflect/packages/packages.zirr:21`</small>

```zirric
fn mainPackage() -> Package
```

The package the running project itself declares.
Importing this module compiles every module in that package, including ones no import reaches, so that they can be looked up by name later.
Left out are the entry module, since the running program cannot be loaded again, and any module whose unqualified name is already taken by a loaded module, since it could never be imported anyway.

---

### `module` {#module}

<small>`reflect/packages/packages.zirr:34`</small>

```zirric
fn module(p: Package, name: String) -> Option
```

The module of p called name, or None if p declares no such module.
Loading a module runs its top-level code the first time it is reached.

---

### `modulesExcept` {#modulesexcept}

<small>`reflect/packages/packages.zirr:65`</small>

```zirric
fn modulesExcept(p: Package, names: [String]) -> [AnyModule]
```

Every module of p except those named.
Loading a module runs its top-level code, so naming the ones to leave out is the point: there is deliberately no way to ask for all of them without saying so.

---

### `modulesWhere` {#moduleswhere}

<small>`reflect/packages/packages.zirr:46`</small>

```zirric
fn modulesWhere(p: Package, predicate: fn(String) -> Bool) -> [AnyModule]
```

Every module of p whose name satisfies predicate.
Each name is tested before its module is loaded, so a module the predicate rejects never runs.
