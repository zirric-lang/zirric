---
title: Getting Started with Zirric
description: A step-by-step guide to getting started with the Zirric programming language.
---

# Getting Started

Zirric is a declaration-driven language with an expression-first feel. It aims to stay small while keeping enough structure to build real programs. Values are dynamic, but conversions are explicit, and behavior is described through attributes rather than interfaces.

::: callout warning Experimental
Zirric is still evolving. The [Specification](/specification/syntax) is the authoritative account of the language as it is implemented; the [proposals](/proposals) are a record of how it got there and are not kept in step with it. The standard library covers I/O and the common data types — `io`, `fmt`, `os`, `strings`, `arrays`, `dicts`, `math`, `json` and more — but names and signatures can still change, so expect gaps and changes.
:::

## Your first program

After [installing Zirric](/guides/installation), put this in `main.zirr`:

```zirric
import fmt
import os

fmt.fprintln("Hello, Zirric!", os.stdout())
```

```bash
$ zirric run main.zirr
Hello, Zirric!
```

No project or `Cavefile` is needed for a single file: a loose script may leave out `mod` entirely. See the [Zirric CLI](/tooling/zirric-cli) for the rest of the commands.

## A whole project

One file is enough to try something out. Anything larger is a **package**: a directory with a `Cavefile` at its root. Each directory in it is one module, holding whatever `.zirr` files sit directly inside. Here is the smallest complete package, with a module of its own and a test.

```
greeter/
├── Cavefile
├── main.zirr
└── greeting/
    ├── greeting.zirr
    └── _t/
        └── greeting_t.zirr
```

### The Cavefile

`zirric cave new` writes it for you. The package needs a module path to be named after — pass `--mod`, or `--git-url` to derive it from the repository:

```bash
$ mkdir greeter && cd greeter
$ zirric cave new --mod example.greeter --description "A greeter, as a worked example."
created Cavefile
```

In a directory that already has a Git remote, `zirric cave new` reads the remote and needs neither flag.

The `Cavefile` it writes is Zirric, not a separate config language:

```zirric
mod example.greeter

import cave

@cave.Package()
@cave.LanguageVersion(">=0.1.0")
@cave.Description("A greeter, as a worked example.")
data Greeter {
}
```

Its `mod` declaration is the package's **base module path**, and every source file's `mod` starts with it. The `data` carrying `@cave.Package()` is the manifest itself: its attributes describe the package, and its fields — none yet — are the dependencies. Tasks are `data` declarations of their own. See [the Cavefile](/cavefile) for both, and `zirric cave new --help` for the rest of the flags.

### A module in a subdirectory

A file's module path is the package base joined with the path from the package root to the file's directory. `greeting/greeting.zirr` therefore declares `example.greeter.greeting`:

```zirric
mod example.greeter.greeting

fn greet(name: String) -> String {
	return "Hello, " + name + "!"
}
```

Every file in one directory declares the same path. A directory name becomes a path segment lowercased, with `-` replaced by `_`.

### Importing your own modules

A module of your own package is imported by its **path relative to the package root** — for a directory at the top of the package, that is just the directory name. There is nothing special about being local; the form is the same one the standard library uses.

```zirric
mod example.greeter

import fmt
import os
import greeting // example.greeter.greeting

fmt.fprintln(greeting.greet("Zirric"), os.stdout())
```

A module further down is reached by joining the segments: a `report/html/` directory is `import report.html`, and the alias is the last segment (`html`). The fully qualified path always works too, and either form can be renamed or narrowed:

```zirric
import html = example.greeter.report.html // any path, under a name you choose
import greeting { greet }                 // greet(…) directly, no qualifier
```

```bash
$ zirric run .
Hello, Zirric!
```

`zirric run .` runs the module at that directory: every top-level statement of every file in it, in file-name order. `zirric run main.zirr` runs that one file instead.

### A test module

`zirric test` runs every module of the project whose name ends in `_t`. That is the whole convention — nothing is registered. A `_t` subdirectory beside the code it tests is the usual shape; a sibling module named `foo_t` works as well.

```zirric
mod example.greeter.greeting._t

import tests { Test }
import tests.assert
import greeting

@Test()
fn testGreetsByName() {
	assert.equal("Hello, Zirric!", greeting.greet("Zirric"))
}
```

