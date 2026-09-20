---
title: "ZE-020 - Standard Library"
description: "Basic Standard Library"
---

# Basic Standard Library

::: callout warning In Progress
This proposal has been accepted in principle.
It is currently under active development.
Parts might be incomplete or missing in Zirric.
:::

## Introduction

Adds a basic set of helpers to the standard library modules.

## Motivation

Currently Zirric lacks the most basic operations regarding `prelude` types. For example, it is impossible to actually work with `Binary` or `String`. Furthermore it is currently impossible to actually write tests in Zirric.

## Proposed Solution

Add a new set of modules around every logical group of types:

- `arrays` provides helpers for `prelude.Array`
- `bytes` for working with `prelude.Byte` and `prelude.Binary`
- `clock` reads the time, kept apart from `time` so that nothing pure depends on a clock
- `dicts` supports `prelude.Dict`
- `errors` is centered around `@prelude.Error`
- `fmt` has been cleaned up from `os`
- `fs` abstracts a filesystem, with an in-memory one so that code touching files stays testable
- `io` adds new attributes for dependency management
- `json` reads and writes JSON as Zirric's own values
- `math` covers `prelude.Int` and `prelude.Float`, which had no operations beyond the arithmetic operators
- `options` adds helpers around `prelude.Option` and `@prelude.AnyOption`
- `paths` manipulates slash-separated paths as text, without touching a filesystem
- `prelude` got some slight adjustments
- `random` supplies randomness, in a fast, a reproducible and a cryptographic flavour
- `ranges` covers `prelude.Range`, `prelude.ClosedRange` and `prelude.OpenRange`
- `reflect` inspects modules, types and their fields, with `reflect.packages` reaching the modules of a package
- `results` adds helpers around `prelude.Result` and `@prelude.AnyResult`
- `scripts` collects the conveniences that assume a process, such as printing to stdout
- `fun` holds function-level helpers and lazy sequence operations
- `strings` for working with `prelude.Char` and `prelude.String`
- `time` supplies `Duration`, `Instant` and `Timestamp` as primitives, with no way to read a clock
- `tests` makes it possible to write tests in Zirric, with `tests.runner` discovering and running them through `reflect`

## Detailed Design

### One module per type group, named after the domain

Every module is named after the group of `prelude` types it serves, in the plural, and its directory name matches its `mod` line one-to-one. `Byte` and `Binary` share `bytes` rather than being split into `bytes` and `binaries`, because a `Binary` _is_ a sequence of `Byte` and splitting them would force an arbitrary call on every future helper. The same reasoning keeps `Char` and `String` together in `strings`.

Modules are embedded into the compiled binary through an `embed.go` per directory and a corresponding entry in `DefaultStdlibProvider`, so they are always available without being declared as a dependency.

### The `Like` unions

Several modules accept more than one concrete type for the same conceptual argument. Rather than overloading, each such module declares a `Like` union and a `from(Like)` normalizer used by every other function in that module:

| Module    | `Like` members                      | Normalizes to                       |
| --------- | ----------------------------------- | ----------------------------------- |
| `bytes`   | `Byte`, `Binary`                    | `Binary`                            |
| `strings` | `Char`, `String`                    | `String`                            |
| `ranges`  | `Range`, `ClosedRange`, `OpenRange` | `ClosedRange` (via `toClosedRange`) |

This keeps `bytes.contains(someBinary, 'x')` and `bytes.contains(someBinary, otherBinary)` a single declaration.

`arrays` deliberately has no `Like` union: an `Any | Array` union would not be meaningful, since any value can already be an array element.

### Zirric or Go

Helpers are written in Zirric wherever the language can express them, and as `extern fn` only where it cannot, or where doing so in Zirric would be quadratic:

- `bytes` and `arrays` are mostly Zirric. `Binary` and `Array` both support `append`, which grows amortized in constant time, so building a result incrementally is efficient. Only the operations needing raw byte access — reinterpretation, the hex codec, slicing and substring search — are `extern fn`.
- `strings` is mostly Go. Character indexing has to decode UTF-8, and repeated `result = result + part` is quadratic because strings are immutable, so anything that builds a string from parts is implemented with a Go `strings.Builder` instead.
- `dicts`, `errors`, `fun`, `options`, `ranges` and `results` are pure Zirric.
- `math` is almost entirely Go, since the VM implements the arithmetic operators and nothing else. Only `clamp` is Zirric, composed from `min` and `max`.
- `paths` is entirely Go, wrapping the standard `path` package so that behaviour matches an established implementation rather than a hand-rolled one.
- `json` is entirely Go, wrapping `encoding/json`.
- `fs` splits evenly. The filesystem operations are Go, wrapping billy so that an in-memory filesystem and the host's own are the same code; `walk`, `glob`, `copy`, `withFile` and the string helpers are Zirric, composed from those operations.
- `random` follows the same split: the three generators are Go, everything derived from them is Zirric.
- `time` is Go, but unusually its arithmetic lives in the VM rather than the module, since its three types are primitives rather than values built on top of one.
- `clock` is entirely Zirric. A clock is a closure over a reading, so the test clocks and the host's alike are built in the language; `os` supplies only the two raw readings.
- `reflect` is Go only where it must be. Enumerating a module's members needs access to the module value's exports, describing a type's fields needs what the compiler recorded about the declaration, and reaching a package's modules needs the compiler itself; everything built on top of those — `member`, `hasMember`, `typeOf`, `modulesWhere`, `modulesExcept` — is Zirric, as are the `Field` and `TypeRef` types the Go side fills in.

### Characters versus bytes

`len` counts bytes, and indexing a `String` yields a `Byte`. Everything in `strings` that takes or returns a position instead counts characters, so `count`, `charAt`, `slice`, `range`, `firstIndexOf` and `lastIndexOf` all operate on Unicode code points. For `"café"`, `len` is `5` while `strings.count` is `4`.

### Absence is an `Option`

Any operation that can fail to find something returns `Option` rather than a sentinel: `arrays.first`, `arrays.last`, `bytes.firstIndexOf`, `strings.lastIndexOf`, `ranges.intersect` and `errors.join` all follow this. The `-1` returned by the underlying `extern fn` search primitives is converted at the Zirric boundary and never surfaces to callers.

### Laziness in `fun`

