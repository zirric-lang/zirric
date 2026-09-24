---
title: "ZE-029 - Documentation Structure"
description: "A docs module that turns the modules of a package into a documentation structure, leaving rendering to later proposals."
---

# Documentation Structure

::: callout draft Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design. Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

`reflect` already knows everything a package says about itself: the doc comment above every module, declaration and field, the signature of each declaration, the members of each union and the type hint on each field. What it does not know is what documentation is. Which line is the summary, which declarations have fields worth listing, how a type hint reads as text, in which order the kinds of declarations appear, and which modules may be loaded at all are decisions every documentation tool makes again.

This proposal adds a standard library module, `docs`, that makes those decisions once. It turns the modules of a package into a plain tree of `data` values: a `docs.Package` holding `docs.Module`s holding `docs.Section`s holding `docs.Entry`s. It renders nothing and writes nothing. Turning that tree into Markdown, HTML, man pages or terminal output is deliberately left to future proposals, because each format has its own questions.

**The structure describes, the renderer decides.** Nothing in `docs` knows about a format.

## Motivation

Documentation is generated from source today: the standard library reference pages are built from doc comments, signatures and source positions. Anything else that wants the same — a package's own reference pages, a README section, a `docs` task in a Cavefile, a TUI that browses a package, a CI check for undocumented declarations — has to walk `reflect` by hand.

```zirric
import arrays
import reflect
import reflect.packages
import strings

for m <- packages.modulesWhere(packages.mainPackage(), isDocumented) {
	const meta = reflect.moduleOf(m)
	const summary = arrays.first(strings.split(strings.trim(meta.docs), "\n")) ?? ""
	// … print the module header …
	for d <- meta.declarations {
		const value = reflect.member(m, d.name) ?? void
		if d.kind is reflect.DataDecl || d.kind is reflect.AttrDecl {
			for f <- reflect.fieldsOf(value) {
				// switch over eight TypeRef cases to print the hint
			}
		}
		if d.kind is reflect.UnionDecl {
			for t <- reflect.unionMembers(value) {
				// reflect.typeName(t), reflect.docs(t)
			}
		}
		// … and group all of this by kind before printing anything
	}
}
```

That walk costs more than its length:

- **Every tool decides differently.** One tool takes the first line as the summary, another the first paragraph. One lists the fields of a `const` bound to a data type, another does not. One prints `[String: Int]`, another `Dict`. Documentation of the same package reads differently depending on who rendered it.
- **Structure and format are tangled.** The walk above interleaves what to say with how to print it. Adding HTML next to Markdown means writing the walk a second time, or threading a format flag through it.
- **Loading is a side effect.** Loading a module runs its top-level code. `reflect.packages` makes that explicit (`modulesWhere` tests names before loading, `modulesExcept` has no "all" shortcut). A hand-written walk that starts with "every module" quietly loads test modules and scripts along with the library.
- **Nothing is checkable without a renderer.** "Every public declaration has a summary" is a useful rule for a library, but today it can only be checked by the same code that prints pages.

With `docs`, the walk becomes a value:

```zirric
import docs
import fmt
import os
import reflect.packages
import strings

fn isDocumented(name: String) -> Bool {
	return !strings.hasSuffix(name, "_t")
}

const pkg = docs.packageWhere(packages.mainPackage(), isDocumented)
for m <- pkg.modules {
	fmt.fprintln(m.name + ": " + m.summary, os.stdout())
	for section <- m.sections {
		for entry <- section.entries {
			fmt.fprintln("  " + entry.name + ": " + entry.summary, os.stdout())
		}
	}
}
```

And the rule about summaries becomes an ordinary test, long before any renderer exists:

```zirric
mod code.example.app._t

import arrays
import docs
import reflect.packages
import strings
import tests
import tests.assert

@tests.Test()
fn testEveryDeclarationHasASummary() -> Result {
	const pkg = docs.packageWhere(packages.mainPackage(), fn(name) {
		return !strings.hasSuffix(name, "_t")
	})
	const missing = arrays.flatMap(pkg.modules, fn(m) {
		return arrays.flatMap(m.sections, fn(s) {
			return for e <- s.entries {
				if e.summary == "" { m.name + "." + e.name } else { continue }
			}
		})
	})
	return assert.equal([], missing)
}
```

## Proposed Solution

A new standard library module, `docs`, holds the structure and the functions that build it.

| Declaration                                       | What it is                                                                                                |
| ------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| `docs.Package`                                    | A package: its name, its summary and details from the Cavefile, its manifest, and the documented modules. |
| `docs.Module`                                     | A module: name, summary, details, imports, source files, and its declarations grouped into sections.      |
| `docs.Section`                                    | All declarations of one kind in a module, e.g. every union.                                               |
| `docs.Entry`                                      | One public declaration: summary, details, the `reflect.Declaration` itself, and its fields or cases.      |
| `docs.Field`                                      | One field of a data type or attribute, with its type hint as text.                                        |
| `docs.Case`                                       | One member of a union, with the member's own summary and details.                                         |
| `docs.package(p, modules)`                        | Documents the modules the caller already loaded.                                                          |
| `docs.packageWhere(p, predicate)`                 | Documents every module of `p` whose name satisfies `predicate`, loading only those.                       |
| `docs.module(m)`                                  | Documents one module.                                                                                     |
| `docs.summary(comment)` / `docs.details(comment)` | Splits a doc comment.                                                                                     |
| `docs.hint(ref)`                                  | Turns a `reflect.TypeRef` into the hint as it reads in source.                                            |

Defined terms:

- **Summary**: the first non-empty line of a doc comment, trimmed. The style guide already asks for this line to stand on its own.
- **Details**: everything after the summary line, with surrounding blank lines removed. Both stay Markdown exactly as written.
- **Section**: the entries of a module that were declared with the same keyword. Sections appear in a fixed order and empty ones are left out.

The whole module is built from `reflect`, `reflect.packages`, `strings` and `arrays`. It adds no syntax, no attribute and no new doc-comment convention.

## Detailed Design

### The structure

```zirric
mod code.knabel.dev.zirric_lang.zirric.docs

import arrays
import reflect
import reflect.packages
import strings

// The documentation of a package: what it says about itself, and the modules it was built from.
data Package {
	// The canonical name, e.g. "code.knabel.dev.zirric_lang.zirric".
	name: String
	// The first line of the comment above the Cavefile's mod declaration.
	summary: String
	// The rest of that comment.
	details: String
	// The manifest as reflect.packages reports it, or None for a project without a Cavefile.
	cavefile: Option
	// The documented modules, in the order they were given.
	modules: [Module]
}

// The documentation of one module.
data Module {
	// The canonical name, e.g. "code.knabel.dev.zirric_lang.zirric.tests.assert".
	name: String
	// The first line of the module's documentation, typically from its module-docs.zirr.
	summary: String
	// The rest of the module's documentation.
	details: String
	// The name of every module it imports, ordered by name.
	imports: [String]
	// The path of every file it was read from, ordered by name.
	sources: [String]
	// Its public declarations grouped by kind, empty sections left out.
	sections: [Section]
}

// The public declarations of a module that were declared in the same form.
data Section {
	// The form shared by every entry, e.g. a reflect.UnionDecl.
	kind: reflect.DeclarationKind
	// The entries, ordered by name.
	entries: [Entry]
}

// One public declaration.
data Entry {
	// The name it was declared under.
	name: String
	// The first line of its doc comment, or the empty String when it has none.
	summary: String
	// The rest of its doc comment.
	details: String
	// The declaration itself: signature, source position, kind, and the attributes written on it.
	declaration: reflect.Declaration
	// The fields of a data or attribute declaration, in declaration order; empty for every other kind.
	fields: [Field]
	// The members of a union declaration, in declaration order; empty for every other kind.
	cases: [Case]
}

// One field of a data type or an attribute.
data Field {
	// The name the field was declared under.
	name: String
	// The first line of the comment above the field.
	summary: String
	// The rest of that comment.
	details: String
	// The type hint as it reads in source, e.g. "[String: Int]", or the empty String when none was written.
	hint: String
	// The field itself, so that attributes written on it stay readable.
	field: reflect.Field
}

// One member of a union.
data Case {
	// The declared name of the member type.
	name: String
	// The first line of the member type's own doc comment.
	summary: String
	// The rest of that comment.
	details: String
}
```

