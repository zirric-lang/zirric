---
title: Syntax Highlighting with Tree-sitter
description: Syntax highlighting and parsing for Zirric using Tree-sitter.
---

# Syntax Highlighting with Tree-sitter

The official Tree-sitter grammar lives in the separate repository:
https://code.knabel.dev/zirric-lang/tree-sitter-zirric

It provides the core grammar, queries, and generated parser sources used by
editors.

## Build the grammar locally

```bash
npm install
npm run gen
```

- `grammar.js` contains the grammar definition.
- `queries/` contains editor query files.
- `src/` contains generated artifacts.

## Helix setup

Add a Zirric entry to `~/.config/helix/languages.toml` (or your system config):

```toml
[[language]]
name = "zirric"
scope = "source.zirric"
file-types = ["zirr", "cavefile"]
grammar = "zirric"

[[grammar]]
name = "zirric"
source = { git = "https://code.knabel.dev/zirric-lang/tree-sitter-zirric", rev = "v0.1.0" }
```

Fetch and build the grammar:

```bash
hx --grammar fetch
hx --grammar build
```

Tip: update `rev` to the latest tag after each release.

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

Add filetype detection (example for `init.lua`):

```lua
vim.filetype.add({
  extension = { zirr = "zirric" },
  filename = { ["cavefile"] = "zirric" },
})
```
