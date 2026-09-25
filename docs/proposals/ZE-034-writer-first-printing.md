---
title: "ZE-034 - Writer-First Printing"
description: "Swapping the parameters of fmt.fprint and fmt.fprintln, so the writer comes first as it does everywhere else in the standard library."
---

# Writer-First Printing

::: callout tip Implemented
This proposal has been accepted and implemented. You can use this feature since Zirric [v0.2.0](/changelog/v0.2.0).
:::

## Introduction

`fmt.fprint` and `fmt.fprintln` take the value first and the writer second. Every other function in the standard library that acts on a capability takes the capability first. This proposal swaps the two parameters so `fmt` follows the same rule:

```zirric
// before
fn fprintln(value: @Printable, writer: @io.Writer) -> Int

// after
fn fprintln(writer: @io.Writer, value: @Printable) -> Int
```

Nothing else changes. No new syntax, no new functions, no new concepts.

## Motivation

The [styleguide](https://zirric.knabel.dev/guides/styleguide/) states the rule plainly: **the subject comes first.** A function that operates on a value takes it as the first parameter, so the module reads as the namespace and the first argument as the subject. Its own examples are `fs.readFile(fsys, "notes.txt")` and `random.int(source, 100)`.

For printing, the subject is the writer. It is the thing being acted on: its state changes, bytes leave the program through it, and it is the capability a test swaps out. The value is only read. The standard library already agrees everywhere except in `fmt`:

| Call                                  | First parameter | Acted on   |
| ------------------------------------- | --------------- | ---------- |
| `fs.readFile(fsys, path)`             | capability      | capability |
| `fs.writeString(fsys, path, content)` | capability      | capability |
| `random.int(source, bound)`           | capability      | capability |
| `io.Writer.write(self, buf)`          | writer          | writer     |
| `tests.tap.reporter(writer)`          | writer          | writer     |
| `fmt.fprintln(value, writer)`         | **value**       | **writer** |

The inconsistency has four concrete costs.

1. **The styleguide contradicts itself.** Its section on `Has…` attributes shows a function that correctly takes the capability first, then calls `fmt` in the opposite order two lines later:

   ```zirric
   fn report(env: @HasStandardWriter, value: @Printable) {
   	const writer = HasStandardWriter(env).writer(env)
   	fmt.fprintln(value, writer)
   }
   ```

2. **The standard library is what people copy.** ZE-018's own example already spread the order into user code: `fn greet(name: String, out: Writer)`. A reader who learns the rule from `fs` and the order from `fmt` has to remember which one is the exception.

3. **Output code reads backwards.** A function that writes several lines repeats the writer on every call. With the writer last, the stable argument moves around at the end of lines of varying length, and the part that differs between calls is at the front:

   ```zirric
   // today
   fn summary(items: [Item], out: @io.Writer) {
   	fmt.fprintln("Open items:", out)
   	for item <- items {
   		fmt.fprintln("- " + item.text, out)
   	}
   }
   ```

   ```zirric
   // proposed
   fn summary(out: @io.Writer, items: [Item]) {
   	fmt.fprintln(out, "Open items:")
   	for item <- items {
   		fmt.fprintln(out, "- " + item.text)
   	}
   }
   ```

4. **The name promises the other order.** The `f` prefix comes from C's `fprintf(FILE *, …)` and Go's `fmt.Fprintln(w, …)`, both writer-first. Anyone who recognizes the name guesses wrong today.

Zirric is experimental and v0.1.0 has just shipped. This is the cheapest this change will ever be.

## Proposed Solution

Swap the parameters of the two writing functions in `fmt`. `sprint` has one parameter and is unaffected.

| Function   | Before                                                      | After                                                       |
| ---------- | ----------------------------------------------------------- | ----------------------------------------------------------- |
| `fprint`   | `fn fprint(value: @Printable, writer: @io.Writer) -> Int`   | `fn fprint(writer: @io.Writer, value: @Printable) -> Int`   |
| `fprintln` | `fn fprintln(value: @Printable, writer: @io.Writer) -> Int` | `fn fprintln(writer: @io.Writer, value: @Printable) -> Int` |
| `sprint`   | `extern fn sprint(value: Any) -> String`                    | unchanged                                                   |

A script that prints to standard output becomes:

```zirric
mod hello

import fmt
import os

fmt.fprintln(os.stdout(), "Hello, Zirric!")
```

Deliberately not part of this proposal:

- **No new printing functions.** No `print`, no `println`, no writer-bound printers. `fmt` stays free of `os`, as ZE-020 decided.
- **No renames.** The names stay `fprint` and `fprintln`.
- **No change to `@Printable`, `@io.Writer` or the return types.**
- **No transition period with both orders.** See Alternatives Considered.

## Detailed Design

The change is confined to `fmt/print.zirr`. Both functions swap their parameters, and `fprintln` passes them on in the new order:

```zirric
// Writes the string representation of a printable value to a writer.
fn fprint(writer: @io.Writer, value: @Printable) -> Int {
	// body unchanged
}

// Writes the string representation of a printable value followed by a newline to a writer.
fn fprintln(writer: @io.Writer, value: @Printable) -> Int {
	return fprint(writer, sprint(value) + "\n")
}
```

There is no grammar change and no change to the compiler, the VM, the formatter, the language server or the tree-sitter grammar.

### What happens to old call sites

An unmigrated call passes the value where the writer is expected. It never silently prints the wrong thing in the common cases:

| Old call                                             | What happens                                                                                                     |
| ---------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `fmt.fprintln("literal", os.stdout())`               | Reported by the static checker: a `String` where `@io.Writer` is declared (ZE-022, attribute constraint).        |
| `fmt.fprintln(name, out)` with `name: String` hinted | Reported by the static checker for the same reason.                                                              |
| `fmt.fprintln(value, out)` without hints             | Fails at runtime on the first call: the value has no `@io.Writer` to write through.                              |
| Both arguments carry `@Printable` and `@io.Writer`   | Runs with the roles swapped. This needs values carrying both attributes in both positions, which should be rare. |

Most real call sites are the first row: a string literal or a string concatenation followed by `os.stdout()` or `os.stderr()`. ZE-022's rule, **report only certainties**, covers exactly these, so the checker reports them before the program runs.

## Changes to the Standard Library

**Breaking:** `fmt.fprint` and `fmt.fprintln` swap their parameters, as shown in Proposed Solution.

Documentation that must follow the change:

| Page                                                                 | Change                                                    |
| -------------------------------------------------------------------- | --------------------------------------------------------- |
| [`fmt` reference](https://zirric.knabel.dev/stdlib/fmt/)             | Regenerated from the updated doc comments.                |
| [Getting Started](https://zirric.knabel.dev/guides/getting-started/) | `fmt.fprintln(os.stdout(), "Hello, Zirric!")`             |
| [Styleguide](https://zirric.knabel.dev/guides/styleguide/)           | The `report` example calls `fmt.fprintln(writer, value)`. |
| [Zirric CLI](https://zirric.knabel.dev/tooling/zirric-cli/)          | Example updated to `fmt.fprintln(os.stdout(), arg)`.      |

The [`scripts`](https://code.knabel.dev/zirric-lang/scripts) package calls `fmt` internally and needs a matching release. Its own public functions take only a value and are unaffected for its users.

The v0.1.0 changelog, ZE-018 and ZE-020 are not updated. Proposals are a witness of their time.

## Compatibility

This is a source-breaking change to two functions. Every program that writes output through `fmt` must swap two arguments. The migration is mechanical, and the static checker finds the common cases. It belongs in the release notes of the version that ships it, with a one-line before/after.

## Dependencies on Other Proposals

| Proposal                                                              | Dependency                                                            |
| --------------------------------------------------------------------- | --------------------------------------------------------------------- |
| [ZE-018](https://zirric.knabel.dev/proposals/ZE-018-io-fmt-os)        | Introduced `fmt.fprint` and `fmt.fprintln` in the order changed here. |
| [ZE-020](https://zirric.knabel.dev/proposals/ZE-020-standard-library) | Made `io.Writer` an attribute and kept `fmt` free of `os`.            |
| [ZE-022](https://zirric.knabel.dev/proposals/ZE-022-static-checks)    | Reports most unmigrated call sites before they run.                   |

## Alternatives Considered

### Do nothing

The strongest case for the current order: `fprintln(value, writer)` reads like the English sentence "print this value to that writer". Every existing program, guide and example uses it. `scripts` already hides the writer for quick scripts, so beginners rarely see the order at all. A breaking change for two parameters is churn that buys consistency, not capability.

It still loses. The styleguide states its rule without exceptions, and the standard library is where people learn the rules. Consistency is the point of a styleguide: the reader should never have to remember which module is different. The English reading also depends on a word Zirric cannot put in the call. Swift gets away with `print(value, to: &stream)` because of the `to:` label, and Python with `print(value, file=f)` because of the keyword. Zirric rejected named arguments in ZE-003, so the call site shows two bare arguments and nothing says which is which. Finally, the cost of the change only grows with every program written.

### Add an exception to the styleguide

The styleguide could say "sinks go last". This turns a rule with no exceptions into one with a category that needs defining: is a file system a sink when writing? `fs.writeString(fsys, path, content)` already puts the capability first while writing, so `fmt` would be the only member of that category. An exception for one module is the inconsistency, written down.

### New names, deprecate the old ones

Add writer-first functions under new names, for example `fmt.write` and `fmt.writeln`, mark `fprint` and `fprintln` with `@Deprecated`, and remove them a release later. This avoids a hard break, because both names can coexist.

It is the right fallback if a transition window is required. It is not the default because it leaves two spellings of the same operation in the standard library for a release, gives up the familiar `f`-prefixed names, and makes every program change anyway, just later. With the static checker catching the common cases, the direct swap is the smaller change.

### Accept both orders

`fprintln` could check which argument carries `@io.Writer` and act accordingly. This is magic: the call site no longer says what happens, a value that is both printable and a writer becomes ambiguous, and the static checker can no longer report a wrong call with certainty. Rejected.

### Writer-bound printers

A function like `fmt.printer(writer)` returning `fn(@Printable) -> Int` would remove the repeated writer from multi-line output. It is an addition, not a fix: it leaves `fprintln` inconsistent. It can be proposed separately if real programs need it.

## Acknowledgements

The printing functions and the `io`/`fmt`/`os` split come from ZE-018 and ZE-020 by Valentin Knabel. The rule this proposal applies is the styleguide's "The subject comes first".

Prior art for writer-first printing: C's `fprintf(FILE *stream, const char *format, …)`, Go's `fmt.Fprintln(w io.Writer, a ...any)` and Rust's `writeln!(w, …)`. Swift's `print(_:to:)` and Python's `print(…, file=)` put the target last, but only behind an argument label or keyword that Zirric does not have.
