---
title: "ZE-007 - Annotation Binding"
description: "Allow annotation functions to bind to their target value automatically."
---

# Annotation Binding

::: callout draft <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-search-slash-icon lucide-search-slash"><path d="m13.5 8.5-5 5"/><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg> Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design.
Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

The proposed annotation binding feature allows a simplified way to call functions of annotations on actual values.
This enables ergonomic, method-like APIs for annotations without duplicating call the receiver.

## Motivation

Zirric are not only used for metadata but to design contracts.
In practice this leads to unnecessarily complex calls of annotation functions.

```zirric
Annotation(value).annoFunc(value, otherArgs)
```

## Proposed Solution

Introduce a `@Bound()` annotation for functions declared inside annotations.
This allows the compiler to bind the target value automatically as the first argument
when calling the function through the annotation instance accessor.

```zirric
annotation Error {
  @Bound()
  @Returns(String)
  debug(err)
}

@Error({ err -> err.message })
data MessageError {
  @String message
}

MessageError("msg")[@Error].debug() // `err` is passed automatically
```

## Detailed Design

- `@Bound()` can be applied to functions declared inside an annotation.
- A bound function is invoked through the annotation instance accessor
  (`value[@Annotation].method()`), and the target value is passed as the first
  argument.
- The function signature remains explicit; tooling can still infer the argument
  types from the annotation body.
- `@Bound()` can only be applied to functions with at least one argument.
- `@Bound()` can only be applied to functions inside annotations.

## Changes to the Standard Library

- Add a `@Bound` annotation in `prelude` to mark bound functions.
- Bindings are opt-in and do not change existing annotations.

## Alternatives Considered

- Manual receiver arguments in every annotation function, which is verbose.
- Special syntax for annotation methods, which would add new language surface
  area without clear benefits over an annotation-based marker.

## Acknowledgements

- Inspired by method receivers in languages like Go and extension methods in C#.
