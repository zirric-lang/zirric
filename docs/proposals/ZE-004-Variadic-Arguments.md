---
title: ZE-004 - Variadic Attributes
description: Introduce variadic attributes to allow attributes to accept a variable number of arguments.
---

# Variadic Attributes

::: callout draft Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design. Features described here may not be implemented as described and cannot be used right now.
:::

::: callout warning Outdated
While this proposal has not been rejected, it is currently outdated and requires an overhaul to reflect the latest design decisions.

- [ ] Reflect latest syntax changes
- [ ] Reflect latest stdlib changes
- [ ] Attributes are no longer used for types
- [ ] Evaluate the use cases of this proposal
      :::

## Introduction

This proposal introduces **variadic attributes** to Zirric, enabling attributes and functions to accept a variable number of arguments. This feature addresses the need for multi-instance constraints (e.g., `@Requires(Countable, Iterable)`) while maintaining a clean and expressive syntax.

Variadic attributes are particularly useful for:

- Representing requirements or constraints on fields, parameters, or types.
- Improving readability and reducing verbosity compared to alternatives like `@HasAll([Countable, Iterable])`.

## Motivation

Zirric currently lacks a concise way to specify multiple constraints or requirements in attributes. For example, to enforce that a parameter must satisfy both `Countable` and `Iterable`, users must use workarounds like `@HasAll([Countable, Iterable])`, which is less readable and feels unnatural.

This could be useful in scenarios such as:

1. **Type Constraints**: Annotating parameters or fields to require multiple traits or interfaces.
   ```zirric
   fn process(@Requires(Countable, Iterable) data) {}
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

Introduce a `@Variadic` attribute to mark attribute fields as accepting a variable number of arguments. When used, the field's type is treated as an `Array` of the specified element type.

#### Example: Defining a Variadic Attribute

```zirric
attr Requires {
  @Variadic()
  @ArgType(Type)  // Accepts types or type-like attributes
  constraints
}
```

#### Example: Using a Variadic Attribute

```zirric
// Function with variadic constraints
fn process(@Requires(Countable, Iterable) data) {
  // `data` must satisfy both `Countable` and `Iterable`
}

// Valid calls
process([1, 2, 3])  // Array of Int (assuming Int is Countable and Iterable)
process([])         // Empty array

// Invalid call (if String doesn't satisfy Iterable)
process("hello")    // Compiler error: String is not Iterable
```

### Behavior

- **Type Handling**: Variadic arguments are always treated as an `Array` of the specified element type (e.g., `@ArgType(Type)`).
- **Empty Lists**: Supported for optional constraints (e.g., `@Requires()`).
- **Type Safety**: The compiler enforces that all provided arguments match the element type (e.g., `Type`).

---

## Detailed Design

### Attribute Definition

- The `@Variadic` attribute marks a field within another attribute as variadic.
- The field's type is implicitly an `Array` of the type specified by `@ArgType`.
  ```zirric
  attr VariadicExample {
    @Variadic()
    @ArgType(Int)  // Accepts variadic Int arguments
    numbers
  }
  ```

### Usage in Functions

- Variadic attributes can be applied to function parameters, fields, or types.
  ```zirric
  fn example(@VariadicExample(1, 2, 3) nums) {}
  ```
- Functions themselves can also have a variadic parameter syntax similar to attributes.
  ```zirric
  fn sum(@Variadic() @ArgType(Int) numbers) {
    // `numbers` is an Array of Int
  }
  sum(1, 2, 3, 4)  // Valid call
  ```

### Compiler and Tooling Support

- **Parsing**: The compiler treats variadic attribute fields as arrays during parsing.
- **Type Checking**: Ensures all arguments match the element type specified by `@ArgType`.
- **Error Messages**: Provides clear errors for type mismatches or invalid usage.

### Edge Cases

- **Empty Lists**: `@Requires()` is valid and results in an empty array of constraints.
- **Single Argument**: `@Requires(Countable)` is treated as an array with one element.
- **Type Mismatches**: The compiler rejects arguments that don't match `@ArgType`.

## Changes to the Standard Library

### New Attributes

1. **`@Variadic`**:
   - Marks a field as accepting a variable number of arguments.
   - Only valid within attribute definitions.

2. **`@ArgType`**:
   - Specifies the type of elements in a variadic field (e.g., `@ArgType(Type)`).
   - Required for variadic fields to enforce type safety.

### **Example Additions**

```zirric
// Standard library addition
attr Variadic {}
attr ArgType {
  type
}
```

## Alternatives Considered

1. **`@HasAll([...])` Syntax**:
   - **Pros**: No new syntax required.
   - **Cons**: Less readable and feels like a workaround.

2. **Dedicated Grouping Syntax (e.g., `@Requires{Countable, Iterable}`)**:
   - **Pros**: Clean and intuitive.
   - **Cons**: Doesn't translate to function arguments and requires new grammar rules.

3. **Multiple Attributes (e.g., `@Requires(Countable) @Requires(Iterable)`)**:
   - **Pros**: Simple and explicit.
   - **Cons**: Verbose and ambiguous for single-instance attributes.

**Why `@Variadic` Was Chosen**:

- Balances flexibility and readability.
- Works seamlessly with function arguments and fields.
- Aligns with Zirric's existing attribute system.

## Acknowledgements

- **Prior Art**: Inspired by variadic functions in languages like C, JavaScript, and Rust.
- **Contributors**: [@vknabel](https://github.com/vknabel) for designing the proposal.

## Open Questions

1. Should there be spreads to pass arrays as variadic arguments (e.g., `@Requires(...myArray)`)?
2. Should `@Variadic` require an optional `min` or `max` parameter or a `range` to enforce bounds on the number of arguments?
