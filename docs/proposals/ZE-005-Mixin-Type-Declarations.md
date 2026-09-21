---
title: ZE-005 - Mixin Type Declarations
description: Introduce mixin type declarations to group and reuse attributes in Zirric.
---

# Mixin Type Declarations

::: callout draft Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design. Features described here may not be implemented as described and cannot be used right now.
:::

::: callout warning Outdated
While this proposal has not been rejected, it is currently outdated and requires an overhaul to reflect the latest design decisions.

- [ ] Reflect latest syntax changes
- [ ] Reflect latest stdlib changes
- [ ] Attributes are no longer used for types
- [ ] Evaluate if this is still necessary given the current design of types and attributes
      :::

## Introduction

This proposal introduces **`mixin` type declarations** to Zirric, enabling the definition of reusable groups of attributes. Mixins allow developers to apply a set of attributes to a declaration in a concise and explicit manner, promoting code reuse and reducing boilerplate.

## Motivation

### Problem

Zirric currently lacks a mechanism to group and reuse attributes across multiple declarations. For example, to enforce that a type is both `Serializable` and `Comparable`, developers must manually repeat the same set of attributes on each declaration. This leads to boilerplate and reduces maintainability.

### Use Cases

1. **Reusable Attribute Groups**: Define a set of attributes once (e.g., `@ToString`, `@FromString`) and apply them to multiple declarations.
2. **Traits/Mixins**: Simulate trait-like behavior by grouping related attributes (e.g., `Comparable`, `Serializable`).
3. **Library Design**: Library authors can provide mixins for common patterns, making APIs more ergonomic and consistent.

### Benefits

- **Reduced Boilerplate**: Avoid repeating the same attributes across multiple declarations.
- **Explicit and Intentional**: Mixins are applied explicitly, making the code easier to understand and maintain.
- **Consistency**: Encourages reuse of attribute patterns, leading to more consistent codebases.

## Proposed Solution

### Syntax

Introduce a new `mixin` declaration that contains only attributes (no fields or methods). Mixins are applied to declarations using the `@[MixinA, MixinB]` syntax.

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

- **Attribute Injection**: When a mixin is applied to a declaration (e.g., `@[Serializable]`), all attributes inside the mixin are injected into the declaration.
- **Overridability**: Attributes in the final declaration override those from mixins, except for explicit `@Type` attributes.
- **Conflict Resolution**: If two mixins inject conflicting attributes (e.g., `@ToString(A)` and `@ToString(B)`), the user must explicitly resolve the conflict in the final declaration.

## Detailed Design

### Mixin Declaration

- A `mixin` is a new type of declaration that can only contain attributes.
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

- **Compiler**: During AST transformation, the compiler injects the attributes from the mixins into the annotated declaration.
- **Type Checking**: The compiler enforces that explicit attributes in the final declaration override those from mixins, except for `@Type`.
- **Error Messages**: Provide clear error messages for conflicts or invalid mixin usage.
- **Tooling**: Update the LSP to recognize mixins and suggest them during autocompletion.

### Edge Cases

- **Empty Mixins**: Mixins with no attributes are allowed but have no effect.
- **Circular Dependencies**: The compiler should detect and reject circular dependencies between mixins.
- **Conflicts**: If two mixins inject conflicting attributes, the user must explicitly resolve the conflict in the final declaration.

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

### 2. Overriding Mixin Attributes

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
   - **Cons**: Risk of unintentionally copying unrelated attributes; less explicit.
   - **Why Not Chosen**: Mixins provide a more explicit and safer mechanism for attribute reuse.

2. **Attribute Composition**:
   - **Pros**: No new syntax required.
   - **Cons**: Less readable and more verbose (e.g., `@HasAll([@ToString, @FromString])`).
   - **Why Not Chosen**: Mixins offer a cleaner and more intuitive syntax.

3. **Trait-Like Interfaces**:
   - **Pros**: Familiar to developers from other languages.
   - **Cons**: Introduces complexity and deviates from Zirric's attribute-centric design.
   - **Why Not Chosen**: Mixins align better with Zirric's existing attribute system.

## Acknowledgements

- **Prior Art**: Inspired by mixins in languages like Scala and Dart, and trait-like behavior in Rust.
- **Contributors**: [@vknabel](https://github.com/vknabel) for designing the proposal.
