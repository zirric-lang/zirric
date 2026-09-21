---
title: Formatter
description: "zirric fmt writes Zirric's canonical layout, in the terminal and in your editor."
---

# Formatter

`zirric fmt` writes Zirric's canonical layout. Editors apply the same layout on save, so the terminal and your editor never disagree.

Formatting only ever changes whitespace.

## Usage

```bash
zirric fmt                     # rewrite every source below the current directory
zirric fmt src/ main.zirr      # rewrite specific paths
zirric fmt --check             # exit 1 if anything is unformatted, write nothing
zirric fmt --list              # print the paths that would change
zirric fmt --diff              # print a unified diff
zirric fmt --stdin             # format stdin to stdout
zirric fmt --no-excludes       # ignore the project's formatting excludes
```

With no arguments it rewrites every `.zirr` file and every `Cavefile` below the current directory, skipping `.git`, `node_modules`, `site` and `testdata`. A path you name explicitly is always formatted, whatever it is called.

Use `--check` in continuous integration:

```yaml
- name: Format
  run: go run ./cmd/zirric fmt --check .
```

In editors, enable format-on-save. There is nothing else to configure.

## The style

Zirric has one canonical style and no options to change it. The formatter's value is that layout is never up for discussion.

- Indent with **tabs**, one per level. Bodies opened on the same line count as one level.
- **K&R braces**: `{` ends the line that opens the block, `}` sits at the indent of the construct that opened it.
- **`switch` cases** sit at the indent of their `switch`, with bodies one level deeper.
- One space between tokens, except before `)`, `]`, `,`, `:` and `.`, after `(`, `[`, `.`, `@` and `!`, between `{` and `}`, and before a `(` or `[` that calls or indexes.
- At most **one consecutive blank line**. Blank lines directly after `{`, directly before `}`, and at the start of a file are removed.
- Comments keep their marker and text exactly; trailing comments stay on their line.
- The deprecated `=>` return arrow is rewritten to `->`.

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

fn classify(v) {
	switch v {
	case is String:
		return "text"
	case _:
		return "other"
	}
}
```

### Line breaks are yours

The formatter never changes whether two tokens share a line. It does not wrap long lines, join short ones, or align columns.

This is not only taste. Zirric has no statement terminator, and `(`, `[`, `-`, `+` and `.` are infix operators, so a line starting with one of them continues the previous expression:

```zirric
const total = count
- 1 // this is `count - 1`, not a new statement
```

A formatter that moved line breaks could silently change what a program means.

## Excluding paths

Vendored and generated sources are not yours to format. Declare them in your `Cavefile`:

```zirric
@cave.FormattingExcludes(["vendor/**", "**/*.generated.zirr"])
data Formatting {}
```

The attribute can sit on any declaration, including `mod`, and several occurrences accumulate. Patterns are relative to the package root and use forward slashes:

| Pattern               | Matches                                      |
| --------------------- | -------------------------------------------- |
| `vendor/**`           | everything below `vendor/`                   |
| `vendor`              | the same — naming a directory excludes it    |
| `**/*.generated.zirr` | that suffix at any depth                     |
| `src/*.zirr`          | `.zirr` files directly in `src/`, not deeper |
| `Cavefile`            | exactly that path                            |

`*` matches within one path segment and `**` matches any number of segments, including none.

Excluded files never count as unformatted, so `--check` stays green. Naming an excluded file explicitly reports why it was skipped:

```
vendor/dep.zirr: skipped by @cave.FormattingExcludes (use --no-excludes to format it)
```

There is no setting for indentation or width — only for which paths to leave alone.

### When the Cavefile cannot be read

A project with **no `Cavefile`** simply has no excludes, and formatting proceeds.

A `Cavefile` that **does not parse** stops formatting, with an error:

```
Error: Cavefile:4:1: unexpected "data": want one of [!, (, +, -, char, …]
Cavefile is malformed, so its formatting excludes cannot be read; fix it or pass --no-excludes
```

It may declare excludes that cannot be seen, and reformatting a file you meant to exclude is worse than formatting nothing. `--no-excludes` skips the check when you need it.

## Guarantees

A file the formatter cannot account for is reported and left exactly as it was, so a defect becomes a refusal to format rather than damaged source. Formatting twice gives the same result as formatting once.

See [ZE-023 Code Formatting](/proposals/ZE-023-code-formatting) for the design and the reasoning behind it.
