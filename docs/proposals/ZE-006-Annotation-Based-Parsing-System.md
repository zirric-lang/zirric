---
title: ZE-006 - Annotation-Based Parsing System
description: Annotation-based parsing system in Zirric.
---

# Annotation-Based Parsing System

::: callout draft <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-search-slash-icon lucide-search-slash"><path d="m13.5 8.5-5 5"/><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg> Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design.
Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

This proposal introduces an **annotation-based parsing system** for Zirric, enabling flexible and format-agnostic encoding/decoding of data types. The system leverages Zirric’s annotation capabilities to define how data should be serialized and deserialized, supporting multiple formats (e.g., JSON, YAML, Protobuf) through modular extensions.

## Motivation

Zirric currently lacks a unified mechanism for encoding/decoding data types to/from various formats. This proposal aims to:

- Provide a **format-agnostic** core system for encoding/decoding.
- Support **format-specific** extensions (e.g., JSON, YAML, Protobuf).
- Use **annotations** to define serialization rules, enabling fine-grained control over encoding/decoding behavior.
- Allow **mixins** for reusable annotation groups, reducing boilerplate.

## Proposed Solution

### 1. Core Modules and Types

#### `prelude` Module

Core annotations and types used across the codebase.

```zirric
module prelude

extern type Binary {
    @Int length
}

data None

annotation Optional
annotation Default {
    @Any value
}

annotation ItemType
    @AnyType type
```

#### `reflect` Module

Core reflection capabilities.

```zirric
module reflect

data Type
data Field
    @String name
    @Type type
    @Array(Annotation) annotations

data Annotation
    @Type annotationType
    @Array(Any) args

annotation Name
    @String name

@Returns(Type) func typeOf(@Any value)
@Returns(Array(Field)) func fieldsOf(@Type type)
@Returns(Annotation?) func annotation(@Field field, @Type annotationType)
@Returns(Bool) func hasAnnotation(@Field field, @Type annotationType)
```

#### `coding` Module

Central module for encoding/decoding logic.

```zirric
module coding

enum Result
    Ok(@Any value)
    Error(@Error error)

data Error
    @String message

mixin Codable
    @Encodable(encode)
    @Decodable(decode)

annotation Encodable
    @Func(@Any) @Returns(Result) encode

annotation Decodable
    @Func(@Type @Any) @Returns(Result) decode

annotation Key
    @String name

@Returns(Result) func encode(@Any value)
@Returns(Result) func decode(@Type type, @Any data)
```

#### `coding.json` Module

JSON-specific annotations and functions.

```zirric
module coding.json

annotation Inline
annotation RawType
    @AnyType type

annotation Key
    @String name

annotation Decode
    @Func(@Any) @Returns(Any) decode

annotation Encode
    @Func(@Any) @Returns(Any) encode

annotation Type
    @AnyType type

@Returns(Result) func encode(@Any value)
@Returns(Result) func decode(@Type type, @Any data)
```

#### `coding.yaml` Module

YAML-specific annotations and functions.

```zirric
module coding.yaml

annotation RawType
    @AnyType type

annotation Key
    @String name

annotation Decode
    @Func(@Any) @Returns(Any) decode

annotation Encode
    @Func(@Any) @Returns(Any) encode

@Returns(Result) func encode(@Any value)
@Returns(Result) func decode(@Type type, @Any data)
```

#### `coding.proto` Module

Protobuf-specific annotations and functions.

```zirric
module coding.proto

annotation RawType
    @AnyType type

annotation Decode
    @Func(@Any) @Returns(Any) decode

annotation Encode
    @Func(@Any) @Returns(Any) encode

@Returns(Result) func encode(@Any value)
@Returns(Result) func decode(@Type type, @Binary data)
```

## Detailed Design

### 1. Core Annotations and Mixins

- **`coding.Codable`**: A mixin combining `coding.Encodable` and `coding.Decodable`.
- **`coding.Encodable` and `coding.Decodable`**: Annotations to mark types/fields as encodable/decodable, specifying the encoding/decoding functions.
- **`coding.Key`**: Specifies the key to use for encoding/decoding a field.

### 2. Format-Specific Annotations

- **`json.Inline`**: Explicitly inlines (flattens) the fields of a `data` type or `enum` case during JSON encoding/decoding.
- **`json.RawType`**: Overrides the raw type for JSON encoding/decoding.
- **`json.Decode` and `json.Encode`**: Specifies custom decoding/encoding functions for a field.
- **`json.Type`**: Specifies the JSON type to use for encoding/decoding a specific enum case.

