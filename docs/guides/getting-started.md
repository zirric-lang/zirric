---
title: Getting Started with Zirric
description: A step-by-step guide to getting started with the Zirric programming language.
---

# Getting Started

Zirric is a declaration-driven language with an expression-first feel. It aims
to stay small while keeping enough structure to build real programs. Values are
dynamic, but conversions are explicit, and behavior is described through
annotations rather than interfaces.

::: callout warning Experimental
Zirric is still evolving. Some features are specified but not fully implemented
yet. Use the proposals for authoritative intent.
:::

## What Zirric emphasizes

- Data and enum types for structured modeling
- First-class functions with concise syntax
- Modules as the unit of organization and import
- Annotations as the primary capability mechanism
- Expression-first control flow (`if`, `for`)

## Declarations at a glance

Zirric code is built from a small set of declarations:

```zirric
module app

import strings

annotation Returns {
    type
}

let answer = 42

func greet(name) {
    return "Hello, " + name
}

data Person {
    name
    age
}

enum Result {
    data Ok { value }
    data Err { message }
}
```

## Values and literals

Zirric supports basic literals you should be familiar with:

```zirric
42                 // Int
3.14               // Float
true               // Bool
"Hello"            // String
[1, 2, 3]          // Array
{ "key": "value" } // Dict
{ a, b -> a + b }  // Function literal
```

## Variables and functions

Declare variables with `let` and functions with `func`. Functions can return
any value.

```zirric
let answer = 42

func greet(name) {
    return "Hello, " + name
}

let message = greet("Zirric")
```

Functions are values and can be passed around like any other expression:

```zirric
func applyTwice(f, value) {
    return f(f(value))
}
```

## Data and enums

Zirric models records with `data` and tagged unions with `enum`. Enum cases can
be nested `data` declarations for structured variants.

```zirric
data Person {
    name
    age
}

enum Result {
    data Ok { value }
    data Err { message }
}

let person = Person("Avery", 30)
let ok = Ok("Done")
```

## Control flow

`if` and `for` come in expression and statement forms. Expression forms return
values; statement forms are for side effects.

```zirric
let status = if answer == 42 {
    "yes"
} else {
    "no"
}

if answer == 42 {
    print("yes")
} else {
    print("no")
}

for item <- [1, 2, 3] {
    print(item)
}

let oddNumbers = for item <- [1, 2, 3] {
    if item % 2 != 0 {
        item
    } else {
        continue // skip to next iteration
    }
}

// oddNumbers is [1, 3]
```

## Annotations and capabilities

Zirric does not use interfaces. Instead, annotations describe capabilities and
attach metadata to declarations. They are a core part of the language and
tooling story.

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
```

Annotations are central to tooling, defaults, and protocol-like behavior.

## Modules and imports

Zirric code is organized into modules. Use `module` to declare the namespace
and `import` to access other modules.

```zirric
module http
import strings

func statusLine(code) {
    return "HTTP " + strings.fromInt(code)
}
```

## What Zirric avoids

- Interfaces or inheritance as a primary abstraction
- Implicit conversions between types

Zirric favors explicit declarations and annotations instead.

## Learn more

- Explore the [Syntax references](/syntax/expressions) for precise grammar.
- Read the [Zirric Evolution Proposals](/proposals) for future design notes.
- Follow the [Styleguide](/guides/styleguide) to keep code consistent.
