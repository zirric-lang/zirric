---
title: YAML
description: Reading and writing YAML text, and per-field YAML overrides.
---

# Module `yaml`

```zirric
import yaml
```

|            |                                                                                                                                                                                                                                                                                                                       |
| ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `yaml`                                                                                                                                                                                                                                                                                                                |
| **Source** | [`yaml/attributes.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/yaml/attributes.zirr), [`yaml/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/yaml/module-docs.zirr), [`yaml/yaml.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/yaml/yaml.zirr) |

> YAML over the same value tree as JSON.

`yaml` reads and writes the same native tree [`json`](../json/index.md) does — [`prelude.Dict`](../prelude/index.md#dict), [`prelude.Array`](../prelude/index.md#array), `String`, `Int`, `Float`, `Bool` and `void` — so the two formats are interchangeable as far as your own types are concerned.

[`parseAll`](#parseall) is the one addition: a YAML file may hold several documents separated by `---`, and it returns all of them.

Pass this module to [`coding`](../coding/index.md) to encode and decode your own `data` types through it; the attributes below override [`@coding`](../coding/index.md)'s for YAML only.

## Contents

- **Attributes** — [`Decode`](#decode), [`Default`](#default), [`Encode`](#encode), [`Ignore`](#ignore), [`Name`](#name)
- **Functions** — [`format`](#format), [`formatIndented`](#formatindented), [`parse`](#parse), [`parseAll`](#parseall)

---

## Attributes

### `Decode` {#decode}

<small>`yaml/attributes.zirr:20`</small>

```zirric
attr Decode {
	parse
}
```

Decodes a field from its raw YAML tree, overriding [`@coding.Decode`](../coding/index.md#decode-attr).

#### Fields

| Field   | Description               |
| ------- | ------------------------- |
| `parse` | `fn(raw: Any) -> Result`. |

---

### `Default` {#default}

<small>`yaml/attributes.zirr:14`</small>

```zirric
attr Default {
	value
}
```

The value a field takes when its YAML key is absent, overriding [`@coding.Default`](../coding/index.md#default).

#### Fields

| Field   | Description       |
| ------- | ----------------- |
| `value` | The value to use. |

---

### `Encode` {#encode}

<small>`yaml/attributes.zirr:26`</small>

```zirric
attr Encode {
	format
}
```

Encodes a field to a YAML tree, overriding [`@coding.Encode`](../coding/index.md#encode-attr).

#### Fields

| Field    | Description                 |
| -------- | --------------------------- |
| `format` | `fn(value: Any) -> Result`. |

---

### `Ignore` {#ignore}

<small>`yaml/attributes.zirr:11`</small>

```zirric
attr Ignore
```

Marks a field as one that does not travel as YAML, overriding [`@coding.Ignore`](../coding/index.md#ignore).

---

### `Name` {#name}

<small>`yaml/attributes.zirr:5`</small>

```zirric
attr Name {
	text
}
```

The YAML key a field is read and written under, overriding [`@coding.Name`](../coding/index.md#name) for YAML only.
[`coding`](../coding/index.md) finds these by name, so declaring them here is all it takes for them to win.

#### Fields

| Field  | Description     |
| ------ | --------------- |
| `text` | The key to use. |

---

## Functions

### `format` {#format}

<small>`yaml/yaml.zirr:11`</small>

```zirric
extern fn format(value: Any) -> Result
```

Renders a native tree as YAML text, indented by two spaces.

---

### `formatIndented` {#formatindented}

<small>`yaml/yaml.zirr:14`</small>

```zirric
extern fn formatIndented(value: Any, indent: Int) -> Result
```

As [`format`](#format), indented by a given number of spaces.

---

### `parse` {#parse}

<small>`yaml/yaml.zirr:5`</small>

```zirric
extern fn parse(text: String) -> Result
```

Reads one YAML document into the native tree: a mapping is a Dict, a sequence an Array, and null is void.
A stream holding several documents is an `Err`, since that is not one value; use [`parseAll`](#parseall) for those.

---

### `parseAll` {#parseall}

<small>`yaml/yaml.zirr:8`</small>

```zirric
extern fn parseAll(text: String) -> Result
```

Reads every document in a stream, in order.
