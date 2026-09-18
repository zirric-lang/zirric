---
title: ZE-015 - Extern Type Constructors
description: Add optional constructors to extern type declarations, allowing Zirric code to instantiate native-backed types directly. Type narrowing remains the responsibility of switch statements.
---

# Extern Type Constructors

::: callout draft Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design.
Features described here may not be implemented as described and cannot be used right now.
:::

::: callout warning Outdated
While this proposal has not been rejected, it is currently outdated and requires an overhaul to reflect the latest design decisions.

- [ ] Reflect latest syntax changes
- [ ] Reflect latest stdlib changes
- [ ] Attributes are no longer used for types
      :::

## Introduction

This proposal adds optional constructors to `extern type` declarations and introduces the constructor syntax:

```zirric
extern type SomeType(arg1, arg2) {
  // fields
}
```

A constructor lets Zirric code instantiate a native-backed type directly by calling the type name as a function. Without a constructor the type remains opaque — instances can only be obtained from the runtime, e.g. as return values of `extern fn` declarations.

This proposal introduces and defines constructors for the extern types in `prelude/shim.zirr` that have meaningful construction semantics.

## Motivation

Several core extern types in `prelude/shim.zirr` cannot be created from Zirric code at all. Creating an `Array`, a `Dict`, or a `String` is possible through literals, but more complex future types would not. There is no direct, uniform way to construct them. Adding constructors closes this gap and makes the type system feel more complete.

Constructors also serve as the natural place to define conversion semantics between related types, such as converting a `Float` to an `Int` or building a `String` from an `Array` of `Char` values.

## Proposed Solution

The following extern types in `prelude/shim.zirr` gain constructors:

| Type     | Constructor                        | Behaviour                                         |
| -------- | ---------------------------------- | ------------------------------------------------- |
| `Array`  | `Array(@Has(Iterable) iterable)`   | Collect any iterable into a new array             |
| `Bool`   | `Bool(@Any value)`                 | Test a value for truthiness                       |
| `Char`   | `Char(@Type(Int) codePoint)`       | Create a character from a Unicode code point      |
| `Dict`   | `Dict(@Array keys, @Array values)` | Build a dictionary from parallel key/value arrays |
| `Float`  | `Float(@Type(Number) number)`      | Convert a `Number` to a floating-point value      |
| `Int`    | `Int(@Type(Number) number)`        | Convert a `Number` to an integer, truncating      |
| `String` | `String(@Array chars)`             | Build a string from an array of `Char` values     |

The meta-level types — `Any`, `AnyType`, `Attribute`, `AttributeType`, `Module`, `ModuleType`, and `Func` — do not receive constructors. Narrowing to these types from a broader type is expressed with `switch`; see _Alternatives Considered_.

```zirric
let digits  = Array(someIterable)
let lookup  = Dict(keys, values)
let letter  = Char(65)             // 'A'
let pi      = Float(3)             // 3.0
let rounded = Int(3.9)             // 3
let hello   = String(charArray)
```

## Detailed Design

### Syntax

The EBNF production for `extern_type` from [ZE-001](/proposals/ZE-001-base-language) is extended with an optional constructor clause:

```ebnf
extern_type        = EXTERN, TYPE, extern_type_name,
                     [LPAREN, [extern_type_params], RPAREN],
                     [LBRACE, extern_type_fields, RBRACE] ;
extern_type_params = { parameter, [_list_separator] } ;
```

Constructor parameters follow the same attribute-based syntax as regular function parameters. When a constructor is present, calling the type as a function invokes it. When no constructor is declared, the type is not callable.

### Semantics per type

#### `Array(@Has(Iterable) iterable)`

Evaluates `iterable` using the protocol defined in [ZE-010](/proposals/ZE-010-iterable) and collects all produced values into a new, fully-realised `Array`. The order of elements matches the iteration order.

#### `Bool(@Any value)`

Returns `false` if `value` is one of the following _falsy_ values; returns `true` otherwise:

- `false`
- `void`

All other values are truthy. This might change in the future.

#### `Char(@Int codePoint)`

Returns the Unicode character whose code point equals `codePoint`. Behaviour is undefined for values outside the valid Unicode scalar value range.

#### `Dict(@Array keys, @Array values)`

Returns a new `Dict` mapping `keys[i]` to `values[i]` for each index. Both arrays must have the same length; passing arrays of unequal length is a runtime error. If `keys` contains duplicate entries the last mapping for a given key wins.

