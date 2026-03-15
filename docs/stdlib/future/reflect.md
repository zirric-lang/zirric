---
title: Future.Reflect
description: Experimental reflection stubs for inspecting Zirric declarations.
---

# Future.Reflect

The `reflect` module provides proposal-era reflection stubs for fields,
attributes, and types.

## Attributes

### attr Name

```zirric
attr Name
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
- `@Array @ItemType(Attribute) attributes`

### data Attribute

```zirric
data Attribute
```

No documentation.

Fields:

- `@AnyAttribute attributeType`
- `@Array args`

## Functions

### fn typeOf

```zirric
@Returns(Type)
fn typeOf(@Any value)
```

No documentation.

### fn fieldsOf

```zirric
@Returns(Array(Field))
fn fieldsOf(@Type type)
```

No documentation.

### fn attribute

```zirric
@Returns(Result)
@OkType(AnyAttribute)
fn attribute(@Field field, @AttributeType attributeType)
```

No documentation.

### fn hasAttribute

```zirric
@Returns(Bool)
fn hasAttribute(@Field field, @AttributeType attributeType)
```

No documentation.
