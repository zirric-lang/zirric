---
title: "ZE-009 - Option Values"
description: "Introduce a standard option type for values that may be present or absent."
---

# Option Values

::: callout warning <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-dot-icon lucide-circle-dot"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="3" fill="currentColor" stroke="none"/></svg> In Progress
This proposal has been accepted in principle.
It is currently under active development.
Parts might be incomplete or missing in Zirric.
:::

## Introduction

This proposal adds a standard `Option` union type to model values that may be present or absent. It defines the `@AnyOption` attribute for marking option unions.

Syntax sugar for working with optional values (`?.`, `??`, `T?`) is covered separately in [ZE-019 Result and Option Sugar](/proposals/ZE-019-result-and-option-sugar).

## Motivation

Zirric currently relies on ad-hoc conventions for missing values. A canonical `Option` type helps communicate intent and enables tooling to recognize optional flows consistently. By separating the type definitions from the syntax sugar, this proposal focuses on the foundational types.

## Proposed Solution

Define an `Option` union with `Some` and `None` variants and a marker attribute.

```zirric
attr AnyOption {}

@AnyOption()
union Option {
    data Some {
        value
    }

    data None
}
```

### Usage

```zirric
fn findUser(id: Int) -> Option {
    // returns Some(user) or None
}

const result = findUser(42)
switch result {
case Some(user):
    // use user.value
case None:
    // handle missing
}
```

Custom option types can be defined for specific domains:

```zirric
@AnyOption()
union PersonOption {
    Person
    None
}
```

## Detailed Design

### `@AnyOption`

The `@AnyOption` attribute is a marker applied to union types. It signals that the union represents an optional value — one variant for presence and one for absence. This attribute is used by [ZE-019](/proposals/ZE-019-result-and-option-sugar) to enable syntactic sugar.

### `Option`

The standard `Option` union has two variants:

- `Some { value }` — the present value, wrapping the inner value.
- `None` — the absent value, a data type with no fields.

## Changes to the Standard Library

| Declaration | Kind    | Description                         |
| ----------- | ------- | ----------------------------------- |
| `AnyOption` | `attr`  | Marker for option union types       |
| `Option`    | `union` | Standard `Some \| None` option type |
| `Some`      | `data`  | Present variant with `value` field  |
| `None`      | `data`  | Absent variant (no fields)          |

## Alternatives Considered

- **Using `null` or `none` literals** — makes optional flows implicit and harder to check.
- **Reusing `Result`** — conflates absence with failure. `Result` ([ZE-008](/proposals/ZE-008-error-handling)) is for errors, not absence.

## Acknowledgements

- Influenced by Swift's `Optional` and Rust's `Option<T>`.
- Syntax sugar for option values is defined in [ZE-019 Result and Option Sugar](/proposals/ZE-019-result-and-option-sugar).
