---
title: ZE-017 - Type Hints and Type Matching
description: Replace attribute-based type hints with first-class type syntax, introduce `is` type matching, and add built-in collection and function type expressions.
---

# Type Hints and Type Matching

::: callout tip Implemented
This proposal has been accepted and implemented. You can use this feature since Zirric [v0.1.0](/changelog/v0.1.0).
:::

## Introduction

This proposal replaces the attribute-based type hint system (`@Type`, `@Returns`, `@Has`, `@ItemType`, `@OkType`, `@ErrType`, `@SomeType`) with first-class, optional **type expression** syntax. It introduces a unified type expression grammar used for type hints, `is` type matching, and switch case patterns. Zirric remains dynamically typed — all type hints are optional and can be omitted.

This supersedes [ZE-012 Type and Returns Sugar](/proposals/ZE-012-type-and-returns-sugar), which treated `: Type` and `-> Type` as syntactic sugar that desugared to `@Type` and `@Returns`. Under this proposal, type hints are a core language feature, and the underlying attributes are removed entirely.

## Motivation

Zirric's current type system expresses types through the attribute system. While this demonstrated the flexibility of attributes, it creates several problems:

1. **Conceptual confusion**: Type hints are not metadata — they are fundamental to a value's contract. Expressing them as `@Type(String)` or `@Returns(Int)` conflates two distinct concerns.
2. **Verbose syntax**: `@Returns(String) fn greet(@String name)` is noisy compared to `fn greet(name: String) -> String`.
3. **Ambiguous `@` prefix**: `@String` currently means `@Type(String)`, making `@` serve double duty as both "attribute" and "type hint." Removing this overload lets `@` exclusively mean "attribute/metadata."
4. **Fragmented type expressions**: Type matching (`is`), type hints (`: T`), and collection types (`[T]`, `[K: V]`) were parsed by separate code paths with different grammars. Unifying them under a single type expression grammar makes the language consistent and composable.

## Proposed Solution

### Type Expressions

A **type expression** is a unified syntax for describing types. Type expressions appear in type hints (`: T`), return types (`-> T`), `is` expressions, and switch case patterns. Every position that accepts a type uses the same grammar.

Type expressions come in four forms:

1. **Named types** — a reference to a `data`, `extern type`, or `union` declaration, optionally module-qualified:
   ```zirric
   String
   prelude.String
   ```

2. **Composite types** — built-in collection and function type expressions:
   ```zirric
   [Item]              // Array with Item elements
   [Key: Value]        // Dict with Key keys and Value values
   fn(a: A, b: B) -> R // Function taking A, B and returning R
   ```

3. **Attribute constraints** — one or more `@Attr` references meaning "a value whose type carries these attributes":
   ```zirric
   @Numeric
   @Iterable @Countable
   @prelude.Iterable
   ```

4. **Nested** — composite types may contain any type expression:
   ```zirric
   [fn(arg: String) -> Int]       // Array of functions
   [String: [Int]]                // Dict from String to Array of Int
   fn(items: [String]) -> [String: Int]
   ```

### Type Hints

Type hints use `: TypeExpr` for values and `-> TypeExpr` for return types. **All type hints are optional.** Zirric remains a dynamically typed language, and omitting a hint is perfectly valid.

> Amended by [ZE-022](/proposals/ZE-022-static-checks): a hint that _is_ written is checked during analysis. It was originally documentation only.

```zirric
// With type hints
const name: String = "Zirric"
fn greet(name: String) -> String {
  "Hello, " + name
}

// Without type hints — equally valid
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

// Composite type hints
const names: [String] = Array("Alice", "Bob")
const ages: [String: Int] = Dict()
fn transform(f: fn(Int) -> String) -> [String] { ... }
```

### Attribute Constraints in Type Position

Attributes can be referenced in type positions to express capability constraints. When used in hints, attributes are written without parentheses:

```zirric
// Parameter must be of a type that has @Numeric
fn double(value: @Numeric) -> Number {
  value * 2
}

// Multiple attribute constraints
fn process(value: @Iterable @Countable) -> Int {
  value.length
}
```

This replaces the old `@Has(Numeric)` pattern. When `@Attr` appears in a type position, it means "any value whose type carries the `@Attr` attribute."

### `is` Expression

The `is` keyword is a binary expression returning `Bool`. It takes a type expression on the right-hand side:

```zirric
// Named type checks
name is String
value is prelude.Float

// Composite type checks (checks the container type)
items is [Int]        // true if items is an Array
lookup is [String: Int]  // true if lookup is a Dict
callback is fn(Int) -> String  // true if callback is a Function

// Attribute checks — all must be present
value is @Numeric
value is @Iterable @Countable
value is @prelude.Iterable
```

For composite types, `is` checks the **container type** at runtime, so `x is [Int]` and `x is [String]` both check that `x` is an `Array`.

