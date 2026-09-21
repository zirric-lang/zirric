---
title: Coding
description: Encoding and decoding your own types through a native value tree.
---

# Module `coding`

> Between your `data` types and a tree any format can carry.

```zirric
import coding
```

|            |                                                                                                       |
| ---------- | ----------------------------------------------------------------------------------------------------- |
| **Module** | `coding`                                                                                              |
| **Source** | [`coding/coding.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/coding/coding.zirr) |

`coding` converts between your own declarations and a native tree of [`Dict`](../prelude/index.md#dict), [`Array`](../prelude/index.md#array), `String`, `Int`, `Float`, `Bool` and `void`. It never touches text: [`json`](../json/index.md) and [`yaml`](../yaml/index.md) take that tree the rest of the way.

It works by [reflection](../reflect/index.md). Fields are read off the type, and the attributes written on them decide the details — the key a field travels under, whether it travels at all, what it becomes when absent, and how it is converted when the general rules are not what you want. A type carrying none of them encodes under the names it declares.

A format can override any of these for itself. [`encodeWith`](#encodewith) and [`decodeWith`](#decodewith) take a format module and consult its own attributes first, which is why `@json.Name` beats `@coding.Name` when encoding as JSON.

## Contents

- **Attributes** — [`Name`](#name), [`Ignore`](#ignore), [`Default`](#default), [`Decode`](#decode), [`Encode`](#encode)
- **Functions** — [`encode`](#encode-fn), [`decode`](#decode-fn), [`encodeWith`](#encodewith), [`decodeWith`](#decodewith)

---

## Attributes

### `Name` {#name}

<small>`coding/coding.zirr:5`</small>

```zirric
attr Name {
	// The key to use.
	text
}
```

The key a field is read and written under, instead of its declared name. Every attribute here is an override, so a type carrying none encodes under the names it declares.

#### Members

| Member | Description     |
| ------ | --------------- |
| `text` | The key to use. |

---

### `Ignore` {#ignore}

<small>`coding/coding.zirr:12`</small>

```zirric
attr Ignore {}
```

Marks a field as one that does not travel. An ignored field is skipped when encoding, and takes its `@Default` — or `void` — when decoding.

---

### `Default` {#default}

<small>`coding/coding.zirr:15`</small>

```zirric
attr Default {
	// The value to use.
	value
}
```

The value a field takes when its key is absent.

#### Members

| Member  | Description       |
| ------- | ----------------- |
| `value` | The value to use. |

---

### `Decode` {#decode}

<small>`coding/coding.zirr:21`</small>

```zirric
attr Decode {
	// `fn(raw: Any) -> Result`.
	parse
}
```

Decodes a field from the raw tree found under its key, instead of by the general rules.

#### Members

| Member  | Description               |
| ------- | ------------------------- |
| `parse` | `fn(raw: Any) -> Result`. |

---

### `Encode` {#encode}

<small>`coding/coding.zirr:27`</small>

```zirric
attr Encode {
	// `fn(value: Any) -> Result`.
	format
}
```

Encodes a field to a tree, instead of by the general rules.

#### Members

| Member   | Description                 |
| -------- | --------------------------- |
| `format` | `fn(value: Any) -> Result`. |

---

## Functions

### `encode` {#encode-fn}

<small>`coding/coding.zirr:33`</small>

```zirric
fn encode(value: Any) -> Result
```

A value as a native tree of `Dict`, `Array`, `String`, `Int`, `Float`, `Bool` and `void`.

---

### `decode` {#decode-fn}

<small>`coding/coding.zirr:38`</small>

```zirric
fn decode(t: Any, tree: Any) -> Result
```

A native tree as a value of type `t`.

---

### `encodeWith` {#encodewith}

<small>`coding/coding.zirr:43`</small>

```zirric
fn encodeWith(format: Module, value: Any) -> Result
```

As `encode`, but consulting the format module's own attributes first, so `@json.Name` overrides `@coding.Name`.

---

### `decodeWith` {#decodewith}

<small>`coding/coding.zirr:48`</small>

```zirric
fn decodeWith(format: Module, t: Any, tree: Any) -> Result
```

As `decode`, but consulting the format module's own attributes first.

---

## Format overrides

Every attribute here has a counterpart in each format module. When you call [`encodeWith`](#encodewith) or [`decodeWith`](#decodewith), the format's own attribute is consulted first and `@coding`'s is the fallback.

| `coding`              | JSON                                        | YAML                                        |
| --------------------- | ------------------------------------------- | ------------------------------------------- |
| [`Name`](#name)       | [`@json.Name`](../json/index.md#name)       | [`@yaml.Name`](../yaml/index.md#name)       |
| [`Ignore`](#ignore)   | [`@json.Ignore`](../json/index.md#ignore)   | [`@yaml.Ignore`](../yaml/index.md#ignore)   |
| [`Default`](#default) | [`@json.Default`](../json/index.md#default) | [`@yaml.Default`](../yaml/index.md#default) |
| [`Decode`](#decode)   | [`@json.Decode`](../json/index.md#decode)   | [`@yaml.Decode`](../yaml/index.md#decode)   |
| [`Encode`](#encode)   | [`@json.Encode`](../json/index.md#encode)   | [`@yaml.Encode`](../yaml/index.md#encode)   |

---

## See also

- [`json`](../json/index.md), [`yaml`](../yaml/index.md) — carrying the tree as text.
- [`reflect`](../reflect/index.md) — the field and type information this reads.
- [ZE-021 Encoding and Decoding](/proposals/ZE-021-encoding-and-decoding) — the design this follows.
