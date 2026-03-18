---
title: "ZE-003: Named Data Construction"
description: "Proposal for named argument syntax in data type construction and function calls."
---

# Named Data Construction

::: callout danger <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-x-icon lucide-circle-x"><circle cx="12" cy="12" r="10"/><path d="m15 9-6 6"/><path d="m9 9 6 6"/></svg> Rejected
This proposal has been rejected and will not be implemented.
:::

## Introduction

Zirric currently constructs data types and calls functions by position. Once a type gains more fields, calls become hard to read and fragile when properties are reordered. Named construction allows each argument to be associated with its field, improving clarity and long-term maintainability.

## Motivation

Using positional arguments for large records forces developers to memorize the declaration order. When the order changes, all call sites must be updated. Named arguments make the relationship between values and properties explicit and enable safer refactoring, especially for data types that evolve over time.

## Proposed Solution

Allow arguments to be supplied by name using the syntax `Type(foo: a, bar: b)` and `func(foo: a, bar: b)`, in addition to the existing ordered form `Type(a, b)`.

- **Unordered by default.** Names may appear in any order. The compiler associates each value with its field name.
- **Positional and named do not mix.** Once a call uses a named argument, all subsequent arguments must also be named. This avoids surprises when argument order changes.
- **Omitted fields.** A field may be left out only when annotated with `@Default(expr)`, which supplies the value when absent. Omitting any other field is a compile-time error.
- **Duplicates.** Supplying the same name more than once is a compile-time error.
- **Functions and data types.** The feature applies to both type construction and ordinary functions for consistency.

This proposal intentionally does not add an ordered variant of named arguments where declaration order must be followed; the existing positional syntax already supports ordered calls. Likewise, a `Type { prop: value }` record literal is not proposed because it conflicts with block and map syntax; it can be explored as future work if needed.

## Detailed Design

Parser changes:

```
Call        ::= Callee '(' (ExprList | NamedArgs)? ')'
ExprList    ::= Expression (',' Expression)*
NamedArgs   ::= NamedArg (',' NamedArg)*
NamedArg    ::= Identifier ':' Expression
```

During type checking, each `NamedArg` is mapped to its corresponding field or parameter. The compiler determines which fields may be omitted by inspecting their declarations:

- A field annotated with `@Default(expr)` is optional and uses the attribute's expression when absent. If no type is written, the type is inferred from `expr`.

Any field lacking a `@Default` attribute is required and must appear in the call. Missing required fields produce errors.

When compiling, named arguments are reordered into positional form to reuse existing call conventions.

## Changes to the Standard Library

Introduce a `@Default` attribute that accepts an expression providing a field's default value. Standard library types and functions may adopt named construction and use `@Default` where appropriate.

```zirric
attr Default {
  // The default value expression for the field. Should be of the field's type.
  value
}
```

## Alternatives Considered

- **Keep positional construction only.** Leaves readability problems unsolved.
- **Record literal `Type { prop: value }`.** Mirrors Go but conflicts with existing block syntax and does not help with function calls.
- **Allow mixing positional and named arguments.** Provides flexibility but makes refactoring dangerous when positional indices shift.
- **Omit fields via `None` types.** Fields whose type admitted `None` could be skipped, implicitly supplying `None`. This ties omission to the optional-type mechanism and obscures the difference between an omitted field and an explicit `None` value, so `@Default` was preferred.

## Acknowledgements

Inspired by languages such as Swift and Go that offer named argument or field initialization.
