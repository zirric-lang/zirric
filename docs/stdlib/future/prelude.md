---
title: Future.Prelude
description: Experimental prelude extensions for proposals (iterables, options, results, and attribute binding).
---

# Future.Prelude

The `prelude` module holds proposal-driven extensions to the prelude, including
iterable helpers, option/result types, and attribute binding.

## Values

### let ZE_007

```zirric
let ZE_007 = "https://zirric.knabel.dev/proposals/ZE-007-attribute-binding"
```

ZE-007: Attribute Binding.

### let ZE_008

```zirric
let ZE_008 = "https://zirric.knabel.dev/proposals/ZE-008-error-handling"
```

ZE-008: Error Handling.

### let ZE_009

```zirric
let ZE_009 = "https://zirric.knabel.dev/proposals/ZE-009-option-values"
```

ZE-009: Option Values.

### let ZE_010

```zirric
let ZE_010 = "https://zirric.knabel.dev/proposals/ZE-010-iterable"
```

ZE-010: Iterable.

## Attributes

### attr Bound

```zirric
@Proposal(ZE_007)
attr Bound
```

A bound function of an attribute receives the targeted value as first argument.

Example:

```zirric
attr Error {
  @Bound()
  @Returns(String)
  debug(err)
}

@Error({ err -> err.message })
data MessageError {
  @String message
}

MessageError("msg")[@Error].debug()
```

### attr Countable

```zirric
@Proposal(ZE_010)
attr Countable
```

Denotes that a type is countable and has a length.

Members:

- `@Returns(Int) length(@Has(Countable) value)`

### attr Iterable

```zirric
@Proposal(ZE_010)
attr Iterable
```

Marks a type as iterable. This allows using `for ... <- ...` on the type.

Members:

- `iterate(@Has(Iterable) value, @Func yield)`

### attr Error

```zirric
@Proposal(ZE_008)
attr Error
```

Marks a type as an error type.

Members:

- `@Bound() @Returns(String) debug(err)` — Returns a debug string representation
  of the error.

### attr AnyResult

```zirric
attr AnyResult
```

Marks a type as a result type. It is expected that types annotated with
`@ResultType` are unions. Marking a type with `@AnyResult` enables additional
syntactic sugar for working with result types:

- `result!.value` resolves to a `Result` with either `Ok(value)` or `Err(error)`.
- `result !! "default value"` resolves to the value of `Ok` or the provided
  default value if `Err`.

### attr OkType

```zirric
@Proposal(ZE_008)
attr OkType
```

Provides a type hint for the `Ok` type of a `Result`.

Fields:

- `@Type(AnyType) type` — The type of `Ok.value`.

### attr ErrType

```zirric
@Proposal(ZE_008)
attr ErrType
```

Provides a type hint for the `Err` type of a `Result`.

Fields:

- `@Type(AnyType) type` — The type of `Err.error`.

### attr AnyOption

```zirric
attr AnyOption
```

Marks a type as an option type. It is expected that types annotated with
`@Option` are unions. Marking a type with `@AnyOption` enables additional
syntactic sugar for working with option types:

- `option?.value` resolves to an `Option` with either `Some(value)` or `None`.
- `option ?? "default value"` resolves to the value of `Some` or the provided
  default value if `None`.

### attr SomeType

```zirric
@Proposal(ZE_009)
attr SomeType
```

Provides a type hint for the `Some` type of an `Option`.

Fields:

- `@Type(AnyType) type` — The type of `Some.value`.

## Data

### data Range

```zirric
@Proposal(ZE_010)
@Countable(_rangeCount)
@Iterable(_rangeIterate)
data Range
```

Represents an open range of integers from start (inclusive) to end (exclusive).
Can be created using the syntax `start..<end`.

Fields:

- `@Int start`
- `@Int end`

### data ClosedRange

```zirric
@Proposal(ZE_010)
@Countable(_rangeCount)
@Iterable(_rangeIterate)
data ClosedRange
```

Represents a closed range of integers from start (inclusive) to end (inclusive).
Can be created using the syntax `start...end`.

Fields:

- `@Int start`
- `@Int end`

### data Ok

```zirric
@Proposal(ZE_008)
data Ok
```

The successful result.

Fields:

- `value` — The value of the successful result.

### data Err

```zirric
@Proposal(ZE_008)
@Error({ err -> err.error.debug(err.error) })
data Err
```

The error result.

Fields:

- `@Has(Error) error` — The error of the result. Must a type annotated with
  `@Error`.

### data Some

```zirric
@Proposal(ZE_009)
data Some
```

The present value.

Fields:

- `value` — The value that is present.

### data None

```zirric
@Proposal(ZE_009)
data None
```

The absent value.

## Union

### union Result

```zirric
@Proposal(ZE_008)
@AnyResult()
union Result
```

A result type that can be either Ok or Err. Used for functions that can fail.
The following syntactic sugar exists for working with Result types:

- `result!.value`
- `result !! "default value"`
- `@String!` which is equivalent to `@Type(Result) @OkType(String)`

Members:

- `Ok`
- `Err`

### union Option

```zirric
@Proposal(ZE_009)
union Option
```

An option type that can be either Some or None. Used for values that may or may
not be present. The following syntactic sugar exists for working with Option
types:

- `option?.value`
- `option ?? "default value"`
- `@String?` which is equivalent to `@Type(Option) @SomeType(String)`

Members:

- `Some`
- `None`

## Functions

### fn _arrayIterate

```zirric
fn _arrayIterate(v, yield)
```

No documentation.

### fn _dictIterate

```zirric
fn _dictIterate(v, yield)
```

No documentation.

### fn _stringIterate

```zirric
fn _stringIterate(v, yield)
```

No documentation.

### fn _rangeCount

```zirric
fn _rangeCount(v)
```

No documentation.

### fn _rangeIterate

```zirric
fn _rangeIterate(v, yield)
```

No documentation.

### fn _closedRangeCount

```zirric
fn _closedRangeCount(v)
```

No documentation.

### fn _closedRangeIterate

```zirric
fn _closedRangeIterate(v, yield)
```

No documentation.
