---
title: Zirric CLI
description: The zirric command — running code, managing a project, and driving every other tool.
---

# Zirric CLI

`zirric` is the only binary you need. The compiler, the language server, the formatter, and the package manager are all subcommands of it rather than separate downloads, so a single [installation](/guides/installation) gives you the whole toolchain.

## Commands

| Command           | Purpose                                        |
| ----------------- | ---------------------------------------------- |
| `zirric run`      | Run a Zirric program                           |
| `zirric repl`     | Start an interactive session                   |
| `zirric init`     | Create a new `Cavefile`                        |
| `zirric install`  | Install the dependencies a `Cavefile` declares |
| `zirric cavefile` | Print the parsed `Cavefile` as YAML or JSON    |
| `zirric task`     | List the tasks a `Cavefile` declares           |
| `zirric task run` | Run one of them (`zirric x` is the shorthand)  |
| `zirric fmt`      | Write the canonical source layout              |
| `zirric lsp`      | Run the language server                        |

## Running code

```bash
zirric run main.zirr             # run a file
zirric run main.zirr a b c       # arguments reach the program through os.args()
zirric run main.zirr --loud      # flags reach it the same way, unparsed by the CLI
```

`run` does not parse flags of its own, so a program's flags never collide with the CLI's. `os.args()` returns the script path followed by everything you passed:

```zirric
import os
import scripts { println }

for arg <- os.args() {
	println(arg)
}
```

```
$ zirric run main.zirr a b
main.zirr
a
b
```

## Exploring interactively

```bash
zirric repl
> "Hello, Zirric!"
- "Hello, Zirric!"
```

The REPL sees the current project's dependencies, so you can import and poke at a module without writing a file first.

## Project commands

A project is a directory with a `Cavefile`. See [Cavefile manifests](/cavefile) for what goes in one, and the [Package Manager](/tooling/package-manager) for how dependencies are resolved.

```bash
zirric init                      # write a starter Cavefile
zirric install                   # fetch what it declares
zirric cavefile                  # show what the tooling actually read
zirric cavefile -o json          # ... as JSON
```

`zirric cavefile` is the one to reach for when a dependency or task does not behave as you expect: it prints the manifest as the tooling understands it, not as you wrote it.

## Tasks

Tasks are declared in the `Cavefile` and become real subcommands with their own typed flags and arguments.

```bash
zirric task                      # list tasks
zirric task run generate         # run one
zirric x generate                # the same, shorter
zirric x generate --dry input    # flags and arguments are the task's own
```

## Choosing the Cavefile

```bash
zirric --cavefile ../other/Cavefile task run build
```

`--cavefile` overrides autodetection and **must come before the subcommand**. Task commands deliberately pass their own arguments through untouched, so a `--cavefile` placed after the subcommand would be handed to the task instead of the CLI.

## Errors

Errors carry a real line and column, and the CLI prints the source line with a caret under it. Parse, analysis, and compile errors from one run are reported together rather than one at a time.

```
Error: main/main.zirr:4:2: undefined: prnt is not declared anywhere in scope
4 | 	prnt("hello")
  | 	^
```

The same diagnostics reach your editor through the [Language Server](/tooling/language-server).

## Not there yet

`zirric test`, `zirric lint`, and `zirric docs` are reserved in the CLI but do not run anything yet; they appear under "Additional help topics". The plan, including the intent that a project can override each of them with a `Cavefile` task, is in [ZE-011 Zirric CLI](/proposals/ZE-011-zirric-cli). Until then, declare a task and run it with `zirric x`.
