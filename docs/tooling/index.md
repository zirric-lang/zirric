---
title: Tooling
description: "Editors, generators, and developer tooling for Zirric."
---

# Tooling

Zirric tooling is focused on editor support, documentation, and experimenting with the language implementation. This section collects the available pieces.

## Editor support

- [Editor Support](/tooling/editor-support) with Tree-sitter and LSP for modern editors.
- [Formatter](/tooling/formatter) for `zirric fmt` and the canonical source layout.

## Language implementation

- [Package Manager](/tooling/package-manager) for Cavefile structure and registry layout.
- [Compiler](/tooling/compiler) for bytecode and runtime architecture.

## Documentation pipeline

The documentation site is generated with docmd, configured in `tasks/docs/docmd.config.js`. Markdown sources live under `docs/` and output to `site/`.

::: callout tip Contributing docs
When adding a new page, also add it to `tasks/docs/docmd.config.js` so it appears in navigation.
:::
