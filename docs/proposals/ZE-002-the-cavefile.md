---
title: ZE-002 - The Cavefile
description: A proposal to document the Zirric package manager and the Cavefile manifest.
---

# The Cavefile

::: callout tip Implemented
This proposal has been accepted and implemented. You can use this feature since Zirric [v0.1.0](/changelog/v0.1.0).
:::

## Introduction

This proposal describes the Zirric package manager and the `Cavefile` manifest that drives it. The goal is to document its behaviour so users and contributors understand how dependencies and tasks are declared, resolved, and run.

## Motivation

Zirric projects will rely on external modules for language extensions and tooling. Capturing the present-day and near future behaviour of the package manager clarifies how packages are discovered, how versions are selected, and how registries interact with the filesystem cache. Documenting the `Cavefile` structure likewise gives users a reference for authoring manifests that match the implementation.

## Proposed Solution

Packages declare dependencies in a `Cavefile` module that Zirric parses at build or install time. The manifest itself will not be executed. The package manager solely works on the type system.

The package manager itself caches Git repositories on disk and coordinates one or more registries to provide the requested packages. Currently Git, local files and built-in registries are supported.

As of now the package manager does not traverse transitive dependencies, leaving that for a future enhancement.

### The Package Manifest

The data structure with the `@cave.Package()` attribute is the package manifest: its attributes describe the package itself, its fields declare the dependencies.

```zirric
mod code.knabel.dev.zirric_lang.ui

import cave

@cave.Package()
@cave.Git("https://code.knabel.dev/zirric-lang/ui")
@cave.Version("1.2.3")
@cave.LanguageVersion("^0.1.0")
@cave.Description("Widgets and layout for Zirric programs.")
@cave.Documentation("https://zirric.knabel.dev/ui")
data UI {
  // Standard library packages are available by default
  @cave.Stdlib("prelude")
  prelude

  // References a local package. The path is relative to the Cavefile location.
  @cave.Local("../some-local-package")
  helpers // importable as helpers

  // References a package hosted in a Git repository.
  @cave.Git("https://code.knabel.dev/zirric-lang/zirric")
  @cave.Version(">0.1.0")
  @cave.Description("The Zirric standard library")
  @cave.Documentation("https://zirric.knabel.dev")
  zirric // importable as zirric
}
```

The `Cavefile`'s own `mod` declares the package's base module path; see [ZE-024 Qualified Module Names](/proposals/ZE-024-qualified-module-names) for how every module of the package is named under it.

Only the fields of the `@cave.Package()` declaration participate in dependency resolution, but additional declarations such as tasks can live alongside it.

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

This will declare a task named `generate` that can be executed with `zirric task generate`, or — since no built-in command is called `generate` — with `zirric generate`. Upon execution, the `tasks/generate.zirr` script is run as its own program; it reads its own flags and arguments via `os.args()`, the same way any Zirric script would.

`build` instead runs by calling the inline function passed to `@tasks.Call` directly, with a single argument: an instance of `BuildTask` built from the parsed, typed flag and argument values.

## Detailed Design

### Manifest structure

The `pkgmanager` will use the `Cavefile`, search for the `@cave.Package()` data structure, and read it as follows.

The attributes on the declaration describe the package itself:

- **`@cave.Git`** – the canonical URL the package is published at.
- **`@cave.Version`** – the package's own version.
- **`@cave.LanguageVersion`** – a predicate the running Zirric toolchain must satisfy. A toolchain that does not refuses the project outright, before any package is fetched or any source is touched; only `zirric cave describe` still opens it, since reporting what the manifest says is how the refusal is explained. A development build that carries no version of its own is exempt, since it cannot be told apart from one that is too old. On a dependency the predicate is the dependency's own, and refuses installing it.
- **`@cave.Description`** – a one-line description.
- **`@cave.Documentation`** – the URL of the package's documentation.

Its fields declare the dependencies:

- **Field name** – the import name for the dependency package. Submodules are supported by allowing dot notation, e.g. `foo.bar` imports the `bar` submodule from the `foo` package.
- **Source** - determined by the presence of `@cave.Git`, `@cave.Local`, or `@cave.Stdlib` attributes on the field.
- **Version predicate** – extracted from the `@cave.Version` attribute if present. If omitted, any version is acceptable.
- **Description, documentation and language version** – `@cave.Description`, `@cave.Documentation` and `@cave.LanguageVersion` describe a dependency the same way they describe the package.

`zirric cave describe` prints all of it: the package under `package:`, the dependencies under `dependencies:`.

`zirric cave new` writes the manifest. A package must be named, and where possible it is named by what the checkout already knows: the Git remote is recorded as `@cave.Git` and canonicalized into the `mod` declaration, so a repository at `git@code.knabel.dev:example/widgets.git` becomes the package `code.knabel.dev.example.widgets`. Every attribute has a flag of its own; `@cave.LanguageVersion` defaults to `>=` the running Zirric, and an attribute nothing was said about is left out rather than written empty.

### Package manager workflow

`pkgmanager.New` initialises a `PackageManager` with the configured registries. The default constructor mounts a Git registry under the `git/` directory of the provided filesystem so cached repositories are stored beneath that path.

`PackageManager.Install` creates an `InstallationTask` bound to a parsed Cavefile. Running the task performs the following steps:

