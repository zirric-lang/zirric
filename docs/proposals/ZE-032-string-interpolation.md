---
title: "ZE-032 - String Interpolation"
description: "A string literal may embed an expression as \\(expression), rendered exactly as fmt.sprint renders it."
---

# String Interpolation

::: callout draft Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design. Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

A string literal may embed an expression as `\(expression)`. The value is rendered exactly as `fmt.sprint` renders it and spliced into the string.

**`"\(x)"` means `fmt.sprint(x)`. Nothing more.** There are no format specifiers, no custom interpolation handlers and no new literal kind.

## Motivation

Zirric's target is CLI and TUI programs, and most of what such a program does with its values is turn them into lines of text. Today that takes `+` and `fmt.sprint`, because `+` only concatenates strings and there are no implicit conversions:

```zirric
mod code.example.sync.report

import fmt

fn summary(copied: Int, total: Int, target: String, ms: Int) -> String {
	return "Copied " + fmt.sprint(copied) + " of " + fmt.sprint(total) + " files to " + target + " in " + fmt.sprint(ms) + "ms"
}
```

With interpolation:

```zirric
mod code.example.sync.report

fn summary(copied: Int, total: Int, target: String, ms: Int) -> String {
	return "Copied \(copied) of \(total) files to \(target) in \(ms)ms"
}
```

The costs of the current form are concrete:

1. **The message is unreadable.** The text a user will see is scattered across quote pairs; spaces at the edges of fragments are easy to drop (`"files to" + target`).
2. **A missing `fmt.sprint` fails at runtime in scripts.** `"Copied " + copied` is an unsupported operator. With a hint on `copied`, ZE-022 reports it. In a hint-less script the checker cannot know, and the program stops at runtime, on the line that was meant to report progress.
3. **Pure modules import `fmt` just to build text.** The module above only computes a string, yet its import list says it formats output. The stdlib's design makes imports signal capabilities ("you notice bad habits right at the import"); `fmt` in a data module is noise that hides real signal.
4. **The checker learns less.** A chain of `+` over unknown values has an unknown type. An interpolated literal is certainly a `String`.

## Proposed Solution

Inside a string literal, `\(` starts an interpolation and the matching `)` ends it. Between them is any expression.

| Source                                  | Value                                                    |
| --------------------------------------- | -------------------------------------------------------- |
| `"n = \(3)"`                            | `"n = 3"`                                                |
| `"\(a) + \(b) = \(a + b)"`              | each expression rendered in turn                         |
| `"Hello, \(user?.name ?? "stranger")!"` | nested string literals are fine                          |
| `"\(if ok { "ok" } else { "failed" })"` | expression forms are fine                                |
| `"\\(literal)"`                         | the text `\(literal)`; the backslash is escaped as today |

Rendering is `fmt.sprint`'s rendering: a value whose type carries `@Printable` uses its `toString`; built-in types fall back to their trivial conversion (`Int`, `Float`, `Char`, `Byte` as hex, and so on). Formatting is done with ordinary functions:

```zirric
const bar = "[\(strings.repeat("#", done))\(strings.repeat(".", width - done))]"
const cell = "\(strings.trim(name)) (\(strings.count(name)) chars)"
const list = "Missing: \(strings.join(missing, ", "))"
```

A user type decides how it looks by carrying `@Printable`, exactly as it does for `fmt.sprint` and `fmt.fprintln` today:

```zirric
@Printable(fn(v) { return "v\(v.major).\(v.minor).\(v.patch)" })
data Version {
	major: Int
	minor: Int
	patch: Int
}

const line = "Updated to \(version)"   // "Updated to v1.4.0"
```

### Deliberately not included

