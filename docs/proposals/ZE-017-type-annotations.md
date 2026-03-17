---
title: ZE-017 - Type Annotations and Type Matching
description: Replace attribute-based type annotations with first-class type syntax, introduce `is` type matching, and add built-in collection type expressions.
---

# Type Annotations and Type Matching

::: callout warning <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-dot-icon lucide-circle-dot"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="3" fill="currentColor" stroke="none"/></svg> In Progress
This proposal has been accepted in principle.
It is currently under active development.
Parts might be incomplete or missing in Zirric.
:::

## Introduction

This proposal replaces the attribute-based type annotation system (`@Type`, `@Returns`, `@Has`, `@ItemType`, `@OkType`, `@ErrType`, `@SomeType`) with first-class, optional type annotation syntax. It also introduces the `is` keyword for type matching in `switch` cases and as a standalone expression, and adds built-in collection type expressions for `Array` and `Dict`. Zirric remains dynamically typed — all type annotations are optional and can be omitted.

This supersedes [ZE-012 Type and Returns Sugar](/proposals/ZE-012-type-and-returns-sugar), which treated `: Type` and `-> Type` as syntactic sugar that desugared to `@Type` and `@Returns`. Under this proposal, type annotations are a core language feature, and the underlying attributes are removed entirely.

## Motivation

Zirric's current type system expresses types through the attribute system. While this demonstrated the flexibility of attributes, it creates several problems:

1. **Conceptual confusion**: Type annotations are not metadata — they are fundamental to a value's contract. Expressing them as `@Type(String)` or `@Returns(Int)` conflates two distinct concerns.
2. **Verbose syntax**: `@Returns(String) fn greet(@String name)` is noisy compared to `fn greet(name: String) -> String`.
3. **Ambiguous `@` prefix**: `@String` currently means `@Type(String)`, making `@` serve double duty as both "attribute" and "type annotation." Removing this overload lets `@` exclusively mean "attribute/metadata."
4. **Switch case confusion**: `case @String:` looks like an attribute check but actually performs type matching. This hides the distinction between type matching and attribute matching.

## Proposed Solution

### Type Annotations

Type annotations use `: Type` for values and `-> Type` for return types. These are now first-class syntax, not sugar for attributes. **All type annotations are optional.** Zirric remains a dynamically typed language — type annotations serve as documentation, tooling hints, and optional runtime checks, but omitting them is perfectly valid.

```zirric
// With type annotations
const name: String = "Zirric"
fn greet(name: String) -> String {
  "Hello, " + name
}

// Without type annotations — equally valid
const name = "Zirric"
fn greet(name) {
  "Hello, " + name
}

// Data fields — types optional
data Greeter {
  greet(name: String) -> String
  greeting: String
}

data Greeter {
  greet(name)
  greeting
}

// Attribute fields
attr Formatter {
  format(value: String) -> String
}
```

### Attribute References in Type Position

Attributes can be referenced in type positions to express capability constraints. When used in parameters or fields, attributes are written without parentheses:

```zirric
// Parameter must be of a type that has @Numeric
fn double(value: @Numeric) -> Number {
  value * 2
}

// Field constrained by attribute
data Calculator {
  compute(input: @Numeric) -> Number
}
```

This replaces the old `@Has(Numeric)` pattern. When `@Attr` appears in a type position, it means "any value whose type carries the `@Attr` attribute."

### Collection Type Expressions

Two built-in type expressions are provided for `Array` and `Dict`, mirroring their special status as language built-ins:

```zirric
// Array type: [ElementType]
const names: [String] = Array("Alice", "Bob")
fn first(items: [String]) -> String { items.0 }

// Dict type: {KeyType: ValueType}
const ages: {String: Int} = Dict()
```

These are not generics — they are fixed syntax for exactly two built-in types that the language already treats specially. No user-defined types can use this syntax.

### `is` in Switch Cases

The `is` keyword replaces the old `case @TypeName:` syntax for type matching in switch expressions:

```zirric
switch value {
  case is String:
    // value is a String
  case is Int:
    // value is an Int
  case @Numeric:
    // value's type has @Numeric attribute
  case 42:
    // value equals 42
  case _:
    // default
}
```

This makes switch cases unambiguous:

