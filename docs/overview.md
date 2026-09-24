---
title: "Overview"
description: "Start here for Zirric language docs, guides, and references."
---

Zirric is an experimental programming language with a reference implementation in Go. This documentation focuses on how the language feels to use, how the tooling is shaped, and where to dig deeper into the implementation.

::: callout warning Experimental
Zirric is evolving quickly. Expect incomplete features and shifting syntax. The [Specification](/specification/syntax) describes the language as it is implemented; the [proposals](/proposals) record how it got there and are not kept in step with it.
:::

## Philosophy

Zirric is built on composability, simplicity and transparency. Just [data types](/specification/typesystem#data-types), [unions](/specification/typesystem#union-types) and [attributes](/specification/typesystem#attribute-types). No classes, inheritance, generics, interfaces, optional conformances or ambiguity. No magic. No exceptions.

Zirric lacks many features and has only a few powerful ones, and those that exist are curated to work well together:

| Declaration                                  | What it is for                     |
| -------------------------------------------- | ---------------------------------- |
| [`data`](/specification/declarations#data)   | holds value                        |
| [`union`](/specification/declarations#union) | gives structure                    |
| [`attr`](/specification/declarations#attr)   | provides context                   |
| [`fn`](/specification/declarations#fn)       | does stuff                         |
| [`mod`](/specification/declarations#mod)     | prevents you from losing your mind |

The other half of the philosophy is to leverage features that already exist rather than to add new ones. Take `for` loops. Used as an expression, a loop collects what its body produces, and `continue` drops a value — so the same loop you would write for its side effects is also the map and the filter. Literally.

```zirric
const evenSquares = for n <- Range(0, 10) {
	if n % 2 == 0 {
		n * n
	} else {
		continue
	}
}
// [0, 4, 16, 36, 64]
```

The pattern repeats. `for element <- value` is a call into the [`Iterable`](/stdlib/prelude#iterable) attribute rather than a built-in over built-in types, so any type carrying it can be looped over. `len` is a call into [`Countable`](/stdlib/prelude#countable). The [Cavefile](/cavefile) is Zirric source rather than a config format — a `data` declaration carrying attributes, read back with the same [reflection](/stdlib/reflect) a program can use on itself. Nothing in that list needed a language feature of its own.

The standard library is basic, but it tries to make the right choices easier than the ones that bite you. Everything is built with testing in mind, and you notice bad habits right at the import: [`os`](/stdlib/os) is the only module that talks to the machine, so a function reaching for it directly cannot be tested without one. Take a [`fs.FileSystem`](/stdlib/fs), a [`clock.SystemClock`](/stdlib/clock) or a [`random.Source`](/stdlib/random) as a parameter instead, and a test hands over `fs.memory()`, `clock.fixed(…)` or `random.seeded(…)` without anything else changing.

## Zirric in a nutshell

```zirric
import math

attr Area {
	of(shape: @Area) -> Float
}

union Shape {
	@Area(fn(c) { math.pi * c.radius * c.radius })
	data Circle { radius: Float }

	@Area(fn(r) { r.width * r.height })
	data Rect { width: Float, height: Float }
}

fn largest(shapes: [Shape]) -> Shape? {
	var best = None()
	for shape <- shapes {
		if best is None || Area(shape).of(shape) > Area(best.value).of(best.value) {
			best = Some(shape)
		}
	}
	return best
}
```

A `data` type holds the values, the `union` says which shapes there are, the attribute says what each one can do, and `Shape?` says the answer may be absent. No interface was declared and no base type was inherited from.

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
