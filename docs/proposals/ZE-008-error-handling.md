---
title: "ZE-008 - Error Handling"
description: "Standardize error types and result values for failure-aware APIs."
---

# Error Handling

::: callout tip <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-check-icon lucide-circle-check"><circle cx="12" cy="12" r="10"/><path d="m9 12 2 2 4-4"/></svg> Implemented
This proposal has been accepted and implemented.
You can use this feature in the latest version of Zirric.
:::

## Introduction

This proposal introduces a standard error protocol and a `Result` union type for failure-aware APIs. It defines the `@Error` attribute for marking error types and the `@AnyResult` attribute for marking result unions.

Syntax sugar for working with result values (`!.`, `!!`, `T!`) is covered separately in [ZE-019 Result and Option Sugar](/proposals/ZE-019-result-and-option-sugar).

## Motivation

Zirric currently lacks a canonical way to represent failures. A shared `Result` model makes error flows explicit and composable. By separating the type definitions from the syntax sugar, this proposal focuses on the foundational types that other proposals and modules can build upon.

## Proposed Solution

Introduce three declarations:

1. `@Error` — an attribute marking types that represent errors, requiring a `debug` function.
2. `@AnyResult` — a marker attribute for union types that represent success-or-failure results.
3. `Result` — the standard result union with `Ok` and `Err` variants.

```zirric
attr Error {
    debug(err) -> String
}

attr AnyResult {}

@AnyResult()
union Result {
    data Ok {
        value
    }

    @Error(fn(err) { return err.error.debug(err.error) })
    data Err {
        error: @Error
    }
}
```

### Usage

```zirric
fn readFile(path: String) -> Result {
    // returns Ok("contents") or Err(someError)
}

const result = readFile("notes.txt")
switch result {
case Ok(text):
    // use text.value
case Err(error):
    panic(Error(error).debug(error))
}
```

Custom error types implement the `@Error` attribute:

```zirric
@Error(fn(err) { return "file not found: " + err.path })
data FileNotFound {
    path: String
}
```

Custom result types can be defined for specific domains:

```zirric
@AnyResult()
union FileResult {
    String
    FileNotFound
}
```

## Detailed Design

### `@Error`

The `@Error` attribute marks a type as an error. It requires a `debug` function that returns a string representation of the error for diagnostics.

### `@AnyResult`

The `@AnyResult` attribute is a marker applied to union types. It signals that the union represents a result — one variant for success and one (or more) for failure. This attribute is used by [ZE-019](/proposals/ZE-019-result-and-option-sugar) to enable syntactic sugar.

### `Result`

The standard `Result` union has two variants:

- `Ok { value }` — the successful result, wrapping the success value.
- `Err { error: @Error }` — the error result. The `error` field must satisfy the `@Error` attribute.

`Err` itself is annotated with `@Error`, delegating its `debug` to the inner error's `debug` function. This means `Err` values are composable as errors.

### `panic`

A `panic` function is added for unrecoverable errors:

```zirric
extern fn panic(message: String) -> Void
```

`panic` immediately terminates execution with the given message. It should be used sparingly — `Result` is preferred for recoverable errors.

## Changes to the Standard Library

| Declaration | Kind        | Description                                        |
| ----------- | ----------- | -------------------------------------------------- |
| `Error`     | `attr`      | Marks error types, requires `debug(err) -> String` |
| `AnyResult` | `attr`      | Marker for result union types                      |
| `Result`    | `union`     | Standard `Ok \| Err` result type                   |
| `Ok`        | `data`      | Success variant with `value` field                 |
| `Err`       | `data`      | Error variant with `error: @Error` field           |
| `panic`     | `extern fn` | Terminate execution with error message             |

## Alternatives Considered

- **Exceptions or panics only** — do not fit Zirric's explicit, attribute-driven model.
- **Returning `Option`** — loses error context and diagnostics. `Option` ([ZE-009](/proposals/ZE-009-option-values)) is for absence, not failure.

## Acknowledgements

- Influenced by Rust's `Result<T, E>` and Swift's `Result` type.
- Syntax sugar for result values is defined in [ZE-019 Result and Option Sugar](/proposals/ZE-019-result-and-option-sugar).