`fun.map`, `fun.filter`, `fun.flatMap`, `fun.take`, `fun.skip` and `fun.zip` accept any `@Iterable` and return a lazy `@Iterable`. Nothing runs until the result is actually iterated, so the operations compose without building an intermediate array per step, and they terminate against infinite sources: `fun.take(counter, 3)` stops after three elements rather than hanging.

Laziness is carried by a single unexported `_lazy` type that wraps one step closure and implements `@Iterable`. Only one type is needed because the operations differ in what their step closure does, not in the shape of the protocol — so any future operation is another function returning a `_lazy`, with no new type.

`fun.reduce` is the exception and consumes its input eagerly, since folding to a single value has to visit every element. `fun.zip` is lazy in its first argument only; its second is buffered up front, because pairing by position needs random access that the push-based `@Iterable` protocol cannot provide incrementally for two independent sources.

The eager `[Any]`-specific equivalents in `arrays` are kept alongside these. They return arrays directly, which is what is wanted when the input is already an array and the result is about to be indexed or measured.

### Time is three primitives, not one number

`Duration`, `Instant` and `Timestamp` are `extern type` declarations, which makes them primitives like `Int` or `Binary` rather than records wrapping one. Three things follow, and no other shape gives all three.

They get operators. Spans add, subtract and scale with `+`, `-` and `*`, which matters because this language has no operator overloading: a record would have turned every one of those into a function call, and durations are the most arithmetic-heavy thing here.

They print as themselves. A duration reads as `1.5s` rather than `1500000000`, both when inspected and when concatenated into text.

They carry a unit the VM enforces. The accepted combinations are the ones that mean something — a span plus a span, a span shifting a point, two points of the same kind differing by a span — and everything else is reported rather than coerced:

| Expression                                                   | Result        |
| ------------------------------------------------------------ | ------------- |
| `Duration ± Duration`                                        | `Duration`    |
| `Duration * Int`, `Int * Duration`, `Duration / Int`         | `Duration`    |
| `Instant - Instant`, `Timestamp - Timestamp`                 | `Duration`    |
| `Instant ± Duration`, `Timestamp ± Duration`                 | the left type |
| `Duration + Int`, `Instant + Instant`, `Instant - Timestamp` | error         |

The last row is the reason `Instant` and `Timestamp` are separate at all. A monotonic reading and a wall-clock one are both a count of nanoseconds, so as plain numbers they would mix silently, and measuring elapsed time with a clock that can jump backwards is a bug that survives every test until the day the host is corrected.

`time` itself never reads a clock. Reading one is `clock`'s job, which is what leaves every function here pure and testable.

### Randomness is a value too

`random` repeats the shape `fs` uses: a `Source` is a value, and the three constructors differ only in where the bits come from. Code written against a `Source` neither knows nor cares which it was handed.

That is what makes randomness testable. `random.seeded(n)` produces the same sequence every run, so a function that shuffles, samples or picks can be tested for its actual behaviour rather than merely for not crashing — pass a seeded source and the outcome is fixed. `fs.memory()` does the same job for files.

The split between `fast` and `strong` is deliberate and named rather than implied. `fast` is seeded from the clock and is a pseudo-random generator; `strong` draws from the operating system's cryptographic generator. Because both are a `Source`, moving from one to the other is a one-line change at the point of construction, and nothing downstream is rewritten.

The capabilities are split the same way, and for a sharper reason than tidiness. A single `HasRandom` would be ambiguous exactly where the difference matters: code generating a token would accept whatever the environment offered, including a seeded generator. `@HasStrongRandom` and `@HasFastRandom` make the requirement part of the signature, so an environment providing only reproducible randomness cannot satisfy code that needs secrecy. A test environment can offer the fast capability alone and will be rejected, at the point of use, by anything asking for the strong one.

### A filesystem is a value

`fs` describes a filesystem as a value rather than as a set of free functions, so a program can be handed one and never learn which it received. `fs.memory()` is a complete filesystem that touches no disk, which is what makes code that reads and writes files testable without fixtures or cleanup — every test in `fs/_t` runs against one.

