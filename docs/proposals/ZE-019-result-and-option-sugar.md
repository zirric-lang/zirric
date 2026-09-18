---
title: "ZE-019 - Result and Option Sugar"
description: "Syntax sugar for ergonomic result unwrapping, optional chaining, fallback operators, and type shorthands"
---

# Result and Option Sugar

::: callout draft Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design.
Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

This proposal introduces syntax sugar for working with `Result` ([ZE-008](/proposals/ZE-008-error-handling)) and `Option` ([ZE-009](/proposals/ZE-009-option-values)) values. It defines operators for unwrapping, chaining, providing fallbacks, and type shorthands in signatures.

These operators were previously part of ZE-008 and ZE-009 respectively. They are combined here because the operators form a symmetric pair and share implementation concerns.

## Motivation

While `Result` and `Option` types make success/failure and presence/absence explicit, working with them using only `switch` statements is verbose. Consider accessing a nested optional field:

```zirric
// Without sugar — verbose and deeply nested
const result = findUser(42)
const city = switch result {
case None: "Unknown"
default:
    const address = result.value.address
    switch address {
    case None: "Unknown"
    default: address.value.city
    }
}
```

```zirric
// With sugar — clear and flat
const city = findUser(42)?.value.address?.value.city ?? "Unknown"
```

Ergonomic operators reduce boilerplate for common patterns — unwrapping a success value, providing a fallback, or chaining through optional fields — without sacrificing the explicit error model.

## Proposed Solution

Six new syntactic forms, grouped into three symmetric pairs:

### Result Operators

| Syntax               | Name                  | Description                              |
| -------------------- | --------------------- | ---------------------------------------- |
| `result!.field`      | Result unwrapping     | Access field on success, propagate error |
| `result !! fallback` | Result fallback       | Unwrap success value or use fallback     |
| `T!`                 | Result type shorthand | Type hint for a result yielding `T`      |

### Option Operators

| Syntax               | Name                  | Description                               |
| -------------------- | --------------------- | ----------------------------------------- |
| `option?.field`      | Optional chaining     | Access field if present, propagate `None` |
| `option ?? fallback` | Option fallback       | Unwrap present value or use fallback      |
| `T?`                 | Option type shorthand | Type hint for an optional `T`             |

### Examples

```zirric
// Result unwrapping — propagates Err to the caller
fn processFile(path: String) -> Result {
    const content = readFile(path)!.value
    return Ok(parse(content))
}

// Result fallback
const text = readFile("config.txt") !! "default config"

// Optional chaining — propagates None
const city = findUser(42)?.value.address?.value.city

// Option fallback
const name = findUser(42)?.value.name ?? "Unknown"

// Type shorthands in signatures
fn readFile(path: String) -> String! { ... }
fn findUser(id: Int) -> User? { ... }
```

## Detailed Design

### Result Unwrapping (`!.`)

The `!.` operator unwraps a value marked with `@AnyResult`. It inspects whether the value is an error (has the `@Error` attribute on its type) or a success:

- **Success**: unwraps the value and accesses the field.
- **Error**: propagates the error as the return value of the enclosing function.

```zirric
const content = readFile(path)!.value

// equivalent to:
const _tmp = readFile(path)
switch _tmp {
case Err(e): return Err(e)
}
const content = _tmp.value
```

The VM detects errors by checking for the `@Error` attribute on the value's type. Non-error values are treated as success and accessed directly. If the value is not already wrapped in `Ok`, it is used as-is.

### Result Fallback (`!!`)

The `!!` operator provides a fallback value when the result is an error:

```zirric
const text = readFile("config.txt") !! "default"

// equivalent to:
const _tmp = readFile("config.txt")
const text = switch _tmp {
case Err(_): "default"
default: _tmp.value
}
```

### Optional Chaining (`?.`)

The `?.` operator accesses a field on a value marked with `@AnyOption`. It checks whether the value is `None`:

- **Present**: accesses the field and wraps the result in `Some` if needed.
- **Absent**: short-circuits and returns `None`.

```zirric
const city = user?.value.address?.value.city

// equivalent to:
const city = switch user {
case None: None
default:
    switch user.value.address {
    case None: None
    default: Some(user.value.address.value.city)
    }
}
```

