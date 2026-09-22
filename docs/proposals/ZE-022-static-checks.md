---
title: "ZE-022 - Static Checks"
description: "Static Checks"
---

# Static Checks

::: callout tip Implemented
This proposal has been accepted and implemented. You can use this feature since Zirric [v0.1.0](/changelog/v0.1.0).
:::

## Introduction

Analysis reports mistakes it is certain about, instead of leaving them to fail at runtime.

## Motivation

A typo, a miscounted argument or an operator with no meaning for its operands already stops a program — just later, without a position, and one at a time. Type hints were worse off still: the language asked for them and then checked nothing.

## Proposed Solution

One rule governs everything: **report only certainties.** Anything the checker cannot see through fits, and only a combination it is sure about is rejected. In a language where types are optional, a false alarm costs more than a missed one.

Two kinds of thing are reported.

**Certain runtime failures** — the VM would stop:

| Reported                      | Example                     |
| ----------------------------- | --------------------------- |
| undefined name                | `nosuchthing()`             |
| unknown module member         | `scripts.printlnn(…)`       |
| unknown field                 | `person.nmae`               |
| wrong number of arguments     | `two(1)` for `fn two(a, b)` |
| calling a non-callable        | `const x = 5` then `x()`    |
| unsupported operator          | `P(1) + 3`, `-"a"`, `!5`    |
| indexing a non-indexable      | `5[0]`                      |
| iterating without `@Iterable` | `for x <- 5`                |

**Broken declarations** — the VM tolerates these, the program does not mean them:

| Reported             | Example                                |
| -------------------- | -------------------------------------- |
| argument type        | `fn(x: Int)` called with a `String`    |
| return type          | `fn f() -> Int { "s" }`                |
| declared value       | `const x: Int = "hello"`               |
| assignment           | `x = "hello"` where `x: Int`           |
| attribute constraint | an `Int` where `@Iterable` is declared |

## Detailed Design

### What is known

A check only fires when the checker knows a type. It knows:

| From                        | Type                                                                     |
| --------------------------- | ------------------------------------------------------------------------ |
| a literal                   | its own                                                                  |
| a type hint                 | what it names, following imports                                         |
| `const`                     | its hint, else its value                                                 |
| `var`                       | its hint, else its value and every assignment — when they all agree      |
| `if` / `switch` expressions | what every branch yields — when they agree, and a `switch` has a default |
| a collection literal        | its elements — when they agree                                           |
| `xs[0]`, `d[k]`             | the element or value type; a `String` or `Binary` gives a `Byte`         |
| a call                      | the callee's declared return type                                        |
| a field, a module member    | what it was declared as                                                  |

Everything else is unknown, and unknown fits everywhere.

### Narrowing

Inside `case is T:` the value switched on is a `T`, so a branch that can only run for one member of a union is checked as that member.

### Unions and attributes

A union fits when any member does, in either direction. An attribute constraint is satisfied when the type's declaration carries that attribute.

### Deliberately not reported

- A non-`Bool` `if` condition — the VM accepts it.
- Element types in a mixed collection, which describe nothing.
- Anything depending on the order statements run in.

## Changes to the Standard Library

None. Every module already passes.

## Compatibility

Catching a runtime failure at compile time is **not a breaking change**. From here on, a new check needs a release note, not a proposal — provided it reports only certainties.

This amends [ZE-017](/proposals/ZE-017-type-hints), which describes type hints as documentation that is never verified, including the element and key types of composite hints. Hints are still optional; written down, they are now binding.
