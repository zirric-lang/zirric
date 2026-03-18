---
title: Prelude
description: Core types, attributes, and primitives available in every Zirric program.
---

# Prelude

The `prelude` module defines Zirric's core types, builtins, and foundational attributes.

## Attributes

### attr Default

```zirric
attr Default {
  value
}
```

Transparently indicates the assumed default value of a parameter or field. Can be
used by tooling and libraries.

Fields:

- `value` — The default value for a parameter.

### attr Doc

```zirric
attr Doc {
  description: String
}
```

Provides access to the documentation string of a declaration.

Fields:

- `description: String` — The documentation string without leading comment markers
  and whitespace.

### attr Deprecated

```zirric
attr Deprecated {
  @Default("without alternative") reason: String
}
```

Marks a declaration as deprecated with a reason. IDEs and other tools can use
this information to warn users about deprecated declarations.

Fields:

- `@Default("without alternative") reason: String`

### attr Numeric

```zirric
attr Numeric {
  toNumber(value: @Numeric) -> Number
}
```

Marks a declaration as numeric, providing a way to convert it to a number.

Members:

- `toNumber(value: @Numeric) -> Number`

## Types

### extern type Any

```zirric
extern type Any {}
```

Anything is a value of type `Any`.

### extern type AnyType

```zirric
extern type AnyType {}
```

All types are of type `AnyType`.

### extern type Attribute

```zirric
extern type Attribute {}
```

All attributes are of type `Attribute`.

### extern type AttributeType

```zirric
extern type AttributeType {}
```

All attribute types are of type `AttributeType`.

### extern type Module

```zirric
extern type Module {}
```

All modules are of type `Module`.

### extern type ModuleType

```zirric
extern type ModuleType {}
```

All module types are of type `ModuleType`.

### extern type Array

```zirric
extern type Array {
  length: Int
}
```

A finite list of values.

Members:

- `length: Int` — The length of the array.

### extern type Bool

```zirric
extern type Bool {
  toggle() -> Bool
}
```

Represents boolean values like `true` and `false`. Typically used for
conditionals and flags.

Members:

- `toggle() -> Bool` — Negates a boolean value.

### extern type Char

```zirric
extern type Char {}
```

A single character from a string.

### extern type Dict

```zirric
extern type Dict {
  length: Int
}
```

An associative array of keys and their values.

Members:

- `length: Int` — The length of the dictionary.

### extern type Func

```zirric
extern type Func {
  arity: Int
}
```

A callable function.

Members:

- `arity: Int` — The amount of function parameters to be passed.

### extern type Float

```zirric
@Numeric(fn(f: Float) -> Float { f })
extern type Float {}
```

A floating point number.

### extern type Int

```zirric
@Numeric(fn(i: Int) -> Int { i })
extern type Int {}
```

A whole integer number.

### extern type String

```zirric
extern type String {
  length: Int
}
```

A string of characters.

Members:

- `length: Int` — The length of the string.

### extern type Void

```zirric
extern type Void {}
```

The type of the `void` value.

## Union

### union Number

```zirric
union Number {
  Float
  Int
}
```

A union of `Float` and `Int`, representing any numeric value.

## Values

### const true

```zirric
const true = 0 == 0
```

The boolean true value.

### const false

```zirric
const false = 0 != 0
```

The boolean false value.

### extern const void

```zirric
extern const void: Void
```

Represents the absence of a value.
