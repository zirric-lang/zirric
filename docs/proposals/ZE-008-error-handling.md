---
title: "ZE-008 - Error Handling"
description: "Standardize error types and result values with ergonomic syntax."
---

# Error Handling

::: callout warning <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-dot-icon lucide-circle-dot"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="3" fill="currentColor" stroke="none"/></svg> In Progress
This proposal has been accepted in principle.
It is currently under active development.
Parts might be incomplete or missing in Zirric.
:::

## Introduction

This proposal introduces a standard error protocol and a `Result` union type for
failure-aware APIs. It also defines syntactic sugar for extracting values or
fallbacks from results.

## Motivation

Zirric currently lacks a canonical way to represent failures. A shared `Result` model makes error flows explicit and composable.

The usage of results should be ergonomic, minimizing boilerplate when extracting values or providing fallbacks.

## Proposed Solution

First of all there will be a `@Error` annotation. This annotation can be enforced.
Introduce an `@AnyResult` annotation to be defined on unions.
Then there will be a standard `Result` union with `Ok` and `Err` variants.

```zirric
@Returns(String!) // Syntactic sugar for @Type(Result) @OkType(String)
func readFile(path) {
  // returns Ok(String) or Err(Error)
}

let text = "Contents:\n" + readFile("notes.txt")!.value // @Type(Result) @OkType(String)
switch text {
case Ok(value):
  // use value
case Err(error):
  panic(error.debug(error))
}
```

### Syntactic Sugar

- `result!.value` unwraps `Result` into `Ok(value)` or `Err(error)`.
- `result !! "default"` unwraps to `Ok.value` or returns the default for `Err`.
- `@String!` expands to `@Type(Result) @OkType(String)`.

## Detailed Design

- `@Error` marks types that represent errors and must provide a bound `debug`
  function returning a string.
- `Result` is a union with `Ok` and `Err` data variants.
- `@AnyResult` semantic documentation.
- `@OkType` and `@ErrType` provide type hints for `Result` payloads.
- `Err.error` must satisfy `@Has(Error)`.
- `panic` to immediately terminate execution with an error message.

For the `!!`, `!.` operators, the VM needs to look at the presence of the `@prelude.Error` annotation. This should be doable with a deref of the underlying type and checking for the annotation presence using a Type ID.
If present, the value is treated as an error. Otherwise it is treated as a normal value.
If these values are not nested in `Ok` or `Err`, they will be wrapped accordingly.

The goal is to support strongly typed result types like `PersonResult`.

```zirric
annotation Error {
	@Bound()
	@Returns(String)
	debug(err)
}

annotation AnyResult {}

@AnyResult()
union Result {
  data Ok {
		value
	}

	@Error({ err -> err.error.debug(err.error) })
  data Err {
		@Has(Error)
		error
	}
}

annotation OkType {
	@Type(AnyType) type
}

annotation ErrType {
	@Type(AnyType) type
}
```

## Changes to the Standard Library

- Add `@Error`, `@AnyResult`, `Result`, `Ok`, `Err`, `@OkType`, and `@ErrType` in
  `prelude`.
- `panic(@String str)` to immediately terminate execution with an error message.

## Alternatives Considered

- Exceptions or panics, which do not fit the explicit, annotation-driven model.
- Returning `Option`, which loses error context and diagnostics.

## Acknowledgements

- Influenced by Rust's `Result<T, E>` and Swift's `Result` type.
