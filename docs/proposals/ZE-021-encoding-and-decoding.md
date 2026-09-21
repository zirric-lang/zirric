---
title: "ZE-021 - Encoding and Decoding"
description: "Encoding and Decoding"
---

# Encoding and Decoding

::: callout warning In Progress
This proposal has been accepted in principle. It is currently under active development. Parts might be incomplete or missing in Zirric.
:::

## Introduction

Adds the `json` and `yaml` modules, and a way to turn Zirric values into either and back, driven by attributes on the declaration rather than by code written per type.

## Motivation

Reading JSON into Zirric's own values and writing them back is enough to work with a document, but not enough to work with a program's own types: turning a `Dict` into a `Person` is a hand-written function per type, and it has to be kept in step with the declaration it mirrors. Every field added to a type is a field that can be forgotten in two places.

The declaration already says almost everything an encoder needs: the fields, their order, their names, and their types. What it does not say is the handful of things that differ between a type and its serialized form — a key spelled differently, a field that should not travel, a value that has a default when absent. Those are exactly what attributes are for.

## Proposed Solution

Three modules:

- `coding` converts between a value and a **native tree** — the `Dict`/`Array`/`String`/`Int`/`Float`/`Bool`/`void` representation that `json.parse` produces. It knows nothing about JSON or YAML.
- `json` and `yaml` convert between a native tree and text.

Separating them means one set of rules for mapping types onto trees, and one small module per format. Adding a format adds a parser and a printer, not a second encoder.

Mapping is driven by ZE-020's `reflect`, which needs no addition here: `fieldsOf` gives the fields of a data type in declaration order, each carrying the attributes written on it, while `fieldValues` and `construct` take a value apart and put one back together, and `isInstance` checks a value against a type known only at runtime. `dicts.hasKey` is what tells an absent key from one holding `void`.

### Attributes

Generic attributes live in `coding` and apply to every format:

| Attribute         | Effect                                                            |
| ----------------- | ----------------------------------------------------------------- |
| `@coding.Name`    | The key to read and write, instead of the field's declared name.  |
| `@coding.Ignore`  | The field does not travel.                                        |
| `@coding.Default` | The value to use when the key is absent.                          |
| `@coding.Decode`  | A function that decodes this field, instead of the default rules. |
| `@coding.Encode`  | A function that encodes this field, instead of the default rules. |

A format module may declare attributes of the same names — `@json.Name`, `@yaml.Name` — which take precedence over the generic ones for that format only. The chain is therefore:

**format-specific attribute → generic `@coding` attribute → the field's declared name**

Every attribute is an override. A data type carrying no attributes at all encodes under its declared field names, which is the case worth optimizing for.

### Types decide how a tree is read

Encoding needs only the value. Decoding needs to know what to build, so it takes a type:

```zirric
const tree = coding.encodeWith(json, person).value
const decoded = coding.decodeWith(json, Person, tree).value
```

The type hints on the fields are what make this work, which is why ZE-020's `reflect` describes a hint structurally rather than as text: `[String: Person]` arrives as a `DictType` of two `NamedType`s, and the decoder walks it.

A union is decoded by trying each member in turn and keeping the first that succeeds. A field with no hint, or one hinted `Any`, keeps the native tree untouched.

## Detailed Design

### The native tree is the only intermediate

There is no `coding.Value` union. A tree is made of the values the language already has, so a program can inspect one, build one by hand, and pass it to any format without a conversion step. It also means `json.parse` composes with `coding.decode` without either module knowing about the other.

A tree means the same thing whichever format produced it, and YAML gives up two things to keep that true. A YAML timestamp stays the text it was written as, so a field decodes the same way through `json` and through `yaml`; converting one is `@coding.Decode`'s job. A mapping key becomes a `String` whatever it was written as, so `1: one` and `"1": one` are one and the same — which also matches what decoding into a data type needs, since fields are named by text.

### Attributes are read off a field like any other value

