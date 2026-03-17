---
title: ZE-016 - Unified Function and Closure Syntax
description: Unify the syntax of closures and named functions by using the fn keyword for both.
---

# Unified Function and Closure Syntax

::: callout draft <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-icon lucide-circle"><circle cx="12" cy="12" r="10"/></svg> Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design.
Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

This proposal replaces the brace-arrow closure syntax `{ args -> body }` with a `fn`-based syntax `fn(args) { body }`, making closures and named functions share the same structural shape.
All other shorthand forms for function literals are discontinued. This mostly affects function declarations.

## Motivation

The current closure syntax `{ a, b -> a + b }` differs significantly from named function declarations `fn add(a, b) { ... }`.
This creates two distinct mental models for the same concept and introduces ambiguity with `Dict` literals (both start with `{`).

The new unified syntax:

- **Reduces cognitive load**: one syntactic pattern for all functions.
- **Removes parser ambiguity**: `{` always introduces a block, never a function
  literal.
- **Improves readability**: closures are immediately recognisable by the `fn`
  keyword.
- **Simplifies the grammar**: a single production covers both named and anonymous
  functions.
- **Feels more consistent**: Zirric prefers keywords over punctuation and a minimal syntax set.

## Proposed Solution

### Named functions (unchanged shape)

```zirric
fn add(a, b) {
    return a + b
}
```

### Named functions in literal syntax (removed)

```zirric
fn add { a, b ->
    return a + b
}
```

### Anonymous functions (closures) — new syntax

```zirric
const add = fn(a, b) {
    a + b
}

constconstconstconstconstconstt = fn(name) {
    "Hello, " + name
}
```

### Zero-parameter closures

```zirric
constconstconstconstconstconstk = fn() {
    42
}
```

### Discontinued syntax

The following forms are **removed**:

| Old form                     | Replacement          |
| ---------------------------- | -------------------- |
| `{ a, b -> a + b }`          | `fn(a, b) { a + b }` |
| `{ -> expr }`                | `fn() { expr }`      |
| `{ expr }` (implicit no-arg) | `fn() { expr }`      |

### Summary of the unified grammar

```ebnf
decl_fn      = "fn", identifier, "(", [ parameter_list ], ")", block ;
fn_literal   = "fn", "(", [ parameter_list ], ")", block ;

parameter_list = parameter, { ",", parameter } ;
parameter      = [ attr_chain ], identifier ;
block          = "{", { statement }, "}" ;
```

Both productions share the `"fn", "(", params, ")", block` core; the only difference is the presence of a name.

## Detailed Design

### Parser changes

1. Remove the `fn_literal` rule that starts with `{` and uses `->`.
2. When the parser encounters `fn` followed by `(`, it produces an anonymous function node.
   When followed by an identifier and then `(`, it produces a named function declaration as today.
3. `{` in expression position no longer needs to disambiguate between closures and dict literals.
   It is always a dict (or a block in statement position).

### AST impact

The existing AST node for function literals already stores a parameter list and a body.
The only structural change is that the source representation changes while the AST shape remains the same.

Named functions continue to carry an additional name field.

### Interaction with ZE-012 (Type and Returns Sugar)

[ZE-012 — Type and Returns Sugar](/proposals/ZE-012-type-and-returns-sugar) introduces `: Type` parameter attributes and `-> ReturnType` return attributes as sugar for `@Type` and `@Returns`.
In case both proposals are accepted, the full signature syntax becomes:

```zirric
// Named function with type sugar
fn greet(name: String) -> String {
    "Hello, " + name
}

// Anonymous function (closure) with type sugar
const greet = fn(name: String) -> String {
    "Hello, " + name
}
```

The desugaring rules of ZE-012 apply identically to both named and anonymous functions and no special-casing is needed, because they now share the same syntactic structure.

### Migration

Existing code using the old closure syntax must be rewritten:

```zirric
// Before
const add = { a, b -> a + b }
items.map({ x -> x + 1 })

// After
const add = fn(a, b) { a + b }
items.map(fn(x) { x + 1 })
```

## Changes to the Standard Library

The standard library (`prelude/`, `future/`) will need its closure usages updated to the new syntax. No new types, modules or functions are introduced.

## Alternatives Considered

- **Keep both syntaxes**: Allow `{ a -> ... }` alongside `fn(a) { ... }`.
  Rejected because it preserves the ambiguity and cognitive overhead this proposal aims to eliminate.
- **Status quo**: The brace-arrow form is already implemented but conflicts with the dict literal syntax and diverges from named function declarations.

## Acknowledgements

- The previous syntax for closures was inspired by Swift's `{ arg in }` and JavaScript's `(arg) => {}` closure syntaxes.
- Now it's inspired by Go (`func(a int) { ... }`).
