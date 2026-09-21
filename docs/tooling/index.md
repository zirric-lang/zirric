---
title: Tooling
description: "The zirric CLI, editor setup, language server, formatter, package manager, and compiler."
---

# Tooling

The whole toolchain is one binary. The compiler, language server, formatter, and package manager are subcommands of `zirric` rather than separate downloads.

- [Zirric CLI](/tooling/zirric-cli) — every command, and how a project is run.

## Working in an editor

- [Editor Configuration](/tooling/editor-configuration) — per-editor setup for Helix, Neovim, and any other LSP client.
- [Language Server](/tooling/language-server) — `zirric lsp`, its transports, and the features it provides.
- [Code Formatter](/tooling/code-formatter) — `zirric fmt` and the canonical source layout.
- [Tree Sitter](https://code.knabel.dev/zirric-lang/tree-sitter-zirric) — the grammar repository behind syntax highlighting.

## Language implementation

- [Package Manager](/tooling/package-manager) — Cavefile structure, module discovery, and registry layout.
- [Compiler](/tooling/compiler) — bytecode and runtime architecture.

## Documentation pipeline

The documentation site is generated with docmd, configured in `tasks/docmd/docmd.config.js`. Markdown sources live under `docs/` and output to `site/`.

::: callout tip Contributing docs
When adding a new page, also add it to `tasks/docmd/docmd.config.js` so it appears in navigation.
:::
