---
title: Language Server
description: zirric lsp — diagnostics, navigation, completion, and formatting for any LSP client.
---

# Language Server

The language server is built into the `zirric` binary. There is nothing extra to install: if you have the compiler, you have the server.

```bash
zirric lsp                       # stdio, the default
```

It understands `.zirr` files and `Cavefile`s, and it reads the project's `Cavefile` to resolve dependencies — so completion and go-to-definition reach into the packages you actually depend on, not just the file in front of you.

## Transports

Most editors speak stdio and need no more than `zirric lsp stdio`. The others exist for clients that cannot spawn a process.

| Command                                | Transport                        |
| -------------------------------------- | -------------------------------- |
| `zirric lsp` / `zirric lsp stdio`      | stdin and stdout                 |
| `zirric lsp socket --listen HOST:PORT` | Socket, default `127.0.0.1:7998` |
| `zirric lsp tcp`                       | TCP, default `127.0.0.1:7998`    |
| `zirric lsp ipc`                       | Node.js IPC channel              |

## What it provides

| Feature           | Notes                                                                    |
| ----------------- | ------------------------------------------------------------------------ |
| Diagnostics       | Parse, analysis, and compile errors, refreshed on open, change, and save |
| Hover             | Declaration signature and doc comment                                    |
| Completion        | Triggered by typing, and by `@` for attributes and `.` for members       |
| Go to definition  | Across modules, including into dependencies                              |
| Find references   | Within the project                                                       |
| Rename            | With a prepare step, so the client can reject an invalid target          |
| Document symbols  | Outline of one file                                                      |
| Workspace symbols | Search declarations across the project                                   |
| Signature help    | Triggered by `(` and `,`                                                 |
| Formatting        | `textDocument/formatting`, the same layout `zirric fmt` writes           |

Documents sync incrementally, so the server sees your edits as you type rather than only on save.

## Diagnostics

Diagnostics are the same ones the compiler reports, at the same positions — the editor and `zirric run` never disagree about whether a program is valid. That includes the [static checks](/proposals/ZE-022-static-checks): undefined names, wrong arity, unsupported operators, unknown fields and members, and values that do not match a declared type hint.

Errors in an imported module surface too. A project module that does not compile is reported rather than silently dropped.

## Formatting on save

The server implements `textDocument/formatting`, so format-on-save works in any LSP client without configuring a separate formatter binary. The result is byte-for-byte what `zirric fmt` writes, including the project's `@cave.FormattingExcludes`. See the [Code Formatter](/tooling/code-formatter) for the layout itself.

## Commands

The server advertises a few workspace commands, which clients surface in a command palette:

| Command              | Effect                                           |
| -------------------- | ------------------------------------------------ |
| `zirric.install`     | Install the dependencies the `Cavefile` declares |
| `zirric.tasks`       | List the tasks the `Cavefile` declares           |
| `zirric.task.<name>` | Run that task — one command per declared task    |

Because each task gets its own command, a client's palette lists them by name instead of asking you which task to run.

## Setting it up

Editor-by-editor configuration lives in [Editor Configuration](/tooling/editor-configuration). Any client works: point it at `zirric lsp stdio` and associate it with `.zirr` and `Cavefile`.
