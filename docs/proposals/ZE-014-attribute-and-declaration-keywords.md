---
title: ZE-014 - Attribute and Declaration Keywords
description: Replace `func`, `annotation`, and `module` with `fn`, `attr`, and `mod`; and rename the "Annotation" concept to "Attribute" throughout the language and standard library.
---

# Attribute and Declaration Keywords

::: callout tip <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-check-icon lucide-circle-check"><circle cx="12" cy="12" r="10"/><path d="m9 12 2 2 4-4"/></svg> Implemented
This proposal has been accepted and implemented.
You can use this feature in the latest version of Zirric.
:::

## Introduction

This proposal makes two related changes:

1. **Declaration keyword replacements**: Replace three verbose declaration keywords — `func`, `annotation`, and `module` — with `fn`, `attr`, and `mod`. The old keywords are removed and no longer recognised by the parser. The remaining declaration keywords (`let`, `data`, `union`, `import`, `extern`) are unchanged.
2. **Concept rename**: Rename the "Annotation" concept to "Attribute" throughout the language and standard library. This affects all identifiers that use the word `Annotation` — including the `AnnotationType` runtime type, which becomes `AttributeType`, and the corresponding Go compiler/runtime internals.

## Motivation

`func`, `annotation`, and `module` are among the most frequently written declaration keywords in Zirric source code. `annotation` in particular is ten characters long and is repeated at the top of every attribute declaration. These verbose forms carry a readability cost that compounds over time.

The replacements `fn`, `attr`, and `mod` are already well-established in the wider language ecosystem:

- `fn` is the function keyword in Rust and Gleam.
- `attr` aligns with the common "attribute" terminology used in Python, .NET, and Rust's `#[...]` syntax.
- `mod` is the module keyword in Rust and is widely understood.

Adopting these forms makes Zirric code more concise without sacrificing clarity. Readers familiar with any of those languages will recognise the keywords immediately.

Renaming the concept from "Annotation" to "Attribute" aligns the terminology with `attr` and with the broader ecosystem convention. The `@Name` application syntax already reads naturally as applying an attribute, and naming the underlying type `AttributeType` removes the mismatch between the keyword and the concept.

The keywords left unchanged — `let`, `data`, `union`, `import`, `extern` — are already short or have no widely-adopted equivalent, so they are not replaced. `extern type` is likewise kept as-is; see _Alternatives Considered_ for the reasoning.

## Proposed Solution

The following table shows every affected keyword:

| Old keyword  | New keyword | Typical usage            |
| ------------ | ----------- | ------------------------ |
| `func`       | `fn`        | `fn greet(name) { ... }` |
| `annotation` | `attr`      | `attr Numeric {}`        |
| `module`     | `mod`       | `mod math { ... }`       |

Everything else — `let`, `data`, `union`, `import`, `extern` — remains as-is.

```zirric
// Before
module math

annotation Numeric {}

func add(a, b) {
    return a + b
}

func square(n) {
    return n * n
}

// After
mod math

attr Numeric {}

fn add(a, b) {
    return a + b
}

fn square(n) {
    return n * n
}
```

`extern` function declarations follow the same pattern:

```zirric
// Before
extern func print(value)

// After
extern fn print(value)
```

## Detailed Design

### Syntax

The following EBNF productions from [ZE-001](/proposals/ZE-001-base-language) are updated:

```ebnf
decl_func      = [attribute_chain], "fn",   identifier, "(", param_list, ")", block ;
decl_attribute = [attribute_chain], "attr", identifier, ["{", { field_decl }, "}"] ;
decl_module    = [attribute_chain], "mod",  identifier, "{", { decl }, "}" ;

decl_extern_func = "extern", "fn", identifier, "(", [ param_list ], ")" ;
```

All other productions in `decl` are unchanged.

### Breaking changes

The three old keywords become invalid. Any source file using `func`, `annotation`, or `module` will fail to parse. The migration is purely mechanical:

| Old form                  | New form               |
| ------------------------- | ---------------------- |
| `func name(...) { ... }`  | `fn name(...) { ... }` |
| `extern func name(...)`   | `extern fn name(...)`  |
| `annotation Name { ... }` | `attr Name { ... }`    |
| `module name { ... }`     | `mod name { ... }`     |

## Changes to the Standard Library

All `.zirr` files in `prelude/` and `future/` must be updated:

- Every `func` declaration is renamed to `fn`.
- Every `annotation` declaration is renamed to `attr`.
- Every `module` declaration is renamed to `mod`.

### Attribute concept rename

As part of the broader "Annotation → Attribute" rename, the following identifiers in `prelude/` and `future/` are also updated:

| Old name                | New name               | Location                   | Kind                                                |
| ----------------------- | ---------------------- | -------------------------- | --------------------------------------------------- |
| `AnnotationType`        | `AttributeType`        | `prelude/shim.zirr`        | `extern type` declaration                           |
| `AnnotationType`        | `AttributeType`        | `prelude/attributes.zirr` | type reference (field type and attribute argument) |
| `annotationType`        | `attributeType`        | `prelude/attributes.zirr` | field name in `Has`                                 |
| `Annotation`            | `Attribute`            | `future/reflect/stub.zirr` | `data` declaration                                  |
| `annotations`           | `attributes`           | `future/reflect/stub.zirr` | field name in `Field`                               |
| `@ItemType(Annotation)` | `@ItemType(Attribute)` | `future/reflect/stub.zirr` | annotation argument                                 |
| `@AnyAnnotation`        | `@AnyAttribute`        | `future/reflect/stub.zirr` | annotation application                              |
| `annotation`            | `attribute`            | `future/reflect/stub.zirr` | function name                                       |
| `hasAnnotation`         | `hasAttribute`         | `future/reflect/stub.zirr` | function name                                       |
| `annotationType`        | `attributeType`        | `future/reflect/stub.zirr` | parameter name                                      |
| `@AnnotationType`       | `@AttributeType`       | `future/reflect/stub.zirr` | annotation application                              |

The corresponding Go compiler and runtime internals are renamed in the same way — `AnnotationType` → `AttributeType`, `AnnotationValue` → `AttributeValue`, `DeclAnnotation` → `DeclAttribute`, `AnnotationChain` → `AttributeChain`, and so on.

## Alternatives Considered

### Replace only `func` → `fn`, leave others unchanged

`fn` alone would already reduce the most common source of verbosity. However, leaving `annotation` at ten characters would be a missed opportunity — attribute declarations appear at the top of every `attr` definition, so the reduction is meaningful. `mod` follows naturally from the same principle.

### Use `fun` instead of `fn`

`fun` (used by Kotlin) is another common alternative for functions. It reads more naturally in prose but saves only one character compared to `func`. `fn` is shorter and is the form used by the Rust ecosystem, which has broader reach.

### Rename `annotation` to `meta`

`meta` is shorter than `attr` but is less precise. It describes the general concept of metadata without specifying that the declaration creates a named type that can be applied with `@` syntax. `attr` (attribute) aligns with the established usage in Python and Rust and makes the connection to the `@Name` application syntax more obvious.

### Also rename `extern`

`extern` is used very sparingly in application code — only language extension authors write it. Brevity is less valuable for rare keywords, and `extern` is already unambiguous at six characters. It is left unchanged.

### Rename `extern type` to `extern data`

`extern type` declares a native-backed type whose internal representation is opaque to Zirric code. Renaming it to `extern data` was considered to mirror the `data` keyword, but this would be misleading: `data` carries a strong implication of a fully Zirric-defined, constructible record type. An `extern type` has its implementation outside of Zirric entirely — constructibility is a separate concern determined by the declaration itself, not by the keyword. `extern type` is the more honest and neutral term, and it remains unchanged.

## Acknowledgements

- `fn` is the function keyword in Rust and Gleam.
- `mod` is the module keyword in Rust.
- `attr` draws from the attribute terminology used in Python (`__attribute__`), Rust (`#[attr]`), and .NET (`[Attribute]`).