`Entry.declaration` and `Field.field` keep the `reflect` values rather than copying them into strings. Attributes are read off a `reflect.Declaration` or a `reflect.Field` exactly as off any other value, so a renderer can write `entry.declaration is @Deprecated` and `Deprecated(entry.declaration).reason`, or read `@json.HasKey` off a field, without `docs` knowing that either attribute exists.

### Building a module

```zirric
// The documentation of m: its summary and details, and its public declarations grouped into sections.
fn module(m: AnyModule) -> Module {
	const meta = reflect.moduleOf(m)
	const entries = for d <- meta.declarations { _entry(m, d) }
	return Module(meta.name, summary(meta.docs), details(meta.docs), meta.imports, meta.sources, _sections(entries))
}

fn _entry(m: AnyModule, d: reflect.Declaration) -> Entry {
	const value = reflect.member(m, d.name) ?? void
	const fields = if d.kind is reflect.DataDecl || d.kind is reflect.AttrDecl {
		for f <- reflect.fieldsOf(value) { _field(f) }
	} else {
		[]
	}
	const cases = if d.kind is reflect.UnionDecl {
		for t <- reflect.unionMembers(value) { _case(t) }
	} else {
		[]
	}
	return Entry(d.name, summary(d.docs), details(d.docs), d, fields, cases)
}

fn _field(f: reflect.Field) -> Field {
	return Field(f.name, summary(f.docs), details(f.docs), hint(f.declaredType), f)
}

fn _case(t: Any) -> Case {
	const comment = reflect.docs(t)
	return Case(reflect.typeName(t), summary(comment), details(comment))
}
```

Fields and cases are taken by **declaration kind, not by value**. A `const Alias = Person` is a constant whose value happens to be a data type; it is documented as a constant, and it does not repeat `Person`'s fields.

### Sections

```zirric
// The order sections appear in, which is the order the standard library reference pages already use.
const _order = [reflect.UnionDecl, reflect.DataDecl, reflect.AttrDecl, reflect.TypeDecl, reflect.FuncDecl, reflect.ConstDecl, reflect.VarDecl, reflect.UnknownDecl]

fn _sections(entries: [Entry]) -> [Section] {
	const grouped = for kind <- _order {
		for e <- entries {
			if reflect.isInstance(e.declaration.kind, kind) { e } else { continue }
		}
	}
	return for group <- grouped {
		if len(group) == 0 { continue } else { Section(group[0].declaration.kind, group) }
	}
}
```

| Order | Kind                  | Has fields | Has cases |
| ----- | --------------------- | ---------- | --------- |
| 1     | `reflect.UnionDecl`   | no         | yes       |
| 2     | `reflect.DataDecl`    | yes        | no        |
| 3     | `reflect.AttrDecl`    | yes        | no        |
| 4     | `reflect.TypeDecl`    | no         | no        |
| 5     | `reflect.FuncDecl`    | no         | no        |
| 6     | `reflect.ConstDecl`   | no         | no        |
| 7     | `reflect.VarDecl`     | no         | no        |
| 8     | `reflect.UnknownDecl` | no         | no        |

Entries keep the order `reflect.moduleOf` reports, which is by name. A section carries a `reflect.DeclarationKind` rather than a title, so no English heading is baked into the structure; the renderer names it, typically with a `switch` over `is reflect.UnionDecl`, `is reflect.DataDecl` and so on.

