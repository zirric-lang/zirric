---
title: Future.Reflect
description: The original reflection sketch, superseded by the reflect module.
---

# Module `future.reflect`

```zirric
import future.reflect
```

|            |                                                                                                                                                                                                                                                    |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `future.reflect`                                                                                                                                                                                                                                   |
| **Source** | [`future/reflect/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/future/reflect/module-docs.zirr), [`future/reflect/stub.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/future/reflect/stub.zirr) |

> An early sketch, kept for reference.

This is the reflection surface as it was first sketched, with empty bodies standing in for behaviour that did not exist yet.

It has since been implemented for real: use [`reflect`](../../reflect/index.md), which covers everything here and more. The sketch is kept because proposals and their early implementations are a record of their time.

## Contents

- **Data** — [`Attribute`](#attribute-data), [`Field`](#field)
- **Attributes** — [`Name`](#name)
- **Functions** — [`attribute`](#attribute-fn), [`fieldsOf`](#fieldsof), [`hasAttribute`](#hasattribute), [`typeOf`](#typeof)

---

## Data

### `Attribute` {#attribute-data}

<small>`future/reflect/stub.zirr:9`</small>

```zirric
data Attribute {
	attributeType: AnyAttribute
	args: [String: Any]
}
```

#### Fields

| Field           | Description |
| --------------- | ----------- |
| `attributeType` |             |
| `args`          |             |

---

### `Field` {#field}

<small>`future/reflect/stub.zirr:3`</small>

```zirric
data Field {
	name: String
	type: AnyType
	attributes: [Attribute]
}
```

#### Fields

| Field        | Description |
| ------------ | ----------- |
| `name`       |             |
| `type`       |             |
| `attributes` |             |

---

## Attributes

### `Name` {#name}

<small>`future/reflect/stub.zirr:14`</small>

```zirric
attr Name {
	name: String
}
```

#### Fields

| Field  | Description |
| ------ | ----------- |
| `name` |             |

---

## Functions

### `attribute` {#attribute-fn}

<small>`future/reflect/stub.zirr:22`</small>

```zirric
fn attribute(field: Field, attributeType: AttributeType) -> Result
```

---

### `fieldsOf` {#fieldsof}

<small>`future/reflect/stub.zirr:20`</small>

```zirric
fn fieldsOf(typeValue: AnyType) -> [Field]
```

---

### `hasAttribute` {#hasattribute}

<small>`future/reflect/stub.zirr:24`</small>

```zirric
fn hasAttribute(field: Field, attributeType: AttributeType) -> Bool
```

---

### `typeOf` {#typeof}

<small>`future/reflect/stub.zirr:18`</small>

```zirric
fn typeOf(value: Any) -> AnyType
```
