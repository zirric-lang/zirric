---
title: Editor Configuration
description: Set up Zirric in Helix, Neovim, or any other LSP-capable editor.
---

# Editor Configuration

A complete Zirric setup has three parts:

1. The [Tree-sitter grammar](https://code.knabel.dev/zirric-lang/tree-sitter-zirric), for syntax highlighting, folding, and text objects.
2. Its queries, which editors keep in their own runtime directory rather than alongside the parser.
3. The [Language Server](/tooling/language-server) — `zirric lsp stdio` — for diagnostics, navigation, completion, and format-on-save.

The language server ships inside the `zirric` binary, so [installing Zirric](/guides/installation) is all that is needed for the third part. Zirric files are `*.zirr`, plus the `Cavefile` at the root of every package.

## Helix

Add a Zirric entry to `~/.config/helix/languages.toml`:

```toml
[[language]]
name = "zirric"
scope = "source.zirric"
grammar = "zirric"
language-servers = ["zirric"]
roots = ["Cavefile"]
auto-format = true
indent = { unit = "\t", tab-width = 4 }
file-types = [
  "zirr",
  { glob = "Cavefile" },
]

[[grammar]]
name = "zirric"
source = { git = "https://code.knabel.dev/zirric-lang/tree-sitter-zirric", rev = "v0.3.0" }

[language-server.zirric]
command = "zirric"
args = ["lsp", "stdio"]
```

Fetch and build the grammar:

```bash
hx --grammar fetch
hx --grammar build
```

Download the queries into your Helix runtime directory:

```bash
mkdir -p ~/.config/helix/runtime/queries/zirric && curl -L https://code.knabel.dev/zirric-lang/tree-sitter-zirric/archive/main:queries.tar.gz | tar -xz --strip-components=1 -C ~/.config/helix/runtime/queries/zirric
```

`auto-format = true` routes saves through the language server's formatter, which writes exactly what `zirric fmt` writes.

## Neovim

With `nvim-treesitter`, add this to your config:

```lua
local parser_config = require("nvim-treesitter.parsers").get_parser_configs()
parser_config.zirric = {
  install_info = {
    url = "https://code.knabel.dev/zirric-lang/tree-sitter-zirric",
    files = { "src/parser.c", "src/scanner.c" },
  },
  filetype = "zirric",
}
```

Then install the parser:

```vim
:TSInstall zirric
```

Download the queries into your Neovim config directory:

```bash
mkdir -p ~/.config/nvim/queries/zirric && curl -L https://code.knabel.dev/zirric-lang/tree-sitter-zirric/archive/main:queries.tar.gz | tar -xz --strip-components=1 -C ~/.config/nvim/queries/zirric
```

Add filetype detection (example for `init.lua`):

```lua
vim.filetype.add({
  extension = { zirr = "zirric" },
  filename = { ["Cavefile"] = "zirric" },
})
```

Configure the language server (example with `lspconfig`):

```lua
require("lspconfig").zirric.setup({
  cmd = { "zirric", "lsp", "stdio" },
  filetypes = { "zirric" },
  root_dir = require("lspconfig.util").root_pattern("Cavefile"),
})
```

Format on save:

```lua
vim.api.nvim_create_autocmd("BufWritePre", {
  pattern = { "*.zirr", "Cavefile" },
  callback = function() vim.lsp.buf.format() end,
})
```

## Any other editor

Nothing about the server is editor-specific. Point your client at:

```
command: zirric
args:    ["lsp", "stdio"]
```

associate it with `*.zirr` and `Cavefile`, and use `Cavefile` as the root marker so the server resolves the project's dependencies. If your client cannot spawn a process, the server also speaks TCP, sockets, and Node IPC — see [Language Server](/tooling/language-server).

Without the Tree-sitter grammar you lose highlighting but keep everything else, since diagnostics, navigation, and formatting all come from the server.

## The grammar and its queries

The grammar lives in [zirric-lang/tree-sitter-zirric](https://code.knabel.dev/zirric-lang/tree-sitter-zirric) and is released under its own `v*` tags, independently of the compiler. It ships `highlights`, `folds`, `indents`, `injection`, `locals` and `textobjects` queries under `queries/`, which editors expect in their own runtime directory rather than next to the parser — which is why installing the grammar and installing the queries are two separate steps above.

Pin a tag rather than tracking `main`, and update the pin after each Zirric release: new syntax needs a grammar that knows about it, and an old grammar highlights new code incorrectly rather than failing loudly. The latest published tag is `v0.3.0`.

It is separate from the compiler's own parser on purpose. Tree-sitter parses incrementally and recovers from errors, which is what an editor needs while you type; the compiler's parser is what decides whether a program is valid.

## Indentation

Zirric indents with tabs, one per level, and the formatter writes it that way. Configure your editor to insert tabs in Zirric files so that what you type and what `zirric fmt` writes agree. See the [Code Formatter](/tooling/code-formatter) for the rest of the canonical layout.
