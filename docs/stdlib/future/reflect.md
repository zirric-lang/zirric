---
title: Future.Reflect
description: Experimental reflection stubs for inspecting Zirric declarations.
---

# Future.Reflect

The `reflect` module provides proposal-era reflection stubs for fields,
annotations, and types.

## Annotations

### annotation Name

```zirric
annotation Name
```

No documentation.

Fields:

- `@String name`

## Data

### data Field

```zirric
data Field
```

No documentation.

Fields:

- `@String name`
- `@Type(AnyType) type`
- `@Array @ItemType(Annotation) annotations`

### data Annotation

```zirric
data Annotation
```

No documentation.

Fields:

- `@AnyAnnotation annotationType`
- `@Array args`

## Functions

### func typeOf

```zirric
@Returns(Type)
func typeOf(@Any value)
```

No documentation.

### func fieldsOf

```zirric
@Returns(Array(Field))
func fieldsOf(@Type type)
```

No documentation.

### func annotation

```zirric
@Returns(Result)
@OkType(AnyAnnotation)
func annotation(@Field field, @AnnotationType annotationType)
```

No documentation.

### func hasAnnotation

```zirric
@Returns(Bool)
func hasAnnotation(@Field field, @AnnotationType annotationType)
```

No documentation.
