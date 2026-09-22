---
title: Prelude
description: The built-in types, attributes and values every Zirric program starts with.
---

# Module `prelude`

```zirric
import prelude
```

|            |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `prelude`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| **Source** | [`prelude/attributes.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/prelude/attributes.zirr), [`prelude/countable.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/prelude/countable.zirr), [`prelude/iterable.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/prelude/iterable.zirr), [`prelude/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/prelude/module-docs.zirr), [`prelude/option.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/prelude/option.zirr), [`prelude/result.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/prelude/result.zirr), [`prelude/shim.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/prelude/shim.zirr) |

> The types, attributes and values that are always in scope.

`prelude` is imported into every module automatically. It defines the built-in types the runtime provides, the attributes that describe what a value can do, and the two unions — [`Option`](#option) and [`Result`](#result) — that the rest of the standard library returns.

Nothing here has to be imported, and nothing here can be avoided: [`Int`](#int), [`String`](#string), [`Array`](#array) and the rest are prelude declarations, and the `for element <- value` loop is a call into [`Iterable`](#iterable).

## Contents

- **Unions** — [`Number`](#number), [`Option`](#option), [`Result`](#result)
- **Data** — [`ClosedRange`](#closedrange), [`Err`](#err), [`None`](#none), [`Ok`](#ok), [`OpenRange`](#openrange), [`Pair`](#pair), [`Range`](#range), [`Some`](#some)
- **Attributes** — [`AnyOption`](#anyoption), [`AnyResult`](#anyresult), [`Countable`](#countable), [`Default`](#default), [`Deprecated`](#deprecated), [`Error`](#error), [`Iterable`](#iterable), [`Numeric`](#numeric), [`Printable`](#printable)
- **Types** — [`Any`](#any), [`AnyModule`](#anymodule), [`AnyType`](#anytype), [`Array`](#array), [`Attribute`](#attribute), [`AttributeType`](#attributetype), [`Binary`](#binary), [`Bool`](#bool), [`Byte`](#byte), [`Char`](#char), [`Dict`](#dict), [`Float`](#float), [`Func`](#func), [`Int`](#int), [`ModuleType`](#moduletype), [`String`](#string), [`Void`](#void-type)
- **Functions** — [`append`](#append), [`len`](#len), [`panic`](#panic)
- **Constants** — [`false`](#false), [`true`](#true), [`void`](#void-const)

---

## Unions

### `Number` {#number}

<small>`prelude/shim.zirr:61`</small>

```zirric
union Number {
	Float
	Int
}
```

#### Cases

| Case    | Interpretation           |
| ------- | ------------------------ |
| `Float` | A floating point number. |
| `Int`   | A whole integer number.  |

---

### `Option` {#option}

<small>`prelude/option.zirr:21`</small>

```zirric
union Option {
	Some
	None
}
```

An option type that can be either Some or None.
Used for values that may or may not be present.

#### Cases

| Case   | Interpretation     |
| ------ | ------------------ |
| `Some` | The present value. |
| `None` | The absent value.  |

---

### `Result` {#result}

<small>`prelude/result.zirr:26`</small>

```zirric
union Result {
	Ok
	Err
}
```

A result type that can be either Ok or Err.
Used for functions that can fail.

#### Cases

| Case  | Interpretation         |
| ----- | ---------------------- |
| `Ok`  | The successful result. |
| `Err` | The error result.      |

---

## Data

### `ClosedRange` {#closedrange}

<small>`prelude/iterable.zirr:14`</small>

```zirric
data ClosedRange {
	start: Int
	end: Int
}
```

Represents a closed range of integers from start (inclusive) to end (inclusive).

#### Fields

| Field   | Description |
| ------- | ----------- |
| `start` |             |
| `end`   |             |

---

### `Err` {#err}

<small>`prelude/result.zirr:48`</small>

```zirric
data Err {
	reason
}
```

The error result.

#### Fields

| Field    | Description        |
| -------- | ------------------ |
| `reason` | The failure reason |

---

### `None` {#none}

<small>`prelude/option.zirr:31`</small>

```zirric
data None
```

The absent value.

---

### `Ok` {#ok}

<small>`prelude/result.zirr:29`</small>

```zirric
data Ok {
	value
}
```

The successful result.

#### Fields

| Field   | Description                         |
| ------- | ----------------------------------- |
| `value` | The value of the successful result. |

---

### `OpenRange` {#openrange}

<small>`prelude/iterable.zirr:22`</small>

```zirric
data OpenRange {
	start: Int
	end: Int
}
```

Represents an open range of integers strictly between start and end (both exclusive).

#### Fields

| Field   | Description |
| ------- | ----------- |
| `start` |             |
| `end`   |             |

---

### `Pair` {#pair}

<small>`prelude/iterable.zirr:28`</small>

```zirric
data Pair {
	key
	value
}
```

A key/value pair, as yielded when iterating a Dict.

#### Fields

| Field   | Description |
| ------- | ----------- |
| `key`   |             |
| `value` |             |

---

### `Range` {#range}

<small>`prelude/iterable.zirr:6`</small>

```zirric
data Range {
	start: Int
	end: Int
}
```

Represents an open range of integers from start (inclusive) to end (exclusive).

#### Fields

| Field   | Description |
| ------- | ----------- |
| `start` |             |
| `end`   |             |

---

### `Some` {#some}

<small>`prelude/option.zirr:24`</small>

```zirric
data Some {
	value
}
```

The present value.

#### Fields

| Field   | Description                |
| ------- | -------------------------- |
| `value` | The value that is present. |

---

## Attributes

### `AnyOption` {#anyoption}

<small>`prelude/option.zirr:12`</small>

```zirric
attr AnyOption {
	toOption(self: @AnyOption) -> Option
}
```

Marks a union as an option type, and says how to read one as an [`Option`](#option).
Types annotated with `@AnyOption` are expected to be unions that follow the
Some/None pattern, or to convert themselves into one. This is what `?.` and
`??` read a value through when it is not already a [`Some`](#some) or a [`None`](#none), so
annotating a union of your own is what makes those operators work on it.

Write it on each member type as well as on the union: a union's attributes
are not read off a value of one of its members, which is why [`Option`](#option)
annotates [`Some`](#some) and [`None`](#none) individually.

#### Fields

| Field      | Description                                                                                                  |
| ---------- | ------------------------------------------------------------------------------------------------------------ |
| `toOption` | Returns this value as an [`Option`](#option), which for a union already shaped like one is the value itself. |

---

### `AnyResult` {#anyresult}

<small>`prelude/result.zirr:17`</small>

```zirric
attr AnyResult {
	toResult(self: @AnyResult) -> Result
}
```

Marks a union as a result type, and says how to read one as a [`Result`](#result).
Types annotated with `@AnyResult` are expected to be unions that follow the
Ok/Err pattern, or to convert themselves into one. This is what `!.` and
`!!` read a value through when it is not already an [`Ok`](#ok) or an [`Err`](#err).

Write it on each member type as well as on the union, the way this module
annotates [`Ok`](#ok) and [`Err`](#err) individually: a union's attributes are not read off
a value of one of its members.

#### Fields

| Field      | Description                                                                                                 |
| ---------- | ----------------------------------------------------------------------------------------------------------- |
| `toResult` | Returns this value as a [`Result`](#result), which for a union already shaped like one is the value itself. |

---

### `Countable` {#countable}

<small>`prelude/attributes.zirr:29`</small>

```zirric
attr Countable {
	length(value: @Countable) -> Int
}
```

Marks a type as countable, providing a way to get its length.

#### Fields

| Field    | Description                    |
| -------- | ------------------------------ |
| `length` | Returns the length of a value. |

---

### `Default` {#default}

<small>`prelude/attributes.zirr:5`</small>

```zirric
attr Default {
	value
}
```

Transparently indicates the assumed default value of a parameter or field.
Can be used by tooling and libraries.

#### Fields

| Field   | Description                        |
| ------- | ---------------------------------- |
| `value` | The default value for a parameter. |

---

### `Deprecated` {#deprecated}

<small>`prelude/attributes.zirr:12`</small>

```zirric
attr Deprecated {
	reason: String
}
```

Marks a declaration as deprecated with a reason.
IDEs and other tools can use this information to warn users about deprecated declarations.

#### Fields

| Field    | Description |
| -------- | ----------- |
| `reason` |             |

---

### `Error` {#error}

<small>`prelude/result.zirr:4`</small>

```zirric
attr Error {
	debug(err) -> String
}
```

Marks a type as an error type.

#### Fields

| Field   | Description                                         |
| ------- | --------------------------------------------------- |
| `debug` | Returns a debug string representation of the error. |

---

### `Iterable` {#iterable}

<small>`prelude/attributes.zirr:35`</small>

```zirric
attr Iterable {
	iterate(value: @Iterable, yield: fn(Any) -> Bool)
}
```

Marks a type as iterable, allowing `for element <- value`.

#### Fields

| Field     | Description                                                                           |
| --------- | ------------------------------------------------------------------------------------- |
| `iterate` | Calls `yield` with each element, stopping early if `yield` returns [`false`](#false). |

---

### `Numeric` {#numeric}

<small>`prelude/attributes.zirr:17`</small>

```zirric
attr Numeric {
	toNumber(value: @Numeric) -> Number
}
```

Marks a declaration as numeric, providing a way to convert it to a number.

#### Fields

| Field      | Description                                  |
| ---------- | -------------------------------------------- |
| `toNumber` | A function to convert the value to a number. |

---

### `Printable` {#printable}

<small>`prelude/attributes.zirr:23`</small>

```zirric
attr Printable {
	toString(self: @Printable) -> String
}
```

Marks a type as printable, providing a way to convert it to a string.

#### Fields

| Field      | Description                                  |
| ---------- | -------------------------------------------- |
| `toString` | A function to convert the value to a string. |

---

## Types

### `Any` {#any}

<small>`prelude/shim.zirr:4`</small>

```zirric
extern type Any
```

Anything is a value of type `Any`.

---

### `AnyModule` {#anymodule}

<small>`prelude/shim.zirr:14`</small>

```zirric
extern type AnyModule
```

All modules are of type `AnyModule`.

---

### `AnyType` {#anytype}

<small>`prelude/shim.zirr:6`</small>

```zirric
extern type AnyType
```

All types are of type `AnyType`.

---

### `Array` {#array}

<small>`prelude/shim.zirr:23`</small>

```zirric
extern type Array
```

A finite list of values.

---

### `Attribute` {#attribute}

<small>`prelude/shim.zirr:9`</small>

```zirric
extern type Attribute
```

All attributes are of type `Attribute`.

---

### `AttributeType` {#attributetype}

<small>`prelude/shim.zirr:11`</small>

```zirric
extern type AttributeType
```

All attribute types are of type `AttributeType`.

---

### `Binary` {#binary}

<small>`prelude/shim.zirr:80`</small>

```zirric
extern type Binary
```

A sequence of raw bytes.
Can be indexed and iterates over Byte.

---

### `Bool` {#bool}

<small>`prelude/shim.zirr:30`</small>

```zirric
extern type Bool
```

Represents boolean values like `True` and `False`.
Typically used for conditionals and flags.

---

### `Byte` {#byte}

<small>`prelude/shim.zirr:83`</small>

```zirric
extern type Byte
```

A single byte.

---

### `Char` {#char}

<small>`prelude/shim.zirr:33`</small>

```zirric
extern type Char
```

A single character from a string.

---

### `Dict` {#dict}

<small>`prelude/shim.zirr:40`</small>

```zirric
extern type Dict {
	keys: Array
}
```

An associative array of keys and their values.
Can be indexed by keys.
Iterates over Pair.

---

### `Float` {#float}

<small>`prelude/shim.zirr:55`</small>

```zirric
extern type Float
```

A floating point number.

---

### `Func` {#func}

<small>`prelude/shim.zirr:46`</small>

```zirric
extern type Func {
	name: String
	arity: Int
}
```

A callable function.

---

### `Int` {#int}

<small>`prelude/shim.zirr:59`</small>

```zirric
extern type Int
```

A whole integer number.

---

### `ModuleType` {#moduletype}

<small>`prelude/shim.zirr:16`</small>

```zirric
extern type ModuleType
```

All module types are of type `ModuleType`.

---

### `String` {#string}

<small>`prelude/shim.zirr:72`</small>

```zirric
extern type String {
	chars: [Char]
}
```

A regular String.
Can be indexed to get the Byte.
Iterates over Char.

---

### `Void` {#void-type}

<small>`prelude/shim.zirr:88`</small>

```zirric
extern type Void
```

The type of the [`void`](#void-const) value.

---

## Functions

### `append` {#append}

<small>`prelude/shim.zirr:18`</small>

```zirric
extern fn append(Array, Any) -> Array
```

---

### `len` {#len}

<small>`prelude/countable.zirr:3`</small>

```zirric
fn len(v: @Countable) -> Int
```

---

### `panic` {#panic}

<small>`prelude/shim.zirr:91`</small>

```zirric
extern fn panic(message: String) -> Void
```

Terminates execution with the given message.

---

## Constants

### `false` {#false}

<small>`prelude/shim.zirr:26`</small>

```zirric
const false
```

---

### `true` {#true}

<small>`prelude/shim.zirr:25`</small>

```zirric
const true
```

---

### `void` {#void-const}

<small>`prelude/shim.zirr:86`</small>

```zirric
extern const void: Void
```

Represents the absence of a value.