A data type nested in a union is hoisted to module scope, so it appears twice: as a case of the union and as an entry of the Data section. That matches the language, where it is both.

### Summaries, details and hints

```zirric
// The first non-empty line of a doc comment, which is its summary.
fn summary(comment: String) -> String {
	return strings.trim(arrays.first(strings.split(strings.trim(comment), "\n")) ?? "")
}

// Everything in a doc comment after its summary line.
fn details(comment: String) -> String {
	const lines = strings.split(strings.trim(comment), "\n")
	return strings.trim(strings.join(arrays.slice(lines, 1, len(lines)), "\n"))
}

// A type hint as it reads in source, or the empty String when none was written.
fn hint(ref: reflect.TypeRef) -> String {
	return switch ref {
	case is reflect.NamedType:
		ref.name
	case is reflect.ArrayType:
		"[" + hint(ref.element) + "]"
	case is reflect.OptionType:
		hint(ref.element) + "?"
	case is reflect.ResultType:
		hint(ref.element) + "!"
	case is reflect.DictType:
		"[" + hint(ref.key) + ": " + hint(ref.value) + "]"
	case is reflect.FuncType:
		"fn(" + strings.join(for p <- ref.parameters { hint(p) }, ", ") + ")" + _returns(ref.returns)
	case is reflect.AttrsType:
		strings.join(for a <- ref.attributes { "@" + hint(a) }, " ")
	case _:
		""
	}
}

fn _returns(ref: reflect.TypeRef) -> String {
	return if ref is reflect.UnknownType { "" } else { " -> " + hint(ref) }
}
```

