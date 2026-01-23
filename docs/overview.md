---
title: "Overview"
description: "Start here for Zirric language docs, guides, and references."
---

Zirric is an experimental programming language with a reference implementation in Go.
This documentation focuses on how the language feels to use, how the tooling is
shaped, and where to dig deeper into the implementation.

It favors small, explicit building blocks: declarations over inheritance,
annotations over interfaces, and expression-oriented control flow. The standard
library is written in Zirric itself, and the language is designed so Zirric code
is easy to reason about.

::: callout warning Experimental
Zirric is evolving quickly. Expect incomplete features, shifting syntax, and
ongoing proposals. Use the proposals as the authoritative roadmap.
:::

## Zirric in a nutshell

```zirric
annotation Countable {
    @Returns(Int)
    length(@Has(Countable) value)
}

@Countable({ v -> v.length })
data Bag {
    items
    length
}

@Returns(Result)
func summarize(@Bag bag) {
    let length = Countable(bag).length(bag)

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

## Language references

The core language surface is documented in the syntax references:

- [Expressions](/syntax/expressions)
- [Declarations](/syntax/declarations)
- [Control flow](/syntax/control-flow)
- [Annotations](/syntax/annotations)
- [Typesystem](/syntax/typesystem)

When you want more depth or future-facing design notes, read the
[Zirric Evolution Proposals](/proposals).

## Runtime and packages

Zirric ships with a standard library written in Zirric itself under `stdlib/` and
a package system called Cavefile.

- [Cavefile manifests](/cavefile)
- [Compiler architecture](/tooling/compiler)

## Tooling

Tooling documentation lives in the [Tooling](/tooling) section. Start with the
Tree-sitter grammar if you want syntax highlighting in editors.

## Working on Zirric

If you are hacking on the compiler or VM, the repository README explains the
build and test workflow. The `docs/` folder is built with docmd and outputs to
`site/`.
