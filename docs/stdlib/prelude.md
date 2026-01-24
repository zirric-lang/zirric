---
title: Prelude
description: Core types, annotations, and primitives available in every Zirric program.
---

# Prelude

The `prelude` module defines Zirric's core types, builtins, and foundational annotations.

## Annotations

### Type

```zirric
annotation Type
```

Annotates a declaration to be of a given type. Instead of annotating declarations
with `@Type(SomeType)`, the shorthand of `@SomeType` can be used.

Fields:

- `@Type(AnyType) type` — The type of the annotation.

### Has

```zirric
annotation Has
```

Has requests passed values to have the given annotation type present. For example,
`@Has(Numeric)` requests that the passed value has the `@Numeric` annotation. Do
not annotate types with `@Has`, as it does not make sense there.

Fields:

- `@Type(AnnotationType) annotationType` — The required annotation type.

### Returns

```zirric
annotation Returns
```

Annotates a function declaration to return a value of the given type.

Fields:

- `@Type(AnyType) type`

### Default

```zirric
annotation Default
```

Transparently indicates the assumed default value of a parameter or field. Can be
used by tooling and libraries.

Fields:

- `@Type(AnyType) value` — The default value for a parameter.

### Doc

```zirric
annotation Doc
```

Provides access to the documentation string of a declaration.

Fields:

- `@String description` — The documentation string without leading comment markers
  and whitespace.

### Deprecated

```zirric
annotation Deprecated
```

Annotates a declaration as deprecated with a reason. IDEs and other tools can use
this information to warn users about deprecated declarations.

Fields:

- `@String @Default("without alternative") reason`

### Numeric

```zirric
annotation Numeric
```

A numeric value, either floating point or integer.

Members:

- `@Returns(Number) get()`

### Numeric (conversion)

```zirric
annotation Numeric
```

Annotates a declaration as numeric, providing a way to convert it to a number.

Members:

- `@Returns(Number) toNumber(@Has(Numeric) value)`

## Types

### Any

```zirric
extern type Any
```

Anything is a value of type `Any`.

### AnyType

```zirric
extern type AnyType
```

All types are of type `AnyType`.

### Annotation

```zirric
extern type Annotation
```

All annotations are of type `Annotation`.

### AnnotationType

```zirric
extern type AnnotationType
```

All annotation types are of type `AnnotationType`.

### Module

```zirric
extern type Module
```

All modules are of type `Module`.

### ModuleType

```zirric
extern type ModuleType
```

All module types are of type `ModuleType`.

### Array

```zirric
extern type Array
```

A finite list of values.

Members:

- `@Type(Int) length` — The length of the array.

### Bool

```zirric
extern type Bool
```

Represents boolean values like `True` and `False`. Typically used for
conditionals and flags.

Members:

- `toggle()` — Negates a boolean value.

### Char

```zirric
extern type Char
```

A single character from a string.

### Dict

```zirric
extern type Dict
```

An associative array of keys and their values.

Members:

- `@Type(Int) length` — The length of the dictionary.

### Func

```zirric
extern type Func
```

A callable function.

Members:

- `@Type(Int) arity` — The amount of function parameters to be passed.

### Float

```zirric
@Numeric({ f -> f })
extern type Float
```

A floating point number.

### Int

```zirric
@Numeric({ i -> i })
extern type Int
```

A whole integer number.

### String

```zirric
extern type String
```

No documentation.

Members:

- `@Type(Int) length`

### Void

```zirric
extern type Void
```

The type of the `void` value.

## Union

### Number

```zirric
union Number
```

No documentation.

Members:

- `Float`
- `Int`

## Values

### true

```zirric
@Bool
let true = 0 == 0
```

No documentation.

### false

```zirric
@Bool
let false = 0 != 0
```

No documentation.

### void

```zirric
@Type(Void)
extern let void
```

Represents the absence of a value.