#### `Float(@Has(Numeric) number)`

- If `number` is already a `Float`, returns it unchanged.
- If `number` is an `Int`, promotes it to the nearest representable `Float`.
- If the type of the passed value has the `Numeric` attribute, its `toNumber` function will be used.

#### `Int(@Has(Numeric) number)`

- If `number` is already an `Int`, returns it unchanged.
- If `number` is a `Float`, truncates toward zero (i.e. the fractional part is discarded).
- If the type of the passed value has the `Numeric` attribute, its `toNumber` function will be used.

#### `String(@Array chars)`

Returns a new `String` whose contents are the `Char` values in `chars`, in order. Each element of `chars` must be a `Char`; passing other value types is a runtime error.

This might change in the future.

## Changes to the Standard Library

`prelude/shim.zirr` is updated to add constructors to the affected `extern type` declarations:

```zirric
// A finite list of values.
extern type Array(iterable: @Iterable) {
  length: Int
}

// Represents boolean values.
extern type Bool(value) {
  toggle() -> Bool
}

// A single Unicode character.
extern type Char(@Type(Int) codePoint) {}

// An associative array of keys and their values.
extern type Dict(keys [String], values: Array) {
  length: Int
}

// A floating point number.
@Numeric(fn(f) { return f })
extern type Float(@Has(Numeric) number) {}

// A whole integer number.
@Numeric(fn(i) { return i })
extern type Int(number: @Numeric) {}

extern type String(chars: [Char]) {
  length: Int
}
```

The declarations for `Any`, `AnyType`, `Attribute`, `AttributeType`, `Module`, `ModuleType`, `Func`, and `Void` are unchanged — they remain opaque and non-callable.

## Alternatives Considered

### Panicking assertion constructors for meta-level types

Constructors such as `AnyType(@Any value)` that assert the incoming value is of the correct type and panic otherwise were considered for `Any`, `AnyType`, `Module`, `ModuleType`, `Attribute`, `AttributeType`, and `Func`. This was rejected because Zirric already has a principled mechanism for type narrowing: `switch` statements. `switch` enforces that all relevant cases are handled, making type narrowing safe and exhaustive. A panicking constructor would bypass this safety guarantee, introducing an implicit failure mode inconsistent with the language's approach to type safety.

### A language-level cast syntax

Introducing a dedicated cast expression (e.g. `value as SomeType`) was considered. This is also unnecessary for the same reason: `switch` already covers safe narrowing. Adding a separate cast syntax would duplicate that responsibility and create two ways to do the same thing.

### Mandatory constructors for all extern types

Requiring a constructor on every `extern type` was considered, with the intent of giving every type a uniform callable interface. This was rejected in favour of keeping construction and type narrowing as distinct concepts. Types that cannot be meaningfully created from Zirric code should not be callable; the `switch` statement handles narrowing for those types.

### Variadic `Array` constructor: `Array(item1, item2, ...)`

A variadic constructor — `Array(1, 2, 3)` — is the most ergonomic form and is referenced in [ZE-004](/proposals/ZE-004-Variadic-Arguments). However, ZE-004 is still a draft. `Array(@Has(Iterable) iterable)` achieves the same goal without a dependency on unaccepted proposals. If ZE-004 is accepted, a variadic form could be added alongside.

### Truthy/falsy rules for `Bool`

An alternative is to require an explicit type — e.g. `Bool(@Type(Int) int)` — so that only integers can be tested for zero. The current proposal prefers the flexible `@Any` form because it mirrors the way Zirric conditions already work in `if`/`while` expressions, keeping `Bool(value)` a reliable way to reify a condition as a named value.

### `Dict` with a single array of key-value pairs

Instead of two parallel arrays, a single array of alternating keys and values (`Dict([k1, v1, k2, v2, ...])`) was considered. This is less readable and more error-prone (a mismatched pair count is harder to spot) compared to two explicit arrays of equal length.

### `String` from a single `Char`

`String(@Type(Char) char)` would allow creating a one-character string. This is a common enough operation, but it can be expressed with an array literal once array literal syntax is available. A dedicated single-character form can be added in a later proposal if it proves necessary.

## Acknowledgements

- `Array(@Has(Iterable) iterable)` builds on the iterable protocol defined in [ZE-010](/proposals/ZE-010-iterable).
