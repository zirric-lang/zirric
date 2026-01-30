---
title: ZE-005 - Mixin Type Declarations
description: Introduce mixin type declarations to group and reuse annotations in Zirric.
---

# Mixin Type Declarations

::: callout draft <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-icon lucide-circle"><circle cx="12" cy="12" r="10"/></svg> Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design.
Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

This proposal introduces **`mixin` type declarations** to Zirric, enabling the definition of reusable groups of annotations. Mixins allow developers to apply a set of annotations to a declaration in a concise and explicit manner, promoting code reuse and reducing boilerplate.

## Motivation

### Problem

Zirric currently lacks a mechanism to group and reuse annotations across multiple declarations. For example, to enforce that a type is both `Serializable` and `Comparable`, developers must manually repeat the same set of annotations on each declaration. This leads to boilerplate and reduces maintainability.

### Use Cases

1. **Reusable Annotation Groups**: Define a set of annotations once (e.g., `@ToString`, `@FromString`) and apply them to multiple declarations.
2. **Traits/Mixins**: Simulate trait-like behavior by grouping related annotations (e.g., `Comparable`, `Serializable`).
3. **Library Design**: Library authors can provide mixins for common patterns, making APIs more ergonomic and consistent.

### Benefits

- **Reduced Boilerplate**: Avoid repeating the same annotations across multiple declarations.
- **Explicit and Intentional**: Mixins are applied explicitly, making the code easier to understand and maintain.
- **Consistency**: Encourages reuse of annotation patterns, leading to more consistent codebases.

## Proposed Solution

### Syntax

Introduce a new `mixin` declaration that contains only annotations (no fields or methods). Mixins are applied to declarations using the `@[MixinA, MixinB]` syntax.

#### Example: Defining a Mixin

```zirric
mixin Serializable {
    @ToString(json.Encode)
    @FromString(json.Decode)
}
```

#### Example: Applying a Mixin

```zirric
@[Serializable]
data Config {
    key
    value
}
```

### Behavior

- **Annotation Injection**: When a mixin is applied to a declaration (e.g., `@[Serializable]`), all annotations inside the mixin are injected into the declaration.
- **Overridability**: Annotations in the final declaration override those from mixins, except for explicit `@Type` annotations.
- **Conflict Resolution**: If two mixins inject conflicting annotations (e.g., `@ToString(A)` and `@ToString(B)`), the user must explicitly resolve the conflict in the final declaration.

## Detailed Design

### Mixin Declaration

- A `mixin` is a new type of declaration that can only contain annotations.
- Mixins cannot contain fields, methods, or other members.
- Example:
  ```zirric
  mixin Comparable {
      @Has(LessThan)
      @Has(Equals)
  }
  ```

### Applying Mixins

- Mixins are applied using the `@[MixinA, MixinB]` syntax.
- Example:
  ```zirric
  @[Comparable, Serializable]
  data Person {
      name
      age
  }
  ```

### Compiler and Tooling Support

- **Compiler**: During AST transformation, the compiler injects the annotations from the mixins into the annotated declaration.
- **Type Checking**: The compiler enforces that explicit annotations in the final declaration override those from mixins, except for `@Type`.
- **Error Messages**: Provide clear error messages for conflicts or invalid mixin usage.
- **Tooling**: Update the LSP to recognize mixins and suggest them during autocompletion.

### Edge Cases

- **Empty Mixins**: Mixins with no annotations are allowed but have no effect.
- **Circular Dependencies**: The compiler should detect and reject circular dependencies between mixins.
- **Conflicts**: If two mixins inject conflicting annotations, the user must explicitly resolve the conflict in the final declaration.

## Examples

### 1. Defining and Applying a Mixin

```zirric
mixin Serializable {
    @ToString(json.Encode)
    @FromString(json.Decode)
}

@[Serializable]
data Config {
    key
    value
}
```

### 2. Overriding Mixin Annotations

```zirric
mixin Loggable {
    @ToString(default.ToString)
}

@[Loggable]
@ToString(custom.ToString)  // Overrides `@ToString(default.ToString)`
data Example {
    field
}
```

### 3. Resolving Conflicts

```zirric
mixin MixinA {
    @ToString(A)
}

mixin MixinB {
    @ToString(B)  // Conflict with MixinA
}

@[MixinA, MixinB]
@ToString(Custom)  // Explicitly resolve the conflict
data Example {
    field
}
```

## Changes to the Standard Library

No changes to the standard library are required for this feature. However, future libraries may provide commonly used mixins (e.g., `Comparable`, `Serializable`).

## Alternatives Considered

1. **`@Inherit(anyType)`**:
   - **Pros**: Simple and requires no new syntax.
   - **Cons**: Risk of unintentionally copying unrelated annotations; less explicit.
   - **Why Not Chosen**: Mixins provide a more explicit and safer mechanism for annotation reuse.

2. **Annotation Composition**:
   - **Pros**: No new syntax required.
   - **Cons**: Less readable and more verbose (e.g., `@HasAll([@ToString, @FromString])`).
   - **Why Not Chosen**: Mixins offer a cleaner and more intuitive syntax.

3. **Trait-Like Interfaces**:
   - **Pros**: Familiar to developers from other languages.
   - **Cons**: Introduces complexity and deviates from Zirric's annotation-centric design.
   - **Why Not Chosen**: Mixins align better with Zirric's existing annotation system.

## Acknowledgements

- **Prior Art**: Inspired by mixins in languages like Scala and Dart, and trait-like behavior in Rust.
- **Contributors**: [@vknabel](https://github.com/vknabel) for designing the proposal.
