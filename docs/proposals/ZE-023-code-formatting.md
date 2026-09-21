---
title: "ZE-023 - Code Formatting"
description: "A canonical whitespace layout for Zirric source, enforced by `zirric fmt` and applied by editors on save."
---

# Code Formatting

::: callout tip Implemented
This proposal has been accepted and implemented. You can use this feature since Zirric [v0.1.0](/changelog/v0.1.0).
:::

## Introduction

Zirric source has one canonical layout. `zirric fmt` writes it, and editors apply the same layout on save.

Formatting changes whitespace and nothing else:

> The formatter never changes whether two adjacent tokens share a line. It normalizes indentation, spacing between tokens, runs of blank lines, trailing whitespace, and the final newline. It does not wrap, join, split, align, reorder, or rewrite tokens.

The single exception is the deprecated `=>` return arrow, which is rewritten to `->`.

## Motivation

Before this proposal nothing formatted Zirric. The standard library had drifted into three styles at once: most files used tabs, the most-read files in `prelude/` used two spaces, and `prelude/result.zirr` mixed both within one declaration. The style guide covered naming and said nothing about whitespace, so every discussion about layout started from scratch.

A formatter ends those discussions, but only if it is not itself configurable. Projects that pick their own indentation reproduce the original problem one repository at a time, and code moved between them reformats on arrival. Zirric therefore offers no style options. The one thing a project may configure is which paths to leave alone, because vendored and generated sources are not the project's code to format.

Preserving the author's line breaks matters as much as choosing a style. Zirric has no statement terminator, and `(`, `[`, `-`, `+` and `.` are all infix operators, so a line beginning with one of them continues the previous expression rather than starting a new statement. A formatter that chose line breaks could silently change what a program means. Leaving them alone makes that impossible.

## Proposed Solution

### Canonical style

- **Indentation** is one tab per level. Bodies opened on the same line count as one level, so a closure inside an attribute argument indents once, not twice.
- **Braces** follow K&R: `{` ends the line that opens the block, and `}` sits at the indent of the construct that opened it.
- **`switch` cases** sit at the indent of their `switch`, with bodies one level deeper, ending at the next `case` or the closing brace.
- **Spacing** is one space between tokens, except before `)`, `]`, `,`, `:` and `.`; after `(`, `[`, `.`, `@` and `!`; between `{` and `}`; and before a `(` or `[` that calls or indexes rather than groups.
- **Blank lines** collapse to at most one. Blank lines directly after `{`, directly before `}`, and at the start of a file are removed.
- **Comments** keep their marker and text exactly. A comment introducing a `case` aligns with that `case`; trailing comments stay on their line.
- **Line length** is not enforced.

```zirric
@AnyResult(fn(r) { r })
union Result {
	// The successful result.
	data Ok {
		value
	}

	data Err {
		reason
	}
}
```

### Excluding paths

A project may exclude paths by declaring `@cave.FormattingExcludes` in its `Cavefile`:

```zirric
@cave.FormattingExcludes(["vendor/**", "**/*.generated.zirr"])
data Formatting {}
```

The attribute may sit on any declaration, including `mod`, so a project needs no placeholder type to carry it. Several occurrences accumulate.

Patterns are relative to the package root and use forward slashes. `*` matches within one path segment, `**` matches any number of segments including none, and a pattern naming a directory excludes everything beneath it.

Excluded files never count as unformatted, so a formatting check stays green. There is no setting for indentation, width, or any other aspect of style.

### When the Cavefile cannot be read

A project with **no `Cavefile`** has no excludes, and formatting proceeds.

A `Cavefile` that **does not parse** stops formatting, with an error. It may declare excludes that cannot be seen, and reformatting a file the project meant to exclude is worse than formatting nothing. `zirric fmt --no-excludes` bypasses the check.

## Detailed Design

Formatting is verified rather than assumed: the formatter checks that its output carries the same code and comments as its input, and a file that fails is reported and left exactly as it was. A defect therefore becomes a refusal to format rather than damaged source. Formatting is idempotent.

`zirric fmt` rewrites sources in place, and offers `--check`, `--list`, `--diff` and `--stdin` for scripting. Directory walks cover `.zirr` files and `Cavefile`s; a path named explicitly is always formatted. Editors need no configuration beyond enabling format-on-save, and produce identical results to the command line.

See [Code Formatter](/tooling/code-formatter) for usage.

## Changes to the Standard Library

The `cave` module gains one attribute:

```zirric
// Excludes paths from `zirric fmt` and from editor formatting.
attr FormattingExcludes {
	patterns: [String]
}
```

There are no breaking changes. A `Cavefile` using `@cave.FormattingExcludes` is ignored rather than rejected by an older `zirric`.

## Alternatives Considered

**Deciding line breaks.** A pretty printer that chooses where lines break to fit a width is both risky and unwanted here: with no statement terminator a wrongly placed break changes meaning, and the existing sources deliberately keep long lines, most of them unwrapped documentation sentences.

**Implying precedence through spacing**, as Go does when it writes `a + b*3`. Measured against the standard library, one expression in 349 mixes additive and multiplicative operators, and the rule cannot be reproduced faithfully because Zirric erases parentheses during parsing — `(a*b) + c` and `a*b + c` are indistinguishable afterwards. Not worth one line.

**Configurable style.** Rejected for the reason given under [Motivation](#motivation): configurable formatters recreate the problem they exist to solve.

**A separate configuration file.** A `Cavefile` is ordinary Zirric and already describes the package, so a second file and format would be added for one list of patterns.

## Acknowledgements

The approach follows `gofmt`: one canonical style, no options, and a tool expected to run on every save — including its choice to preserve the author's line breaks rather than compute them. The `**` pattern syntax follows the convention established by `.gitignore`.
