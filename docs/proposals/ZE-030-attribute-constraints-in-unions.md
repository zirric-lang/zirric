---
title: "ZE-030 - Attribute Constraints in Unions"
description: "A union member may be an attribute constraint, letting a union mix a closed set of types with an open capability."
---

# Attribute Constraints in Unions

> **Draft.** This proposal is still a draft and is subject to change. It cannot be used right now.

## Introduction

A union member may be an attribute constraint: `@A`, or a chain such as `@A @B`. A value belongs to the union if its type is one of the named members **or its type carries every attribute of a constraint member.**

```zirric
union Target {
	String
	@io.Writer
}
```

This lets a union mix a closed set of types with an open capability, which today needs either an unhinted parameter or wrapper types. It adds no keyword, operator or concept: `@A @B` is the attribute constraint [ZE-017](https://zirric.knabel.dev/proposals/ZE-017-type-hints/) already defines for hints, `is` expressions and `switch` cases. Union bodies are the one type position that doesn't accept it yet.

## Motivation

Zirric has two ways to describe what a value may be. A union is a closed set of types. An attribute constraint is an open set: any type, in any module, that carries the attribute. Programs often need both at once, and there is no way to say so.

A CLI writes its report to the path given with `--out`, or to a writer: standard output in production, a buffer in tests.

**Today**, the parameter can't be hinted:

```zirric
// target: a path, or anything carrying @io.Writer.
fn save(fsys: fs.FileSystem, report: String, target) -> Result {
	switch target {
	case is String:
		return fs.writeString(fsys, target, report)
	case is @io.Writer:
		fmt.fprint(report, target)
		return Ok(void)
	case _:
		panic("target must be a path or an @io.Writer")
	}
}
```

The runtime handles this fine; `case is @io.Writer:` already works. The costs are elsewhere:

1. `save(fsys, report, 42)` is not reported, although the checker could know it is wrong.
2. The language server shows `target` as `Any`. The contract lives in a comment.
3. The `panic` branch exists only because the parameter admits everything.

Or the union wraps:

```zirric
union Target {
	data ToFile { path: String }
	data ToWriter { writer: @io.Writer }
}

save(fsys, report, ToWriter(os.stdout()))
save(fs.memory(), report, ToFile("report.txt"))
```

This keeps the hint but contradicts how unions work everywhere else: [a value does not need to be wrapped](https://zirric.knabel.dev/specification/typesystem/#union-types) to be a member. Every call site and every test pays for it, and the function body has to unwrap `target.writer` again.

**Proposed:**

```zirric
// Where a report goes: a file path, or any writer.
union Target {
	String
	@io.Writer
}

fn save(fsys: fs.FileSystem, report: String, target: Target) -> Result {
	switch target {
	case is String:
		return fs.writeString(fsys, target, report)
	case is @io.Writer:
		fmt.fprint(report, target)
		return Ok(void)
	}
}

save(os.fs(), report, "report.txt")
save(os.fs(), report, os.stdout())
save(fs.memory(), report, 42)   // reported: Int is not a member of Target
```

Callers pass what they have. Tests pass `fs.memory()` and a capturing writer without wrapping either.

## Proposed Solution

A union member is one of:

| Member                   | Written                 | A value is a member when…                                     |
| ------------------------ | ----------------------- | ------------------------------------------------------------- |
| Inline data              | `data Leaf { value }`   | its type is that `data`                                       |
| Named type               | `String`, `fs.Entry`    | its type is that `data` or `extern type`                      |
| Named union              | `Number`                | it is a member of that union, transitively                    |
| Named attribute type     | `tasks.Exec`            | it is an instance of that attribute, as reflection returns it |
| **Attribute constraint** | `@io.Writer`            | its type carries `@io.Writer`                                 |
| **Attribute chain**      | `@io.Reader @io.Writer` | its type carries every listed attribute                       |

The last two rows are new. They mean exactly what they mean after `:` or `is`, so the union is a named way of writing "one of these type expressions".

Note the difference between the named attribute type and the constraint. `Exec` matches the attribute instance itself; `@Exec` matches values whose type carries it. This is the same distinction `x is Exec` and `x is @Exec` already make.

A single-member union names a chain:

```zirric
// Something that can be both read and written, like an open fs.File.
union Duplex {
	@io.Reader @io.Writer
}
```

This is a consequence, not a goal: `fn pipe(stream: Duplex)` and `fn pipe(stream: @io.Reader @io.Writer)` accept exactly the same values.

## Detailed Design

### Grammar

Illustrative, extending the Syntax page's `UnionMember` with the `AttrRef` of ZE-017:

```ebnf
Union        = "union", Identifier, "{", { UnionMember }, "}" ;
UnionMember  = {Attribute}, Data, [","]
             | StaticReference, [","]
             | AttrRef, {AttrRef}, [","] ;          (* new *)
AttrRef      = "@", StaticReference ;               (* from ZE-017, no parentheses *)
```

- A chain ends at a newline or a comma. `@A @B` is one member; `@A, @B` and `@A` on one line followed by `@B` on the next are two.
- `@Name(` begins an attribute application to the following nested `data`, as today. `@Name` not followed by `(` begins a constraint member. Because applications always require parentheses, the parser needs one token of lookahead and nothing more.
- Member references, named or constraint, still cannot carry attributes. The grammar above drops `{Attribute}` from `StaticReference` members to match the [Declarations page](https://zirric.knabel.dev/specification/declarations/#union).
- Only `attr` declarations may follow `@`, as ZE-017's strict validation already requires. `@String` is a compile error here as everywhere.

### Runtime

`IsType` on a union checks each member in declaration order and stops at the first match. For a constraint member it performs the attribute check `is @A @B` already performs.

Attributes belong to closed declarations and never change while a program runs, so membership of a type in a union is a fixed fact. The VM may cache it per (union, type) and keep membership checks as cheap as today after the first lookup.

### Switch and narrowing

Nothing changes. `case is Target:` matches members; `case is @io.Writer:` narrows to the constraint as it does today. Switch expressions still require `_`, and statements still don't check exhaustiveness.

### Static checks

The rules extend [ZE-022](https://zirric.knabel.dev/proposals/ZE-022-static-checks/) and still **report only certainties.**

| Value known as                         | Fits `Target` when…                                                                        | Reported when…                                                                |
| -------------------------------------- | ------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------- |
| A named type (`Int`, `fs.Entry`)       | it is a named member, or its declaration carries every attribute of some constraint member | neither holds; the declaration is closed, so this is certain                  |
| A constraint (`@io.Writer @io.Reader`) | some constraint member's attributes are all among the known ones                           | never; the concrete type may still be a named member or carry more attributes |
| Another union                          | any of its members fits                                                                    | every one of its members is certainly excluded                                |
| Unknown                                | always                                                                                     | never                                                                         |

In the other direction, a `Target` value passed to a hint fits when any of its members fits. A constraint member counts as fitting any named type, since some type carrying the attribute might be it. This keeps the existing "in either direction" rule and adds no false alarms.

### Reflection

`reflect.unionMembers(t)` returns member types in declaration order. For a constraint member it returns a `reflect.AttrsType` whose `attributes` hold one `NamedType` per attribute, each `resolved` to the attribute type. `reflect.isInstance(value, t)` accepts an `AttrsType` and decides it as `is` does, so code that tests values against each member keeps working.

Code that assumes every element is a type (for example calling `reflect.typeName` on each) sees a new kind of value. See Compatibility.

### Tooling

- **Formatter:** one member per line, as today. A chain stays on one line. This keeps `@A @B` and `@A` / `@B` visibly different.
- **Language server:** hover on a constraint member shows the attributes; completion after `@` in a union body offers attribute types only.
- **Tree-sitter:** a new `union_constraint_member` node reusing the attribute-constraint node from type expressions.

### Edge cases

- **Overlapping members.** A named member that also carries a constraint's attribute is a member twice. That is allowed and harmless: first match wins in `switch`, and membership is a yes/no question.
- **Missing parentheses.** `@Printable` written directly above a nested `data`, meant as an application, now parses as a constraint member followed by an unannotated `data` instead of failing to parse. The formatter should insert a blank line after a constraint member that precedes a nested `data`, and the language server shows the member, so the mistake is visible. See open question 2.
- **Attributes from other modules.** `@io.Writer` resolves like any qualified reference and follows imports.
- **Nested unions.** Membership is transitive through named unions, including unions whose members are constraints.
- **Result and option shorthands.** `Target?` and `Target!` work as with any union.

### Deliberately not included

- **Composite type expressions as members** (`[Int]`, `fn(Event)`). `Array`, `Dict` and `Func` can already be named. Element types would promise a runtime check that `is` does not make.
- **Inline union types in hints** (`target: String | @io.Writer`). That is a new operator. Name the union instead.
- **Attributes on constraint members.**
- **Negation or exclusion** (`not @A`).
- **Exhaustiveness checking.** Zirric has none, and this proposal doesn't need it.
- **Types joining a union from their own declaration.** Membership is decided at the union, nowhere else.

## Changes to the Standard Library

- `reflect.unionMembers` may return `reflect.AttrsType` values; `reflect.isInstance` accepts them.
- `coding` must decide what to do when decoding into a union that has a constraint member it can't construct: skip the member, or report an error. _(verify how `coding` decodes unions today)_
- No existing declarations change. The API design guidance "closed set → union, open set → attribute" gains a third line: "a union may mix both".

## Compatibility

The syntax is purely additive: every union that parses today parses and behaves the same.

The reflection change is visible to code that walks `unionMembers` and treats every element as a type. No such code is known outside `coding` _(verify)_.

## Dependencies on Other Proposals

| Proposal                                                                                      | Dependency                                              |
| --------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| [ZE-017 Type Hints and Type Matching](https://zirric.knabel.dev/proposals/ZE-017-type-hints/) | `AttrRef`, attribute constraints, `is` semantics        |
| [ZE-022 Static Checks](https://zirric.knabel.dev/proposals/ZE-022-static-checks/)             | "Report only certainties"; union and constraint fitting |
| [ZE-023 Code Formatting](https://zirric.knabel.dev/proposals/ZE-023-code-formatting/)         | One member per line keeps chains and lists distinct     |

## Alternatives Considered

### Do nothing

Leave the parameter unhinted and `switch` with `case is @io.Writer:`. This costs nothing, and the runtime already supports it. It gives up the hint, the check and the documentation for exactly the parameters where a mistake is most likely: those that accept several shapes. In a language that asks library code to hint its public API, a common signature shape that can't be hinted is a gap, not a style choice.

### Wrapper data members

`union Target { data ToFile { path }, data ToWriter { writer } }` works today and is explicit. It is the right tool when the variants carry different meaning. Here it only exists to smuggle a constraint into a union: every caller wraps, every test wraps, the body unwraps, and a value that carries several capabilities (an open `fs.File` is both a reader and a writer) must be wrapped differently depending on where it goes. It contradicts the non-wrapping design of unions.

### Inline union type expressions

`fn save(target: String | @io.Writer)` expresses the same thing without a declaration. It adds an operator to type expressions, invites long anonymous types in signatures, and gives the language server nothing to name. A named union costs three lines and documents the concept.

### A `@MemberOf(Target)` attribute on member types

Reverses the direction: types opt into a union. Types from other modules, including every extern type, can't opt in, and a union's members would be scattered across the program. This breaks closed declarations.

### A keyword for constraint members

`union Target { String, has io.Writer }` avoids the parenthesis typo but adds a keyword for something `@` already means in every other type position.

## Open Questions

1. Should `reflect.unionMembers` use `reflect.AttrsType`, or a dedicated type for constraint members?
2. Is the missing-parentheses case worth a compile error, for example "a constraint member directly followed by a nested `data` on the next line"? It would be a style rule, not a certainty.
3. Attribute types as union members (`tasks.Task { Exec Call }`) are used in the standard library but not listed on the Declarations page. This proposal assumes they are intended and documents them alongside constraint members.

## Acknowledgements

- Builds on the attribute constraints of [ZE-017](https://zirric.knabel.dev/proposals/ZE-017-type-hints/).
- Prior art: TypeScript unions whose members may be interfaces (`string | Writable`), and Python's `Union[str, SupportsWrite]` with `typing.Protocol`. Zirric's version is nominal: a type carries the attribute on its declaration instead of matching a shape.
