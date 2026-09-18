---
title: ZE-002 - The Cavefile
description: A proposal to document the Zirric package manager and the Cavefile manifest.
---

# The Cavefile

::: callout tip Implemented
This proposal has been accepted and implemented.
You can use this feature since Zirric [v0.1.0](/changelog/v0.1.0).
:::

## Introduction

This proposal describes the Zirric package manager and the `Cavefile` manifest that drives it. The goal is to document its behaviour so users and contributors understand how dependencies and tasks are declared, resolved, and run.

## Motivation

Zirric projects will rely on external modules for language extensions and tooling.
Capturing the present-day and near future behaviour of the package manager clarifies how packages are discovered, how versions are selected, and how registries interact with the filesystem cache. Documenting the `Cavefile` structure likewise gives
users a reference for authoring manifests that match the implementation.

## Proposed Solution

Packages declare dependencies in a `Cavefile` module that Zirric parses at build or install time. The manifest itself will not be executed. The package manager solely works on the type system.

The package manager itself caches Git repositories on disk and coordinates one or more registries to provide the requested packages. Currently Git, local files and built-in registries are supported.

As of now the package manager does not traverse transitive dependencies, leaving that for a future enhancement.

### Cavefile Dependencies

The data structure with the `@cave.Dependencies()` attribute will be used to declare dependencies in a `Cavefile`.

```zirric
import cave

@cave.Dependencies()
data Dependencies {
  // Standard library packages are available by default
  @cave.Stdlib("prelude")
  prelude

  // References a local package. The path is relative to the Cavefile location.
  @cave.Local("../some-local-package")
  helpers // importable as helpers

  // References a package hosted in a Git repository.
  @cave.Git("https://code.knabel.dev/zirric-lang/zirric")
  @cave.Version(">0.1.0")
  zirric // importable as zirric
}
```

Only the entries produced by `@cave.Dependencies` participate in dependency
resolution, but additional declarations such as tasks can live alongside them.

### Cavefile Tasks

Additionally to dependencies, a `Cavefile` can declare tasks that the Zirric CLI can execute.

```zirric
@tasks.Name("generate")
@tasks.Help("Generates something")
@tasks.Exec("tasks/generate.zirr")
data GenerateTask {
  @tasks.Flag()
  @tasks.Name("dry")
  @tasks.Help("If true, only simulates the generation")
  isDryRun: Bool

  @tasks.Arg()
  positional: String
}

@tasks.Name("build")
@tasks.Call(fn(opts: BuildTask) {
  // ...
})
data BuildTask {
  @tasks.Flag()
  dry: Bool

  @tasks.Arg()
  target: String
}
```

This will declare a task named `generate` that can be executed with `zirric task run generate` (or the `zirric x generate` shorthand).
Upon execution, the `tasks/generate.zirr` script is run as its own program; it reads its own flags and arguments via `os.args()`, the same way any Zirric script would.

`build` instead runs by calling the inline function passed to `@tasks.Call` directly, with a single argument: an instance of `BuildTask` built from the parsed, typed flag and argument values.

## Detailed Design

### Dependencies manifest structure

The `pkgmanager` will use the `Cavefile`, search for the `@cave.Dependencies()` data structure and parse its fields as follows:

- **Field name** – the import name for the dependency package. Submodules are supported by
  allowing dot notation, e.g. `foo.bar` imports the `bar` submodule from the
  `foo` package.
- **Source** - determined by the presence of `@cave.Git`, `@cave.Local`, or
  `@cave.Stdlib` attributes on the field.
- **Version predicate** – extracted from the `@cave.Version` attribute if
  present. If omitted, any version is acceptable.

### Package manager workflow

`pkgmanager.New` initialises a `PackageManager` with the configured registries.
The default constructor mounts a Git registry under the `git/` directory of the
provided filesystem so cached repositories are stored beneath that path.

`PackageManager.Install` creates an `InstallationTask` bound to a parsed
Cavefile. Running the task performs the following steps:

