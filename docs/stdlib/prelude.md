---
title: Prelude
description: Core types, annotations, and helpers available in every Zirric program.
---

# Prelude

The `prelude` module defines Zirric's core types and foundational annotations.

## Annotations

### Countable

```zirric
annotation Countable
```

No documentation.

Members:

- `@Returns(Int) length(@Has(Countable) value)`

### Default

```zirric
annotation Default
```

Transparently indicates the assumed default value of a parameter or field.

Fields:

- `@Type(AnyType) value` — The default value for a parameter.

### Deprecated

```zirric
annotation Deprecated
```

Annotates a declaration as deprecated with a reason.

Fields:

- `@String @Default("without alternative") reason`

### Doc

```zirric
annotation Doc
```

Provides access to the documentation string of a declaration.

Fields:

- `@String description` — The documentation string without leading comment
  markers and whitespace.

### ErrType

```zirric
annotation ErrType
```

No documentation.

Fields:

- `@Type(AnyType) type`

### Error

```zirric
annotation Error
```

No documentation.

Members:

- `@Returns(String) toString(@Has(Error) err)`

### Has

```zirric
annotation Has
```

Requests passed values to have the given annotation type present. Do not
annotate types with `@Has`.

Fields:

- `@Type(AnnotationType) annotationType` — The required annotation type.

### Iterable

```zirric
annotation Iterable
```

No documentation.

Members:

- `iterate(@Has(Iterable) value, @Func yield)`

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

### OkType

```zirric
annotation OkType
```

No documentation.

Fields:

- `@Type(AnyType) type`

### Returns

```zirric
annotation Returns
```

Annotates a function declaration to return a value of the given type.

Fields:

- `@Type(AnyType) type`

### Type

```zirric
annotation Type
```

Annotates a declaration to be of a given type. Instead of annotating
`@Type(SomeType)`, the shorthand `@SomeType` can be used.

Fields:

- `@Type(AnyType) type` — The type of the annotation.

## Enum

### Number

```zirric
enum Number
```

No documentation.

Cases:

- `Float`
- `Int`

### Optional

```zirric
@json.Inline()
enum Optional
```

No documentation.

Cases:

- `@json.Type(json.Null) None`
- `@json.Inline Some { value }`

### Result

```zirric
enum Result
```

No documentation.

Cases:

- `Ok { @Any value }`
- `Err { @Has(Error) error }`

## Extern Types

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

### Array

```zirric
extern type Array
```

A finite list of values.

Fields:

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

Fields:

- `@Type(Int) length` — The length of the dictionary.

### Float

```zirric
@Numeric({ f -> f })
extern type Float
```

A floating point number.

### Func

```zirric
extern type Func
```

A callable function.

Fields:

- `@Type(Int) arity` — The amount of function parameters to be passed.

### Int

```zirric
@Numeric({ i -> i })
extern type Int
```

A whole integer number.

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

### Null

```zirric
extern type Null
```

The type of the `null` value.

### String

```zirric
extern type String
```

No documentation.

Fields:

- `@Type(Int) length`

## Data

### Range

```zirric
data Range
```

No documentation.

Fields:

- `@Int start`
- `@Int end`

## Extern Constants

### null

```zirric
@Type(Null)
extern let null
```

Represents the absence of a value.

## Variables

### false

```zirric
@Bool
let false = 0 != 0
```

No documentation.

### true

```zirric
@Bool
let true = 0 == 0
```

No documentation.
