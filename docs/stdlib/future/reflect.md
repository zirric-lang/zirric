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
attr Name {
  name: String
}
```

No documentation.

Fields:

- `name: String`

## Data

### data Field

```zirric
data Field {
  name: String
  type: AnyType
  attributes: [Attribute]
}
```

No documentation.

Fields:

- `name: String`
- `type: AnyType`
- `attributes: [Attribute]`

### data Attribute

```zirric
data Attribute {
  attributeType: AnyAttribute
  args: Array
}
```

No documentation.

Fields:

- `attributeType: AnyAttribute`
- `args: Array`

## Functions

### fn typeOf

```zirric
fn typeOf(value: Any) -> AnyType
```

No documentation.

### fn fieldsOf

```zirric
fn fieldsOf(typeValue: AnyType) -> [Field]
```

No documentation.

### fn attribute

```zirric
fn attribute(field: Field, attributeType: AttributeType) -> Result
```

No documentation.

### fn hasAttribute

```zirric
fn hasAttribute(field: Field, attributeType: AttributeType) -> Bool
```

No documentation.
