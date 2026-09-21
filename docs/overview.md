---
title: "Overview"
description: "Start here for Zirric language docs, guides, and references."
---

Zirric is an experimental programming language with a reference implementation in Go. This documentation focuses on how the language feels to use, how the tooling is shaped, and where to dig deeper into the implementation.

It favors small, explicit building blocks: declarations over inheritance, attributes over interfaces, and expression-oriented control flow. The standard library is written in Zirric itself, and the language is designed so Zirric code is easy to reason about.

::: callout warning Experimental
Zirric is evolving quickly. Expect incomplete features, shifting syntax, and ongoing proposals. Use the proposals as the authoritative roadmap.
:::

## Zirric in a nutshell

```zirric
attr Countable {
	length(value: @Countable) -> Int
}

@Countable(fn(v) { return v.length })
data Bag {
	items
	length
}

fn summarize(bag: Bag) -> Result {
	const length = Countable(bag).length(bag)

	return if length > 0 {
		Ok(length)
	} else {
		Err("empty")
	}
}
```

## Start here

- New to Zirric? Begin with the [Getting Started](/guides/getting-started) guide.
- Need install instructions? Jump to the [Installation](/guides/installation) page.
- Want conventions before writing code? Read the [Styleguide](/guides/styleguide).

## Language specification

The language is formally specified in the specification section:

- [Syntax](/specification/syntax) — formal grammar reference
- [Declarations](/specification/declarations) — declaration forms, scoping, attributes
- [Expressions](/specification/expressions) — expressions, control flow, closures
- [Type System](/specification/typesystem) — types, type hints, type checking

When you want more depth or future-facing design notes, read the [Zirric Evolution Proposals](/proposals).

## Runtime and packages

Zirric ships with a [standard library](/stdlib) written in Zirric itself — from `prelude` and the collection modules through `io`, `fs` and `os` to reflection, serialization and testing — plus a package system called Cavefile.

- [Standard Library](/stdlib) — every module, grouped by what it is for
- [Cavefile manifests](/cavefile) — declaring dependencies and tasks
- [Compiler architecture](/tooling/compiler) — bytecode and the VM

## Tooling

The whole toolchain is one binary. See the [Tooling](/tooling) section, starting with the [Zirric CLI](/tooling/zirric-cli) for what `zirric` does and [Editor Configuration](/tooling/editor-configuration) for setting up your editor.

## Working on Zirric

If you are hacking on the compiler or VM, the repository README explains the build and test workflow. The `docs/` folder is built with docmd and outputs to `site/`.
