---
title: Future.Reflect
description: The original reflection sketch, superseded by the reflect module.
---

# Module `future.reflect`

> An early sketch, kept for reference.

```zirric
import future.reflect
```

|            |                                                                                                                   |
| ---------- | ----------------------------------------------------------------------------------------------------------------- |
| **Module** | `future.reflect`                                                                                                  |
| **Source** | [`future/reflect/stub.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/future/reflect/stub.zirr) |

This is the reflection surface as it was first sketched, with empty bodies standing in for behaviour that did not exist yet.

It has since been implemented for real: use [`reflect`](../../reflect/index.md), which covers everything here and more. The sketch is kept because proposals and their early implementations are a record of their time.

## Contents

- **Attributes** — [`Name`](#name)
- **Data** — [`Field`](#field), [`Attribute`](#attribute)
- **Functions** — [`typeOf`](#typeof), [`fieldsOf`](#fieldsof), [`attribute`](#attribute-fn), [`hasAttribute`](#hasattribute)

---

## Attributes

### `Name` {#name}

<small>`future/reflect/stub.zirr:14`</small>

```zirric
attr Name {
	name: String
}
```

The declared name of a field.

#### Members

| Member | Signature      | Description                                  |
| ------ | -------------- | -------------------------------------------- |
| `name` | `name: String` | The name to use instead of the declared one. |

---

## Data

### `Field` {#field}

<small>`future/reflect/stub.zirr:3`</small>

```zirric
data Field {
	name: String
	type: AnyType
	attributes: [Attribute]
}
```

A field of a data type, as the sketch modelled it.

#### Fields

| Field        | Signature                 | Description                           |
| ------------ | ------------------------- | ------------------------------------- |
| `name`       | `name: String`            | The declared name.                    |
| `type`       | `type: AnyType`           | The declared type.                    |
| `attributes` | `attributes: [Attribute]` | Every attribute written on the field. |

---

### `Attribute` {#attribute}

<small>`future/reflect/stub.zirr:9`</small>

```zirric
data Attribute {
	attributeType: AnyAttribute
	args: [String: Any]
}
```

An attribute applied to a declaration, as the sketch modelled it.

#### Fields

| Field           | Signature                     | Description                                 |
| --------------- | ----------------------------- | ------------------------------------------- |
| `attributeType` | `attributeType: AnyAttribute` | Which attribute this is.                    |
| `args`          | `args: [String: Any]`         | The arguments it was applied with, by name. |

---

## Functions

### `typeOf` {#typeof}

<small>`future/reflect/stub.zirr:18`</small>

```zirric
fn typeOf(value: Any) -> AnyType {}
```

The type of a value. Implemented for real as [`reflect.typeOf`](../../reflect/index.md#typeof), which returns an `Option`.

---

### `fieldsOf` {#fieldsof}

<small>`future/reflect/stub.zirr:20`</small>

```zirric
fn fieldsOf(typeValue: AnyType) -> [Field] {}
```

The fields of a type. Implemented for real as [`reflect.fieldsOf`](../../reflect/index.md#fieldsof).

---

### `attribute` {#attribute-fn}

<small>`future/reflect/stub.zirr:22`</small>

```zirric
fn attribute(field: Field, attributeType: AttributeType) -> Result {}
```

The attribute of the given type applied to a field, if any.

---

### `hasAttribute` {#hasattribute}

<small>`future/reflect/stub.zirr:24`</small>

```zirric
fn hasAttribute(field: Field, attributeType: AttributeType) -> Bool {}
```

Whether the given attribute is applied to a field.

---

## See also

- [`reflect`](../../reflect/index.md) — the module that replaced this.
- [`future`](../index.md) — what this module is part of.
