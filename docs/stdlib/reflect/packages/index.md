---
title: Reflect.Packages
description: Discovering the modules that make up the running package.
---

# Module `reflect.packages`

> Finding modules by name, or by asking.

```zirric
import reflect.packages
// or: import pkgs = reflect.packages
```

|            |                                                                                                                               |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `reflect.packages`                                                                                                            |
| **Source** | [`reflect/packages/packages.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/reflect/packages/packages.zirr) |

Where [`reflect`](../index.md) inspects a module you already hold, `reflect.packages` finds the modules in the first place. [`mainPackage()`](#mainpackage) is the package being run, and [`modulesWhere`](#moduleswhere) selects from it by name.

This is how test discovery works: the [runner](../../tests/runner/index.md) asks for every module whose name matches a convention, then reflects over each one to find the functions carrying [`@tests.Test`](../../tests/index.md#test).

## Contents

- **Data** — [`Package`](#package)
- **Functions** — [`mainPackage`](#mainpackage), [`module`](#module), [`modulesWhere`](#moduleswhere), [`modulesExcept`](#modulesexcept)

---

## Data

### `Package` {#package}

<small>`reflect/packages/packages.zirr:6`</small>

```zirric
data Package {
	name: String
	moduleNames: [String]
}
```

A package and the modules it declares.

#### Fields

| Field         | Signature               | Description                                    |
| ------------- | ----------------------- | ---------------------------------------------- |
| `name`        | `name: String`          | The package name, as the Cavefile declares it. |
| `moduleNames` | `moduleNames: [String]` | Every module the package contains.             |

---

## Functions

### `mainPackage` {#mainpackage}

<small>`reflect/packages/packages.zirr:18`</small>

```zirric
fn mainPackage() -> Package
```

The package the running project itself declares. Importing this module compiles every module in that package, including ones no import reaches, so that they can be looked up by name later. Left out are the entry module, since the running program cannot be loaded again, and any module whose unqualified name is already taken by a loaded module, since it could never be imported anyway.

---

### `module` {#module}

<small>`reflect/packages/packages.zirr:24`</small>

```zirric
fn module(p: Package, name: String) -> Option
```

The module of p called name, or None if p declares no such module. Loading a module runs its top-level code the first time it is reached.

---

### `modulesWhere` {#moduleswhere}

<small>`reflect/packages/packages.zirr:36`</small>

```zirric
fn modulesWhere(p: Package, predicate: fn(String) -> Bool) -> [Module]
```

Every module of p whose name satisfies predicate. Each name is tested before its module is loaded, so a module the predicate rejects never runs.

---

### `modulesExcept` {#modulesexcept}

<small>`reflect/packages/packages.zirr:55`</small>

```zirric
fn modulesExcept(p: Package, names: [String]) -> [Module]
```

Every module of p except those named. Loading a module runs its top-level code, so naming the ones to leave out is the point: there is deliberately no way to ask for all of them without saying so.

---

## See also

- [`reflect`](../index.md) — inspecting the modules this returns.
- [`tests.runner`](../../tests/runner/index.md) — discovery built on top of this.
