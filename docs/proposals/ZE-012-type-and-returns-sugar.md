---
title: ZE-012 - Type and Returns Sugar
description: Add typed parameter and return signatures as sugar for @Type and @Returns annotations.
---

# Type and Returns Sugar

::: callout warning <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-dot-icon lucide-circle-dot"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="3" fill="currentColor" stroke="none"/></svg> In Progress
This proposal has been accepted in principle.
It is currently under active development.
Parts might be incomplete or missing in Zirric.
:::

## Introduction

This proposal introduces a compact signature syntax that desugars to existing annotations:
`greet(name: String) -> String` becomes `@Returns(String) greet(@String name)`.
The goal is to reduce annotation noise while staying fully compatible with the
attr system.

## Motivation

Zirric uses `@Type` and `@Returns` to describe function signatures. While explicit
annotations are flexible, simple signatures are verbose to write and read,
especially in data and annotation declarations where signatures serve as
documentation. A concise syntax improves readability without changing runtime
semantics.

## Proposed Solution

Add a signature sugar that is allowed in:

- `let` declarations
- `fn` declarations
- function fields inside `data`
- function fields inside `attr`

### Examples

```zirric
// fn declaration
fn greet(name: String) -> String {
  "Hello, " + name
}
```

Desugars to:

```zirric
@Returns(String)
fn greet(@String name) {
  "Hello, " + name
}
```

```zirric
// let declaration
let name: String = "Zirric"
```

```zirric
// data fields
data Greeter {
  greet(name: String) -> String
  greeting: String
}
```

```zirric
// annotation fields
attr Formatter {
  format(value: String) -> String
}
```

### Desugaring rules

- `let var: T` desugars to `@Type(T) let var`
- `var: T` desugars to `@Type(T) var` for parameters and in fields.
- `-> T` desugars to `@Returns(T)` on the declaration or field.

## Detailed Design

### Scope and limitations

- The `:` and `->` syntax is sugar only; it has no runtime effect beyond the
  existing annotations it produces.
- If both the sugar and explicit annotations specify the same data (e.g.,
  `@Returns` and `->`), the compiler should report a duplicate/ambiguous annotation
  error to avoid hidden conflicts.
- The type reference after `:` or `->` uses the same rules as `@Type`, so
  `String`, `module.Type`, and other static references are valid.

## Changes to the Standard Library

None. The feature only uses existing `@Type` and `@Returns` annotations.

## Alternatives Considered

- **Status quo**: keep only explicit `@Type` and `@Returns`.
  - Simple implementation, but verbose and harder to read.

## Acknowledgements

- Inspired by type annotation syntax in languages like TypeScript and Swift.
