---
title: "ZE-011 - Zirric CLI"
description: "Planning document for the Zirric Command Line Interface (CLI)"
---

# Zirric CLI Design

::: callout warning In Progress
This proposal has been accepted in principle.
It is currently under active development.
Parts might be incomplete or missing in Zirric.
:::

## Introduction

This proposal outlines the design and functionality of the Zirric Command Line Interface (CLI). The CLI aims to provide a seamless and efficient way for developers to interact with the Zirric programming language. The goal is the integration of all necessary tools and commands into a single, cohesive CLI, enhancing the overall developer experience.

## Motivation

Currently Zirric solely exists in unit tests, not as a usable programming language. This is one central step to getting started and to actually use Zirric.

## Proposed Solution

The goal is a single CLI tool named `zirric` that encompasses all functionalities required for developing, managing, and running Zirric projects. This includes commands for running code, managing dependencies, handling tasks, and interacting with the Language Server Protocol (LSP).

```bash
# global flag: overrides the Cavefile used by any command below (must precede the subcommand)
$ zirric --cavefile <path> ...

# default command
$ zirric

# Running Code
$ zirric run <path-to-file-or-module> [-- args...] [--flags...]

# REPL command
$ zirric repl
> "Hello, Zirric!"
- "Hello, Zirric!"

# Cave commands
$ zirric init     # creates a new Cavefile
$ zirric install  # installs dependencies from Cavefile
$ zirric cavefile # shows the parsed Cavefile contents

# Task commands
$ zirric task                                             # lists available tasks
$ zirric task run [--flags...] <task-name> [--] [args...]
$ zirric x <task-name> [--flags...] [--] [args...]        # shorthand for `task run`

# Project commands
$ zirric test # runs tests, can be overridden by user-defined task
$ zirric lint # lints code, can be overridden by user-defined task
$ zirric fmt  # formats code, can be overridden by user-defined task
$ zirric docs # generates documentation, can be overridden by user-defined task

# LSP subcommands
$ zirric lsp # defaults to stdio
$ zirric lsp stdio
$ zirric lsp ipc
$ zirric lsp tcp --port 7998 --bind 127.0.0.1 # both flags optional
$ zirric lsp socket --port 7998 --bind 127.0.0.1 # both flags optional

# Boring commands
$ zirric version
$ zirric help
$ zirric completion
```

## Detailed Design

When starting the `zirric` CLI, it needs to determine which command is being invoked before passing it to the appropriate handler.
If the `Cavefile` needs to be parsed for the command to come, this needs to be done first.

The CLI should only read arguments and flags until `--`. Remaining arguments are passed to the executed command or task as-is.

`--cavefile <path>` overrides autodetection of the project's Cavefile and must precede the subcommand: several task commands disable their own flag parsing so a task's own arguments pass through untouched, which means a `--cavefile` placed after the subcommand would never be recognized.

| Command      | Requires Cavefile? | Why?                |
| ------------ | ------------------ | ------------------- |
| root command | Yes                | For help            |
| `run`        | Lazy               | For dependencies    |
| `repl`       | Lazy               | For dependencies    |
| `init`       | No                 | Generates it        |
| `install`    | Lazy               | For dependencies    |
| `cavefile`   | Lazy               | To show it          |
| `task *`     | Yes                | For tasks           |
| `x`          | Yes                | For tasks           |
| `test`       | Yes                | For tasks and flags |
| `lint`       | Yes                | For tasks and flags |
| `fmt`        | Yes                | For tasks and flags |
| `docs`       | Yes                | For tasks and flags |
| `lsp *`      | No                 | Managed by LSP      |
| `version`    | No                 | Irrelevant          |
| `help`       | Yes                | For tasks           |
| `completion` | No                 | Irrelevant          |

## Changes to the Standard Library

None.

## Alternatives Considered

None.

## Acknowledgements

[Lithia](https://code.knabel.dev/lithia-lang/lithia) with the embedded `lsp`, `npm` and `task` with their scripts.