### 3. Encoding/Decoding Functions

- **`coding.encode` and `coding.decode`**: Generic functions for encoding/decoding values.
- **Format-specific functions**: `json.encode`, `json.decode`, `yaml.encode`, etc.

## Changes to the Standard Library

- **New Modules**: `coding`, `coding.json`, `coding.yaml`, `coding.proto`.
- **New Types**: `coding.Result`, `coding.Error`, `prelude.Binary`, `prelude.None`.
- **New Annotations**: `coding.Encodable`, `coding.Decodable`, `coding.Key`, `json.Inline`, etc.

## Behavior and Rules

### 1. Inlining

- **Explicit Inlining**: `@json.Inline` must be explicitly applied to enable inlining. Nested inlining also requires explicit `@json.Inline` annotations.

### 2. Conflict Resolution

- **Parsing Errors on Conflicts**: If multiple inlined fields have the same name, the parser will raise an error.

### 3. Optional Fields

- **`@prelude.Optional`**: When used with an enum, the `None` case should be parsed due to the `@json.Type(json.Null)` annotation.

## Examples

### 1. Basic Usage with `Codable` Mixin

```zirric
module example

import coding
import coding.json
import reflect

mixin coding.Codable

data Person
    @String name
    @Int age

func main
    let person = Person("Alice" 30)
    let jsonResult = coding.json.encode(person)
```

### 2. Custom Encoding/Decoding with `@json.Inline`

```zirric
module example

import coding
import coding.json
import reflect

@json.Inline()
enum Optional
    @json.Type(json.Null)
    data None

    @json.Inline
    data Some
        @Any value

data Person
    @String name
    @Optional nickname

func main
    let personWithNickname = Person("Alice" Optional.Some("Bob"))
    let personWithoutNickname = Person("Alice" Optional.None)

    let jsonWithNickname = coding.json.encode(personWithNickname)
    // Output: {"name": "Alice", "value": "Bob"}

    let jsonWithoutNickname = coding.json.encode(personWithoutNickname)
    // Output: {"name": "Alice", "nickname": null}
```

### 3. Custom Date Encoding/Decoding

```zirric
module example

import coding
import coding.json
import reflect

@Returns(Date)
func dateFromISODateString(@String raw) {
    // Implementation
}

@Returns(String)
func dateToISODateString(@Date value) {
    // Implementation
}

data PersonWithDate
    @String name
    @Date
    @json.RawType(String)
    @json.Decode(dateFromISODateString)
    @json.Encode(dateToISODateString)
    birthday

func main
    let person = PersonWithDate("Alice" dateFromISODateString("2023-01-01"))
    let jsonResult = coding.json.encode(person)
    // Output: {"name": "Alice", "birthday": "2023-01-01"}
```

### 4. Conflict Example

```zirric
module example

import coding
import coding.json
import reflect

data Child1
    @String field

data Child2
    @String field

data Parent
    @json.Inline
    @Child1 child1

    @json.Inline
    @Child2 child2

func main
    let parent = Parent(Child1("value1") Child2("value2"))
    let jsonResult = coding.json.encode(parent)
    // Error: Conflicting field names "field" in inlined types Child1 and Child2
```

### 5. Optional Enum Example

```zirric
module example

import coding
import coding.json
import reflect

@json.Inline()
enum Optional
    @json.Type(json.Null)
    data None

    @json.Inline
    data Some
        @Any value

data Person
    @String name
    @Optional nickname

func main
    let personWithoutNickname = Person("Alice" Optional.None)
    let jsonWithoutNickname = coding.json.encode(personWithoutNickname)
    // Output: {"name": "Alice", "nickname": null}
```

## Open Questions (Resolved)

1. **Inlining must be explicit**: Yes, `@json.Inline` must be explicitly applied to enable inlining.
2. **Parsing errors on conflicts**: Yes, the parser will raise an error if multiple inlined fields have the same name.
3. **Optional Fields**: `@prelude.Optional` fields with the `None` case should be parsed as `null` due to the `@json.Type(json.Null)` annotation.

## Acknowledgements

- **Inspiration**: Swift’s `Codable` protocol, Go’s format-specific parsing.
- **Contributors**: Valentin Knabel.
