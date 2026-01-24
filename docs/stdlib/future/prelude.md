---
title: Future.Prelude
description: Experimental prelude extensions for proposals (iterables, options, results, and annotation binding).
---

# Future.Prelude

The `prelude` module holds proposal-driven extensions to the prelude, including
iterable helpers, option/result types, and annotation binding.

## Values

### ZE_007

```zirric
let ZE_007 = "https://zirric.knabel.dev/proposals/ZE-007-annotation-binding"
```

ZE-007: Annotation Binding.

### ZE_008

```zirric
let ZE_008 = "https://zirric.knabel.dev/proposals/ZE-008-error-handling"
```

ZE-008: Error Handling.

### ZE_009

```zirric
let ZE_009 = "https://zirric.knabel.dev/proposals/ZE-009-option-values"
```

ZE-009: Option Values.

### ZE_010

```zirric
let ZE_010 = "https://zirric.knabel.dev/proposals/ZE-010-iterable"
```

ZE-010: Iterable.

## Annotations

### Bound

```zirric
@Proposal(ZE_007)
annotation Bound
```

A bound function of an annotation receives the targeted value as first argument.

Example:

```zirric
annotation Error {
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

### Countable

```zirric
@Proposal(ZE_010)
annotation Countable
```

Denotes that a type is countable and has a length.

Members:

- `@Returns(Int) length(@Has(Countable) value)`

### Iterable

```zirric
@Proposal(ZE_010)
annotation Iterable
```

Marks a type as iterable. This allows using `for ... <- ...` on the type.

Members:

- `iterate(@Has(Iterable) value, @Func yield)`

### Error

```zirric
@Proposal(ZE_008)
annotation Error
```

Marks a type as an error type.

Members:

- `@Bound() @Returns(String) debug(err)` — Returns a debug string representation
  of the error.

### AnyResult

```zirric
annotation AnyResult
```

Marks a type as a result type. It is expected that types annotated with
`@ResultType` are unions. Marking a type with `@AnyResult` enables additional
syntactic sugar for working with result types:

- `result!.value` resolves to a `Result` with either `Ok(value)` or `Err(error)`.
- `result !! "default value"` resolves to the value of `Ok` or the provided
  default value if `Err`.

### OkType

```zirric
@Proposal(ZE_008)
annotation OkType
```

Provides a type hint for the `Ok` type of a `Result`.

Fields:

- `@Type(AnyType) type` — The type of `Ok.value`.

### ErrType

```zirric
@Proposal(ZE_008)
annotation ErrType
```

Provides a type hint for the `Err` type of a `Result`.

Fields:

- `@Type(AnyType) type` — The type of `Err.error`.

### AnyOption

```zirric
annotation AnyOption
```

Marks a type as an option type. It is expected that types annotated with
`@Option` are unions. Marking a type with `@AnyOption` enables additional
syntactic sugar for working with option types:

- `option?.value` resolves to an `Option` with either `Some(value)` or `None`.
- `option ?? "default value"` resolves to the value of `Some` or the provided
  default value if `None`.

### SomeType

```zirric
@Proposal(ZE_009)
annotation SomeType
```

Provides a type hint for the `Some` type of an `Option`.

Fields:

- `@Type(AnyType) type` — The type of `Some.value`.

## Data

### Range

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

### ClosedRange

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

### Ok

```zirric
@Proposal(ZE_008)
data Ok
```

The successful result.

Fields:

- `value` — The value of the successful result.

### Err

```zirric
@Proposal(ZE_008)
@Error({ err -> err.error.debug(err.error) })
data Err
```

The error result.

Fields:

- `@Has(Error) error` — The error of the result. Must a type annotated with
  `@Error`.

### Some

```zirric
@Proposal(ZE_009)
data Some
```

The present value.

Fields:

- `value` — The value that is present.

### None

```zirric
@Proposal(ZE_009)
data None
```

The absent value.

## Union

### Result

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

### Option

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

### _arrayIterate

```zirric
func _arrayIterate(v, yield)
```

No documentation.

### _dictIterate

```zirric
func _dictIterate(v, yield)
```

No documentation.

### _stringIterate

```zirric
func _stringIterate(v, yield)
```

No documentation.

### _rangeCount

```zirric
func _rangeCount(v)
```

No documentation.

### _rangeIterate

```zirric
func _rangeIterate(v, yield)
```

No documentation.

### _closedRangeCount

```zirric
func _closedRangeCount(v)
```

No documentation.

### _closedRangeIterate

```zirric
func _closedRangeIterate(v, yield)
```

No documentation.
