---
title: Tooling
description: "Editors, generators, and developer tooling for Zirric."
---

# Tooling

Zirric tooling is focused on editor support, documentation, and experimenting
with the language implementation. This section collects the available pieces.

## Editor support

- [Syntax Highlighting](./tooling/syntax-highlighting) with Tree-sitter for modern editors.

## Language implementation

- [Package Manager](./tooling/package-manager) for Cavefile structure and registry layout.
- [Compiler](./tooling/compiler) for bytecode and runtime architecture.

## Documentation pipeline

The documentation site is generated with docmd, configured in
`docs/docmd.config.js`. Markdown sources live under `docs/` and output to
`site/`.

::: callout tip Contributing docs
When adding a new page, also add it to `docs/docmd.config.js` so it appears in
navigation.
:::