> Amended by [ZE-022](/proposals/ZE-022-static-checks): element, key and value types are checked during analysis where they are known, even though `is` still compares only the container.

### `is` in Switch Cases

Switch cases use `case is <type_expr>:` for type matching. All type expressions are supported:

```zirric
switch value {
  case is String:
    // value is a String
  case is Int:
    // value is an Int
  case is @Numeric:
    // value's type has @Numeric attribute
  case is @Iterable @Countable:
    // value's type has both attributes
  case is [Int]:
    // value is an Array
  case is prelude.Float:
    // static reference to a type from another module
  case 42:
    // value equals 42
  case _:
    // default
}
```

This makes switch cases unambiguous:

- `case is <type_expr>:` — type/attribute match using a type expression
- `case <expr>:` — equality match
- `case _:` — default

Note that `case @Attr:` (without `is`) is **no longer valid**. All type matching in switch cases requires the `is` keyword. This eliminates the previous ambiguity where `@` had different meanings in case patterns vs. declarations.

## Detailed Design

### Dropped Attributes

The following attributes are removed from the language and standard library:

| Attribute      | Replacement                      |
| -------------- | -------------------------------- |
| `@Type(T)`     | `: T` type hint                  |
| `@Returns(T)`  | `-> T` return type               |
| `@Has(Attr)`   | `: @Attr` in type position       |
| `@ItemType(T)` | `[T]` collection type expression |
| `@OkType(T)`   | Use concrete union types instead |
| `@ErrType(T)`  | Use concrete union types instead |
| `@SomeType(T)` | Use concrete union types instead |

The removal of `@Type(T)` also means that all former "shorthand" uses like `@String`, `@Bool`, `@Int` are invalid — these are not attributes and must not appear as `@Name()` attribute instances. They are replaced by `: String`, `: Bool`, `: Int` type hints.

### Strict Attribute Instance Validation

The `@Name()` syntax is exclusively for attribute instances — only types declared with `attr` may appear after `@`. The compiler **rejects** any `@Name()` where `Name` resolves to a non-attribute declaration (e.g., `data`, `extern type`, `union`).

```zirric
// COMPILE ERROR: Bool is an extern type, not an attribute
data GenerateTask {
  @Bool()      // ← rejected: "Bool" does not refer to attribute type
  isDryRun
}

// Correct: use type hint syntax
data GenerateTask {
  isDryRun: Bool
}

// COMPILE ERROR: String is an extern type, not an attribute
@String()
fn greet(name) { "Hello, " + name }

// Correct: use return type hint
fn greet(name) -> String { "Hello, " + name }
```

This applies everywhere attribute instances appear: on declarations (`data`, `union`, `fn`, `extern`, `attr`, `mod`), on fields, and on parameters. The old implicit conversion of non-attribute types to `@Type(Type)` is removed entirely. Use `: Type` for type hints and `case is Type:` for switch cases.

### Attribute References Without Parentheses

When attributes appear in type expressions (hints, `is` patterns, switch cases), they are written without parentheses and without arguments. This indicates a constraint — "the value must be of a type that has this attribute" — rather than constructing an attribute instance. Attribute instances with arguments are only created on `data`, `union`, `fn`, `mod`, and other declarations.

```zirric
// Type expression: constraint (no parens)
fn sum(a: @Numeric, b: @Numeric) -> Number { ... }

// Declaration position: instance (with parens)
@Deprecated("use newSum instead")
fn sum(a: @Numeric, b: @Numeric) -> Number { ... }
```

### Runtime Semantics of `is`

The `is` operator and `case is:` patterns use a single runtime check (`IsType` opcode). The VM determines the check to perform based on what the constant resolves to:

| Constant type                    | Runtime behavior                                                                     |
| -------------------------------- | ------------------------------------------------------------------------------------ |
| `data`, `extern type`            | Direct type ID comparison: `value.TypeConstantId() == typeId`                        |
| `union`                          | Union membership check: `union.IsMember(value.TypeConstantId())`                     |
| `attr`                           | Attribute check: does the value's type carry this attribute?                         |
| Composite (`[T]`, `[K:V]`, `fn`) | Check the container type (Array, Dict, Function); inner types are documentation only |

For multiple attribute constraints (`is @A @B @C`), each attribute is checked with short-circuit AND semantics. All attributes must be present for the expression to return `true`.

### Composite Type Expressions

Composite types are fixed syntax for built-in types:

- **Array**: `[ElementType]` — the element type is any type expression
- **Dict**: `[KeyType: ValueType]` — both key and value types are type expressions; mirrors the dict literal syntax `[k: v]`
- **Function**: `fn(params) -> ReturnType` — parameters follow the same syntax as function declarations (name with optional `: Type`), return type is optional (defaults to `Any`)

