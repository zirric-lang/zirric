---
title: Getting Started with Zirric
description: A step-by-step guide to getting started with the Zirric programming language.
---

# Getting Started

Zirric is a declaration-driven language with an expression-first feel. It aims to stay small while keeping enough structure to build real programs. Values are dynamic, but conversions are explicit, and behavior is described through attributes rather than interfaces.

::: callout warning Experimental
Zirric is still evolving. Some features are specified but not fully implemented yet. Use the proposals for authoritative intent. The standard library now covers I/O and the common data types — `io`, `fmt`, `os`, `strings`, `arrays`, `dicts`, `math`, `json` and more — but [ZE-020 Standard Library](/proposals/ZE-020-standard-library) is still in progress, so expect gaps and changes.
:::

## Your first program

After [installing Zirric](/guides/installation), put this in `main.zirr`:

```zirric
import scripts { println }

println("Hello, Zirric!")
```

```bash
$ zirric run main.zirr
Hello, Zirric!
```

No project or `Cavefile` is needed for a single file. See the [Zirric CLI](/tooling/zirric-cli) for the rest of the commands.

## What Zirric emphasizes

- Data and union types for structured modeling
- Attributes as the primary capability mechanism
- Closed declarations for least surprise
- Modules as the unit of organization and import
- First-class functions with concise syntax
- Expression-first control flow (`if`, `for`)

## The Zirric mindset

Your development starts with a new module. You simply create a new folder and place your files there. Then you begin defining the shape of your data. Declare every `data` type you need and group them into `union`s. Try to make your data match your mental model of the relationship between the `data` types and the `union`s.

If the `union` itself is the important thing and your `data` types are just an implementation detail, nest them. If the `union` is just supportive, make the `data` top level. This communicates how important these structures are.

For example, let's model a simple binary tree, that highlights the relationship between `data` and `union`. The `union` is the important thing here, so we nest the `data` declarations inside it.

```zirric
union BinaryTree {
	data Branch {
		left
		right
	}

	data Leaf { value }
}
```

Now that we modeled our domain, we can start writing functions that operate on our `BinaryTree`. For example, a function to calculate the depth of the tree:

```zirric
import math

fn depth(tree: BinaryTree) -> Int {
	return switch tree {
	case is Branch:
		1 + math.max(depth(tree.left), depth(tree.right))
	case is Leaf:
		1
	}
}
```

Once we want to integrate our `BinaryTree` with other parts of our codebase, we can add attributes that describe its capabilities. For example, we could add a `Countable` attribute that allows us to count the number of leaves in the tree:

```zirric
// this could be defined in another module
attr Counted {
	count(value: @Counted) -> Int
}

fn count(val: @Counted) -> Int {
	return Counted(val).count(val)
}

// in your module

union BinaryTree {
	@Counted(fn(tree) {
		return count(tree.left) + count(tree.right)
	})
	data Branch {
		left
		right
	}

	@Counted(fn(tree) {
		return 1
	})
	data Leaf { value }
}
```

Each member carries its own implementation. A value of the union finds the one belonging to its concrete type, so `count` works on a whole tree without the union itself having to know how counting is done.

In the same way we could also add attributes for JSON parsing. Then the JSON parsing library would lookup your attributes like `@Key` or `@Default` to figure out how to parse your data.

## Declarations at a glance

Zirric code is built from a small set of declarations:

```zirric
mod myapp

import strings

attr Returns {
	type
}

const answer = 42

fn greet(name) {
	return "Hello, " + name
}

data Person {
	name
	age
}

union Result {
	data Ok { value }
	data Err { message }
}
```

## Values and literals

Zirric supports basic literals you should be familiar with:

```zirric
42 // Int
3.14 // Float
true // Bool
"Hello" // String
[1, 2, 3] // Array
["key": "value"] // Dict
fn(a, b) { return a + b } // Function literal
```

## Variables and functions

Declare constants with `const`, variables with `var` and functions with `fn`.

```zirric
const answer = 42

fn greet(name) {
	return "Hello, " + name
}

const message = greet("Zirric")
```

Functions are values and can be passed around like any other expression:

```zirric
fn applyTwice(f, value) {
	return f(f(value))
}
```

## Data and unions

Zirric models records with `data` and tagged unions with `union`. Union members can be nested `data` declarations for structured variants.

```zirric
data Person {
	name
	age
}

union Result {
	data Ok { value }
	data Err { message }
}

const person = Person("Avery", 30)
const ok = Ok("Done")
```

## Control flow

`if` and `for` come in expression and statement forms. Expression forms return values; statement forms are for side effects.

```zirric
import fmt
import scripts { println }

const status = if answer == 42 {
	"yes"
} else {
	"no"
}

if answer == 42 {
	println("yes")
} else {
	println("no")
}

for item <- [1, 2, 3] {
	println(fmt.sprint(item))
}

const oddNumbers = for item <- [1, 2, 3] {
	if item % 2 != 0 {
		item
	} else {
		continue // skip to next iteration
	}
}

// oddNumbers is [1, 3]
```

## Attributes and capabilities

Zirric does not use interfaces. Instead, attributes describe capabilities and attach metadata to declarations. They are a core part of the language and tooling story.

```zirric
attr Countable {
	length(value: @Countable) -> Int
}

@Countable(fn(v) { return v.length })
data Bag {
	items
	length
}
```

Attributes are central to tooling, defaults, and protocol-like behavior.

## Modules and imports

Zirric code is organized into modules. Every file declares the module it belongs to with `mod`, by its fully qualified path, and uses `import` to reach other modules.

```zirric
mod myapp.http

import fmt

fn statusLine(code) {
	return "HTTP " + fmt.sprint(code)
}
```

`scripts.println` writes to stdout and `fmt.sprint` turns any value into a string; `strings`, `math`, `arrays` and the rest are imported the same way.

## What Zirric avoids

- Interfaces or inheritance as a primary abstraction.
- Implicit conversions between types.
- Generics. Types should be easy to reason about.
- Scattering members and capabilities across multiple declarations.

Zirric favors explicit declarations and attributes instead.

## Learn more

- Explore the [Specification](/specification/syntax) for precise grammar and semantics.
- Read the [Zirric Evolution Proposals](/proposals) for future design notes.
- Follow the [Styleguide](/guides/styleguide) to keep code consistent.
