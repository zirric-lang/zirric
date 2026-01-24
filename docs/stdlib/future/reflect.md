---
title: Future.Reflect
description: Experimental reflection stubs for inspecting Zirric declarations.
---

# Future.Reflect

The `reflect` module provides proposal-era reflection stubs for fields,
annotations, and types.

## Annotations

### Name

```zirric
annotation Name
```

No documentation.

Fields:

- `@String name`

## Data

### Field

```zirric
data Field
```

No documentation.

Fields:

- `@String name`
- `@Type(AnyType) type`
- `@Array @ItemType(Annotation) annotations`

### Annotation

```zirric
data Annotation
```

No documentation.

Fields:

- `@AnyAnnotation annotationType`
- `@Array args`

## Functions

### typeOf

```zirric
@Returns(Type)
func typeOf(@Any value)
```

No documentation.

### fieldsOf

```zirric
@Returns(Array(Field))
func fieldsOf(@Type type)
```

No documentation.

### annotation

```zirric
@Returns(Result)
@OkType(AnyAnnotation)
func annotation(@Field field, @AnnotationType annotationType)
```

No documentation.

### hasAnnotation

```zirric
@Returns(Bool)
func hasAnnotation(@Field field, @AnnotationType annotationType)
```

No documentation.