`reflect.fieldsOf` returns `reflect.Field` values that carry the attributes written on the declaration as their own, so `coding.Name(field).text` reads an attribute off a field exactly as it reads one off a function. Resolving the precedence chain is then three lookups, not a second attribute system.

A format is passed as a `Module`, and its attributes are found by name through `reflect.member`. A format module needs no registration: declaring `attr Name` in it is what makes `@json.Name` take precedence.

### What encoding does

Encoding is driven by the value, never by a type hint, since the value is what is actually there. Each kind of value has one rule:

| Value                                | Tree                                                          |
| ------------------------------------ | ------------------------------------------------------------- |
| `Int` `Float` `String` `Bool` `void` | Itself.                                                       |
| `Array`                              | An `Array`, each element encoded.                             |
| `Dict`                               | A `Dict`, each value encoded. A non-`String` key is an `Err`. |
| `Some`                               | Whatever it holds, encoded — never an object of its own.      |
| `None`                               | No key at all as a field, and `void` on its own.              |
| a data value                         | A `Dict` keyed by the fields that travel.                     |
| anything else                        | An `Err` naming what it found.                                |

### What decoding does

Decoding is driven by the type, because the tree alone cannot say what to build. For each field of a data type, in declaration order:

1. If the field is ignored, it takes its default, or `None` when optional, or `void`. Ignoring is never an error, however the field is typed.
2. Otherwise the key is resolved — format attribute, then `@coding` attribute, then the declared name — and looked up. A key holding `void` counts as present, which is why `dicts.hasKey` exists.
3. If the key is absent: the default if there is one, else `None` if the field is optional, else an `Err` naming the key. An untyped field is not an optional one — leniency is asked for with `@coding.Default`, not inferred.
4. If a decode function is attached, it is given the raw tree and its result is used unchanged.
5. Otherwise the raw tree is decoded against the field's declared type.

Decoding a raw tree against a type hint walks the hint: `[T]` needs an array and decodes each element against `T`, `[K: V]` needs an object and decodes each value against `V`, and a name resolves to its type and recurses. A hint that was never written, one that says `Any`, and one whose name resolves to nothing all keep the tree untouched — the last because a name that resolves to nothing cannot describe what to build. That is about a key that is _present_; an absent one is decided by the rules above, whatever the hint.

### Options are prelude's `Option`, and absence is `None`

A present `Option` travels as the value it holds and an absent one leaves no key behind, so a field that was `None` decodes back to `None` through nothing more than the missing-key rule. That symmetry is the reason `None` is omitted rather than written as null.

`Option` carries no element type, so a present value is wrapped without being decoded further; a field needing more than that uses `@coding.Decode`.

This applies to prelude's `Option` alone, not to everything carrying `@AnyOption`. Another option-shaped union cannot be built by wrapping in `Some` — that would produce a value of the wrong type — so those go through the ordinary union rules instead.

### Unions are tried in declaration order

A union is decoded by trying each member in turn and keeping the first that succeeds. There is no discriminator: members are told apart by whether their fields can be built from the tree at all.

Two consequences are worth stating plainly. Extra keys are ignored, so a member matches when the tree carries everything it needs, not when the tree carries exactly that. And members whose required fields overlap resolve by declaration order: a `Shape` union of `data Circle { radius }` and `data Square { side }` tells itself apart, but a tree carrying both `radius` and `side` decodes as whichever member was declared first.

This is also why an absent key is an error rather than a `void`: a member whose fields all defaulted would match every object, and try-each would always stop at the first member. A union that must be told apart by a tag rather than by shape wants `@coding.Decode` on the field holding it.

### Custom coding is per field

`@coding.Decode` takes a function and hands it the raw tree for that field. This is the escape hatch for anything the general rules cannot express — a date as a string, an enum as a number, a field whose shape depends on another. `@coding.Encode` is its mirror, handed the field's value. Both are deliberately per field rather than per type, since a type that needs custom treatment usually needs it in one place.

## Changes to the Standard Library

### coding

