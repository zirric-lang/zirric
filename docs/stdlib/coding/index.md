---
title: Coding
description: Encoding and decoding your own types through a native value tree.
---

# Module `coding`

```zirric
import coding
```

|            |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Module** | `coding`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| **Source** | [`coding/attributes.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/coding/attributes.zirr), [`coding/coding.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/coding/coding.zirr), [`coding/decode.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/coding/decode.zirr), [`coding/encode.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/coding/encode.zirr), [`coding/helpers.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/coding/helpers.zirr), [`coding/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/coding/module-docs.zirr) |

> Between your `data` types and a tree any format can carry.

`coding` converts between your own declarations and a native tree of [`prelude.Dict`](../prelude/index.md#dict), [`prelude.Array`](../prelude/index.md#array), `String`, `Int`, `Float`, `Bool` and `void`. It never touches text: [`json`](../json/index.md) and [`yaml`](../yaml/index.md) take that tree the rest of the way.

It works through [`reflect`](../reflect/index.md). Fields are read off the type, and the attributes written on them decide the details — the key a field travels under, whether it travels at all, what it becomes when absent, and how it is converted when the general rules are not what you want. A type carrying none of them encodes under the names it declares.

A format can override any of these for itself. [`encodeWith`](#encodewith) and [`decodeWith`](#decodewith) take a format module and consult its own attributes first, which is why [`@json.Name`](../json/index.md#name) beats [`@coding.Name`](#name) when encoding as JSON.

## Dependencies

- [`dicts`](../dicts/index.md)
- [`reflect`](../reflect/index.md)
- [`results`](../results/index.md)

---

## Contents

- **Attributes** — [`Decode`](#decode-attr), [`Default`](#default), [`Encode`](#encode-attr), [`Ignore`](#ignore), [`Name`](#name)
- **Functions** — [`decode`](#decode-fn), [`decodeWith`](#decodewith), [`encode`](#encode-fn), [`encodeWith`](#encodewith)

---

## Attributes

### `Decode` {#decode-attr}

<small>`coding/coding.zirr:21`</small>

```zirric
attr Decode {
	parse
}
```

Decodes a field from the raw tree found under its key, instead of by the general rules.

#### Fields

| Field   | Description               |
| ------- | ------------------------- |
| `parse` | `fn(raw: Any) -> Result`. |

---

### `Default` {#default}

<small>`coding/coding.zirr:15`</small>

```zirric
attr Default {
	value
}
```

The value a field takes when its key is absent.

#### Fields

| Field   | Description       |
| ------- | ----------------- |
| `value` | The value to use. |

---

### `Encode` {#encode-attr}

<small>`coding/coding.zirr:27`</small>

```zirric
attr Encode {
	format
}
```

Encodes a field to a tree, instead of by the general rules.

#### Fields

| Field    | Description                 |
| -------- | --------------------------- |
| `format` | `fn(value: Any) -> Result`. |

---

### `Ignore` {#ignore}

<small>`coding/coding.zirr:12`</small>

```zirric
attr Ignore
```

Marks a field as one that does not travel.
An ignored field is skipped when encoding, and takes its [`@Default`](#default) — or `void` — when decoding.

---

### `Name` {#name}

<small>`coding/coding.zirr:5`</small>

```zirric
attr Name {
	text
}
```

The key a field is read and written under, instead of its declared name.
Every attribute here is an override, so a type carrying none encodes under the names it declares.

#### Fields

| Field  | Description     |
| ------ | --------------- |
| `text` | The key to use. |

---

## Functions

### `decode` {#decode-fn}

<small>`coding/coding.zirr:38`</small>

```zirric
fn decode(t: Any, tree: Any) -> Result
```

A native tree as a value of type `t`.

---

### `decodeWith` {#decodewith}

<small>`coding/coding.zirr:48`</small>

```zirric
fn decodeWith(format: AnyModule, t: Any, tree: Any) -> Result
```

As [`decode`](#decode-fn), but consulting the format module's own attributes first.

---

### `encode` {#encode-fn}

<small>`coding/coding.zirr:33`</small>

```zirric
fn encode(value: Any) -> Result
```

A value as a native tree of `Dict`, `Array`, `String`, `Int`, `Float`, `Bool` and `void`.

---

### `encodeWith` {#encodewith}

<small>`coding/coding.zirr:43`</small>

```zirric
fn encodeWith(format: AnyModule, value: Any) -> Result
```

As [`encode`](#encode-fn), but consulting the format module's own attributes first, so [`@json.Name`](../json/index.md#name) overrides [`@coding.Name`](#name).
