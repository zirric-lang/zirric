---
title: "ZE-033 - Result and Option as Constraint Unions"
description: "Declaring Result and Option as unions of attribute constraints, so a value is a success, failure or present option by its own type as well as by being wrapped."
---

# Result and Option as Constraint Unions

::: callout draft Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design. Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

With attribute constraints as union members ([ZE-030](https://zirric.knabel.dev/proposals/ZE-030-attribute-constraints-in-unions/)), the prelude can say directly which values are results and options:

```zirric
union Result {
	Ok
	@Success
	@Error
}

union Option {
	Some
	None
	@Present
}
```

**A value is a success, a failure or a present option either by being wrapped** (`Ok`, `Err`, `Some`) **or by its own type** (`@Success`, `@Error`, `@Present`). One rule covers both unions: the wrappers are read through `.value`; a value whose type carries the attribute is read as it is.

`@AnyResult` and `@AnyOption` are removed. They were union-level markers that the runtime never reads off a union, and their conversion callbacks become unnecessary. Two zero-field attributes are added: `@Success` and `@Present`. `@Error` keeps its meaning and gains a role: an `@Error` value is a failed result.

## Motivation

### Results

Today a library with a typed page and typed failures defines its own result union and annotates it three times:

```zirric
@AnyResult(fn(r) { return r })
union Fetched {
	@AnyResult(fn(page) { return Ok(page) })
	data Page { body: String }

	@AnyResult(fn(failure) { return Err(failure) })
	@Error(fn(e) { return "HTTP " + fmt.sprint(e.status) })
	data HttpFailure { status: Int }
}

fn fetch(client: Client, url: String) -> Fetched { … }
```

The costs are concrete:

1. **The union-level `@AnyResult` does nothing.** Its callback is never called, because no value is ever a `Fetched` itself. It is required by convention only.
2. **Forgetting a member-level `@AnyResult` is silent.** `!.` on a value carrying neither `Ok`/`Err` nor `@AnyResult` is a runtime error, found on the first failing call rather than at the declaration. [ZE-031](https://zirric.knabel.dev/proposals/ZE-031-attributes-required-on-members/) proposed a checked marker for this and was rejected in favour of this proposal.
3. **Some unions can never work.** ZE-008's own example, `@AnyResult() union FileResult { String, FileNotFound }`, cannot be read by `!.`, because `String` can't carry `@AnyResult`.
4. **Every failure has two spellings.** `Err(HttpFailure(404))` and a member `HttpFailure(404)` are both failures, each with its own attribute plumbing, while `@Error` already says "this is an error".

With this proposal:

```zirric
// A page as it was fetched.
@Success()
data Page { body: String }

// A response the server refused.
@Error(fn(e) { return "HTTP " + fmt.sprint(e.status) })
data HttpFailure { status: Int }

fn fetch(client: Client, url: String) -> Page! {
	…
	if response.status >= 400 {
		return HttpFailure(response.status)
	}
	return Page(response.body)
}

fn titleOf(client: Client, url: String) -> String! {
	const body = fetch(client, url)!.body
	return Ok(firstLine(body))
}
```

Each type says what it is, once, on its own declaration. `titleOf` shows the wrapper is still there for values whose types can't carry an attribute: `Ok(firstLine(body))` wraps a `String`.

### Options

A configuration lookup in a CLI reports where a setting came from:

```zirric
// A setting as it was found, and where it came from.
@Present()
data Setting {
	key: String
	value: String
	source: String
}

fn lookup(config: Config, key: String) -> Setting? {
	for setting <- config.settings {
		if setting.key == key {
			return setting
		}
	}
	return None()
}

const origin = lookup(config, "editor")?.source ?? "default"
```

**Today** `lookup` either wraps every return in `Some`, or defines a custom option union with `@AnyOption` on the union and on each member, including a callback on `Setting` that wraps itself in `Some`. With this proposal `Setting` is present by its own declaration, and `?.` reads its fields directly. `Some` remains for values whose types can't carry `@Present`: `Some("vim")`, `Some(42)`.

## Proposed Solution

### The prelude

```zirric
// A result type: a success, or an error.
// Used for functions that can fail.
union Result {
	Ok
	@Success
	@Error
}

// An option type: a present value, or None.
// Used for values that may be missing.
union Option {
	Some
	None
	@Present
}

// Marks a type whose values are successful results by themselves,
// so they can be returned where a Result is expected without Ok.
attr Success

// Marks a type whose values are present options by themselves,
// so they can be returned where an Option is expected without Some.
attr Present
```

`Err` carries `@Error`, so it is a member of `Result` through the constraint. It stays for turning a reason that is not itself an error, typically a `String`, into one. `Ok`, `Some` and `None` carry none of the new attributes; they are members by name.

`@AnyResult` and `@AnyOption` are removed.

### One rule for both unions

|         | Wrapped, read through `.value`      | Marked, read as it is | Closed |
| ------- | ----------------------------------- | --------------------- | ------ |
| Success | `Ok { value }`                      | `@Success`            | —      |
| Failure | `Err { reason }` (carries `@Error`) | `@Error`              | —      |
| Present | `Some { value }`                    | `@Present`            | —      |
| Absent  | —                                   | —                     | `None` |

The wrappers exist for values whose types can't carry an attribute: strings, numbers, types from other modules. Everything else can say what it is on its own declaration.

### What the operators do

| Operator        | Target is…                         | Result                                      |
| --------------- | ---------------------------------- | ------------------------------------------- |
| `x!.field`      | `Ok`                               | reads `field` off `x.value`                 |
|                 | carries `@Success`                 | reads `field` off `x`                       |
|                 | carries `@Error` (including `Err`) | the enclosing function returns `x` as it is |
|                 | anything else                      | runtime error                               |
| `x !! fallback` | `Ok`                               | `x.value`                                   |
|                 | carries `@Success`                 | `x`                                         |
|                 | carries `@Error`                   | `fallback`                                  |
|                 | anything else                      | runtime error                               |
| `x?.field`      | `Some`                             | reads `field` off `x.value`                 |
|                 | carries `@Present`                 | reads `field` off `x`                       |
|                 | `None`                             | the chain is `None`                         |
|                 | anything else                      | runtime error                               |
| `x ?? fallback` | `Some`                             | `x.value`                                   |
|                 | carries `@Present`                 | `x`                                         |
|                 | `None`                             | `fallback`                                  |
|                 | anything else                      | runtime error                               |

The runtime error cases were already runtime errors; they no longer consult a conversion callback first.

A chain containing `?.` still always produces an `Option`: its result is wrapped in `Some` unless it already is a member of `Option`.

`T!` and `T?` still mean `Result` and `Option`. A function hinted `-> Page!` may return `Ok(x)`, any `@Success` value, `Err("reason")` or any `@Error` value. A function hinted `-> Setting?` may return `Some(x)`, any `@Present` value or `None()`.

## Detailed Design

### Membership

`x is Result` is true for `Ok` and for any value whose type carries `@Success` or `@Error`. `x is Option` is true for `Some`, `None` and any value whose type carries `@Present`. Both follow ZE-030. Membership is a property of the type and is cached per type after the first check.

### Standard library convention

**The standard library returns `@Error` values directly.** It uses `Err` only to wrap a reason that isn't an error itself. Everything that already has an error type (a file system failure, a decoding failure) is returned as it is.

Standard library success and present values stay wrapped in `Ok` and `Some` unless a type is only ever returned as a result. Carrying `@Success` or `@Present` is a statement about every value of a type, and most stdlib types (`fs.Entry`, `String`) are also used outside results and options.

### Static checks

Removing the conversion hooks makes results and options a question the checker can answer from declarations. Under ZE-022's **report only certainties**:

| Reported                                                                      | Example                           |
| ----------------------------------------------------------------------------- | --------------------------------- |
| a declaration carrying both `@Success` and `@Error`                           | `@Success() @Error(…) data Weird` |
| `!.` or `!!` on a value of a known type that is not a member of `Result`      | `const n = 42` then `n!.x`        |
| `?.` or `??` on a value of a known type that is not a member of `Option`      | `"text"?.length`                  |
| returning a known type that isn't a member from a `-> T!` or `-> T?` function | `fn f() -> Int! { return 42 }`    |

A type carrying both `@Success` and `@Error` would make `!.` ambiguous; the checker reports it as a broken declaration, and the runtime treats it as an error. A union-typed value is reported only when none of its members can be a member of `Result` (or `Option`), as for every union today.

### Matching

Code that tells success from failure matches the failure side, and code that tells presence from absence matches `None`. The other side then covers both the wrapped and the marked spelling:

```zirric
switch result {
case is @Error:
	report(errors.debug(result))
case _:
	use(results.value(result))
}

switch found {
case is None:
	useDefault()
case _:
	use(options.value(found))
}
```

`case is Err:` only matches the wrapper and misses bare errors; `case is @Error:` matches both, because `Err` carries `@Error`. `case is Ok:` and `case is Some:` likewise miss marked values.

### Custom result and option types

A custom union whose members are all members of `Result` is a result; one whose members are all members of `Option` is an option. The checker verifies this from the declarations; no annotation is needed:

```zirric
// What fetching a page can produce.
union Fetched {
	Page
	HttpFailure
}

// What looking up a setting can produce.
union Lookup {
	Setting
	None
}
```

A type that is a success or a failure depending on its value can no longer declare that through `toResult`. It converts explicitly with a function, visible at the call site:

```zirric
// Treats a response with an error status as a failure.
fn checked(response: Response) -> Response! {
	if response.status >= 400 {
		return HttpFailure(response.status)
	}
	return Ok(response)
}
```

### Edge cases

- **A marked type that holds its payload in a field.** `?.` and `!.` read fields off a marked value directly. A type `data Cached { value, age }` carrying `@Present` is read as the whole `Cached`, not as `value`. That is the intended meaning: the value _is_ what is present. A type that wants to expose only part of itself returns `Some(part)` or `Ok(part)`.
- **`Ok(x)` where `x` carries `@Success`**, or `Some(x)` where `x` carries `@Present`, is read through the wrapper and reaches the same `x`.
- **A type carrying `@Present` and `@Success` or `@Error`** is a member of both unions. Each operator reads its own union, so there is no ambiguity. It is allowed but rarely meaningful.
- **Marking is all or nothing.** `@Success` says every value of the type is a success. A type that is also used as plain data, outside results, should not carry it; the wrapper is the right tool there.

## Changes to the Standard Library

**Breaking:**

- `prelude`: remove `AnyResult` and `AnyOption`; redeclare `Result` and `Option` as above; add `attr Success` and `attr Present`; remove `@AnyResult`/`@AnyOption` from `Ok`, `Err`, `Some` and `None`.
- `results`: every `r: @AnyResult` parameter becomes `r: Result`, and every `fn(Any) -> @AnyResult` becomes `fn(Any) -> Result`. `results.from(r)` normalises instead of converting: a marked success becomes `Ok(value)` and a bare `@Error` value becomes `Err(value)`, so code that wants the old two-case shape can have it. `results.isOk(r)` and `results.isErr(r)` cover the marked spellings.
- `options`: `flatMap(o, transform: fn(Any) -> @AnyOption)` becomes `fn(Any) -> Option`. `options.from(o)` normalises an `@Present` value to `Some(value)` _(verify what it does for other values today)_. `options.isSome(o)` covers `@Present` values.
- Functions across the standard library that wrap an existing `@Error` value in `Err` return it directly.

**Added:**

- `prelude.Success`, `prelude.Present`.
- `results.value(r: Result) -> Any` and `options.value(o: Option) -> Any`: the value, read through the wrapper or taken as it is. They panic on a failure or on `None`, which is a bug in the caller. _(Names open; `!!` and `??` with a fallback may be enough.)_

**Unchanged:** `errors` already defines an error as "any value carrying `@Error`". This proposal makes `Result` agree with it.

## Compatibility

This is a breaking change to the prelude, `results`, `options`, and the error values the standard library returns.

| Existing code                                                  | After                                                                                                                                        |
| -------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `@AnyResult(…)` / `@AnyOption(…)` anywhere                     | compile error: unknown attribute. Delete it; successes gain `@Success()`, failures keep or gain `@Error`, present values gain `@Present()`.  |
| Custom result union annotated per member                       | works once `@AnyResult` is replaced as above                                                                                                 |
| Custom option union annotated per member                       | works once `@AnyOption` is replaced with `@Present()` on its present members and its absent case is `None`                                   |
| `switch r { case is Ok: … case is Err: … }`                    | misses bare errors, including those the standard library now returns; match `case is @Error:` and `case _:`, or call `results.from(r)` first |
| `switch o { case is Some: … case is None: … }`                 | misses `@Present` values; match `case is None:` and `case _:`                                                                                |
| `Ok(…)`, `Err(…)`, `Some(…)`, `None()`, `?.`, `!.`, `??`, `!!` | unchanged                                                                                                                                    |

The migration is mechanical except for `switch` on the wrappers; a migration note should list those first. The checker points at every `@AnyResult`/`@AnyOption`, since they become unknown names.

## Dependencies on Other Proposals

| Proposal                                                                                                              | Dependency                                                                                  |
| --------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| [ZE-030 Attribute Constraints in Unions](https://zirric.knabel.dev/proposals/ZE-030-attribute-constraints-in-unions/) | `@Success`, `@Error` and `@Present` as union members                                        |
| [ZE-008 Error Handling](https://zirric.knabel.dev/proposals/ZE-008-error-handling/)                                   | Amended: `@AnyResult` removed; `Result` redeclared; `@Success` added                        |
| [ZE-009 Option Values](https://zirric.knabel.dev/proposals/ZE-009-option-values/)                                     | Amended: `@AnyOption` removed; `Option` redeclared; `@Present` added                        |
| [ZE-019 Result and Option Sugar](https://zirric.knabel.dev/proposals/ZE-019-result-and-option-sugar/)                 | Amended: operators read membership instead of conversion attributes                         |
| [ZE-022 Static Checks](https://zirric.knabel.dev/proposals/ZE-022-static-checks/)                                     | New certain reports for `!.`, `!!`, `?.`, `??`, hinted returns and `@Success` with `@Error` |

## Alternatives Considered

### Do nothing

Keep `@AnyResult` and `@AnyOption` with the documented rule to annotate the union and every member. This is correct for readers who know the rule and costs no migration. It keeps union-level attributes that do nothing, a failure mode that only shows at runtime, and two unrelated ways of saying "this is an error" (`@Error`, and a `toResult` returning `Err`).

### Check the markers instead (ZE-031, rejected)

A prelude marker, `@RequiredOnMembers()`, would report unions carrying `@AnyResult` whose members don't. It fixes the silent failure but keeps all the annotations and the dead union-level callback, and adds attributes about attributes. Removing the markers removes the problem it checked.

### Open only the error side

`union Result { Ok, @Error }` and `union Option { Some, None }` add no attribute at all. Every success and every present value then has to be wrapped, and custom option unions, which `@AnyOption` existed for, disappear. Opening one side of each union but not the others would also need its own explanation; one rule for all three sides is easier to learn than two.

### Name the attributes after the wrappers: `@OkValue`, `@SomeValue`

These make the link to `Ok` and `Some` obvious. They were not chosen for three reasons:

1. **The failure side is already named by what a value is, not by its wrapper.** It is `@Error`, not `@ErrValue`. `@Success` pairs with `@Error`; `@OkValue` would be the only attribute named after a wrapper.
2. **They read as accessors.** `@OkValue()` suggests the attribute supplies a value, like `Ok`'s `value` field, and invites `OkValue(x).value`. The attributes have no fields; they state what a value is.
3. **The prelude already names state with adjectives and nouns.** `@Deprecated`, `@Default` and `@Error` describe their subject. `@Present` and `@Success` follow that shape.

`@Some` and `@Ok` are not available: an `attr` can't share a name with the `data` types in the same module.

### Attributes with an unwrap callback

`attr Success { value(self: @Success) -> Any }` would let a type say which of its fields is the value, so `!.` reads through it as through `Ok`. Every marked type would then need a callback, usually the identity (`@Success(fn(page) { return page })`). The zero-field markers cover the common case; a type that wants to expose only part of itself returns `Ok(part)` or `Some(part)`.

### Open the absent side too

`union Option { Some, @Present, @Absent }`, so that custom types can be absences (`NotConfigured`, `Unset`). Absence carries no information by definition, and a reason for being absent is a failure, which is what `Result` and `@Error` are for. It can be added later without breaking anything.

### Keep conversion hooks alongside membership

Keep `@AnyResult(toResult)` as an escape hatch for types that are sometimes failures. That is two mechanisms for one question, and it keeps results undecidable for the checker. An explicit `checked(response)` function does the same job visibly.

## Other Cases of the Pattern

The pattern is: **a concept whose members are partly fixed types and partly "anything carrying an attribute", written as a union with a constraint member instead of a marker attribute on every type.** These candidates were found in the v0.1.0 standard library and its target domain.

| Case                                                                       | Today                                                                                      | With ZE-030                                                                                                                      | Status                                                                                                   |
| -------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| **Result**                                                                 | `@AnyResult` on union and members                                                          | `union Result { Ok, @Success, @Error }`                                                                                          | This proposal                                                                                            |
| **Option**                                                                 | `@AnyOption` on union and members                                                          | `union Option { Some, None, @Present }`                                                                                          | This proposal                                                                                            |
| **Either randomness capability**                                           | a function takes `@HasFastRandom` and rejects an environment that only has a strong source | `union AnyRandom { @HasFastRandom, @HasStrongRandom }` in `random`                                                               | Proposed in [ZE-030](https://zirric.knabel.dev/proposals/ZE-030-attribute-constraints-in-unions/)        |
| **Output targets** in CLIs (`--out file` or stdout)                        | unhinted parameter or wrapper types                                                        | `union Output { String, @io.Writer }`                                                                                            | Application or library code; ZE-030's motivating example                                                 |
| **Input sources** (`-` means stdin)                                        | same                                                                                       | `union Input { String, @io.Reader }`                                                                                             | Same                                                                                                     |
| **A capability or its provider**: a writer, or an environment that has one | callers extract the writer                                                                 | `union WriterSource { @io.Writer, @io.HasStandardWriter }`                                                                       | Probably not: one explicit `HasStandardWriter(env).writer(env)` at the call site is clearer              |
| **Named chains**                                                           | repeated `@io.Reader @io.Writer` hints                                                     | `union Duplex { @io.Reader @io.Writer }`                                                                                         | Only where the name adds meaning; `fs.File` is the one stdlib type it describes                          |
| **Task declarations**                                                      | `union Task { Exec, Call }` matches attribute _instances_ found by reflection              | `union TaskDecl { @tasks.Exec, @tasks.Call }` would match _values of task types_, such as the instance a `Call` handler receives | Worth considering for handler hints. The existing union keeps its different meaning                      |
| **Numbers**                                                                | `union Number { Float, Int }` plus `@Numeric(toNumber)`                                    | `union NumberLike { Number, @Numeric }`                                                                                          | No. `@Numeric` is a conversion that `Int` and `Float` already carry, so the constraint alone covers both |
| **String-likes**                                                           | `union strings.Like { Char, String }`                                                      | could add `@Printable`                                                                                                           | No. It would change what `strings` functions accept and hide a conversion                                |
| **UI content** (zirric-ui, hypothetical)                                   | —                                                                                          | `union Child { String, @ui.View }`                                                                                               | For the ecosystem to decide. Same shape as output targets                                                |

Only `Result` and `Option` replace marker attributes. The others are new expressiveness and belong to the modules that own them.

## Open Questions

1. Final names for the markers: `@Success` and `@Present`, or the wrapper-based `@OkValue` and `@SomeValue` (see Alternatives)?
2. Are `results.value` and `options.value` worth adding, or are `!!` and `??` enough?
3. Which standard library functions currently wrap an existing `@Error` in `Err`, and do any callers inside the standard library match on `Err`?

## Acknowledgements

- Grew out of the discussion of [ZE-031](https://zirric.knabel.dev/proposals/ZE-031-attributes-required-on-members/), rejected in favour of this proposal. Defining `Result` and `Option` as unions of attribute constraints, and opening their success and present sides, came up in its review.
- Close to [ZE-019](https://zirric.knabel.dev/proposals/ZE-019-result-and-option-sugar/)'s original design, which detected errors by `@Error` alone.
- Prior art: Go's `error` interface, where any type implementing it is an error without wrapping. Zirric's version is nominal: a type carries `@Error` on its declaration.