| Declaration  | Kind   | Description                                       |
| ------------ | ------ | ------------------------------------------------- |
| `Name`       | `attr` | The key to read and write.                        |
| `Ignore`     | `attr` | The field does not travel.                        |
| `Default`    | `attr` | The value to use when the key is absent.          |
| `Decode`     | `attr` | Decodes this field from its raw tree.             |
| `Encode`     | `attr` | Encodes this field to a tree.                     |
| `encode`     | `fn`   | A value as a native tree.                         |
| `decode`     | `fn`   | A native tree as a value of a given type.         |
| `encodeWith` | `fn`   | As `encode`, honouring a format's own attributes. |
| `decodeWith` | `fn`   | As `decode`, honouring a format's own attributes. |

### json

Over `encoding/json`. Both format modules are Go throughout, while `coding` is Zirric.

JSON maps onto Zirric's own values rather than onto a tree of its own: an object is a `Dict` with `String` keys, an array an `Array`, and `null` is `void`.

A number is an `Int` when it was written as one and a `Float` otherwise, so `1` and `1.0` stay apart. Object keys come out sorted, so the same value always produces the same text, and text is not HTML-escaped, since a Zirric `String` is text.

Only the types JSON has a form for can be written; a `Char`, a function or a `Dict` with non-`String` keys is an `Err` naming what it found. The attributes are one per generic one, each overriding it for JSON alone.

| Declaration      | Kind        | Description                                         |
| ---------------- | ----------- | --------------------------------------------------- |
| `parse`          | `extern fn` | Reads JSON text, or `Err`.                          |
| `format`         | `extern fn` | Renders a value as JSON text, or `Err`.             |
| `formatIndented` | `extern fn` | As `format`, spread over lines with a given indent. |
| `Name`           | `attr`      | The JSON key for a field.                           |
| `Ignore`         | `attr`      | The field does not travel as JSON.                  |
| `Default`        | `attr`      | The value to use when the JSON key is absent.       |
| `Decode`         | `attr`      | Decodes this field from its raw JSON tree.          |
| `Encode`         | `attr`      | Encodes this field to a JSON tree.                  |

### yaml

Over `goccy/go-yaml`, which also replaced `gopkg.in/yaml.v3` in `zirric cavefile`, so that one implementation decides what YAML means everywhere in the toolchain.

`parse` reads exactly one document and reports a stream of several as an `Err` pointing at `parseAll`, mirroring the trailing-content rule `json.parse` already applies. `formatIndented` takes a number of spaces rather than an indent string, since YAML indentation is only ever spaces.

| Declaration      | Kind        | Description                                        |
| ---------------- | ----------- | -------------------------------------------------- |
| `parse`          | `extern fn` | Reads one YAML document, or `Err`.                 |
| `parseAll`       | `extern fn` | Reads every document in a stream, in order.        |
| `format`         | `extern fn` | Renders a tree as YAML, indented by two spaces.    |
| `formatIndented` | `extern fn` | As `format`, indented by a given number of spaces. |
| `Name`           | `attr`      | The YAML key for a field.                          |
| `Ignore`         | `attr`      | The field does not travel as YAML.                 |
| `Default`        | `attr`      | The value to use when the YAML key is absent.      |
| `Decode`         | `attr`      | Decodes this field from its raw YAML tree.         |
| `Encode`         | `attr`      | Encodes this field to a YAML tree.                 |

## Alternatives Considered

**A `Codable` attribute per type.** Swift requires conformance; Zirric does not need it, because reflection can already describe any data type. Requiring an attribute would mean a type has to opt in to being encodable, which buys nothing when encoding is a function rather than a protocol.

**One encoder per format.** Would let each format see the declaration directly, but every format would then reimplement the attribute rules, and they would drift.

**A tree type of its own.** Rejected for the format modules and for `coding` alike: a `Dict` is already a tree node, and introducing a parallel one would mean converting at both ends.

## Acknowledgements

Swift's `Codable` is the direct model, in particular the split between what a format does and what the generic machinery does. Go's struct tags are the prior art for naming fields by annotation, and its `encoding/json` for treating the absence of a tag as the field name.
