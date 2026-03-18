---
title: ZE-006 - Attribute-Based Parsing System
description: Attribute-based parsing system in Zirric.
---

# Attribute-Based Parsing System

::: callout draft <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-icon lucide-circle"><circle cx="12" cy="12" r="10"/></svg> Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design.
Features described here may not be implemented as described and cannot be used right now.
:::

::: callout warning Outdated
While this proposal has not been rejected, it is currently outdated and requires an overhaul to reflect the latest design decisions.

- [ ] Reflect latest syntax changes
- [ ] Reflect latest stdlib changes
- [ ] Attributes are no longer used for types
- [ ] Plan everything out
      :::

## Introduction

This proposal introduces an **attribute-based parsing system** for Zirric, enabling flexible and format-agnostic encoding/decoding of data types. The system leverages Zirric's attribute capabilities to define how data should be serialized and deserialized, supporting multiple formats (e.g., JSON, YAML, Protobuf) through modular extensions.

## Motivation

Zirric currently lacks a unified mechanism for encoding/decoding data types to/from various formats. This proposal aims to:

- Provide a **format-agnostic** core system for encoding/decoding.
- Support **format-specific** extensions (e.g., JSON, YAML, Protobuf).
- Use **attributes** to define serialization rules, enabling fine-grained control over encoding/decoding behavior.

## Proposed Solution

### 1. Core Modules and Types

#### `prelude` Module

Core attributes and types used across the codebase.

```zirric
mod prelude

extern type Binary {
    @Int length
}

attr ItemType {
    @AnyType type
}
```

#### `coding` Module

Central module for encoding/decoding logic.

```zirric
mod coding

attr Inline {}

attr Default {
    @Any value
}

attr RawType {
    @AnyType type
}

attr Encodable {
    @Returns(Result) encode(value)
}

attr Decodable {
    @Returns(Result) decode(value)
}

attr Key {
    @String name
}

@Returns(Result)
fn encode(value, encoder)

@Returns(Result)
fn decode(@AnyType type, decoder)
```

#### `coding.json` Module

JSON-specific attributes and functions.

```zirric
mod coding.json

// same as coding, but these will only be respected by the JSON encoder/decoder
```

#### `coding.yaml` Module

YAML-specific attributes and functions.

```zirric
mod coding.yaml

// same as coding, but these will only be respected by the YAML encoder/decoder
```

## Detailed Design

### 1. Core Attributes

- **`coding.Encodable` and `coding.Decodable`**: Attributes to mark types/fields as encodable/decodable, specifying the encoding/decoding functions.
- **`coding.Key`**: Specifies the key to use for encoding/decoding a field.

### 2. Format-Specific Attributes

- **`json.Inline`**: Explicitly inlines (flattens) the fields of a `data` type or `union` member during JSON encoding/decoding.
- **`json.RawType`**: Overrides the raw type for JSON encoding/decoding.
- **`json.Decode` and `json.Encode`**: Specifies custom decoding/encoding functions for a field.
- **`json.Type`**: Specifies the JSON type to use for encoding/decoding a specific union member.

### 3. Encoding/Decoding Functions

- **`coding.encode` and `coding.decode`**: Generic functions for encoding/decoding values.
- **Format-specific functions**: `json.encode`, `json.decode`, `yaml.encode`, etc.

## Changes to the Standard Library

- **New Modules**: `coding`, `coding.json`, `coding.yaml`
- **New Types**: `coding.Error`, `prelude.Binary`
- **New Attributes**: `coding.Encodable`, `coding.Decodable`, `coding.Key`, `json.Inline`, etc.

## Behavior and Rules

### 1. Inlining

- **Explicit Inlining**: `@json.Inline` must be explicitly applied to enable inlining. Nested inlining also requires explicit `@json.Inline` attributes.

### 2. Conflict Resolution

- **Parsing Errors on Conflicts**: If multiple inlined fields have the same name, the parser will raise an error.

### 3. Optional Fields

- **`@prelude.Option`**: When used with an union, the `None` member should be parsed due to the `@json.Type(json.Null)` attribute.

## Acknowledgements

- **Inspiration**: Swift's `Codable` protocol, Go's format-specific parsing.
- **Contributors**: Valentin Knabel.

## Open Questions

- How should the Decoder and Encoder types be defined?