1. **Queue setup** – the first execution copies the manifest dependencies into a work queue so repeated runs reuse the same slice.
2. **Local discovery** – each registry reports the packages already cached on disk. Results are grouped by source so matching versions can be reused.
3. **Remote resolution** – unmet dependencies trigger `DiscoverPackageVersions` on every registry. The installer selects the first offered version (registries return results sorted from newest to oldest), resolves it to a local clone, and records the package.

If no registry can satisfy a dependency, the run terminates with an error. The installer does not yet traverse transitive dependencies, leaving that work for a future enhancement.

### Registry abstraction

Registries implement the `registry.Provider` interface. They surface:

- `Discover`, which returns all locally available packages and versions.
- `DiscoverPackageVersions`, which lists remote versions matching supplied predicates.

Packages resolved from either path expose module discovery helpers so the rest of the toolchain can load `.zirr` sources. Modules advertise their logical URI and enumerate the files contained within the checkout.

### Git registry provider

The bundled `GitRegistry` manages repositories inside its root filesystem. Local packages are stored as `git/<source>/<version>/` beneath the registry root. Local discovery iterates those directories, opens each Git worktree, and lists any tags that match the checked-out commit. Remote discovery connects to the upstream repository, collects available tags, filters them by the provided predicates, and sorts them in descending semantic-version order before returning packages to the installer.

When a remote version is selected, the registry clones the tagged commit into a mangled `<source>/<version>/` directory. Module enumeration delegates to the filesystem module discovery helper so every `.zirr` source within the repo is published to the toolchain.

### Task execution and parsing

The `future.tasks` package provides attributes and helpers to declare and execute tasks. Data structures may be tasks when annotated with `@tasks.Exec` to execute files or `@tasks.Call` to call functions.

They will be parsed by the CLI and registered as commands. By default the command name is the lowercased data name, but it may be overridden with `@tasks.Name`. A help text may be provided with `@tasks.Help`.

Tasks may declare flags and positional arguments by annotating fields with `@tasks.Flag` and `@tasks.Arg`. For a `@tasks.Call` task, each field's type must be `Bool`, `String`, or `Int`; the CLI registers a real, typed flag or positional argument per field (via Cobra), which is also what makes `--help` output and shell completions for task-defined commands possible. A `@tasks.Exec` task's declared fields do not drive its own flag parsing: flag parsing is disabled for it, and the script reads its own flags and arguments from `os.args()`. They still say what the task accepts, which is what lets a built-in command decide whether it may be called through to.

### Where a task appears in the CLI

Every task is a subcommand of `task`, so `zirric task` lists them and `zirric task <name>` runs one.

A task is also registered at the top level, so that `zirric <name>` runs it — but only when nothing built in already answers to that name or alias. The built-in always wins: `zirric run` and `zirric fmt` cannot be redefined by declaring a task. A task whose name is taken stays reachable as `zirric task <name>`.

Since a name a built-in holds is also the most natural one for a project to want, a built-in command may **call a task of its own name through itself**. `zirric fmt` formats `.zirr` sources and then runs a declared `fmt` task, so that one command covers the whole project.

Calling through is conditional, because a task that cannot be told what was asked of it must not be run:

- The task runs only when it declares a flag for **every flag the caller actually set**. `zirric fmt --check` runs a `fmt` task that declares `check`; against one that does not, the task is left unrun and the reason is printed. Running it would write where the caller asked only to look.
- The check is by flag name, whatever the value: `--write=false` needs the task to declare `write` just as `--write` does.
- A `@tasks.Exec` task receives the flags in its `os.args()`, written back the way they were passed. A `@tasks.Call` task receives them as its declared fields.
- The command exits zero only when both the built-in's own work and the task succeeded.

There is a second shape, where the task runs **instead of** the built-in rather than beside it. `zirric test` takes it: a declared `test` task is what `zirric test` means, and with no such task the command runs the standard library's own runner. It suits a command whose built-in behaviour is a default rather than a foundation.

`fmt` and `test` are the first commands to work this way; others follow the same two rules.

## Changes to the Standard Library

Introduces the `cave` and `future.tasks` modules to the standard library.

- `cave`:
  - `Package` attribute, marking the package manifest
  - a union for `Source` with values `Stdlib`, `Local`, and `Git`
  - `Stdlib`, `Local`, and `Git` attributes for dependency sources
  - `Version`, `LanguageVersion`, `Description`, and `Documentation` attributes, which describe either the package or one of its dependencies
  - `FormattingExcludes` attribute, excluding paths from `zirric fmt`
- `future.tasks`:
  - a union for `Task` with values `Exec` and `Call`
  - `Exec` and `Call` attributes for task declarations
  - `Name`, `Alias`, `Help`, `Short`, `Flag`, and `Arg` attributes for tasks and their fields

## Alternatives Considered

- Using a different manifest format such as JSON or YAML was considered, but Zirric's strong typing and attribute system makes it straightforward to declare dependencies and tasks directly in Zirric code.
- Implementing transitive dependency resolution was considered, but deferred to a future enhancement to keep the initial implementation simpler and focused on direct dependencies only.
- Supporting additional registry types (e.g., HTTP-based registries) was considered, but the initial implementation focuses on Git and local files to establish a solid foundation before expanding to other sources.
- Using a different approach for task declaration, such as a dedicated task configuration file, was considered, but integrating tasks into the `Cavefile` keeps related configurations together and leverages Zirric's type system.
- A `package.zirr` file was considered, but the `.zirr` file extension would require additional tooling support and could lead to confusion with regular Zirric source files.

## Acknowledgements

Thanks to the Zirric maintainers for building the initial package manager and Cavefile tooling that this document captures.
