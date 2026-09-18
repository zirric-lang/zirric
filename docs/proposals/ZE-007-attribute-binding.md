---
title: "ZE-007 - Attribute Binding"
description: "Allow attribute functions to bind to their target value automatically."
---

# Attribute Binding

::: callout draft Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design.
Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

The proposed attribute binding feature allows a simplified way to call functions of attributes on actual values.
This enables ergonomic, method-like APIs for attributes without duplicating call the receiver.

## Motivation

Zirric attributes are not only used for metadata but to design contracts.
In practice this leads to unnecessarily complex calls of attribute functions.

```zirric
Attribute(value).attrFunc(value, otherArgs)
```

## Proposed Solution

Introduce a `@Bound()` attribute for functions declared inside attributes.
This allows the compiler to bind the target value automatically as the first argument
when calling the function through the attribute instance accessor.

```zirric
attr Error {
  @Bound()
  debug(err) -> String
}

@Error(fn(err) { err.message })
data MessageError {
  message: String
}

MessageError("msg")[@Error].debug() // `err` is passed automatically
```

## Detailed Design

- `@Bound()` can be applied to functions declared inside an attribute.
- A bound function is invoked through the attribute instance accessor
  (`value[@Attribute].method()`), and the target value is passed as the first
  argument.
- The function signature remains explicit; tooling can still infer the argument
  types from the attribute body.
- `@Bound()` can only be applied to functions with at least one argument.
- `@Bound()` can only be applied to functions inside attributes.

## Changes to the Standard Library

- Add a `@Bound` attribute in `prelude` to mark bound functions.
- Bindings are opt-in and do not change existing attributes.

## Alternatives Considered

- Manual receiver arguments in every attribute function, which is verbose.
- Special syntax for attribute methods, which would add new language surface
  area without clear benefits over an attribute-based marker.

## Acknowledgements

- Inspired by method receivers in languages like Go and extension methods in C#.
