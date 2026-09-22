---
title: JSON
description: Reading and writing JSON text, and per-field JSON overrides.
---

# Module `json`

```zirric
import json
```

|            |                                                                                                                                                                                                                                                                                                                       |
| ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `json`                                                                                                                                                                                                                                                                                                                |
| **Source** | [`json/attributes.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/json/attributes.zirr), [`json/json.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/json/json.zirr), [`json/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/json/module-docs.zirr) |

> JSON as Zirric values, not as a tree of its own.

JSON maps onto Zirric's own values: an object is a [`prelude.Dict`](../prelude/index.md#dict) with `String` keys, an array is an [`prelude.Array`](../prelude/index.md#array), and `null` is `void`. A number is an [`prelude.Int`](../prelude/index.md#int) when it was written as one and a [`prelude.Float`](../prelude/index.md#float) otherwise, so `1` and `1.0` stay apart.

[`parse`](#parse) and [`format`](#format) move between text and that tree. To move between the tree and your own `data` types, hand this module to [`coding`](../coding/index.md): `coding.encodeWith(json, value)` consults the attributes below before falling back to [`@coding`](../coding/index.md)'s.

## Contents

- **Attributes** — [`Decode`](#decode), [`Default`](#default), [`Encode`](#encode), [`Ignore`](#ignore), [`Name`](#name)
- **Functions** — [`format`](#format), [`formatIndented`](#formatindented), [`parse`](#parse)

---

## Attributes

### `Decode` {#decode}

<small>`json/attributes.zirr:20`</small>

```zirric
attr Decode {
	parse
}
```

Decodes a field from its raw JSON tree, overriding [`@coding.Decode`](../coding/index.md#decode-attr).

#### Fields

| Field   | Description               |
| ------- | ------------------------- |
| `parse` | `fn(raw: Any) -> Result`. |

---

### `Default` {#default}

<small>`json/attributes.zirr:14`</small>

```zirric
attr Default {
	value
}
```

The value a field takes when its JSON key is absent, overriding [`@coding.Default`](../coding/index.md#default).

#### Fields

| Field   | Description       |
| ------- | ----------------- |
| `value` | The value to use. |

---

### `Encode` {#encode}

<small>`json/attributes.zirr:26`</small>

```zirric
attr Encode {
	format
}
```

Encodes a field to a JSON tree, overriding [`@coding.Encode`](../coding/index.md#encode-attr).

#### Fields

| Field    | Description                 |
| -------- | --------------------------- |
| `format` | `fn(value: Any) -> Result`. |

---

### `Ignore` {#ignore}

<small>`json/attributes.zirr:11`</small>

```zirric
attr Ignore
```

Marks a field as one that does not travel as JSON, overriding [`@coding.Ignore`](../coding/index.md#ignore).

---

### `Name` {#name}

<small>`json/attributes.zirr:5`</small>

```zirric
attr Name {
	text
}
```

The JSON key a field is read and written under, overriding [`@coding.Name`](../coding/index.md#name) for JSON only.
[`coding`](../coding/index.md) finds these by name, so declaring them here is all it takes for them to win.

#### Fields

| Field  | Description     |
| ------ | --------------- |
| `text` | The key to use. |

---

## Functions

### `format` {#format}

<small>`json/json.zirr:11`</small>

```zirric
extern fn format(value: Any) -> Result
```

Renders a value as JSON text, failing with Err when it holds something JSON has no form for.
Object keys come out sorted, so the same value always produces the same text.

---

### `formatIndented` {#formatindented}

<small>`json/json.zirr:14`</small>

```zirric
extern fn formatIndented(value: Any, indent: String) -> Result
```

As format, but spread over lines with each level prefixed by indent.

---

### `parse` {#parse}

<small>`json/json.zirr:7`</small>

```zirric
extern fn parse(text: String) -> Result
```

Reads JSON text, failing with Err when the text is not one JSON value.
