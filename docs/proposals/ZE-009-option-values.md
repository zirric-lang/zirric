---
title: "ZE-009 - Option Values"
description: "Introduce a standard option type and optional chaining syntax."
---

# Option Values

::: callout warning <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-dot-icon lucide-circle-dot"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="3" fill="currentColor" stroke="none"/></svg> In Progress
This proposal has been accepted in principle.
It is currently under active development.
Parts might be incomplete or missing in Zirric.
:::

## Introduction

This proposal adds a standard `Option` union type to model values that may be
present or absent, plus syntactic sugar for optional chaining and defaults.

## Motivation

Zirric currently relies on ad-hoc conventions for missing values. A canonical
`Option` type helps communicate intent and enables tooling to recognize optional
flows consistently.

## Proposed Solution

Define an `Option` union with `Some` and `None` variants and a marker annotation
that enables optional operators.

```zirric
@AnyOption()
union Option {
  data Some { value }
  data None
}

@Returns(User?)
func findUser(id) {
  // returns Some(User) or None
}

let name = findUser(42)?.value.name ?? "Unknown"
```

### Syntactic Sugar

- `option?.value` maps to `Some(value)` or `None`.
- `option ?? "default"` unwraps `Some.value` or returns the default for `None`.
- `@String?` expands to `@Type(Option) @SomeType(String)`.

## Detailed Design

- `Option` is a union with `Some` and `None` data variants.
- `@AnyOption` opt-in enables the `?.` and `??` operators for option values.
- `@SomeType` provides a type hint for `Some.value`.

For the `??`, `?.` operators, the VM needs to look if the value is a `@prelude.None`.
If it is `None`, it should return the default value for `??` or skip the member access for `?.`.
In case of `?.` the value will be wrapped into `prelude.Some` if needed if it is not `prelude.None`.

The goal is to support strongly typed optional types like `PersonOption`.

```zirric
annotation AnyOption {}

union Option {
	data Some {
		value
	}

	data None
}

annotation SomeType {
	@Type(AnyType) type
}
```

## Changes to the Standard Library

- Add `@AnyOption`, `Option`, and `@SomeType` in `future/prelude`.

## Alternatives Considered

- Using `null` or `none` literals without a type wrapper, which makes optional
  flows implicit and harder to check.
- Reusing `Result` for optional values, which conflates absence with failure.

## Acknowledgements

- Influenced by Swift's `Optional` and Rust's `Option<T>`.