Both implementations wrap [billy](https://github.com/go-git/go-billy), already a dependency of the package manager, so the in-memory and host filesystems are the same code over a different backing store rather than two implementations that drift.

`cd` returns a filesystem rooted at a subdirectory, which is how a program gives away a confined view of its own. It replaces what would otherwise be a separate `os.dir`.

Permissions are deliberately absent. billy puts them behind a capability its in-memory filesystem does not implement, so exposing them would mean an API that works on one filesystem and fails on another.

### Streams are an attribute, not a type

`io.Reader` and `io.Writer` began as `data` types, which meant only `io` could produce a stream: anything wanting to accept one had to be handed a value of that exact type. As attributes they work the way `@Countable` and `@Iterable` already do — a program annotates its own type and it is a writer, accepted by `fmt.fprint`, the TAP reporter, or anything else asking for `@Writer`.

`ReadStream` and `WriteStream` remain as the types the host hands out, each holding a single function the runtime supplies. They are ordinary types carrying the attribute, with no privileged status.

This is also what makes a file a stream. A file has to be closable, so it cannot be an `io.Writer` value; carrying `@Writer` costs it nothing.

### Module reflection needs only enumeration

A module is already a first-class value: `import tests` binds a `prelude.Module`, and a function value already answers `name` and `arity` and carries the attributes of its own declaration. Matching those attributes is ordinary Zirric — `switch decl { case is @tests.Test: }` — so the only thing missing was a way to ask a module what it declares. `reflect` adds exactly that, and nothing else needs Go.

This is why `tests.discover` is written in Zirric rather than as an extern. It walks `reflect.members`, keeps the declarations carrying `@Test`, and reads `@Skip`, `@Todo` and `@Comment` off each one the same way any other code would.

Only public members are reported, since a module value holds exactly its exports. `_`-prefixed declarations are internal and stay invisible, which is what makes `strings._firstIndexOf` absent from `reflect.memberNames(strings)` while `strings.firstIndexOf` is present.

### A field is a value that carries its attributes

Reflecting a type is not enumeration, because a field is not a value the program can otherwise hold: `Person` has a field called `name`, but nothing in the language hands you that field. `reflect.fieldsOf` therefore builds one, and the important part is that the `Field` it builds carries the attributes written on the declaration as its own. That makes reading them ordinary Zirric — `coding.Name(field).text` and `field is @coding.Ignore` are the same expressions one would write against a function declaration — rather than a second, parallel way of asking about attributes.

A field's declared type is kept structurally rather than as text, so `[String: Person]` arrives as a `DictType` of two `NamedType`s and a decoder can walk it instead of parsing it. Each `NamedType` carries the type its name resolved to, which is the same value an `is` check compares against, so reflection can recurse into a nested data type without going back through a name.

Type hints are not enforced at runtime, which a `TypeRef` does not change: it says what a declaration promised, not what a value turned out to be.

### Loading a package is deliberate

`reflect.packages` reaches the modules a package declares, including ones no import mentions. Reaching them at all means compiling them, which happens when the module is bound rather than when it is called; `reflect.packages` is therefore a separate module from `reflect`, so that ordinary module reflection — the part `tests` depends on — costs nothing.

Two kinds of module are never offered. The entry module is excluded because the running program cannot be loaded a second time. A module whose unqualified name is already taken by a loaded module is excluded because it could never be imported anyway: a project carrying its own `prelude` directory would otherwise compile a second set of core types, and values made by one would fail `@Countable` lookups from the other.

Loading a module runs its top-level code, so the API has no way to ask for every module without saying so. `modulesWhere` tests each name before resolving it, and `modulesExcept` requires the exclusions to be named — `modulesExcept(p, [])` loads everything, but only when written that way on purpose.

## Changes to the Standard Library

### arrays

Operates on `[Any]` and returns arrays eagerly. For lazy equivalents over any `@Iterable`, see `fun`.

| Declaration    | Kind | Description                                                                |
| -------------- | ---- | -------------------------------------------------------------------------- |
| `slice`        | `fn` | The elements from `start` (inclusive) to `end` (exclusive).                |
| `range`        | `fn` | The elements selected by a `ranges.Like`.                                  |
| `concat`       | `fn` | Flattens an array of arrays one level into a single array.                 |
| `repeat`       | `fn` | The array repeated `n` times.                                              |
| `reverse`      | `fn` | The elements in reverse order.                                             |
| `isEmpty`      | `fn` | Whether the array has no elements.                                         |
| `first`        | `fn` | The first element as `Some`, or `None` if empty.                           |
| `last`         | `fn` | The last element as `Some`, or `None` if empty.                            |
| `contains`     | `fn` | Whether a value occurs anywhere in the array.                              |
| `firstIndexOf` | `fn` | The index of a value's first occurrence, or `None` if absent.              |
| `lastIndexOf`  | `fn` | The index of a value's last occurrence, or `None` if absent.               |
| `map`          | `fn` | A new array with a transform applied to each element.                      |
| `flatMap`      | `fn` | A new array with a transform applied to each element, flattened one level. |
| `filter`       | `fn` | The elements for which a predicate returns `true`.                         |
| `reduce`       | `fn` | Folds the elements into a single value, left to right.                     |

### bytes

| Declaration    | Kind        | Description                                                        |
| -------------- | ----------- | ------------------------------------------------------------------ |
| `Like`         | `union`     | `Byte` or `Binary`, accepted wherever a byte sequence is expected. |
| `from`         | `extern fn` | Normalizes a `Like` to `Binary`.                                   |
| `fromString`   | `extern fn` | The UTF-8 bytes of a `String`.                                     |
| `fromChar`     | `extern fn` | The UTF-8 bytes of a `Char`, which may be 1 to 4 bytes.            |
| `toString`     | `extern fn` | Reinterprets the bytes as a `String`.                              |
| `toHex`        | `extern fn` | The hex representation, two lowercase digits per byte.             |
| `fromHex`      | `extern fn` | Decodes a hex string, either case, into `Binary`.                  |
| `slice`        | `extern fn` | The bytes from `start` (inclusive) to `end` (exclusive).           |
| `range`        | `fn`        | The bytes selected by a `ranges.Like`.                             |
| `firstIndexOf` | `fn`        | The index of a needle's first occurrence, or `None` if absent.     |
| `lastIndexOf`  | `fn`        | The index of a needle's last occurrence, or `None` if absent.      |
| `contains`     | `fn`        | Whether a needle occurs anywhere.                                  |
| `hasPrefix`    | `fn`        | Whether the bytes start with a prefix.                             |
| `hasSuffix`    | `fn`        | Whether the bytes end with a suffix.                               |
| `concat`       | `fn`        | Concatenates every part into a single `Binary`, in order.          |
| `repeat`       | `fn`        | The bytes repeated `n` times.                                      |

### clock

Reading the time is a side effect, so it lives apart from `time`. The capabilities are split because a wall clock can jump: code measuring how long something took must not be handed one.

The test clocks are the point. `stepping` and `steppingMonotonic` make time pass exactly as much as a test says it does, with no waiting, which is what `fs.memory()` and `random.seeded()` do for their own domains.

None of this module is Go. A clock is a closure over a reading, so every clock here — including the host's, which `os` builds from two raw readings — is ordinary Zirric.

| Declaration         | Kind        | Description                                                       |
| ------------------- | ----------- | ----------------------------------------------------------------- |
| `SystemClock`       | `data`      | A clock reading civil time, which can jump.                       |
| `MonotonicClock`    | `data`      | A clock that only moves forward, whose origin carries no meaning. |
| `HasSystemClock`    | `attr`      | Provides the wall clock.                                          |
| `HasMonotonicClock` | `attr`      | Provides the monotonic clock.                                     |
| `now`               | `fn`        | The current wall-clock time.                                      |
| `instant`           | `fn`        | The current monotonic reading.                                    |
| `fixed`             | `extern fn` | A wall clock frozen at one time.                                  |
| `stepping`          | `extern fn` | A wall clock advancing by a fixed step on every reading.          |
| `steppingMonotonic` | `extern fn` | A monotonic clock advancing by a fixed step on every reading.     |

### dicts

Iteration order is unspecified, so `reduce` must be given an order-independent combine.

| Declaration | Kind | Description                                                                   |
| ----------- | ---- | ----------------------------------------------------------------------------- |
| `isEmpty`   | `fn` | Whether the dict has no entries.                                              |
| `map`       | `fn` | Maps each value, keeping the keys. The transform receives both key and value. |
| `mapKeys`   | `fn` | Maps each key, keeping the values.                                            |
| `mapValues` | `fn` | Maps each value, keeping the keys. The transform receives only the value.     |
| `mapPairs`  | `fn` | Maps each `Pair`, replacing both key and value.                               |
| `filter`    | `fn` | The entries for which a predicate returns `true`, given key and value.        |
| `reduce`    | `fn` | Folds the entries into a single value via `combine(acc, key, value)`.         |

### errors

The concrete types behind `join` and `wrap` are intentionally unexported, so the only guaranteed surface is `@Error` itself.

| Declaration | Kind | Description                                                              |
| ----------- | ---- | ------------------------------------------------------------------------ |
| `debug`     | `fn` | The debug string of an `@Error`, via its attribute.                      |
| `join`      | `fn` | Combines several errors into one as `Some`, or `None` if there are none. |
| `wrap`      | `fn` | Adds context to an error, debugging as `"<message>: <err>"`.             |
| `unwrap`    | `fn` | The error `wrap` was given, or the error itself if it is not wrapped.    |

### fmt

**Removed:** `println`. Moved to `scripts`.

| Declaration | Kind        | Description                                                                    |
| ----------- | ----------- | ------------------------------------------------------------------------------ |
| `sprint`    | `extern fn` | Any value as a `String`, preferring `@Printable` and falling back to builtins. |
| `fprint`    | `fn`        | Writes a printable value to any `@Writer` and returns the bytes written.       |
| `fprintln`  | `fn`        | As `fprint`, followed by a newline.                                            |

### fs

Every operation that can fail returns a `Result`; nothing here stops the program. A `File` carries both `@io.Reader` and `@io.Writer`, so it can be passed anywhere a stream is expected, including `fmt.fprint`.

| Declaration     | Kind        | Description                                                                 |
| --------------- | ----------- | --------------------------------------------------------------------------- |
| `FileSystem`    | `data`      | A filesystem, as a value that can be passed around.                         |
| `File`          | `data`      | An open file: a reader, a writer, and closable.                             |
| `Entry`         | `data`      | One directory entry: `name`, `isDir`, `size`.                               |
| `HasFileSystem` | `attr`      | Provides a filesystem, so code can require one instead of using `os`.       |
| `memory`        | `extern fn` | An empty in-memory filesystem, touching no disk.                            |
| `close`         | `fn`        | Releases a file; writes are not guaranteed to have landed until it returns. |
| `withFile`      | `fn`        | Runs a body with an open file and closes it afterwards.                     |
| `readString`    | `fn`        | A file's contents decoded as a `String`.                                    |
| `writeString`   | `fn`        | Writes a `String` as UTF-8.                                                 |
| `walk`          | `fn`        | Every file path beneath a directory, descending into subdirectories.        |
| `glob`          | `fn`        | Every file path beneath a base directory matching a pattern.                |
| `copy`          | `fn`        | Copies one file between filesystems, which may be the same one.             |

Every `FileSystem` field also exists as a module function taking the filesystem first — `readFile`, `writeFile`, `open`, `create`, `exists`, `remove`, `move`, `list`, `mkdirAll`, `cd`, `root` — so that `fs.readFile(disk, path)` reads like the rest of the library rather than `disk.readFile(path)`, and so a filesystem can be threaded through `fun.pipe` and friends. The field remains the thing a custom filesystem implements; the function is only a call onto it.

The module is split across `fs.zirr` for the types, `operations.zirr` for those delegations, and `helpers.zirr` for everything composed from them.

### io

`Reader` and `Writer` became attributes rather than `data` types, so being a stream is something any type can carry instead of something only `io` can hand out. A program can annotate its own buffer, and it works with `fmt.fprint` and the TAP reporter like any other writer.

`ReadStream` and `WriteStream` are what the host provides — the concrete types behind `os.stdout` and friends — but they hold no privileged position.

The capability attributes let a value declare which streams it provides, so code can require a capability instead of reaching for `os` directly.

| Declaration         | Kind   | Description                                            |
| ------------------- | ------ | ------------------------------------------------------ |
| `Reader`            | `attr` | Marks a type as a readable stream of bytes.            |
| `Writer`            | `attr` | Marks a type as a writable stream of bytes.            |
| `ReadStream`        | `data` | A readable stream backed by the host.                  |
| `WriteStream`       | `data` | A writable stream backed by the host.                  |
| `HasStandardWriter` | `attr` | Provides a standard out writer, usually `os.stdout`.   |
| `HasErrorWriter`    | `attr` | Provides a standard error writer, usually `os.stderr`. |
| `HasStandardReader` | `attr` | Provides a standard input reader, usually `os.stdin`.  |

### json

JSON maps onto Zirric's own values rather than onto a tree of its own: an object is a `Dict` with `String` keys, an array an `Array`, and `null` is `void`.

A number is an `Int` when it was written as one and a `Float` otherwise, so `1` and `1.0` stay apart. Object keys come out sorted, so the same value always produces the same text, and text is not HTML-escaped, since a Zirric `String` is text.

Only the types JSON has a form for can be written; a `Char`, a function or a `Dict` with non-`String` keys is an `Err` naming what it found. Mapping data types and their attributes onto this tree is a later concern, and will not change what is here.

| Declaration      | Kind        | Description                                         |
| ---------------- | ----------- | --------------------------------------------------- |
| `parse`          | `extern fn` | Reads JSON text, or `Err`.                          |
| `format`         | `extern fn` | Renders a value as JSON text, or `Err`.             |
| `formatIndented` | `extern fn` | As `format`, spread over lines with a given indent. |

### math

`Int` and `Float` mix freely in arithmetic, where the VM promotes to `Float`, and this module follows the same instinct: `abs`, `min`, `max` and `clamp` return whichever type they were given, while the rounding family, `sqrt` and `pow` return `Float` because that is what they genuinely produce. `toInt` is the way back.

Float division is unguarded, so `1.0 / 0.0` and `0.0 / 0.0` yield infinity and NaN without any effort on the caller's part. `isInfinite` and `isNaN` are how those are detected — NaN in particular compares unequal to everything, itself included, so no equality test can find it.

| Declaration | Kind           | Description                                                                |
| ----------- | -------------- | -------------------------------------------------------------------------- |
| `pi`        | `extern const` | The ratio of a circle's circumference to its diameter.                     |
| `e`         | `extern const` | Euler's number, the base of the natural logarithm.                         |
| `toFloat`   | `extern fn`    | A number as a `Float`.                                                     |
| `toInt`     | `extern fn`    | A number as an `Int`, discarding any fractional part rather than rounding. |
| `abs`       | `extern fn`    | A number without its sign, as the type it was given.                       |
| `min`       | `extern fn`    | Whichever of two numbers is smaller, as the type it was given.             |
| `max`       | `extern fn`    | Whichever of two numbers is larger, as the type it was given.              |
| `clamp`     | `fn`           | A number limited to a range, as the type it was given.                     |
| `sign`      | `extern fn`    | `-1`, `0` or `1` according to the number's sign.                           |
| `floor`     | `extern fn`    | The greatest whole number not above the input.                             |
| `ceil`      | `extern fn`    | The least whole number not below the input.                                |
| `round`     | `extern fn`    | The nearest whole number, halves away from zero.                           |
| `trunc`     | `extern fn`    | The input with its fractional part discarded, rounding toward zero.        |
| `sqrt`      | `extern fn`    | The square root.                                                           |
| `pow`       | `extern fn`    | A base raised to an exponent.                                              |

### options

`from` accepts any value: `None` stays `None`, an existing `Some` is left as it is, and anything else is lifted into `Some`. Every other function normalizes through it, so all of them accept bare values too.

| Declaration | Kind | Description                                             |
| ----------- | ---- | ------------------------------------------------------- |
| `from`      | `fn` | Normalizes any value to an `Option`.                    |
| `isSome`    | `fn` | Whether the value is present.                           |
| `isNone`    | `fn` | Whether the value is absent.                            |
| `map`       | `fn` | Maps the contained value, leaving `None` untouched.     |
| `flatMap`   | `fn` | As `map`, for transforms that can themselves be absent. |
| `or`        | `fn` | The contained value, or a default when absent.          |

### os

Only `fs` is new; the standard streams and process accessors are unchanged, though `stdout`, `stdin` and `stderr` now yield values carrying `@io.Writer` or `@io.Reader` rather than values of a `Writer`/`Reader` type.

| Declaration      | Kind        | Description                                                                  |
| ---------------- | ----------- | ---------------------------------------------------------------------------- |
| `fs`             | `extern fn` | The host's own filesystem, rooted so that absolute paths resolve as written. |
| `systemClock`    | `extern fn` | The host's wall clock.                                                       |
| `monotonicClock` | `extern fn` | The host's monotonic clock, for measuring elapsed time.                      |
| `processes`      | `fn`        | The host's own ability to run programs.                                      |
| `stdout`         | `extern fn` | The standard output stream, as an `@io.Writer`.                              |
| `stdin`          | `extern fn` | The standard input stream, as an `@io.Reader`.                               |
| `stderr`         | `extern fn` | The standard error stream, as an `@io.Writer`.                               |
| `exit`           | `extern fn` | Ends the process with a status code. Unchanged.                              |
| `env`            | `extern fn` | An environment variable's value. Unchanged.                                  |
| `args`           | `extern fn` | The command line arguments. Unchanged.                                       |

### paths

Pure text manipulation: nothing here touches a filesystem. Paths are slash-separated regardless of host, matching the filesystem abstraction rather than the machine the program runs on.

| Declaration | Kind        | Description                                                                  |
| ----------- | ----------- | ---------------------------------------------------------------------------- |
| `join`      | `extern fn` | Every part joined into one path, cleaned.                                    |
| `clean`     | `extern fn` | Redundant separators and `.` or `..` elements resolved.                      |
| `base`      | `extern fn` | The last element of a path.                                                  |
| `dir`       | `extern fn` | A path without its last element.                                             |
| `ext`       | `extern fn` | The extension of the last element, including the dot, or `""`.               |
| `stem`      | `extern fn` | The last element without its extension.                                      |
| `isAbs`     | `extern fn` | Whether the path begins at the root.                                         |
| `segments`  | `extern fn` | The path's non-empty elements, in order.                                     |
| `match`     | `extern fn` | Whether a path matches a glob pattern, or `Err` if the pattern is malformed. |

### prelude

| Declaration                         | Kind          | Description                                                                    |
| ----------------------------------- | ------------- | ------------------------------------------------------------------------------ |
| `Binary`                            | `extern type` | Renamed from `Bytes`. Indexes and iterates as `Byte`, and inspects as hex.     |
| `Byte`                              | `extern type` | A single byte. Produced by indexing a `String` or a `Binary`.                  |
| `OpenRange`                         | `data`        | Integers strictly between `start` and `end`, both exclusive.                   |
| `Pair`                              | `data`        | A key/value pair, as yielded when iterating a `Dict`.                          |
| `Module`                            | `extern type` | The type of a module value. `import x` binds one, and `reflect` inspects it.   |
| `append`                            | `extern fn`   | Appends to an `Array` or `Binary`, growing amortized in constant time.         |
| `len`                               | `fn`          | The length of any `@Countable`. Counts bytes for `String` and `Binary`.        |
| `_inspect`                          | `extern fn`   | A debug string for any value, whether or not it is `@Printable`.               |
| `Array`, `Dict`, `String`, `Binary` | `extern type` | All gained `@Countable` and `@Iterable`, so `len` and `for <-` work uniformly. |

**Removed:** `bytesFromString` and `binaryFromString`, superseded by `bytes.fromString` and `bytes.fromChar`. The `length` fields on `Array`, `Dict` and `String`, superseded by `len`. `with` and `pipe`, moved to `fun`. `Bool.toggle`.

**Changed:** `Dict.keys` and `String.chars` became methods, `keys()` and `chars()`. `Func` gained a `name` field.

### random

A `Source` is a value, so code takes one rather than reaching for a generator. Each field has a matching module function taking the source first.

| Declaration       | Kind        | Description                                                             |
| ----------------- | ----------- | ----------------------------------------------------------------------- |
| `Source`          | `data`      | A source of randomness.                                                 |
| `HasFastRandom`   | `attr`      | Provides a fast source, which a seeded one satisfies.                   |
| `HasStrongRandom` | `attr`      | Provides a cryptographic source.                                        |
| `fast`            | `extern fn` | Seeded from the clock. For simulations and sampling, never for secrets. |
| `seeded`          | `extern fn` | Seeded with a given value, producing the same sequence every run.       |
| `strong`          | `extern fn` | The operating system's cryptographic generator.                         |
| `int`             | `fn`        | A whole number from 0 up to but excluding a bound.                      |
| `float`           | `fn`        | A number from 0 up to but excluding 1.                                  |
| `bytes`           | `fn`        | A given number of random bytes.                                         |
| `bool`            | `fn`        | True or false with equal likelihood.                                    |
| `intBetween`      | `fn`        | A whole number within a range.                                          |
| `floatBetween`    | `fn`        | A number within a range.                                                |
| `choice`          | `fn`        | One element of an array, or `None` if it is empty.                      |
| `shuffle`         | `fn`        | The elements in a new order, leaving the original untouched.            |

### ranges

| Declaration     | Kind    | Description                                                                 |
| --------------- | ------- | --------------------------------------------------------------------------- |
| `Like`          | `union` | `Range`, `ClosedRange` or `OpenRange`.                                      |
| `contains`      | `fn`    | Whether a value lies within the range.                                      |
| `toClosedRange` | `fn`    | The equivalent inclusive bounds as `Some`, or `None` if the range is empty. |
| `overlap`       | `fn`    | Whether two ranges share any value.                                         |
| `intersect`     | `fn`    | The values two ranges have in common, or `None`.                            |
| `merge`         | `fn`    | Two ranges combined into one spanning both, or `None` if there is a gap.    |

### reflect

Reports only a module's public members, in name order. Enumeration is the one thing that needs Go; the values it returns already answer `name` and `arity` and carry their own attributes.

| Declaration   | Kind        | Description                                                      |
| ------------- | ----------- | ---------------------------------------------------------------- |
| `moduleName`  | `extern fn` | The canonical name of a module.                                  |
| `members`     | `extern fn` | Every public member of a module, ordered by name.                |
| `memberNames` | `extern fn` | The name of every public member, in the same order as `members`. |
| `member`      | `fn`        | The public member of a module by name, or `None` if it has none. |
| `hasMember`   | `fn`        | Whether a module has a public member of that name.               |

Types are reflected the same way. `fieldsOf` returns values, so the attributes written on a field are read off it exactly as off any other declaration.

| Declaration    | Kind        | Description                                                   |
| -------------- | ----------- | ------------------------------------------------------------- |
| `Field`        | `data`      | A field of a data type, carrying that field's own attributes. |
| `TypeRef`      | `union`     | A type hint as it was written.                                |
| `NamedType`    | `data`      | A type named directly, and the type that name resolves to.    |
| `ArrayType`    | `data`      | An array type, `[Element]`.                                   |
| `DictType`     | `data`      | A dict type, `[Key: Value]`.                                  |
| `FuncType`     | `data`      | A function type, `fn(Parameters) -> Returns`.                 |
| `AttrsType`    | `data`      | An attribute constraint, e.g. `@Iterable`.                    |
| `UnknownType`  | `data`      | No type hint was written.                                     |
| `typeName`     | `extern fn` | The declared name of a type.                                  |
| `isDataType`   | `extern fn` | Whether a value is a data type.                               |
| `isUnionType`  | `extern fn` | Whether a value is a union type.                              |
| `fieldsOf`     | `extern fn` | The fields of a data type, in declaration order.              |
| `unionMembers` | `extern fn` | The member types of a union type, in declaration order.       |
| `typeOf`       | `fn`        | The type a value was built from, or `None`.                   |

#### reflect.packages

Reaching a package's modules compiles them, so this is a separate module: importing `reflect` alone carries none of that cost. Loading a module runs its top-level code, which is why there is no function that loads every module without the caller saying so.

| Declaration     | Kind   | Description                                                                    |
| --------------- | ------ | ------------------------------------------------------------------------------ |
| `Package`       | `data` | A package and the names of the modules it declares.                            |
| `mainPackage`   | `fn`   | The package the running project declares, less the entry and shadowed modules. |
| `module`        | `fn`   | The module of a package by name, or `None` if it declares no such module.      |
| `modulesWhere`  | `fn`   | Every module whose name satisfies a predicate; the rest are never loaded.      |
| `modulesExcept` | `fn`   | Every module except those named.                                               |

### results

`from` normalizes any `@AnyResult` through its `toResult`, so every function accepts any result-shaped type, not just `Result`.

| Declaration  | Kind | Description                                                           |
| ------------ | ---- | --------------------------------------------------------------------- |
| `from`       | `fn` | Normalizes an `@AnyResult` to `Result`.                               |
| `isOk`       | `fn` | Whether the result is `Ok`.                                           |
| `isErr`      | `fn` | Whether the result is `Err`.                                          |
| `map`        | `fn` | Maps the success value, leaving an error untouched.                   |
| `mapErr`     | `fn` | Maps the error reason, leaving a success untouched.                   |
| `flatMap`    | `fn` | As `map`, for transforms that can themselves fail.                    |
| `flatMapErr` | `fn` | As `mapErr`, for recoveries that can themselves fail.                 |
| `catch`      | `fn` | Recovers from an error by handling its reason, yielding `Ok`.         |
| `catchTo`    | `fn` | Recovers from an error by replacing it with a default, yielding `Ok`. |
| `all`        | `fn` | `Ok` of every value if all succeed, otherwise `Err` of every reason.  |

### scripts

Conveniences that assume a process and its standard streams, so that `fmt` itself stays free of `os`.

| Declaration | Kind | Description                                       |
| ----------- | ---- | ------------------------------------------------- |
| `print`     | `fn` | Writes a printable value to stdout.               |
| `println`   | `fn` | Writes a printable value and a newline to stdout. |
| `eprint`    | `fn` | Writes a printable value to stderr.               |
| `eprintln`  | `fn` | Writes a printable value and a newline to stderr. |

The same idea extends to files: these operate on `os.fs()`, so a script need not thread a filesystem through itself. Code that wants to stay testable takes an `fs.FileSystem` instead.

The module is split by what each group wraps — `fmt-scripts.zirr` and `fs-scripts.zirr` — so that adding helpers for another module adds a file rather than lengthening one.

| Declaration   | Kind | Description                                             |
| ------------- | ---- | ------------------------------------------------------- |
| `readFile`    | `fn` | A file's bytes from the host's filesystem.              |
| `writeFile`   | `fn` | Writes bytes to the host's filesystem.                  |
| `readString`  | `fn` | A file's contents as a `String`.                        |
| `writeString` | `fn` | Writes a `String` as UTF-8.                             |
| `exists`      | `fn` | Whether a path exists.                                  |
| `list`        | `fn` | The entries of a directory.                             |
| `remove`      | `fn` | Removes a path.                                         |
| `move`        | `fn` | Moves a file.                                           |
| `mkdirAll`    | `fn` | Creates a directory and any missing parents.            |
| `glob`        | `fn` | Every path beneath a base directory matching a pattern. |

### time

Pure throughout: nothing here reads a clock. The three types are primitives, so their arithmetic is the VM's rather than this module's — see [Time is three primitives](#time-is-three-primitives-not-one-number).

| Declaration                                                                  | Kind          | Description                                                              |
| ---------------------------------------------------------------------------- | ------------- | ------------------------------------------------------------------------ |
| `Duration`                                                                   | `extern type` | A span of time.                                                          |
| `Instant`                                                                    | `extern type` | A monotonic reading, meaningful only next to another.                    |
| `Timestamp`                                                                  | `extern type` | A point on the wall clock.                                               |
| `nanoseconds`, `microseconds`, `milliseconds`, `seconds`, `minutes`, `hours` | `extern fn`   | A span of that many units.                                               |
| `zero`                                                                       | `extern fn`   | A span of no time.                                                       |
| `origin`                                                                     | `extern fn`   | The zero point of the monotonic scale, so an instant can be constructed. |
| `asNanoseconds`, `asMilliseconds`                                            | `extern fn`   | A span as a whole number of units.                                       |
| `asSeconds`, `asMinutes`, `asHours`                                          | `extern fn`   | A span as a number of units, including any fraction.                     |
| `absolute`                                                                   | `extern fn`   | A span without its sign.                                                 |
| `formatDuration`                                                             | `extern fn`   | A span as text, always readable back by `parseDuration`.                 |
| `parseDuration`                                                              | `extern fn`   | Reads a span such as `1.5s`, `2h45m` or `-250ms`, or `Err`.              |
| `since`                                                                      | `extern fn`   | The span between two instants.                                           |
| `between`                                                                    | `extern fn`   | The span between two timestamps.                                         |
| `fromEpoch`, `toEpoch`                                                       | `extern fn`   | Conversion between a timestamp and nanoseconds since the Unix epoch.     |
| `format`                                                                     | `extern fn`   | A timestamp in RFC 3339 form, in UTC.                                    |
| `parse`                                                                      | `extern fn`   | Reads an RFC 3339 timestamp, or `Err`.                                   |
| `year`, `month`, `day`, `hour`, `minute`, `second`                           | `extern fn`   | The calendar parts of a timestamp, in UTC.                               |

Time zones and calendar arithmetic such as "one month later" are deliberately absent: both need a zone database and rules that vary by locale, which is a larger commitment than a first version should make. Everything here is UTC.

### fun

`map`, `filter`, `flatMap`, `take`, `skip` and `zip` take any `@Iterable` and return a lazy `@Iterable`. `reduce` is eager.

| Declaration | Kind   | Description                                                                |
| ----------- | ------ | -------------------------------------------------------------------------- |
| `with`      | `fn`   | Applies a function to a value, without naming an intermediate variable.    |
| `pipe`      | `fn`   | Combines functions into one applying them left to right.                   |
| `identity`  | `fn`   | Returns its argument unchanged.                                            |
| `constant`  | `fn`   | A function always returning the same value, ignoring its argument.         |
| `negate`    | `fn`   | The boolean negation of a predicate.                                       |
| `compose`   | `fn`   | Combines two functions right to left, the mirror of `pipe`.                |
| `flip`      | `fn`   | A function with its two arguments swapped.                                 |
| `map`       | `fn`   | Lazily applies a transform to each element.                                |
| `filter`    | `fn`   | Lazily keeps the elements for which a predicate returns `true`.            |
| `flatMap`   | `fn`   | Lazily applies a transform and flattens each resulting `@Iterable`.        |
| `reduce`    | `fn`   | Folds the elements into a single value, left to right. Eager.              |
| `take`      | `fn`   | Lazily yields at most the first `n` elements.                              |
| `skip`      | `fn`   | Lazily omits the first `n` elements and yields the rest.                   |
| `zip`       | `fn`   | Lazily pairs two sequences by position, stopping when either is exhausted. |
| `Zipped`    | `data` | A pair of values at the same position in `zip`'s two sources.              |

### strings

Positions count characters, not bytes. Most of this module is Go: indexing has to decode UTF-8, and building a string from parts in Zirric would be quadratic.

| Declaration    | Kind        | Description                                                              |
| -------------- | ----------- | ------------------------------------------------------------------------ |
| `Like`         | `union`     | `Char` or `String`, accepted wherever text is expected.                  |
| `from`         | `extern fn` | Normalizes a `Like` to `String`.                                         |
| `count`        | `extern fn` | The number of characters, unlike `len`, which counts bytes.              |
| `charAt`       | `extern fn` | The character at a character index.                                      |
| `slice`        | `extern fn` | The characters from `start` (inclusive) to `end` (exclusive).            |
| `range`        | `fn`        | The characters selected by a `ranges.Like`.                              |
| `isEmpty`      | `fn`        | Whether the text has no characters.                                      |
| `contains`     | `extern fn` | Whether a needle occurs anywhere.                                        |
| `hasPrefix`    | `extern fn` | Whether the text starts with a prefix.                                   |
| `hasSuffix`    | `extern fn` | Whether the text ends with a suffix.                                     |
| `firstIndexOf` | `fn`        | The character index of a needle's first occurrence, or `None` if absent. |
| `lastIndexOf`  | `fn`        | The character index of a needle's last occurrence, or `None` if absent.  |
| `replace`      | `extern fn` | Replaces every occurrence of a target.                                   |
| `replaceFirst` | `extern fn` | Replaces only the first occurrence of a target.                          |
| `replaceLast`  | `extern fn` | Replaces only the last occurrence of a target.                           |
| `split`        | `extern fn` | Splits on every occurrence of a separator.                               |
| `join`         | `extern fn` | Joins parts with a separator between each.                               |
| `concat`       | `extern fn` | Concatenates every part, in order.                                       |
| `repeat`       | `extern fn` | The text repeated `n` times.                                             |
| `toUpper`      | `extern fn` | Uppercased, preserving whether the input was a `Char` or a `String`.     |
| `toLower`      | `extern fn` | Lowercased, preserving whether the input was a `Char` or a `String`.     |
| `trim`         | `extern fn` | Leading and trailing whitespace removed.                                 |
| `trimPrefix`   | `extern fn` | A leading prefix removed, if present.                                    |
| `trimSuffix`   | `extern fn` | A trailing suffix removed, if present.                                   |

### tests

A test is a function annotated with `@Test` returning a `Result`. `run` executes a list of `TestCase` and reports progress through an event callback, which keeps reporting independent of execution.

`tests` is only this vocabulary — declarations, no functions and no imports. Running and finding tests both live in `tests.runner`, so depending on the attributes and types drags in neither reflection nor a reporter.

| Declaration       | Kind    | Description                                                     |
| ----------------- | ------- | --------------------------------------------------------------- |
| `Test`            | `attr`  | Marks a function as a test case.                                |
| `Skip`            | `attr`  | Marks a test to be reported without being executed.             |
| `Only`            | `attr`  | Restricts the run to the annotated tests.                       |
| `Todo`            | `attr`  | Marks a test as expected to be incomplete.                      |
| `Comment`         | `attr`  | Attaches an explanatory note, surfaced by the reporter.         |
| `TestCase`        | `data`  | A single test: its name, implementation, and reporting details. |
| `TestEvent`       | `union` | Discovered, Skipped, Started, Finished and Completed.           |
| `TestCaseDetails` | `union` | Why a case is reported specially: skip, todo or only.           |
| `TestReport`      | `data`  | The passing, failing, skipped and todo cases of a run.          |
| `FailureRecord`   | `data`  | A failing case together with its error.                         |

#### tests.runner

Finds and runs tests. Kept apart from `tests` because it reaches for `reflect`, `reflect.packages`, `os` and a reporter, none of which writing a test requires.

`exec` is the primitive: it runs exactly the cases it is given, and neither discovers nor reports anything. `@Only` is applied there rather than during discovery, so restricting a run behaves the same however the cases were collected. Discovery loads only the modules its predicate accepts, so the rest never run. `runT` is the convention this repository follows and matches any module name ending in `_t`, which covers both a `foo/_t` submodule and a sibling module named `foo_t`; repositories that organize tests differently use `runWhere` instead. Both report TAP to stdout and exit non-zero when a test failed, so a task or CI step fails with the suite.

| Declaration     | Kind    | Description                                                              |
| --------------- | ------- | ------------------------------------------------------------------------ |
| `exec`          | `fn`    | Runs test cases, emitting events, and returns a `TestReport`.            |
| `ignoreEvents`  | `const` | An event callback that discards everything, for runs without output.     |
| `discover`      | `fn`    | The `@tests.Test` functions of a module, as runnable cases.              |
| `discoverAll`   | `fn`    | The `@tests.Test` functions of several modules, as runnable cases.       |
| `discoverWhere` | `fn`    | The cases of every project module whose name satisfies a predicate.      |
| `runWhere`      | `fn`    | Runs the cases of every project module whose name satisfies a predicate. |
| `runT`          | `fn`    | Runs the cases of every project module whose name ends with `_t`.        |

A discovered case is named `<module>.<function>`, so tests sharing a function name across modules stay distinguishable in the report.

#### tests.assert

Every assertion returns a `Result`, so a test body is an expression rather than a sequence of statements.

| Declaration | Kind | Description                                                 |
| ----------- | ---- | ----------------------------------------------------------- |
| `all`       | `fn` | `Ok` if every given result is `Ok`, otherwise the failures. |
| `equal`     | `fn` | Asserts that two values are equal.                          |
| `isTrue`    | `fn` | Asserts that a value is `true`.                             |
| `isFalse`   | `fn` | Asserts that a value is `false`.                            |
| `isOk`      | `fn` | Asserts that a result is `Ok`.                              |
| `isErr`     | `fn` | Asserts that a result is `Err`.                             |
| `isError`   | `fn` | Asserts that a value carries `@Error`.                      |
| `fail`      | `fn` | Fails unconditionally with a reason.                        |

#### tests.tap

| Declaration | Kind | Description                                                    |
| ----------- | ---- | -------------------------------------------------------------- |
| `reporter`  | `fn` | An event callback writing TAP version 14 output to a `Writer`. |

## Deferred

Running other programs is left out. A useful interface needs to write to a program's input while reading its output, and to run more than one at a time, neither of which is expressible while the language is single-threaded: anything built now would buffer everything and deadlock on a pipeline that fills a pipe. The shape it should take — a `Processes` value, a capability, and a separation between a program that fails and a program that will not start — survives in this proposal's history rather than in code, to be revisited once concurrency exists.

Time zones and calendar arithmetic belong to `time` and are described there.

## Alternatives Considered

**Methods on the `prelude` types instead of modules.** Helpers could have been added directly to `Array`, `String` and friends as fields on their `extern type`. That keeps calls shorter, but every helper would then have to be an `extern fn` implemented in Go, even the ones expressible in Zirric, and the `prelude` would grow without bound as the library does. Separate modules keep `prelude` to the types themselves and let most helpers be ordinary Zirric.

**Splitting by type rather than by domain.** `bytes` and `binaries`, or `chars` and `strings`, would mirror the types exactly. It was rejected because the boundary is arbitrary in practice: comparing two `Binary` values or searching a `String` for a `Char` belongs equally to both halves, so every new helper would need a placement decision that carries no information.

**Overloading instead of `Like` unions.** Accepting several types could have been expressed as overloaded declarations. A union plus a `from` normalizer was chosen instead because it needs no new language feature, it names the accepted set in one place that users can read, and it makes the normalization explicit rather than hidden in dispatch.

**Eager `fun` operations returning arrays.** `fun.map` and friends could have returned `[Any]` like their `arrays` counterparts. Laziness was chosen because these accept any `@Iterable`, including infinite ones, where an eager implementation cannot terminate, and because chained operations would otherwise allocate an intermediate array per step. The eager array versions remain available in `arrays` for the common case where the input is already an array.

**Type reflection instead of module reflection.** An earlier sketch (`future/reflect/stub.zirr`) described reflecting over types and fields — `typeOf`, `fieldsOf`, `Field`, `Attribute`. It is left as a sketch because nothing consumes it, while enumerating declarations is what test discovery and documentation generation both need. The two are independent axes and can be added separately.

**A lazy type per operation.** `MapSeq`, `FilterSeq` and so on would mirror the way some languages model lazy pipelines. A single `_lazy` wrapping one step closure was chosen because the operations differ only in that closure, so per-operation types would add declarations without adding capability, and each new operation would require another type.

## Acknowledgements

The module split by domain and the naming of `bytes`, `strings` and `errors` follow Go's standard library. The `Result` and `Option` helper sets — `map`, `mapErr`, `flatMap`, `catch` — follow Rust's `Result` and `Option` combinators, and Swift's `Optional` informs the `Some`-preserving behaviour of `options.from`. The lazy sequence operations in `fun` are modelled on Rust's iterator adapters and Swift's `LazySequence`, both of which likewise keep an eager collection-level API alongside the lazy one.
