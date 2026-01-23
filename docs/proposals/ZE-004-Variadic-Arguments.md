---
title: ZE-004 - Variadic Annotations
description: Introduce variadic annotations to allow annotations to accept a variable number of arguments.
---

# Variadic Annotations

::: callout draft <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-search-slash-icon lucide-search-slash"><path d="m13.5 8.5-5 5"/><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg> Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design.
Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

This proposal introduces **variadic annotations** to Zirric, enabling annotations to accept a variable number of arguments. This feature addresses the need for multi-instance constraints (e.g., `@Requires(Countable, Iterable)`) while maintaining a clean and expressive syntax.

Variadic annotations are particularly useful for:

- Representing requirements or constraints on fields, parameters, or types.
- Improving readability and reducing verbosity compared to alternatives like `@HasAll([Countable, Iterable])`.

---

## Motivation

### Problem

Zirric currently lacks a concise way to specify multiple constraints or requirements in annotations. For example, to enforce that a parameter must satisfy both `Countable` and `Iterable`, users must use workarounds like `@HasAll([Countable, Iterable])`, which is less readable and feels unnatural.

### Use Cases

1. **Type Constraints**: Annotating parameters or fields to require multiple traits or interfaces.
   ```zirric
   func process(@Requires(Countable, Iterable) data) {}
   ```
2. **Tooling and Compile-Time Checks**: Enabling tools (e.g., LSP, compilers) to validate that values meet multiple constraints.
3. **Library Design**: Allowing library authors to define flexible APIs that accept variadic constraints.

### Benefits

- **Readability**: `@Requires(Countable, Iterable)` is more intuitive than `@HasAll([Countable, Iterable])`.
- **Flexibility**: Supports empty lists, optional constraints, and future extensibility.
- **Consistency**: Aligns with variadic functions in other languages, making it familiar to developers.

---

## Proposed Solution

### Syntax

Introduce a `@Variadic` annotation to mark annotation fields as accepting a variable number of arguments. When used, the field’s type is treated as an `Array` of the specified element type.

#### Example: Defining a Variadic Annotation

```zirric
annotation Requires {
  @Variadic()
  @Elements(Type)  // Accepts types or type-like annotations
  constraints
}
```

#### Example: Using a Variadic Annotation

```zirric
// Function with variadic constraints
func process(@Requires(Countable, Iterable) data) {
  // `data` must satisfy both `Countable` and `Iterable`
}

// Valid calls
process([1, 2, 3])  // Array of Int (assuming Int is Countable and Iterable)
process([])         // Empty array

// Invalid call (if String doesn't satisfy Iterable)
process("hello")    // Compiler error: String is not Iterable
```

### Behavior

- **Type Handling**: Variadic arguments are always treated as an `Array` of the specified element type (e.g., `@Elements(Type)`).
- **Empty Lists**: Supported for optional constraints (e.g., `@Requires()`).
- **Type Safety**: The compiler enforces that all provided arguments match the element type (e.g., `Type`).

---

## Detailed Design

### Annotation Definition

- The `@Variadic` annotation marks a field within another annotation as variadic.
- The field’s type is implicitly an `Array` of the type specified by `@Elements`.
  ```zirric
  annotation VariadicExample {
    @Variadic()
    @Elements(Int)  // Accepts variadic Int arguments
    numbers
  }
  ```

### Usage in Functions

- Variadic annotations can be applied to function parameters, fields, or types.
  ```zirric
  func example(@VariadicExample(1, 2, 3) nums) {}
  ```

### Compiler and Tooling Support

- **Parsing**: The compiler treats variadic annotation fields as arrays during parsing.
- **Type Checking**: Ensures all arguments match the element type specified by `@Elements`.
- **Error Messages**: Provides clear errors for type mismatches or invalid usage.

### Edge Cases

- **Empty Lists**: `@Requires()` is valid and results in an empty array of constraints.
- **Single Argument**: `@Requires(Countable)` is treated as an array with one element.
- **Type Mismatches**: The compiler rejects arguments that don’t match `@Elements`.

---

## Changes to the Standard Library

### New Annotations

1. **`@Variadic`**:
   - Marks a field as accepting a variable number of arguments.
   - Only valid within annotation definitions.

2. **`@Elements`**:
   - Specifies the type of elements in a variadic field (e.g., `@Elements(Type)`).
   - Required for variadic fields to enforce type safety.

### **Example Additions**

```zirric
// Standard library addition
annotation Variadic {}
annotation Elements {
  type
}
```

---

## Alternatives Considered

1. **`@HasAll([...])` Syntax**:
   - **Pros**: No new syntax required.
   - **Cons**: Less readable and feels like a workaround.

2. **Dedicated Grouping Syntax (e.g., `@Requires{Countable, Iterable}`)**:
   - **Pros**: Clean and intuitive.
   - **Cons**: Doesn’t translate to function arguments and requires new grammar rules.

3. **Multiple Annotations (e.g., `@Requires(Countable) @Requires(Iterable)`)**:
   - **Pros**: Simple and explicit.
   - **Cons**: Verbose and ambiguous for single-instance annotations.

**Why `@Variadic` Was Chosen**:

- Balances flexibility and readability.
- Works seamlessly with function arguments and fields.
- Aligns with Zirric’s existing annotation system.

---

## Acknowledgements

- **Prior Art**: Inspired by variadic functions in languages like C, JavaScript, and Rust.
- **Contributors**: [@vknabel](https://github.com/vknabel) for designing the proposal.

---

## Open Questions

1. Should `@Variadic` be restricted to specific contexts (e.g., only for constraints)?
2. How should the compiler handle type mismatches in variadic arguments?
3. Should there be a shorthand syntax for common cases (e.g., `@Requires[Countable, Iterable]`)?
4. Should there be spreads to pass arrays as variadic arguments (e.g., `@Requires(...myArray)`)?
5. Should `@Variadic` require an optional `min` or `max` parameter or a `range` to enforce bounds on the number of arguments?
