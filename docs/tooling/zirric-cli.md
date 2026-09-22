---
title: Zirric CLI
description: The zirric command — running code, managing a project, and driving every other tool.
---

# Zirric CLI

`zirric` is the only binary you need. The compiler, the language server, the formatter, and the package manager are all subcommands of it rather than separate downloads, so a single [installation](/guides/installation) gives you the whole toolchain.

## Commands

| Command                | Purpose                                         |
| ---------------------- | ----------------------------------------------- |
| `zirric <file>.zirr`   | Run a program by naming its file                |
| `zirric run`           | Run a Zirric program                            |
| `zirric repl`          | Start an interactive session                    |
| `zirric cave new`      | Create a new `Cavefile`                         |
| `zirric cave install`  | Install the dependencies a `Cavefile` declares  |
| `zirric cave describe` | Print the parsed `Cavefile` as YAML or JSON     |
| `zirric task`          | List the tasks a `Cavefile` declares            |
| `zirric task <name>`   | Run one of them                                 |
| `zirric <name>`        | The same, when nothing built in claims the name |
| `zirric fmt`           | Write the canonical source layout               |
| `zirric lsp`           | Run the language server                         |
| `zirric version`       | Print the version this binary was built as      |

`cave` answers to `cv`, and its subcommands to their first letter, so `zirric cv i` installs and `zirric cv n` starts a package.

## Running code

```bash
zirric main.zirr                 # run a file by naming it
zirric run main.zirr             # the same, said in full
zirric flow/node                 # a directory runs as the module it holds
zirric main.zirr a b c           # arguments reach the program through os.args()
zirric main.zirr --loud          # flags reach it the same way, unparsed by the CLI
```

A first argument ending in `.zirr`, or naming a directory that exists, is a program to run — unless a command, built in or a task, already answers to that name. `run` does not parse flags of its own, so a program's flags never collide with the CLI's. `os.args()` returns the script path followed by everything you passed:

```zirric
import fmt
import os

for arg <- os.args() {
	fmt.fprintln(arg, os.stdout())
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
zirric cave new                  # write a starter Cavefile
zirric cave install              # fetch what it declares
zirric cave describe             # show what the tooling actually read
zirric cave describe -o json     # ... as JSON

zirric cv i                      # cave install, for the hundredth time today
```

`cave new` fills in what the checkout already knows. In a Git repository it reads the remote, records it as `@cave.Git`, and names the package after it — `git@code.knabel.dev:example/widgets.git` becomes `mod code.knabel.dev.example.widgets`. Everything else can be said outright:

| Flag                  | Writes                  | Default                        |
| --------------------- | ----------------------- | ------------------------------ |
| `--mod`               | the `mod` declaration   | derived from the Git remote    |
| `--git-url`           | `@cave.Git`             | the Git remote, `origin` first |
| `--version`           | `@cave.Version`         | left out                       |
| `--language-version`  | `@cave.LanguageVersion` | `>=` the running Zirric        |
| `--description`       | `@cave.Description`     | left out                       |
| `--documentation-url` | `@cave.Documentation`   | left out                       |

A package has to be named, so `cave new` stops when neither `--mod` nor a Git remote gives it one. An attribute nothing was said about is left out rather than written empty.

```bash
$ zirric cave new --mod myapp --version 1.0.0 --description "Widgets and layout"
created Cavefile
```

```zirric
mod myapp

import cave

@cave.Package()
@cave.Version("1.0.0")
@cave.LanguageVersion(">=0.1.0")
@cave.Description("Widgets and layout")
data Myapp {
}
```

`zirric cave describe` is the one to reach for when a dependency or task does not behave as you expect: it prints the manifest as the tooling understands it, not as you wrote it.

## Tasks

Tasks are declared in the `Cavefile` and become real subcommands with their own typed flags and arguments.

```bash
zirric task                      # list tasks
zirric task generate             # run one
zirric generate                  # the same, when no built-in command is called generate
zirric generate --dry input      # flags and arguments are the task's own
```

A task is promoted to the top level only when nothing built in already answers to its name, so `zirric run` and `zirric fmt` never become something else. A task whose name a built-in holds is still reachable as `zirric task <name>`.

### Extending a built-in command

A few built-in commands call a task of the same name through themselves, so that a project can extend what they do:

```bash
zirric fmt                       # formats .zirr sources, then runs the `fmt` task
```

The task only runs when it accepts every flag that was actually passed. `zirric fmt --check` runs a `fmt` task that declares a `check` flag, and says so rather than running one that does not — a task with no way to be told to check would write where you asked only to look. `zirric fmt` exits zero only when both the formatter and the task do.

```
$ zirric fmt --list
main.zirr
note: task "fmt" takes no --list, so it was not run
```

`fmt` is the first of these; more commands will work the same way.

`test` is the other shape: a declared `test` task runs **instead** of the built-in, arguments and all, rather than beside it. With no such task, `zirric test` runs the standard library's runner over every module whose name ends in `_t` — so a project has a working `zirric test` before it has written anything about testing.

```
$ zirric test
TAP version 14
1..1
ok 1 myapp.thing._t.testAdd
```

## Choosing the Cavefile

```bash
zirric --cavefile ../other/Cavefile task build
```

`--cavefile` overrides autodetection and **must come before the subcommand**. Task commands deliberately pass their own arguments through untouched, so a `--cavefile` placed after the subcommand would be handed to the task instead of the CLI.

## Which Zirric is this

```bash
$ zirric version
0.1.0 (a1b2c3d4), v0.1.0, 2026-09-22T06:38:59Z, go1.25
```

`zirric --version` prints the same line. A build made straight from source carries no version of its own and says `devel`; a package that pins a `@cave.LanguageVersion` exempts such a build rather than refusing it.

When a package does pin one this Zirric does not satisfy, every command refuses it:

```
$ zirric fmt
Error: package "myapp" needs Zirric >=9.0.0, but this is 0.1.0
```

`zirric cave describe` is the exception — it still prints the manifest, so you can see what is being asked for.

## Errors

Errors carry a real line and column, and the CLI prints the source line with a caret under it. Parse, analysis, and compile errors from one run are reported together rather than one at a time.

```
Error: main/main.zirr:4:2: undefined: prnt is not declared anywhere in scope
4 | 	prnt("hello")
  | 	^
```

The same diagnostics reach your editor through the [Language Server](/tooling/language-server).

## Not there yet

`zirric lint` and `zirric docs` are not built in. Declare a task by that name and it becomes the command. [ZE-011 Zirric CLI](/proposals/ZE-011-zirric-cli) describes the CLI this one grew out of.
