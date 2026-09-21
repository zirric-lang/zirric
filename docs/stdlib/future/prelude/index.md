---
title: Future.Prelude
description: Proposed additions to the prelude that have not landed yet.
---

# Module `future.prelude`

> Prelude declarations still under proposal.

```zirric
import future.prelude
```

|            |                                                                                                                                                           |
| ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `future.prelude`                                                                                                                                          |
| **Source** | [`future/prelude/ZE-007-attribute-binding.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/future/prelude/ZE-007-attribute-binding.zirr) |

These are prelude declarations from proposals that have not been accepted. They follow the same rules as the rest of [`future`](../index.md): they may change, and they may be removed.

[`Bound`](#bound) belongs to [ZE-007 Attribute Binding](/proposals/ZE-007-attribute-binding), which proposes calling an attribute's members on a value directly instead of going through the attribute.

## Contents

- **Attributes** — [`Bound`](#bound)
- **Constants** — [`ZE_007`](#ze_007)

---

## Attributes

### `Bound` {#bound}

<small>`future/prelude/ZE-007-attribute-binding.zirr:25`</small>

```zirric
@Proposal(ZE_007)
attr Bound {}
```

A bound function of an attribute receives the targeted value as first argument. attr Error { @Bound() debug(err) -> String } @Error(fn(err) { return err.message }) data MessageError { message: String } MessageError("msg")[@Error].debug() // err-receiver passed automatically

---

## Constants

### `ZE_007` {#ze_007}

<small>`future/prelude/ZE-007-attribute-binding.zirr:9`</small>

```zirric
const ZE_007 = "https://zirric.knabel.dev/proposals/ZE-007-attribute-binding"
```

ZE-007: Attribute Binding https://zirric.knabel.dev/proposals/ZE-007-attribute-binding

---

## See also

- [`future`](../index.md) — what this module is part of.
- [ZE-007 Attribute Binding](/proposals/ZE-007-attribute-binding) — the proposal.
