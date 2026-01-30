---
title: Editor Support
description: Editor support for Zirric using Tree-sitter and LSP.
---

# Editor Support

Zirric provides syntax highlighting and language features for modern editors using [Tree-sitter](https://tree-sitter.github.io/) and [LSP](https://microsoft.github.io/language-server-protocol/).

The official Tree-sitter grammar lives in the separate repository:
[zirric-lang/tree-sitter-zirric](https://code.knabel.dev/zirric-lang/tree-sitter-zirric) and provides the core grammar and queries.

The LSP server is built into the Zirric itself.

For the insallation of Zirric, please refer to the [installation instructions](/guides/installation).

::: callout
Note that the Language Server still lacks basic features and is under active development.
:::

## Helix setup

Add a Zirric entry to `~/.config/helix/languages.toml`:

```toml
[[language]]
name = "zirric"
scope = "source.zirric"
grammar = "zirric"
language-servers = ["zirric"]
file-types = [
  "zirr",
  { glob = "Cavefile" },
]

[[grammar]]
name = "zirric"
source = { git = "https://code.knabel.dev/zirric-lang/tree-sitter-zirric", rev = "v0.1.0" }

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

## Neovim setup

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
  filename = { ["cavefile"] = "zirric" },
})
```

Configure LSP (example with `lspconfig`):

```lua
require("lspconfig").zirric.setup({
  cmd = { "zirric", "lsp", "stdio" },
  filetypes = { "zirric" },
})
```
