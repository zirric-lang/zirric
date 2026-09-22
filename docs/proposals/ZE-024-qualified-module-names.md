---
title: "ZE-024 - Qualified Module Names"
description: "Every source file names the module it belongs to, by its fully qualified path."
---

# Qualified Module Names

::: callout tip Implemented
This proposal has been accepted and implemented. You can use this feature since Zirric [v0.1.0](/changelog/v0.1.0).
:::

## Introduction

`mod` used to be an optional convenience that bound the current module to a local name of the author's choosing. It said nothing about which module the file actually belonged to — that was decided entirely by the directory the file sat in.

This proposal makes `mod` required, gives it the module's fully qualified path, and ties that path to the package base the `Cavefile` declares.

## Motivation

A Zirric file used to carry no record of where it belonged. `prelude/shim.zirr` said `mod prelude`, `tests/tap/tap-reporter.zirr` said `mod tap`, and both names were chosen freely: nothing checked them, nothing used them, and a file moved between directories kept saying whatever it said before.

That costs more than tidiness:

- **A file read on its own is ambiguous.** Two modules called `tap` in different packages read identically, and a reviewer looking at a diff has only the file path to tell them apart.
- **Tools have to guess.** An editor opening a single file, a diagnostic naming a module, a stack frame — each had to reconstruct the module from the path relative to a `Cavefile` it first had to find.
- **A rename goes unnoticed.** Moving `flow/node` to `graph/node` left every `mod node` in place and correct-looking, while the module it named had changed.
- **The package had no name of its own.** The project's identity came from its directory name, or from a URL passed to an attribute, which meant the same sources produced different module names depending on where they were checked out.

Naming the module in full, in every file, makes each file say where it belongs and lets that claim be checked.

## Proposed Solution

Every source file declares the module it belongs to, by its fully qualified path:

```zirric
mod code.knabel.dev.zirric_lang.ui.flow.node
```

The last segment binds the module locally, so the file refers to itself as `node`:

```zirric
mod code.knabel.dev.zirric_lang.ui.flow.node

const version = "1.0"

fn describe() -> String {
	node.version
}
```

A file that wants a different local name says so:

```zirric
mod here = code.knabel.dev.zirric_lang.ui.flow.node
```

The base every module is named under is the one the package's `Cavefile` declares with its own `mod`:

```zirric
// Cavefile
mod code.knabel.dev.zirric_lang.ui
```

With that base, a file at `./flow/node/graph.zirr` must declare `mod code.knabel.dev.zirric_lang.ui.flow.node`, and a file at the package root must declare `mod code.knabel.dev.zirric_lang.ui`.

### Rules

1. Every source file of a package declares a module.
2. The declared path is the package base joined with the path from the package root to the file's directory.
3. Every file of one module declares the same path.
4. The last segment binds the module locally, unless `mod local = path` names something else.

## Detailed Design

### Grammar

```ebnf
Module     = "mod", [Identifier, "="], ModulePath;
ModulePath = Identifier, {".", Identifier};
```

`mod` keeps its position: first in the file, after an optional shebang.

### Directory names and path segments

A path segment is the directory name canonicalized the way module URIs already are: lowercased, with `-` replaced by `_`. A directory `flow-nodes` is therefore the segment `flow_nodes`. Directories that cannot become an identifier cannot hold a module.

### What is checked, and when

The rules above are checked while analyzing a module of the package the `Cavefile` describes. They are deliberately not checked elsewhere:

- **A project with no `Cavefile`** — a loose script, the REPL — has no base to qualify against. Its files may declare whatever they like, since there is nothing to hold them to.
- **A dependency** is checked when it is built as its own package, not again through every package that uses it. A package that ships its modules under short names, as the embedded standard library does, keeps working.

A violation is an analysis error, carrying the position of the declaration and the name the file should have declared.

### The Cavefile's own module

The `Cavefile` is a Zirric source file and follows the same rule: its `mod` is the package's base path, and it binds the last segment locally. That declaration is what gives the package its name — `zirric cave describe` reports it as `package.name` — instead of the directory the project happens to sit in.

`zirric cave new` writes a `Cavefile` whose `mod` is derived from the directory it runs in.

### Files run by path

A file run directly (`zirric run flow/node/graph.zirr`) belongs to the module its directory names, exactly as it would when reached through an import. It is the program being run either way, and its top-level code runs.

## Changes to the Standard Library

No new declarations. Every standard library source now names itself in full, e.g. `prelude/shim.zirr` declares `mod code.knabel.dev.zirric_lang.zirric.prelude`. The local binding is unchanged — it remains the last segment, `prelude`.

## Alternatives Considered

- **Keeping `mod` optional and deriving everything from the path.** This is what Zirric did. It works for the compiler, which always knows the path, and fails everyone else: a file, a diff, or a stack frame read on its own still cannot say which module it belongs to.
- **Declaring only the path relative to the package** (`mod flow.node`). Shorter, but it reintroduces the ambiguity between packages that the fully qualified form removes, and it leaves the package's own name undeclared.
- **Binding the whole path rather than its last segment** (`code.knabel.dev.zirric_lang.ui.flow.node.version`). Faithful, unusable. The last segment is what a reader means by "this module", and `mod local = path` covers the cases where it collides.
- **Deriving the package base from the Git URL** rather than from `mod`. The URL is where a package is fetched from, not what it is called; a fork, a mirror or a local checkout would each rename every module in the package.

## Acknowledgements

Go's module paths and Java's package declarations both name a compilation unit's home in full; this proposal follows them, and adds the alias form so that a long path need not become a long local name.