- `case is Type:` — type match (is the value of this type?)
- `case @Attr:` — attribute match (does the value's type have this attribute?)
- `case literal:` — equality match
- `case _:` — default

### `is` as Expression

The `is` keyword also works as a binary expression returning `Bool`:

```zirric
if name is String {
  // name is a String
}

const isNumeric = value is @Numeric
```

## Detailed Design

### Dropped Attributes

The following attributes are removed from the language and standard library:

| Attribute | Replacement |
|---|---|
| `@Type(T)` | `: T` type annotation |
| `@Returns(T)` | `-> T` return type |
| `@Has(Attr)` | `: @Attr` in type position |
| `@ItemType(T)` | `[T]` collection type expression |
| `@OkType(T)` | Use concrete union types instead |
| `@ErrType(T)` | Use concrete union types instead |
| `@SomeType(T)` | Use concrete union types instead |

### Implicit Type Conversion Removed

The implicit conversion of non-attribute types to `@Type(Type)` is removed. Writing `@String` is no longer valid. Use `: String` instead for type annotations, and `case is String:` for switch cases.

### Attribute Matching in Switch

With `@Type` removed, `case @Attr:` in switch cases now exclusively checks whether the value's runtime type carries the `@Attr` attribute. This is a simplification: `@` in case patterns always means "attribute check."

```zirric
// Old: case @Has(Numeric): — explicit @Has wrapper
// New: case @Numeric:      — direct attribute check
switch value {
  case @Numeric:
    // value's type has @Numeric attribute
  case @Iterable:
    // value's type has @Iterable attribute
}
```

### Attribute References Without Parentheses

When attributes appear in type positions (parameters, fields) or switch cases, they are written without parentheses and without arguments. This indicates a constraint — "the value must be of a type that has this attribute" — rather than constructing an attribute instance. Attribute instances with arguments are only created on `data`, `union`, `fn`, `mod`, and other declarations.

```zirric
// Type position: constraint (no parens)
fn sum(a: @Numeric, b: @Numeric) -> Number { ... }

// Declaration position: instance (with parens if needed)
@Deprecated("use newSum instead")
fn sum(a: @Numeric, b: @Numeric) -> Number { ... }
```

### Result and Option Types

Without `@OkType`, `@ErrType`, and `@SomeType`, there is no built-in way to express "a Result containing a String." For use cases requiring this level of type specificity, define concrete union types:

```zirric
data ApiError { message: String }
union ApiResult { ApiResponse ApiError }

fn fetchUser(id: Int) -> ApiResult { ... }
```

The `Result` and `Option` unions from the standard library remain available for generic use, but their contained types are not expressed in the type system.

### Grammar Changes

```ebnf
(* Type expressions *)
type_expr = type_name
          | "@", identifier              (* attribute constraint *)
          | "[", type_expr, "]"          (* array type *)
          | "{", type_expr, ":", type_expr, "}" (* dict type *)
          ;

type_name = identifier, { ".", identifier } ;

(* Type annotation *)
type_annotation = ":", type_expr ;

(* Return type *)
return_type = "->", type_expr ;

(* is expression *)
is_expr = expression, "is", type_expr ;

(* switch case patterns *)
case_pattern = "is", type_expr          (* type match *)
             | "@", identifier          (* attribute match *)
             | expression               (* equality match *)
             | "_"                      (* default *)
             ;
```

## Changes to the Standard Library

### Removed from `prelude/attributes.zirr`

```zirric
// REMOVED:
attr Type { ... }
attr Has { ... }
attr Returns { ... }
```

### Removed from other modules

- `@ItemType` — replaced by `[T]` syntax
- `@OkType`, `@ErrType`, `@SomeType` — removed without direct replacement

### Updated declarations

All standard library declarations using the old attribute-based type annotations will be updated to use the new syntax. For example:

```zirric
// Old:
@Numeric({ f -> f })
extern type Float {}

// New (attribute field syntax may need adjustment):
@Numeric
extern type Float {}
```

```zirric
// Old:
extern type Array { @Type(Int) length }

// New:
extern type Array { length: Int }
```

## Alternatives Considered

- **Keep `: Type` as sugar for `@Type`** (ZE-012 approach): This maintains backwards compatibility but preserves the conceptual confusion of types-as-attributes. Rejected because a clean separation is worth the breaking change.
- **Generics** (`Array[String]`, `Result[String, Error]`): Too powerful for Zirric's goals as a simple, dynamic language. The `[T]` and `{K: V}` syntax covers the two built-in collection types without introducing a generic type system.
- **Bare type names in switch** (`case String:`): Clean but ambiguous when a variable holds a type value. The `is` keyword eliminates this ambiguity.
- **Keep `@ItemType`/`@OkType`/`@SomeType` as metadata**: These would survive as documentation hints, but in practice users who need typed containers should define concrete union types. Dropped for simplicity.

## Acknowledgements

- Builds on [ZE-012 Type and Returns Sugar](/proposals/ZE-012-type-and-returns-sugar) which first introduced the `: Type` and `-> Type` syntax.
- The `is` keyword for type checking is inspired by Swift, Kotlin, and TypeScript.
- Collection type syntax `[T]` and `{K: V}` is inspired by Go's slice and map type expressions.