A test is a function carrying `@Test()` that takes no arguments and returns a `Result` — which the assertions already are, so returning one is the whole body.

```bash
$ zirric test
TAP version 14
1..1
ok 1 example.greeter.greeting._t.testGreetsByName

# PASS	FAIL	SKIP	TODO
# 1	0	0	0

# PASSED!
```

Every test runs, and the command exits non-zero when any of them failed, so it drops into CI unchanged. See [`tests`](/stdlib/tests) for `@Skip`, `@Todo`, `@Only` and `@Comment`, and [`tests.assert`](/stdlib/tests/assert) for the rest of the assertions.

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
	case _:
		0
	}
}
```

A `switch` used as an expression always needs a `case _:`, even where the `is` cases look like they cover every member — nothing checks them for it, and an expression whose cases all fail has no value to produce. A `switch` written as a statement does not need one.

Once we want to integrate our `BinaryTree` with other parts of our codebase, we can add attributes that describe its capabilities. For example, we could add a `Counted` attribute that allows us to count the number of leaves in the tree:

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
mod example.myapp // the package's own base path, plus this file's directory

import strings

attr Table {
	name: String
}

const answer = 42

fn greet(name) {
	return "Hello, " + name
}

@Table("people")
data Person {
	name
	age
}

union Shape {
	data Circle { radius }
	data Rect { width, height }
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

union Shape {
	data Circle { radius }
	data Rect { width, height }
}

const person = Person("Avery", 30)
const circle = Circle(2)
```

`prelude` ships two such unions, `Option` (`Some`/`None`) and `Result` (`Ok`/`Err`), along with operators for working through them. `?.` reads a field off what an option holds, unless it is absent, in which case the whole chain is `None`; `!.` does the same for a result, and returns the error from the enclosing function instead. `??` and `!!` then stand in for a value that is not there, and `T?` and `T!` name the two types in a signature. None of them needs an unwrapping step written out.

```zirric
fn nameOf(person: Person?) -> String {
	return person?.name ?? "Anonymous"
}

fn describe(found: Person!) -> String! {
	const name = found!.name
	return Ok("This is " + name)
}
```

A union of your own joins in by carrying `@AnyOption` or `@AnyResult`, whose callback says how to read it as the standard one.

Give your own types names the prelude does not already use. Declaring a `Result`, `Ok`, `Err`, `Option`, `Some` or `None` of your own shadows the prelude's for the rest of the module — with no warning — and these operators stop working, since they are defined in terms of the prelude's.

See [Expressions § Guarded Member Access](/specification/expressions#guarded-member-access) and [§ Fallback](/specification/expressions#fallback).

## Control flow

`if` and `for` come in expression and statement forms. Expression forms return values; statement forms are for side effects.

```zirric
import fmt
import os

const out = os.stdout()

const status = if answer == 42 {
	"yes"
} else {
	"no"
}

if answer == 42 {
	fmt.fprintln("yes", out)
} else {
	fmt.fprintln("no", out)
}

for item <- [1, 2, 3] {
	fmt.fprintln(item, out)
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
attr Sized {
	size(value: @Sized) -> Int
}

@Sized(fn(bag) { return len(bag.items) })
data Bag {
	items
}

fn sizeOf(value: @Sized) -> Int {
	return Sized(value).size(value)
}
```

Applying `@Sized` supplies the implementation; calling the attribute type on a value reads it back. A hint of `@Sized` then asks for any value whose type carries it, and analysis reports one that does not.

The prelude already defines the two the language itself uses — `@Countable`, behind `len`, and `@Iterable`, behind `for element <- value` — so a type opts into both by applying them rather than by declaring its own. Attributes are central to tooling, defaults, and protocol-like behavior.

## Modules and imports

Zirric code is organized into modules. Every file declares the module it belongs to with `mod`, by its fully qualified path, and uses `import` to reach other modules.

```zirric
mod example.myapp.http

import fmt

fn statusLine(code) {
	return "HTTP " + fmt.sprint(code)
}
```

`fmt.fprintln` writes to a writer such as `os.stdout()` and `fmt.sprint` turns any value into a string; `strings`, `math`, `arrays` and the rest are imported the same way. The [`scripts`](https://code.knabel.dev/zirric-lang/scripts) package, which lives outside the standard library, pairs those functions with standard output for you.

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
