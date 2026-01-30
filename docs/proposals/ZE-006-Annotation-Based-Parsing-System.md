---
title: ZE-006 - Annotation-Based Parsing System
description: Annotation-based parsing system in Zirric.
---

# Annotation-Based Parsing System

::: callout draft <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-icon lucide-circle"><circle cx="12" cy="12" r="10"/></svg> Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design.
Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

This proposal introduces an **annotation-based parsing system** for Zirric, enabling flexible and format-agnostic encoding/decoding of data types. The system leverages Zirric's annotation capabilities to define how data should be serialized and deserialized, supporting multiple formats (e.g., JSON, YAML, Protobuf) through modular extensions.

## Motivation

Zirric currently lacks a unified mechanism for encoding/decoding data types to/from various formats. This proposal aims to:

- Provide a **format-agnostic** core system for encoding/decoding.
- Support **format-specific** extensions (e.g., JSON, YAML, Protobuf).
- Use **annotations** to define serialization rules, enabling fine-grained control over encoding/decoding behavior.

## Proposed Solution

### 1. Core Modules and Types

#### `prelude` Module

Core annotations and types used across the codebase.

```zirric
module prelude

extern type Binary {
    @Int length
}

annotation ItemType {
    @AnyType type
}
```

#### `coding` Module

Central module for encoding/decoding logic.

```zirric
module coding

annotation Inline {}

annotation Default {
    @Any value
}

annotation RawType {
    @AnyType type
}

annotation Encodable {
    @Returns(Result) encode(value)
}

annotation Decodable {
    @Returns(Result) decode(value)
}

annotation Key {
    @String name
}

@Returns(Result)
func encode(value, encoder)

@Returns(Result)
func decode(@AnyType type, decoder)
```

#### `coding.json` Module

JSON-specific annotations and functions.

```zirric
module coding.json

// same as coding, but these will only be respected by the JSON encoder/decoder
```

#### `coding.yaml` Module

YAML-specific annotations and functions.

```zirric
module coding.yaml

// same as coding, but these will only be respected by the YAML encoder/decoder
```

## Detailed Design

### 1. Core Annotations

- **`coding.Encodable` and `coding.Decodable`**: Annotations to mark types/fields as encodable/decodable, specifying the encoding/decoding functions.
- **`coding.Key`**: Specifies the key to use for encoding/decoding a field.

### 2. Format-Specific Annotations

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
- **New Annotations**: `coding.Encodable`, `coding.Decodable`, `coding.Key`, `json.Inline`, etc.

## Behavior and Rules

### 1. Inlining

- **Explicit Inlining**: `@json.Inline` must be explicitly applied to enable inlining. Nested inlining also requires explicit `@json.Inline` annotations.

### 2. Conflict Resolution

- **Parsing Errors on Conflicts**: If multiple inlined fields have the same name, the parser will raise an error.

### 3. Optional Fields

- **`@prelude.Option`**: When used with an union, the `None` member should be parsed due to the `@json.Type(json.Null)` annotation.

## Acknowledgements

- **Inspiration**: Swift's `Codable` protocol, Go's format-specific parsing.
- **Contributors**: Valentin Knabel.

## Open Questions

- How should the Decoder and Encoder types be defined?
