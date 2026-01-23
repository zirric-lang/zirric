---
title: Reflect
description: Reflection types and helpers for inspecting Zirric declarations.
---

# Reflect

The `reflect` module exposes metadata about types, fields, and annotations.

## Annotations

### Name

```zirric
annotation Name
```

No documentation.

Fields:

- `@String name`

## Data

### Annotation

```zirric
data Annotation
```

No documentation.

Fields:

- `@AnyAnnotation annotationType`
- `@Array args`

### Field

```zirric
data Field
```

No documentation.

Fields:

- `@String name`
- `@Type(AnyType) type`
- `@Array @ItemType(Annotation) annotations`

## Functions

### annotation

```zirric
@Returns(Result)
@OkType(AnyAnnotation)
func annotation(@Field field, @AnnotationType annotationType)
```

No documentation.

### fieldsOf

```zirric
@Returns(Array(Field))
func fieldsOf(@Type type)
```

No documentation.

### hasAnnotation

```zirric
@Returns(Bool)
func hasAnnotation(@Field field, @AnnotationType annotationType)
```

No documentation.

### typeOf

```zirric
@Returns(Type)
func typeOf(@Any value)
```

No documentation.