- **Format specifiers** (`\(x, width: 5)`, `{x:.2f}`). That is a second language inside the string. Padding, precision and alignment belong in functions.
- **Custom interpolation handlers** (Swift's `StringInterpolationProtocol`, JavaScript's tagged templates). A literal must mean the same thing everywhere; a hidden handler is magic.
- **Raw or multi-line string literals.** Separate concerns; they would get their own proposal.
- **Interpolation in `Char` literals.** `'\('` stays an error.
- **Implicit conversion in `+`.** `"n = " + 3` stays an error. Interpolation is the one place a value becomes text without a call, and it is visible in the literal.

## Detailed Design

### Grammar

Illustrative only. The `STRING` token of the [Syntax specification](https://zirric.knabel.dev/specification/syntax/) becomes a small production:

```ebnf
StringLiteral = '"' , { string_char | Interpolation } , '"' ;
Interpolation = backslash , "(" , Expression , ")" ;
string_char   = ? any character except '"' or backslash ? | escape | backslash , '"' ;
```

`escape` is unchanged. The lexer switches from string mode to expression mode at `\(` and back at the matching `)`, counting parentheses and entering string mode again for nested literals.

### Rules

| Rule                                          | Consequence                                                                                                         |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| The expression is any `Expression`.           | Calls, operators, `?.`, `!.`, `??`, `!!`, `if`/`switch`/`for` expressions and closures all work.                    |
| The expression must not be empty.             | `"\()"` is a syntax error.                                                                                          |
| The expression must not contain a line break. | Interpolations stay readable on one line; a `//` or `#` inside would swallow the closing `)` and is a syntax error. |
| Parts are evaluated left to right, each once. | Same as every other Zirric expression.                                                                              |
| Each value is rendered as by `fmt.sprint`.    | Rendering never fails; a value without `@Printable` gets the fallback rendering.                                    |
| The result is a `String`.                     | Known to the checker with certainty.                                                                                |

### Semantics

`"a\(x)b\(y)c"` evaluates to the same string as

```zirric
strings.concat(["a", fmt.sprint(x), "b", fmt.sprint(y), "c"])
```

but needs neither import. The compiler may emit a single string-building instruction; that is an implementation detail. The conversion used is the one `fmt.sprint` already calls, moved below `fmt` (into the VM next to `@Printable`, which ZE-018 placed in the prelude). No new public name is introduced; `fmt.sprint` stays, and its documentation gains one line: _`fmt.sprint(v)` is the same as `"\(v)"`._

### Interaction with existing features

| Feature                             | Interaction                                                                                                                                                                                                                     |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `?.` / `??`                         | `"\(user?.name)"` renders the `Option` itself, as `fmt.sprint` would. Write `"\(user?.name ?? "anonymous")"` to render the name.                                                                                                |
| `!.`                                | `"\(config!.path)"` inside a function returning a result returns the `Err` early, exactly as outside a string. The interpolation is an ordinary expression.                                                                     |
| `for` expressions                   | Produce arrays, which render as arrays. Use `strings.join(for …, ", ")` for lists.                                                                                                                                              |
| `switch` / `is`                     | An interpolated string is an ordinary `String` expression. It is not a constant, so it is not a literal wherever the grammar requires one.                                                                                      |
| Attributes                          | `@Printable` controls rendering; nothing else is looked up. Attribute arguments may contain interpolated strings like any expression.                                                                                           |
| Static checks (ZE-022)              | Each embedded expression is checked as usual (undefined names, unknown fields, wrong argument counts). Nothing new is reported about the interpolation itself: every value can be rendered, so there is no certainty to report. |
| Formatter (ZE-023)                  | Formats the embedded expression with the usual rules: `\( x+1 )` becomes `\(x + 1)`. No options.                                                                                                                                |
| Language server                     | Hover, completion, go-to-definition and diagnostics work inside `\( … )`; semantic tokens mark the delimiters.                                                                                                                  |
| Tree-sitter                         | `string` gains `string_content`, `escape_sequence` and `interpolation` children.                                                                                                                                                |
| `strings.quote` / `strings.unquote` | Unchanged. `quote` escapes backslashes, so its output never interpolates. `unquote` reads constant literals only and returns `Err` for text containing `\(`, as it does today for any unknown escape.                           |

## Changes to the Standard Library

None required. Documentation only:

- `fmt.sprint`: note the equivalence with `"\(value)"`.
- Guides and examples may drop `import fmt` where it was only used to build strings.

## Dependencies on Other Proposals

| Proposal                                                                      | Dependency                                                         |
| ----------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| [ZE-018](https://zirric.knabel.dev/proposals/ZE-018-io-fmt-os/)               | `@Printable` in the prelude and the rendering of `fmt.sprint`.     |
| [ZE-019](https://zirric.knabel.dev/proposals/ZE-019-result-and-option-sugar/) | `?.`, `!.`, `??`, `!!` inside interpolations.                      |
| [ZE-022](https://zirric.knabel.dev/proposals/ZE-022-static-checks/)           | Embedded expressions are checked; "report only certainties" holds. |
| [ZE-023](https://zirric.knabel.dev/proposals/ZE-023-code-formatting/)         | The formatter formats embedded expressions.                        |

## Compatibility

**Not a breaking change.** The Syntax specification states that any escape other than the listed ones is an error rather than literal text, so no program that compiles today contains `\(` inside a string. Every existing string keeps its meaning.

## Alternatives Considered

### Do nothing

The strongest alternative. `+` with `fmt.sprint` is explicit, already works, and costs nothing: no lexer modes, no tree-sitter changes, no new rule to teach. Every call is visible.

It loses because the cost lands on the language's main use case. Building output lines is the most common thing a CLI program does, and the current form makes that code hard to read, forces `fmt` into modules that don't format output, and turns a forgotten `fmt.sprint` into a runtime failure in exactly the hint-less scripts Zirric wants to support. Interpolation adds one concept and no new names, binding forms or scoping rules.

### A library function, no syntax

```zirric
const line = fmt.concat(["Copied ", copied, " of ", total, " files to ", target])
```

`fmt.concat(parts: [Any]) -> String` would render every part with `fmt.sprint` and join them. It needs no language change and removes the runtime failure. It keeps the fragmented, quote-heavy message and still requires `import fmt` for pure text. It is a reasonable fallback if this proposal is rejected, and small enough to add with a release note.

### A format function with placeholders

```zirric
const line = strings.format("Copied {} of {} files to {}", [copied, total, target])
```

A mini-language inside a string: placeholder syntax, escaping rules for `{`, and a count mismatch between placeholders and arguments that the checker cannot see. Rejected.

### Implicit conversion in `+`

`"n = " + 3` would just work. It breaks "no implicit conversions" for every use of `+`, not just inside strings, and makes `1 + "2"` a question. Rejected.

### `${expression}` (JavaScript, Kotlin, Dart)

Any existing string containing `${` (shell snippets, template text) would silently change meaning: a breaking change with no error. `\(` is currently an error, so it is free.

### Prefixed literals (`f"… {x} …"`, Python, C#)

A second kind of string literal to learn, braces that must be doubled in ordinary text, and one more token for every tool. `\(` reuses the escape character strings already have.

### Format specifiers or custom handlers

Covered under _Deliberately not included_: both add a sub-language or hidden behavior, and both can be expressed with function calls inside a plain interpolation.

## Open Questions

- **Line breaks.** Is the no-line-break rule worth its edge case, or should the formatter simply keep interpolations on one line?
- **Rendering `Option` and `Result`.** Interpolating `user?.name` renders `Some`/`None`, which is almost never what a user wants to see. Following `fmt.sprint` keeps one rule; a checker note would contradict "report only certainties". Is the documented idiom (`?? fallback`) enough?

## Acknowledgements

The `\(…)` syntax is taken from Swift, which shows it works without a prefix and composes with nested literals. Kotlin, JavaScript and Python informed the alternatives.