`summary`, `details` and `hint` are public because renderers need the same answers for text that is not part of the structure, such as a hint inside a signature they lay out themselves. Doc comments are never parsed: Markdown, links like ``[`Name`](#name)`` and code blocks pass through untouched.

### Building a package

```zirric
// The documentation of package p, built from modules the caller already loaded.
// Nothing else is loaded, so the caller decides which top-level code runs.
fn package(p: packages.Package, modules: [AnyModule]) -> Package {
	const comment = p.cavefile?.docs ?? ""
	const documented = for m <- modules { module(m) }
	return Package(p.name, summary(comment), details(comment), p.cavefile, documented)
}

// The documentation of every module of p whose name satisfies predicate.
// Each name is tested before its module is loaded, exactly as in packages.modulesWhere.
fn packageWhere(p: packages.Package, predicate: fn(String) -> Bool) -> Package {
	return package(p, packages.modulesWhere(p, predicate))
}
```

There is deliberately no `docs.everything(p)`. It would load every module, test modules and scripts included, and `reflect.packages` already decided that asking for all modules must be said out loud. `packages.modulesExcept(p, [...])` is the way to say it.

### Rules

1. Only public declarations are documented. A name starting with `_` is private by language rule; there is no second mechanism for hiding a declaration from documentation.
2. A module is documented only if the caller passed it in, or the predicate accepted its name before it was loaded.
3. Fields are listed for data and attribute declarations only, cases for union declarations only, both in declaration order.
4. Sections follow the fixed order above and empty sections are left out. Entries within a section are ordered by name.
5. Modules keep the order they were given in. Nesting (`tests`, `tests.assert`, `tests.runner`) is visible in the names and left to the renderer.
6. Doc comments are split into summary and details and otherwise passed through as written.

### Composition with the rest of the language

| Feature                     | Interaction                                                                                              |
| --------------------------- | -------------------------------------------------------------------------------------------------------- |
| `data`, `union`             | The structure is plain data; `Section.kind` is a member of the existing `reflect.DeclarationKind` union. |
| Attributes                  | Read off `Entry.declaration` and `Field.field` as off any value. `docs` declares none.                   |
| `is`, `switch`              | Renderers switch on `Section.kind`; `docs` itself narrows `TypeRef` with `case is`.                      |
| `for` expressions           | Every list in the structure is a `for` expression over `reflect` results.                                |
| Static checks               | All declarations carry hints, so a renderer gets field-level checking of its access to the structure.    |
| `==`                        | `data` equality compares fields, so a test can compare a built `docs.Module` against an expected one.    |
| Formatter, LSP, tree-sitter | Untouched.                                                                                               |

### Testing

`docs` takes modules as parameters and returns values; it reads no files and prints nothing. A test documents a fixture module and compares:

```zirric
mod code.knabel.dev.zirric_lang.zirric.docs._t

import docs
import reflect
import tests
import tests.assert
import code.knabel.dev.zirric_lang.zirric.docs._t.fixture

@tests.Test()
fn testSummaryAndDetails() -> Result {
	return assert.all([
		assert.equal("First line.", docs.summary("\nFirst line.\nSecond line.\n")),
		assert.equal("Second line.", docs.details("\nFirst line.\nSecond line.\n")),
		assert.equal("", docs.summary(""))
	])
}

@tests.Test()
fn testUnionCasesCarryMemberDocs() -> Result {
	const m = docs.module(fixture)
	const unions = m.sections[0]
	return assert.all([
		assert.isTrue(unions.kind is reflect.UnionDecl),
		assert.equal("Shape", unions.entries[0].name),
		assert.equal("A circle around the origin.", unions.entries[0].cases[0].summary)
	])
}
```

### Deliberately not part of this proposal

- **Rendering.** No Markdown, HTML, man page or terminal output. Each format raises its own questions (anchors, tables, line width, colors, paging) and deserves its own proposal.
- **Writing.** No files, no directories, no `fs` or `os` import.
- **A CLI command.** `zirric doc` can come later, on top of a renderer.
- **Cross-reference resolution.** Doc-comment links stay text. Resolving `Name` to the module that declares it needs a renderer's notion of a link target.
- **Doc-comment syntax.** No `@param` or `@returns` tags, no doc attributes. Signatures and hints already say what those would say.
- **Private declarations and bodies.** Only what `reflect` exposes as public, and only signatures.

## Changes to the Standard Library

A new module `docs` (`code.knabel.dev.zirric_lang.zirric.docs`), with a `module-docs.zirr` overview, containing:

| Kind      | Declarations                                                      |
| --------- | ----------------------------------------------------------------- |
| Data      | `Case`, `Entry`, `Field`, `Module`, `Package`, `Section`          |
| Functions | `details`, `hint`, `module`, `package`, `packageWhere`, `summary` |

Dependencies: `arrays`, `reflect`, `reflect.packages`, `strings`.

No existing module changes. Nothing is breaking.

## Compatibility

Purely additive. One interaction is worth naming: `packages.mainPackage()` leaves out any module whose unqualified name is already taken by a loaded module. A user package with its own module ending in `.docs` would be hidden from reflection once the standard `docs` module is loaded. This is the existing rule for every standard library name, but `docs` is a more likely directory name than `reflect` or `tasks`. See Open Questions.

## Dependencies on Other Proposals

| Proposal                                                                     | Dependency                                                                    |
| ---------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| [ZE-002](https://zirric.knabel.dev/proposals/ZE-002-the-cavefile/)           | The Cavefile's doc comment becomes the package summary.                       |
| [ZE-017](https://zirric.knabel.dev/proposals/ZE-017-type-hints/)             | The hint forms `docs.hint` turns back into text.                              |
| [ZE-020](https://zirric.knabel.dev/proposals/ZE-020-standard-library/)       | `reflect`, `reflect.packages`, `strings` and `arrays`.                        |
| [ZE-024](https://zirric.knabel.dev/proposals/ZE-024-qualified-module-names/) | Canonical module names, which make `Module.name` unambiguous across packages. |

## Alternatives Considered

**Do nothing.** `reflect` already is a structure, and everything in `docs` fits in a page of Zirric that any package could copy. This is the strongest alternative: the standard library should not grow for convenience alone. It loses because the value is not in the lines but in the agreement. Summary rules, which kinds have fields, hint text, section order and the loading rule should be the same for the reference pages, a package's own docs and any tool in between. Copied code drifts; a module does not.

**Extend `reflect` instead.** Adding `summary` or `hint` to `reflect` would avoid a new module. But `reflect` describes what was declared, and documentation is one view of it with opinions of its own (section order, the summary convention). Keeping those opinions out of `reflect` keeps it neutral for `coding`, `tests.runner` and `tasks`. `hint` is the one borderline case, see Open Questions.

**Structure and Markdown in one step.** A `docs.markdown(pkg, writer)` now would serve the reference pages immediately. It would also fix the structure to what Markdown needs, and the next format would inherit those choices. The `tests` family shows the better split: `tests` is the vocabulary, `tests.runner` and `tests.tap` are machinery that can be replaced. `docs` is the vocabulary; renderers follow.

**One entry type per kind.** A `union Entry { DataEntry, UnionEntry, FuncEntry, … }` would make every field meaningful for its kind. It also adds eight types to learn and a `switch` to every renderer that only wants names and summaries. The kind already exists as `reflect.DeclarationKind`; empty `fields` and `cases` cost less than a parallel union.

**Plain strings only.** Copying signature, source and kind into strings and dropping the `reflect` values would make the structure trivially encodable with `json` for tools outside Zirric. It would also lose the attributes, and with them `@Deprecated` and every attribute a future renderer wants to show. Keeping the `reflect` values is the smaller decision; an encodable projection can be added later without breaking anything.

**Extraction in the compiler (`zirric doc --json`).** Reading doc comments from the parser would run no top-level code and work on packages that do not compile. It is also a second implementation of what `reflect` already does, unusable from Zirric code, and commits to a wire format before any renderer exists. It remains possible later, producing the same structure.

**Doc attributes or tags.** `@docs.Summary("…")`, `@docs.Hidden()` or `@param` tags would give authors finer control. They are a second way to write what a comment and a `_` prefix already express, and tags are a small sub-language. Rejected.

## Future Directions

These are out of scope and named only to show the structure is enough for them:

- **Renderers** as submodules, e.g. `docs.markdown` and `docs.html`, each a function from a `docs.Package` to an `@io.Writer`, or to an `fs.FileSystem` for one file per module. The standard library reference pages would be the first consumer.
- **Terminal output** for CLI and TUI use, building on the external `term` and `md` packages.
- **A `zirric doc` command** or a conventional `docs` Cavefile task calling a renderer.
- **Link resolution** from doc-comment links to entries of the same `docs.Package`.

## Open Questions

- **The name.** `docs` reads best (`docs.module(m)`, `docs.Entry`) and leaves room for `docs.markdown`. It also collides more easily with a package's own `docs` directory, and with the common local name `docs`. `apidocs` is the fallback.
- **Where `hint` lives.** Turning a `TypeRef` into text is useful beyond documentation (error messages, the LSP). It may belong in `reflect` as `reflect.hintText`.
- **Home module of a case.** A union member may be declared in another module, and `reflect` cannot say which. `Case` has no `module` field until it can.
- **Encodability.** Whether `docs.Package` should encode with `json` as-is, which would decide whether `reflect` values may stay inside it.

## Acknowledgements

Go separates [`go/doc`](https://pkg.go.dev/go/doc), which extracts documentation into a structure, from the tools that render it (`go doc`, pkg.go.dev); this proposal follows that split. Rust's rustdoc JSON output and Swift's symbol graphs for DocC make the same point from the other side: once the structure is a value, formats multiply without touching extraction. The shape of `docs` itself follows the existing `tests` family, where vocabulary and machinery are separate modules.
