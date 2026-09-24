---
title: "ZE-031 - Attributes Required on Members"
description: "A prelude attribute, @RequiredOnMembers(), that makes the checker require a union's attribute on every one of its members."
---

# Attributes Required on Members

::: callout danger Rejected
This proposal has been rejected and will not be implemented.

Rejected in favour of [ZE-033 Result and Option as Constraint Unions](https://zirric.knabel.dev/proposals/ZE-033-result-and-option-as-constraint-unions/).

1. **The problem came from the markers, not from unions.** The only union-level attributes the language asks users to repeat on members are `@AnyResult` and `@AnyOption`. Once [ZE-030](https://zirric.knabel.dev/proposals/ZE-030-attribute-constraints-in-unions/) allows attribute constraints as union members, `Result` can be declared as `union Result { Ok, Err, @Error }`. Whether a value is a result is then decided by its own type, and there is no union-level claim left to check.
2. **It checked a rule instead of removing it.** Users would still write the attribute on the union and on every member, and the union-level application would still need a callback the runtime never calls.
3. **It added attributes about attributes.** A marker read by the checker on `attr` declarations is a new kind of rule for the checker to know and for users to learn. Without `@AnyResult` and `@AnyOption`, no known program needs it.
4. **The general case has no use case yet.** "Every member of this closed union carries `@Printable`" is not expressible after this decision. No stdlib or known program needs it. If one does, a union-side form (see Alternatives) can be proposed then.
   :::

## Introduction

A new prelude attribute, `@RequiredOnMembers()`, marks an attribute declaration. When an attribute marked this way is written on a union, **every member of that union must carry it too.** The checker reports a member that doesn't, at the union's declaration.

Union attributes still don't propagate. Nothing changes at runtime. The proposal turns a documented rule ("write `@AnyResult` on the union and on each member") into a checked one.

## Motivation

Attributes are read off a value's concrete type, and no value's concrete type is ever a union. An attribute on a union is therefore never seen by `is @A`, `A(value)`, `for`, `len`, `?.` or `!.`. The prelude documentation of `AnyResult` and `AnyOption` says so and asks users to repeat the attribute on each member. Forgetting it is silent until the program runs:

```zirric
@AnyResult(fn(r) { return r })
union Fetched {
	data Page { body: String }

	@Error(fn(e) { return "HTTP " + fmt.sprint(e.status) })
	data HttpFailure { status: Int }
}

fn titleOf(fetched: Fetched) -> String! {
	const body = fetched!.body   // runtime error: Page carries neither Ok/Err nor @AnyResult
	return Ok(firstLine(body))
}
```

The same holds for every capability attribute: `@Printable`, `@Iterable`, `@Countable` or `@Error` on a union say something about its members that nothing enforces.

**With this proposal**, the prelude declares:

```zirric
@RequiredOnMembers()
attr AnyResult {
	toResult(self: @AnyResult) -> Result
}
```

and the checker reports:

```
Fetched carries @AnyResult, but its member Page does not.
Fetched carries @AnyResult, but its member HttpFailure does not.
```

The fix is local:

```zirric
@AnyResult(fn(r) { return r })
union Fetched {
	@AnyResult(fn(page) { return Ok(page) })
	data Page { body: String }

	@AnyResult(fn(failure) { return Err(failure) })
	@Error(fn(e) { return "HTTP " + fmt.sprint(e.status) })
	data HttpFailure { status: Int }
}
```

The attribute's author decides once. Every union carrying it, in every module, is checked.

## Proposed Solution

```zirric
// Marks an attribute that describes what a value can do.
// Written on a union, such an attribute is a promise about every member,
// because attributes are only ever read off a value's own type.
attr RequiredOnMembers
```

**Rule:** if a union carries `@A`, and the declaration of `A` carries `@RequiredOnMembers()`, then every member of the union must carry `@A`:

| Member                                                                                                     | Carries `@A` when…                                                      |
| ---------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| Inline or named `data`                                                                                     | its declaration carries `@A`                                            |
| Named `extern type`                                                                                        | its declaration carries `@A`                                            |
| Named `attr` type                                                                                          | its declaration carries `@A`                                            |
| Named union                                                                                                | it carries `@A` itself (its members are checked at its own declaration) |
| Constraint `@X @Y` ([ZE-030](https://zirric.knabel.dev/proposals/ZE-030-attribute-constraints-in-unions/)) | `@A` is part of the chain                                               |

A member that doesn't is reported as a broken declaration, in the sense of [ZE-022](https://zirric.knabel.dev/proposals/ZE-022-static-checks/): the VM tolerates it, but the program doesn't mean it.

## Detailed Design

- The check reads declarations only. Members of a union are named at its declaration, and attributes are part of closed declarations, so the outcome is always certain. It never depends on values or on the order statements run in.
- `@RequiredOnMembers()` has an effect only on `attr` declarations. Written on anything else, it is reported as having no effect.
- Attributes without the marker keep today's meaning on unions: metadata about the union itself. `@Deprecated("…")` on a union does not require deprecated members.
- An extern type from another module can't be given new attributes. A union that lists one and carries a marked attribute it lacks is reported. That is correct: `@AnyOption union MaybeText { String, None }` claims a capability its `String` values don't have.
- A union-level application still needs its arguments (`@AnyResult(fn(r) { return r })`), although the runtime never calls them. This proposal doesn't change that; defaults and variadic attributes were rejected ([ZE-003](https://zirric.knabel.dev/proposals/ZE-003-named-data-construction/), [ZE-004](https://zirric.knabel.dev/proposals/ZE-004-Variadic-Attributes/)).

### Checking hints against unions

Separately from this proposal, and needing only a release note under ZE-022: when the checker decides whether a union-typed value satisfies `@A`, it should look at the members, never the union's own attributes, because the runtime never does. The value fits unless no member can carry `@A`. With `@RequiredOnMembers()`, a union carrying a marked attribute then satisfies it statically _because_ all its members do.

## Changes to the Standard Library

- Add `attr RequiredOnMembers` to the prelude (`prelude/attributes.zirr`).
- Mark the capability attributes. Candidates: `AnyOption`, `AnyResult`, `Error`, `Countable`, `Iterable`, `Numeric`, `Printable`. Not `Default` or `Deprecated`, which are metadata.
- Library capability attributes such as `io.Reader` and `io.Writer` can adopt it at their authors' discretion.
- Every stdlib union carrying one of the marked attributes must pass. `Option` and `Result` already annotate their members. _(verify the rest with the checker before marking)_

## Compatibility

Code that already follows the documented rule is unaffected. A union that carries a marked attribute on the union alone becomes a reported error. Such a union was already broken at runtime for every value that reaches `?.`, `!.`, `for` or `len` through that attribute, so the report surfaces an existing bug, in the spirit of ZE-022's note that reporting a certain failure earlier is not a breaking change.

## Dependencies on Other Proposals

| Proposal                                                                                                              | Dependency                                                |
| --------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| [ZE-022 Static Checks](https://zirric.knabel.dev/proposals/ZE-022-static-checks/)                                     | Broken declarations are reported; only certainties        |
| [ZE-030 Attribute Constraints in Unions](https://zirric.knabel.dev/proposals/ZE-030-attribute-constraints-in-unions/) | Optional. Defines how constraint members carry attributes |

## Alternatives Considered

### Do nothing

Keep the rule in the documentation of `AnyResult` and `AnyOption`. Costs nothing and remains correct for readers who know it. The failure it prevents is silent at the declaration and certain at runtime, and it affects user-defined result and option unions, which the sugar was designed to support.

### Propagate union attributes to members

Makes the annotation unnecessary. A type can be a member of several unions, so its capabilities would depend on every union that lists it anywhere in the program. That is action at a distance and breaks closed declarations.

### Fall back to the union's attribute at runtime

When `!.` finds no `@AnyResult` on the value's type, consult the union. A value doesn't know which union it was passed as, and it may belong to several.

### Hard-code the rule for `AnyResult` and `AnyOption`

Covers the two known cases without a new attribute. It is invisible to readers, special-cases the checker, and leaves library authors with the same problem for their own capability attributes.

### Require attributes from the union's side

`@Requires(Printable) union Shape { … }`: the union's author states the requirement. This also allows requiring an attribute the union doesn't carry itself. But the mistake it prevents is forgetting an annotation, and whoever forgets `@AnyResult` on a member will just as easily forget `@Requires`. The attribute's author knows once and for all whether it is a capability. The union-side form can be added later if a real program needs it.

### Report every attribute that is on a union but missing on a member

Needs no marker, but is wrong for metadata attributes (`@Deprecated`, `@json.Name`) and would produce false alarms, which ZE-022 ranks worse than missed reports.

## Acknowledgements

- The rule was already stated in prose on the prelude's `AnyResult` and `AnyOption` documentation; this proposal makes it checkable.
- Prior art: Java's `@Inherited` meta-annotation, where an annotation's declaration decides how it spreads. This proposal borrows the placement (a marker on the attribute's declaration) but not the behaviour: nothing spreads, a promise is checked. Prior art: Java's `@Inherited` meta-annotation, where an annotation's declaration decides how it spreads. This proposal borrows the placement (a marker on the attribute's declaration) but not the behaviour: nothing spreads, a promise is checked.
