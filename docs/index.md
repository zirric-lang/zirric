---
title: Zirric
description: A small language built on data, unions and attributes. No classes, no generics, no magic.
layout: "full"
toc: false
---

![Zirric](/assets/images/zirric.svg){ .size-small .align-center }

::: hero glow:true layout:split

# No magic.

Zirric is built on composability, simplicity and transparency. Just [data types](/specification/typesystem#data-types), [unions](/specification/typesystem#union-types) and [attributes](/specification/typesystem#attribute-types). No classes, inheritance, generics, interfaces, optional conformances or ambiguity. No exceptions.

::: button "Get Started" /guides/getting-started
::: button "Read the Specification" /specification/syntax

== side

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

const shapes = [Circle(1.0), Rect(4.0, 5.0), Circle(3.0)]

// `for` is the map and the filter.
const bigAreas = for shape <- shapes {
	const area = Area(shape).of(shape)
	if area > 10.0 {
		area
	} else {
		continue
	}
}
```

:::

## Five things to learn

Zirric lacks many features and has only a few powerful ones, and those that exist are curated to work well together. `data` types hold value, unions give structure, attributes provide context, functions do stuff, and modules prevent you from losing your mind.

::: grids
::: grid
::: card "`data` holds value" icon:package
Records with named fields, compared field by field, constructed by calling the type. [Data types →](/specification/typesystem#data-types)
:::
:::

::: grid
::: card "`union` gives structure" icon:shapes
A closed set of member types. A value belongs by being one of them — no wrapping step. [Union types →](/specification/typesystem#union-types)
:::
:::

::: grid
::: card "`attr` provides context" icon:tags
Capabilities and metadata, declared on the type and read back by anything that asks. [Attributes →](/specification/typesystem#attribute-types)
:::
:::

::: grid
::: card "`fn` does stuff" icon:braces
First-class functions and closures, with one syntax for both. [Closures →](/specification/expressions#closures)
:::
:::

::: grid
::: card "`mod` keeps you sane" icon:folder-tree
Every file names its module in full, so a name always says where it came from. [Modules →](/specification/declarations#mod)
:::
:::

:::

## Existing features, put to work

Rather than adding a feature, Zirric tends to reach for one that is already there. Take `for` loops: used as an expression, a loop collects what its body produces, and `continue` drops a value. That is `map` and `filter` — literally the same loop you would write for its side effects.

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

The same goes for the rest. `for element <- value` is a call into the [`Iterable`](/stdlib/prelude#iterable) attribute, so anything carrying it can be looped over. `len` is a call into [`Countable`](/stdlib/prelude#countable). The [Cavefile](/cavefile) is Zirric source, not a config format — a `data` declaration with attributes on it, like any other.

::: grids
::: grid
::: card "Errors are values" icon:shield
A failure returns a [`Result`](/stdlib/prelude#result), and `!.`, `!!` and `T!` work through one without an unwrapping step. `panic` is for bugs, and nothing catches it. [Guarded access →](/specification/expressions#guarded-member-access)
:::
:::

::: grid
::: card "Hints are optional, and binding" icon:scissors
Write a type hint and analysis holds you to it. Leave it off and the value flows as it always did. [Type hints →](/specification/typesystem#type-hints)
:::
:::

::: grid
::: card "Built for testing" icon:flask-conical
The filesystem, the clock and randomness are values you pass, not globals you reach for. A test hands over a fake; nothing else changes. [Standard library →](/stdlib)
:::
:::

::: grid
::: card "One binary" icon:rocket
`zirric` runs, formats, tests, serves LSP and manages packages. [The CLI →](/tooling/zirric-cli)
:::
:::

:::

## About the standard library

It is basic, but it tries to make the right choices easier than the ones that bite you. Everything is built with testing in mind, and you notice bad habits right at the import: [`os`](/stdlib/os) is the only module that talks to the machine, so a function that reaches for it directly cannot be tested without one. Take a [`fs.FileSystem`](/stdlib/fs), a [`clock.SystemClock`](/stdlib/clock) or a [`random.Source`](/stdlib/random) as a parameter instead, and a test can hand over `fs.memory()`, `clock.fixed(…)` or `random.seeded(…)`.

::: callout warning Experimental
Zirric is evolving quickly. Expect incomplete features and shifting syntax. The [Specification](/specification/syntax) describes the language as it is implemented; the [proposals](/proposals) record how it got there.
:::