1. **Queue setup** – the first execution copies the manifest dependencies into a
   work queue so repeated runs reuse the same slice.
2. **Local discovery** – each registry reports the packages already cached on
   disk. Results are grouped by source so matching versions can be reused.
3. **Remote resolution** – unmet dependencies trigger `DiscoverPackageVersions`
   on every registry. The installer selects the first offered version (registries
   return results sorted from newest to oldest), resolves it to a local clone,
   and records the package.

If no registry can satisfy a dependency, the run terminates with an error. The
installer does not yet traverse transitive dependencies, leaving that work for a
future enhancement.

### Registry abstraction

Registries implement the `registry.Provider` interface. They surface:

- `Discover`, which returns all locally available packages and versions.
- `DiscoverPackageVersions`, which lists remote versions matching supplied
  predicates.

Packages resolved from either path expose module discovery helpers so the rest
of the toolchain can load `.zirr` sources. Modules advertise their logical URI
and enumerate the files contained within the checkout.

### Git registry provider

The bundled `GitRegistry` manages repositories inside its root filesystem. Local
packages are stored as `git/<source>/<version>/` beneath the registry root.
Local discovery iterates those directories, opens each Git worktree, and lists
any tags that match the checked-out commit. Remote discovery connects to the
upstream repository, collects available tags, filters them by the provided
predicates, and sorts them in descending semantic-version order before returning
packages to the installer.

When a remote version is selected, the registry clones the tagged commit into a
mangled `<source>/<version>/` directory. Module enumeration delegates to the
filesystem module discovery helper so every `.zirr` source within the repo is
published to the toolchain.

### Task execution and parsing

The `future.tasks` package provides attributes and helpers to declare and execute tasks.
Data structures may be tasks when annotated with `@tasks.Exec` to execute files or `@tasks.Call` to call functions.

They will be parsed by the CLI and registered as commands. By default the command name is the lowercased data name, but it may be overridden with `@tasks.Name`. A help text may be provided with `@tasks.Help`.

Tasks may declare flags and positional arguments by annotating fields with `@tasks.Flag` and `@tasks.Arg`. For a `@tasks.Call` task, each field's type must be `Bool`, `String`, or `Int`; the CLI registers a real, typed flag or positional argument per field (via Cobra), which is also what makes `--help` output and shell completions for task-defined commands possible. A `@tasks.Exec` task's declared fields have no effect on the CLI: flag parsing is disabled for it, and the script parses its own flags and arguments from `os.args()`.

## Changes to the Standard Library

Introduces the `cave` and `future.tasks` modules to the standard library.

- `cave`:
  - `Dependencies` attribute
  - a union for `Source` with values `Stdlib`, `Local`, and `Git`
  - `Stdlib`, `Local`, `Git`, and `Version` attributes
- `future.tasks`:
  - a union for `Task` with values `Exec` and `Call`
  - `Exec` and `Call` attributes for task declarations
  - `Name`, `Alias`, `Help`, `Short`, `Flag`, and `Arg` attributes for tasks and their fields

## Alternatives Considered

- Using a different manifest format such as JSON or YAML was considered, but
  Zirric's strong typing and attribute system makes it straightforward to
  declare dependencies and tasks directly in Zirric code.
- Implementing transitive dependency resolution was considered, but deferred to a future enhancement to keep the initial implementation simpler and focused on
  direct dependencies only.
- Supporting additional registry types (e.g., HTTP-based registries) was considered, but the initial implementation focuses on Git and local files to establish a solid foundation before expanding to other sources.
- Using a different approach for task declaration, such as a dedicated task configuration file, was considered, but integrating tasks into the `Cavefile` keeps related configurations together and leverages Zirric's type system.
- A `package.zirr` file was considered, but the `.zirr` file extension would require additional tooling support and could lead to confusion with regular Zirric source files.

## Acknowledgements

Thanks to the Zirric maintainers for building the initial package manager and
Cavefile tooling that this document captures.