These are not generics. No user-defined types can use this bracket/fn syntax. They exist because Array, Dict, and Function are language built-ins that benefit from structural type notation.

In composite type expressions, subtypes default to `Any` when omitted would create ambiguity. For function types, omitting the return type means `-> Any`. The parameter types are individually optional: `fn(a, b)` means `fn(a: Any, b: Any)`.

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
(* Type expressions — unified grammar for all type positions *)
type_expr = type_name                                          (* named type *)
          | attr_ref, { attr_ref }                             (* attribute constraint(s) *)
          | "[", type_expr, "]"                                (* array type *)
          | "[", type_expr, ":", type_expr, "]"                (* dict type *)
          | "fn", "(", [typed_param_list], ")", ["->", type_expr]  (* function type *)
          ;

type_name = identifier, { ".", identifier } ;

(* Attribute reference — fully qualified paths allowed *)
attr_ref = "@", type_name ;

(* Function type parameters — same syntax as function declarations *)
typed_param_list = typed_param, { ",", typed_param } ;
typed_param = identifier, [":", type_expr] ;

(* Type hint and return type *)
type_hint = ":", type_expr ;
return_type = "->", type_expr ;

(* is expression — always takes a type expression *)
is_expr = expression, "is", type_expr ;

(* switch case patterns — is always required for type matching *)
case_pattern = "is", type_expr      (* type/attribute match *)
             | expression           (* equality match *)
             | "_"                  (* default *)
             ;
```

All positions that accept types use the same `type_expr` grammar. Multiple `@attr_ref` in a type expression are combined with short-circuit AND semantics.

### Where Type Expressions Are Allowed

| Position                      | Syntax                                                                                   | Required?                                  |
| ----------------------------- | ---------------------------------------------------------------------------------------- | ------------------------------------------ |
| Composite type subexpressions | `[<type_expr>]`, `[<type_expr>: <type_expr>]`, `fn(<param>: <type_expr>) -> <type_expr>` | Mandatory (to fulfill composite structure) |
| Function parameters           | `fn greet(arg: <type_expr>)`                                                             | Optional                                   |
| Function return types         | `fn greet() -> <type_expr>`                                                              | Optional                                   |
| Constants and variables       | `const x: <type_expr>`, `var y: <type_expr>`                                             | Optional                                   |
| Function/closure declarations | `fn name(arg: <type_expr>) -> <type_expr>`                                               | Optional                                   |
| Data and extern type fields   | `data Foo { field: <type_expr> }`                                                        | Optional                                   |
| `is` expressions              | `x is <type_expr>`, `case is <type_expr>:`                                               | Mandatory (right-hand side of `is`)        |

Type expressions are **not** required for `union` member declarations.

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

All standard library declarations using the old attribute-based type hints will be updated to use the new syntax. For example:

```zirric
// Old:
extern type Array { @Type(Int) length }

// New:
extern type Array { length: Int }
```

## Alternatives Considered

- **Keep `: Type` as sugar for `@Type`** (ZE-012 approach): This maintains backwards compatibility but preserves the conceptual confusion of types-as-attributes. Rejected because a clean separation is worth the breaking change.
- **Generics** (`Array[String]`, `Result[String, Error]`): Too powerful for Zirric's goals as a simple, dynamic language. The `[T]`, `[K: V]`, and `fn() -> T` syntax covers the built-in types without introducing a generic type system.
- **Bare type names in switch** (`case String:`): Clean but ambiguous when a variable holds a type value. The `is` keyword eliminates this ambiguity.
- **`case @Attr:` without `is`**: The previous design allowed `case @Attr:` as a shorthand for attribute matching. This was removed in favour of requiring `case is @Attr:` for consistency — all type matching uses `is`, and `@` in case patterns no longer has special meaning.
- **Keep `@ItemType`/`@OkType`/`@SomeType` as metadata**: These would survive as documentation hints, but in practice users who need typed containers should define concrete union types. Dropped for simplicity.
- **Separate `HasAttribute` opcode**: The earlier implementation used a dedicated `HasAttribute` opcode for attribute checks and `Dup` for multi-attribute short-circuit AND. This was replaced by extending `IsType` to handle attribute constants, unifying all type matching under a single opcode.

## Acknowledgements

- Builds on [ZE-012 Type and Returns Sugar](/proposals/ZE-012-type-and-returns-sugar) which first introduced the `: Type` and `-> Type` syntax.
- The `is` keyword for type checking is inspired by Swift, Kotlin, and TypeScript.
- Collection type syntax `[T]` and `[K: V]` mirrors Zirric's own array and dict literal syntax `[1, 2]` and `[k: v]`.
- Function type syntax `fn(A) -> R` is inspired by Rust's `fn(A) -> R` and Swift's `(A) -> R`.