The VM detects absence by checking if the value is `None`. Non-`None` values are accessed directly. Results of `?.` are wrapped in `Some` if the accessed value is not already `None` or `Some`.

### Option Fallback (`??`)

The `??` operator provides a fallback value when the option is `None`:

```zirric
const name = findUser(42)?.value.name ?? "Unknown"

// equivalent to:
const _tmp = findUser(42)
const name = switch _tmp {
case None: "Unknown"
default: _tmp.value.name
}
```

### Type Shorthands

`T!` and `T?` are type hint shorthands in signatures:

```zirric
fn readFile(path: String) -> String! { ... }
// the return type is a @AnyResult union where success is String

fn findUser(id: Int) -> User? { ... }
// the return type is a @AnyOption union where the present value is User
```

The exact desugaring depends on how type hints ([ZE-017](/proposals/ZE-017-type-hints)) interact with union type parameters. At minimum, `T?` signals that the return type is an `@AnyOption` union and `T!` signals an `@AnyResult` union.

### Operator Precedence

From highest to lowest:

1. Member access (`.`)
2. Optional chaining (`?.`) and result unwrapping (`!.`)
3. Result fallback (`!!`) and option fallback (`??`)
4. Comparison operators
5. Assignment (`=`)

`?.` and `!.` bind tighter than `??` and `!!`, allowing natural chaining:

```zirric
// Parsed as: ((findUser(42))?.value.name) ?? "Unknown"
const name = findUser(42)?.value.name ?? "Unknown"

// Parsed as: ((readFile(path))!.value) !! "default"
const text = readFile(path)!.value !! "default"
```

### VM Implementation Notes

For `!.` and `!!`, the VM inspects the value's type for the `@Error` attribute. This is a type ID check — dereference the underlying type and check for attribute presence. Values with `@Error` are treated as errors; all others as success.

For `?.` and `??`, the VM checks if the value is `None` (a specific type identity check). `None` triggers the fallback or short-circuit path. All non-`None` values are treated as present.

This asymmetry — `@Error` attribute check for results vs. `None` identity check for options — reflects the structural difference: any type can be an error (if annotated), but only `None` represents absence.

## Changes to the Standard Library

None. This proposal only introduces syntax. It depends on the types defined in [ZE-008](/proposals/ZE-008-error-handling) and [ZE-009](/proposals/ZE-009-option-values).

## Dependencies on Other Proposals

| Proposal                                                  | Dependency                                  |
| --------------------------------------------------------- | ------------------------------------------- |
| [ZE-008 Error Handling](/proposals/ZE-008-error-handling) | `@Error`, `@AnyResult`, `Result` types      |
| [ZE-009 Option Values](/proposals/ZE-009-option-values)   | `@AnyOption`, `Option`, `None` types        |
| [ZE-017 Type Hints](/proposals/ZE-017-type-hints)         | `T!` and `T?` build on the type hint system |

## Alternatives Considered

### No Sugar

Users work with `switch` exclusively. Safe but verbose; common patterns like null checks and error propagation become deeply nested.

### Only Fallback Operators (`!!` and `??`)

Fallback operators without chaining. Simpler to implement but misses the most common use case — accessing fields on optional or result values without intermediate `switch` statements.

### Separate Proposals for Result and Option Sugar

Previously, result sugar was part of [ZE-008](/proposals/ZE-008-error-handling) and option sugar was part of [ZE-009](/proposals/ZE-009-option-values). They are combined here because:

1. The operators form a symmetric pair (`!.`/`?.`, `!!`/`??`, `T!`/`T?`).
2. They share VM implementation concerns (value inspection, short-circuit semantics).
3. Designing them together ensures consistent precedence and interaction rules.

### Rust-Style `?` Operator

Rust uses a single `?` operator for both `Result` and `Option` propagation. This was considered but rejected because Zirric's `!.` and `?.` make the distinction between error propagation and optional chaining visually explicit, reducing ambiguity in a dynamically-typed language.

## Acknowledgements

- `?.` and `??` are influenced by Swift's optional chaining and nil-coalescing operators.
- `!.` is influenced by Rust's `?` operator for result propagation.
- Result and option type shorthands draw from Swift's `T?` syntax.
- The underlying types are defined in [ZE-008](/proposals/ZE-008-error-handling) and [ZE-009](/proposals/ZE-009-option-values).
