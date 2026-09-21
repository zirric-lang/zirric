---
title: JSON
description: Reading and writing JSON text, and per-field JSON overrides.
---

# Module `json`

> JSON as Zirric values, not as a tree of its own.

```zirric
import json
```

|            |                                                                                                                                                                                                          |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `json`                                                                                                                                                                                                   |
| **Source** | [`json/json.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/json/json.zirr), [`json/attributes.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/json/attributes.zirr) |

JSON maps onto Zirric's own values: an object is a [`Dict`](../prelude/index.md#dict) with `String` keys, an array is an [`Array`](../prelude/index.md#array), and `null` is `void`. A number is an [`Int`](../prelude/index.md#int) when it was written as one and a [`Float`](../prelude/index.md#float) otherwise, so `1` and `1.0` stay apart.

[`parse`](#parse) and [`format`](#format) move between text and that tree. To move between the tree and your own `data` types, hand this module to [`coding`](../coding/index.md): `coding.encodeWith(json, value)` consults the attributes below before falling back to `@coding`'s.

## Contents

- **Attributes** — [`Name`](#name), [`Ignore`](#ignore), [`Default`](#default), [`Decode`](#decode), [`Encode`](#encode)
- **Functions** — [`parse`](#parse), [`format`](#format), [`formatIndented`](#formatindented)

---

## Attributes

### `Name` {#name}

<small>`json/attributes.zirr:5`</small>

```zirric
attr Name {
	// The key to use.
	text
}
```

The JSON key a field is read and written under, overriding `@coding.Name` for JSON only. `coding` finds these by name, so declaring them here is all it takes for them to win.

#### Members

| Member | Description     |
| ------ | --------------- |
| `text` | The key to use. |

---

### `Ignore` {#ignore}

<small>`json/attributes.zirr:11`</small>

```zirric
attr Ignore {}
```

Marks a field as one that does not travel as JSON, overriding `@coding.Ignore`.

---

### `Default` {#default}

<small>`json/attributes.zirr:14`</small>

```zirric
attr Default {
	// The value to use.
	value
}
```

The value a field takes when its JSON key is absent, overriding `@coding.Default`.

#### Members

| Member  | Description       |
| ------- | ----------------- |
| `value` | The value to use. |

---

### `Decode` {#decode}

<small>`json/attributes.zirr:20`</small>

```zirric
attr Decode {
	// `fn(raw: Any) -> Result`.
	parse
}
```

Decodes a field from its raw JSON tree, overriding `@coding.Decode`.

#### Members

| Member  | Description               |
| ------- | ------------------------- |
| `parse` | `fn(raw: Any) -> Result`. |

---

### `Encode` {#encode}

<small>`json/attributes.zirr:26`</small>

```zirric
attr Encode {
	// `fn(value: Any) -> Result`.
	format
}
```

Encodes a field to a JSON tree, overriding `@coding.Encode`.

#### Members

| Member   | Description                 |
| -------- | --------------------------- |
| `format` | `fn(value: Any) -> Result`. |

---

## Functions

### `parse` {#parse}

<small>`json/json.zirr:7`</small>

```zirric
extern fn parse(text: String) -> Result
```

Reads JSON text, failing with Err when the text is not one JSON value.

---

### `format` {#format}

<small>`json/json.zirr:11`</small>

```zirric
extern fn format(value: Any) -> Result
```

Renders a value as JSON text, failing with Err when it holds something JSON has no form for. Object keys come out sorted, so the same value always produces the same text.

---

### `formatIndented` {#formatindented}

<small>`json/json.zirr:14`</small>

```zirric
extern fn formatIndented(value: Any, indent: String) -> Result
```

As format, but spread over lines with each level prefixed by indent.

---

## See also

- [`coding`](../coding/index.md) — encoding and decoding your own types.
- [`yaml`](../yaml/index.md) — the same shape for YAML.
