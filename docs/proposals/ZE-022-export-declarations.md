---
title: ZE-022 - Export Declarations
status: Draft
---

# ZE-022 - Export Declarations

::: callout draft Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design.
Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

This proposal introduces `export`, a module-level declaration that lets a module re-export a name it has imported, so downstream consumers can
reach that name through the re-exporting module directly rather than importing the original source module themselves.

## Motivation

[ZE-021](./ZE-021-flow-architecture.md) surfaced a concrete case this gap blocks:
`flow.node` is a deliberately self-contained submodule holding `Renderable`/`Node`, and `flow` imports it to build `Component` and `MappedView`. Without a way for `flow` to make `Renderable` visible under its own name, every component and backend author has to write.

```zirric
import flow { Component, Step, Binding }
import flow.node { Renderable }
```

even though, from an application author's point of view, `Renderable` is conceptually part of `flow`'s public surface — its only reason for living in a separate submodule is to keep `flow.node` free of unrelated dependencies (`Cmd`, `Sub`, `Component`, `lift`, `run`), not to hide it from ordinary users. This pattern generalizes beyond `flow`: any module that splits its implementation across internal submodules for dependency or organizational reasons currently has no way to present a single, flat, stable import surface to its consumers. Restructuring a module's internals — moving a declaration into or out of a submodule — becomes a breaking change for every downstream import, because there is no indirection between where a name is _declared_ and where a consumer is expected to _get it from_.

## Proposed Solution

`export` re-exports a name already in scope via `import`. It does not change what is directly importable from a module's own declarations — those remain visible to consumers exactly as they are today, with no `export` needed. `export` only adds visibility for names a module brought in from elsewhere:

```zirric
mod flow

import flow.node { Renderable }

export Renderable
```

With this, `import flow { Component, Renderable }` succeeds for any downstream consumer, without needing to know `Renderable` actually originates in `flow.node`.

## Detailed Design

### Three forms

```zirric
import flow.node { Renderable }

export Renderable          // re-export the directly imported name
export node.Renderable     // re-export via the module's implicit namespace alias
export Rename = Renderable // re-export under a new local name
```

Importing a qualified submodule (`flow.node`) implicitly binds the trailing path segment (`node`) as a namespace alias for that module in addition to whatever names were destructured out of it, so `Renderable` and `node.Renderable` refer to the same value. `export` accepts either form as its source — a bare identifier already bound in scope, or a dotted path rooted at an implicit namespace alias. Both of the first two forms above are equivalent; the namespace-qualified form exists for cases where disambiguating the source is clearer to a reader, or where no bare destructured binding exists for the name being re-exported.

The third form, `export <Alias> = <path>`, re-exports under a different local name than the source uses. This is separate from renaming at `import` time (`import flow.node { Renderable as R }`, if such syntax exists) — it renames what _this module's own consumers_ see, without affecting how the current module refers to the name internally.

### What `export` does not do

- **It does not export a module's own declarations.** A `data`/`union`/ `attr`/`fn` declared directly in a module is importable by consumers today with no `export` statement, and this proposal does not change that. `export` is scoped narrowly to re-exporting _imported_ names, matching the one gap that motivated it.
- **There is no wildcard form.** `export *` is deliberately excluded.
  Every re-exported name is a distinct, explicit statement, consistent with Zirric's general preference for explicitness over implicit surface area (see `const`/`var`, explicit type hints). A module's re-exported surface should be readable as a short, complete list, not inferred from what happens to be imported.
- **It does not change the dependency graph.** `flow` already depends on `flow.node` via its own `import`; `export Renderable` changes what is _visible_ to `flow`'s consumers, not what `flow` itself depends on.
  Import-cycle rules are unaffected — a module cannot use `export` to route around the prohibition on cyclic imports, since the underlying `import` establishing the dependency still has to be acyclic on its own.
- **Multiple exports of the same symbol are allowed.** As shown above, a module may export the same underlying value under more than one name simultaneously; this proposal does not enforce a single canonical export name per symbol.

## Changes to the Standard Library

Once implemented, [ZE-021](./ZE-021-flow-architecture.md) should be revised: `flow` re-exports `flow.node.Renderable` directly, and the "double import" workaround documented there — along with its corresponding Best Practices entry — is removed, since `import flow { Component, Renderable }` becomes sufficient on its own.

## Dependencies on Other Proposals

| Proposal                                                  | Dependency                                                                                          |
| --------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| [ZE-021 Flow Architecture](./ZE-021-flow-architecture.md) | Motivating use case; `flow`'s re-export of `flow.node.Renderable` depends directly on this proposal |

## Best Practices

- **Prefer the bare form (`export Name`) for the common case.** Reach for the namespace-qualified form only when it genuinely improves clarity about where a re-exported name originates, or when no bare binding exists locally.
- **Use rename-on-export sparingly.** A re-exported name that differs from its source name makes it harder for a reader to trace a symbol back to where it's actually declared; keep names aligned unless there is a real naming collision or clarity problem to solve.
- **Treat every `export` as a public API commitment.** Once a name is re-exported, downstream code may depend on reaching it through the re-exporting module specifically — removing an `export` later is a breaking change for those consumers even if the underlying import remains.
- **Use `export` to keep internal reorganization invisible to consumers.**
  If a module's internals get split into submodules for dependency-isolation reasons (as with `flow`/`flow.node`), re-export the names consumers are expected to use from the top-level module, rather than requiring them to know or follow the internal structure.
