---
title: Prelude
description: The built-in types, attributes and values every Zirric program starts with.
---

# Module `prelude`

> The types, attributes and values that are always in scope.

```zirric
// always in scope — no import needed
```

|            |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Module** | `prelude`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| **Source** | [`prelude/shim.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/prelude/shim.zirr), [`prelude/attributes.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/prelude/attributes.zirr), [`prelude/countable.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/prelude/countable.zirr), [`prelude/iterable.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/prelude/iterable.zirr), [`prelude/option.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/prelude/option.zirr), [`prelude/result.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/prelude/result.zirr) |

`prelude` is imported into every module automatically. It defines the built-in types the runtime provides, the attributes that describe what a value can do, and the two unions — [`Option`](#option) and [`Result`](#result) — that the rest of the standard library returns.

Nothing here has to be imported, and nothing here can be avoided: `Int`, `String`, `Array` and the rest are prelude declarations, and the `for element <- value` loop is a call into [`Iterable`](#iterable).

## Contents

- **Attributes** — [`Default`](#default), [`Doc`](#doc), [`Deprecated`](#deprecated), [`Numeric`](#numeric), [`Printable`](#printable), [`Countable`](#countable), [`Iterable`](#iterable), [`AnyOption`](#anyoption), [`Error`](#error), [`AnyResult`](#anyresult)
- **Types** — [`Any`](#any), [`AnyType`](#anytype), [`Attribute`](#attribute), [`AttributeType`](#attributetype), [`Module`](#module), [`ModuleType`](#moduletype), [`Array`](#array), [`Bool`](#bool), [`Char`](#char), [`Dict`](#dict), [`Func`](#func), [`Float`](#float), [`Int`](#int), [`String`](#string), [`Binary`](#binary), [`Byte`](#byte), [`Void`](#void)
- **Unions** — [`Number`](#number), [`Option`](#option), [`Result`](#result)
- **Data** — [`Range`](#range), [`ClosedRange`](#closedrange), [`OpenRange`](#openrange), [`Pair`](#pair)
- **Constants** — [`true`](#true), [`false`](#false), [`void`](#void-const)
- **Functions** — [`append`](#append), [`panic`](#panic), [`len`](#len)

---

## Attributes

### `Default` {#default}

<small>`prelude/attributes.zirr:5`</small>

```zirric
attr Default {
	// The default value for a parameter.
	value
}
```

Transparently indicates the assumed default value of a parameter or field. Can be used by tooling and libraries.

#### Members

| Member  | Description                        |
| ------- | ---------------------------------- |
| `value` | The default value for a parameter. |

---

### `Doc` {#doc}

<small>`prelude/attributes.zirr:11`</small>

```zirric
attr Doc {
	// The documentation string without leading comment markers and whitespace.
	description: String
}
```

Provides access to the documentation string of a declaration.

#### Members

| Member        | Signature             | Description                                                              |
| ------------- | --------------------- | ------------------------------------------------------------------------ |
| `description` | `description: String` | The documentation string without leading comment markers and whitespace. |

---

### `Deprecated` {#deprecated}

<small>`prelude/attributes.zirr:18`</small>

```zirric
attr Deprecated {
	@Default("without alternative") reason: String
}
```

Marks a declaration as deprecated with a reason. IDEs and other tools can use this information to warn users about deprecated declarations.

#### Members

| Member   | Signature        | Description                                                      |
| -------- | ---------------- | ---------------------------------------------------------------- |
| `reason` | `reason: String` | Why, and what to use instead. Defaults to "without alternative". |

---

### `Numeric` {#numeric}

<small>`prelude/attributes.zirr:23`</small>

```zirric
attr Numeric {
	// A function to convert the value to a number.
	toNumber(value: @Numeric) -> Number
}
```

Marks a declaration as numeric, providing a way to convert it to a number.

#### Members

| Member     | Signature                             | Description                                  |
| ---------- | ------------------------------------- | -------------------------------------------- |
| `toNumber` | `toNumber(value: @Numeric) -> Number` | A function to convert the value to a number. |

---

### `Printable` {#printable}

<small>`prelude/attributes.zirr:29`</small>

```zirric
attr Printable {
	// A function to convert the value to a string.
	toString(self: @Printable) -> String
}
```

Marks a type as printable, providing a way to convert it to a string.

#### Members

| Member     | Signature                              | Description                                  |
| ---------- | -------------------------------------- | -------------------------------------------- |
| `toString` | `toString(self: @Printable) -> String` | A function to convert the value to a string. |

---

### `Countable` {#countable}

<small>`prelude/attributes.zirr:35`</small>

```zirric
attr Countable {
	// Returns the length of a value.
	length(value: @Countable) -> Int
}
```

Marks a type as countable, providing a way to get its length.

#### Members

| Member   | Signature                          | Description                    |
| -------- | ---------------------------------- | ------------------------------ |
| `length` | `length(value: @Countable) -> Int` | Returns the length of a value. |

---

### `Iterable` {#iterable}

<small>`prelude/attributes.zirr:41`</small>

```zirric
attr Iterable {
	// Calls `yield` with each element, stopping early if `yield` returns `false`.
	iterate(value: @Iterable, yield: fn(Any) -> Bool)
}
```

Marks a type as iterable, allowing `for element <- value`.

#### Members

| Member    | Signature                                           | Description                                                                 |
| --------- | --------------------------------------------------- | --------------------------------------------------------------------------- |
| `iterate` | `iterate(value: @Iterable, yield: fn(Any) -> Bool)` | Calls `yield` with each element, stopping early if `yield` returns `false`. |

---

### `AnyOption` {#anyoption}

<small>`prelude/option.zirr:6`</small>

```zirric
attr AnyOption {}
```

Marks a union as an option type. Types annotated with `@AnyOption` are expected to be unions that follow the Some/None pattern.

---

### `Error` {#error}

<small>`prelude/result.zirr:4`</small>

```zirric
attr Error {
	// Returns a debug string representation of the error.
	debug(err) -> String
}
```

Marks a type as an error type.

#### Members

| Member  | Signature              | Description                                         |
| ------- | ---------------------- | --------------------------------------------------- |
| `debug` | `debug(err) -> String` | Returns a debug string representation of the error. |

---

### `AnyResult` {#anyresult}

<small>`prelude/result.zirr:12`</small>

```zirric
attr AnyResult {
	toResult(self) -> Result
}
```

Marks a union as a result type. Types annotated with `@AnyResult` are expected to be unions that follow the Ok/Err pattern.

#### Members

| Member     | Signature                  | Description                                             |
| ---------- | -------------------------- | ------------------------------------------------------- |
| `toResult` | `toResult(self) -> Result` | Converts this type to the standard [`Result`](#result). |

---

## Types

### `Any` {#any}

<small>`prelude/shim.zirr:4`</small>

```zirric
extern type Any {}
```

Anything is a value of type `Any`.

---

### `AnyType` {#anytype}

<small>`prelude/shim.zirr:6`</small>

```zirric
extern type AnyType {}
```

All types are of type `AnyType`.

---

### `Attribute` {#attribute}

<small>`prelude/shim.zirr:9`</small>

```zirric
extern type Attribute {}
```

All attributes are of type `Attribute`.

---

### `AttributeType` {#attributetype}

<small>`prelude/shim.zirr:11`</small>

```zirric
extern type AttributeType {}
```

All attribute types are of type `AttributeType`.

---

### `Module` {#module}

<small>`prelude/shim.zirr:14`</small>

```zirric
extern type Module {}
```

All modules are of type `Module`.

---

### `ModuleType` {#moduletype}

<small>`prelude/shim.zirr:16`</small>

```zirric
extern type ModuleType {}
```

All module types are of type `ModuleType`.

---

### `Array` {#array}

<small>`prelude/shim.zirr:23`</small>

```zirric
@Countable(_arrayLen)
@Iterable(_arrayIterate)
extern type Array {}
```

A finite list of values.

---

### `Bool` {#bool}

<small>`prelude/shim.zirr:30`</small>

```zirric
extern type Bool {}
```

Represents boolean values like `True` and `False`. Typically used for conditionals and flags.

---

### `Char` {#char}

<small>`prelude/shim.zirr:33`</small>

```zirric
extern type Char {}
```

A single character from a string.

---

### `Dict` {#dict}

<small>`prelude/shim.zirr:40`</small>

```zirric
@Countable(_dictLen)
@Iterable(_dictIterate)
extern type Dict {
	// The dictionary's keys, in unspecified order.
	keys() -> Array
}
```

An associative array of keys and their values. Can be indexed by keys. Iterates over Pair.

#### Members

| Member | Signature         | Description                                  |
| ------ | ----------------- | -------------------------------------------- |
| `keys` | `keys() -> Array` | The dictionary's keys, in unspecified order. |

---

### `Func` {#func}

<small>`prelude/shim.zirr:46`</small>

```zirric
extern type Func {
	// The name of the function. Might be non-deterministic for closures.
	name: String
	// The amount of function parameters to be passed.
	arity: Int
}
```

A callable function.

#### Members

| Member  | Signature      | Description                                                        |
| ------- | -------------- | ------------------------------------------------------------------ |
| `name`  | `name: String` | The name of the function. Might be non-deterministic for closures. |
| `arity` | `arity: Int`   | The amount of function parameters to be passed.                    |

---

### `Float` {#float}

<small>`prelude/shim.zirr:55`</small>

```zirric
@Numeric(fn(f: Float) -> Float { f })
extern type Float {}
```

A floating point number.

---

### `Int` {#int}

<small>`prelude/shim.zirr:59`</small>

```zirric
@Numeric(fn(i: Int) -> Int { i })
extern type Int {}
```

A whole integer number.

---

### `String` {#string}

<small>`prelude/shim.zirr:72`</small>

```zirric
@Printable(fn(s: String) -> String { return s })
@Countable(_strLen)
@Iterable(_stringIterate)
extern type String {
	chars() -> [Char]
}
```

A regular String. Can be indexed to get the Byte. Iterates over Char.

#### Members

| Member  | Signature           | Description                                 |
| ------- | ------------------- | ------------------------------------------- |
| `chars` | `chars() -> [Char]` | The characters of the string, decoded once. |

---

### `Binary` {#binary}

<small>`prelude/shim.zirr:80`</small>

```zirric
@Countable(_binaryLen)
@Iterable(_binaryIterate)
extern type Binary {}
```

A sequence of raw bytes. Can be indexed and iterates over Byte.

---

### `Byte` {#byte}

<small>`prelude/shim.zirr:83`</small>

```zirric
extern type Byte {}
```

A single byte.

---

### `Void` {#void}

<small>`prelude/shim.zirr:88`</small>

```zirric
extern type Void {}
```

The type of the `void` value.

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

Either kind of number. Arithmetic operators accept both, and `math` is written against this union rather than against `Int` or `Float` alone.

#### Cases

| Case    | Interpretation           |
| ------- | ------------------------ |
| `Float` | A floating point number. |
| `Int`   | A whole integer number.  |

---

### `Option` {#option}

<small>`prelude/option.zirr:11`</small>

```zirric
@AnyOption()
union Option {
	// The present value.
	data Some {
		// The value that is present.
		value
	}

	// The absent value.
	data None
}
```

An option type that can be either Some or None. Used for values that may or may not be present.

#### Cases

| Case   | Interpretation     |
| ------ | ------------------ |
| `Some` | The present value. |
| `None` | The absent value.  |

---

### `Result` {#result}

<small>`prelude/result.zirr:19`</small>

```zirric
@AnyResult(fn(r) { r })
union Result {
	// The successful result.
	@AnyResult(fn(r) { r })
	data Ok {
		// The value of the successful result.
		value
	}

	// The error result.
	@AnyResult(fn(r) { r })
	@Error(fn(err) {
		switch err.reason {
		case is String:
			err.reason
		case is @Error:
			Error(err.reason).debug(err.reason)
		case is @Printable:
			Printable(err.reason).toString(err.reason)
		case _:
			_inspect(err.reason)
		}
	})
	data Err {
		// The failure reason
		reason
	}
}
```

A result type that can be either Ok or Err. Used for functions that can fail.

#### Cases

| Case  | Interpretation         |
| ----- | ---------------------- |
| `Ok`  | The successful result. |
| `Err` | The error result.      |

---

## Data

### `Range` {#range}

<small>`prelude/iterable.zirr:6`</small>

```zirric
@Countable(_rangeCount)
@Iterable(_rangeIterate)
data Range {
	start: Int
	end: Int
}
```

Represents an open range of integers from start (inclusive) to end (exclusive).

#### Fields

| Field   | Signature    | Description            |
| ------- | ------------ | ---------------------- |
| `start` | `start: Int` | First value, included. |
| `end`   | `end: Int`   | Upper bound, excluded. |

---

### `ClosedRange` {#closedrange}

<small>`prelude/iterable.zirr:14`</small>

```zirric
@Countable(_closedRangeCount)
@Iterable(_closedRangeIterate)
data ClosedRange {
	start: Int
	end: Int
}
```

Represents a closed range of integers from start (inclusive) to end (inclusive).

#### Fields

| Field   | Signature    | Description            |
| ------- | ------------ | ---------------------- |
| `start` | `start: Int` | First value, included. |
| `end`   | `end: Int`   | Last value, included.  |

---

### `OpenRange` {#openrange}

<small>`prelude/iterable.zirr:22`</small>

```zirric
@Countable(_openRangeCount)
@Iterable(_openRangeIterate)
data OpenRange {
	start: Int
	end: Int
}
```

Represents an open range of integers strictly between start and end (both exclusive).

#### Fields

| Field   | Signature    | Description            |
| ------- | ------------ | ---------------------- |
| `start` | `start: Int` | Lower bound, excluded. |
| `end`   | `end: Int`   | Upper bound, excluded. |

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

| Field   | Description                |
| ------- | -------------------------- |
| `key`   | The entry's key.           |
| `value` | The value stored under it. |

---

## Constants

### `true` {#true}

<small>`prelude/shim.zirr:25`</small>

```zirric
const true = 0 == 0
```

The true boolean. Declared rather than built in, as `0 == 0`.

---

### `false` {#false}

<small>`prelude/shim.zirr:26`</small>

```zirric
const false = 0 != 0
```

The false boolean. Declared rather than built in, as `0 != 0`.

---

### `void` {#void-const}

<small>`prelude/shim.zirr:86`</small>

```zirric
extern const void: Void
```

Represents the absence of a value.

---

## Functions

### `append` {#append}

<small>`prelude/shim.zirr:18`</small>

```zirric
extern fn append(Array, Any) -> Array
```

Returns a new array with one value added at the end. The original array is unchanged.

---

### `panic` {#panic}

<small>`prelude/shim.zirr:91`</small>

```zirric
extern fn panic(message: String) -> Void
```

Terminates execution with the given message.

---

### `len` {#len}

<small>`prelude/countable.zirr:3`</small>

```zirric
fn len(v: @Countable) -> Int
```

The length of anything carrying [`Countable`](#countable) — an array, a dict, a string in bytes, a binary, or a range. For the number of characters in a string, use [`strings.count`](../strings/index.md#count) instead.

---

## See also

- [`options`](../options/index.md), [`results`](../results/index.md) — helpers for the two unions defined here.
- [`errors`](../errors/index.md) — combining and wrapping values that carry [`Error`](#error).
- [`ranges`](../ranges/index.md) — operations on [`Range`](#range), [`ClosedRange`](#closedrange) and [`OpenRange`](#openrange).
- [Type System](/specification/typesystem) — how these types take part in type hints and `is` matching.
