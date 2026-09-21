---
title: YAML
description: Reading and writing YAML text, and per-field YAML overrides.
---

# Module `yaml`

> YAML over the same value tree as JSON.

```zirric
import yaml
```

|            |                                                                                                                                                                                                          |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `yaml`                                                                                                                                                                                                   |
| **Source** | [`yaml/yaml.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/yaml/yaml.zirr), [`yaml/attributes.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/yaml/attributes.zirr) |

`yaml` reads and writes the same native tree [`json`](../json/index.md) does — [`Dict`](../prelude/index.md#dict), [`Array`](../prelude/index.md#array), `String`, `Int`, `Float`, `Bool` and `void` — so the two formats are interchangeable as far as your own types are concerned.

[`parseAll`](#parseall) is the one addition: a YAML file may hold several documents separated by `---`, and it returns all of them.

Pass this module to [`coding`](../coding/index.md) to encode and decode your own `data` types through it; the attributes below override `@coding`'s for YAML only.

## Contents

- **Attributes** — [`Name`](#name), [`Ignore`](#ignore), [`Default`](#default), [`Decode`](#decode), [`Encode`](#encode)
- **Functions** — [`parse`](#parse), [`parseAll`](#parseall), [`format`](#format), [`formatIndented`](#formatindented)

---

## Attributes

### `Name` {#name}

<small>`yaml/attributes.zirr:5`</small>

```zirric
attr Name {
	// The key to use.
	text
}
```

The YAML key a field is read and written under, overriding `@coding.Name` for YAML only. `coding` finds these by name, so declaring them here is all it takes for them to win.

#### Members

| Member | Description     |
| ------ | --------------- |
| `text` | The key to use. |

---

### `Ignore` {#ignore}

<small>`yaml/attributes.zirr:11`</small>

```zirric
attr Ignore {}
```

Marks a field as one that does not travel as YAML, overriding `@coding.Ignore`.

---

### `Default` {#default}

<small>`yaml/attributes.zirr:14`</small>

```zirric
attr Default {
	// The value to use.
	value
}
```

The value a field takes when its YAML key is absent, overriding `@coding.Default`.

#### Members

| Member  | Description       |
| ------- | ----------------- |
| `value` | The value to use. |

---

### `Decode` {#decode}

<small>`yaml/attributes.zirr:20`</small>

```zirric
attr Decode {
	// `fn(raw: Any) -> Result`.
	parse
}
```

Decodes a field from its raw YAML tree, overriding `@coding.Decode`.

#### Members

| Member  | Description               |
| ------- | ------------------------- |
| `parse` | `fn(raw: Any) -> Result`. |

---

### `Encode` {#encode}

<small>`yaml/attributes.zirr:26`</small>

```zirric
attr Encode {
	// `fn(value: Any) -> Result`.
	format
}
```

Encodes a field to a YAML tree, overriding `@coding.Encode`.

#### Members

| Member   | Description                 |
| -------- | --------------------------- |
| `format` | `fn(value: Any) -> Result`. |

---

## Functions

### `parse` {#parse}

<small>`yaml/yaml.zirr:5`</small>

```zirric
extern fn parse(text: String) -> Result
```

Reads one YAML document into the native tree: a mapping is a Dict, a sequence an Array, and null is void. A stream holding several documents is an `Err`, since that is not one value; use `parseAll` for those.

---

### `parseAll` {#parseall}

<small>`yaml/yaml.zirr:8`</small>

```zirric
extern fn parseAll(text: String) -> Result
```

Reads every document in a stream, in order.

---

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

As `format`, indented by a given number of spaces.

---

## See also

- [`coding`](../coding/index.md) — encoding and decoding your own types.
- [`json`](../json/index.md) — the same shape for JSON.
