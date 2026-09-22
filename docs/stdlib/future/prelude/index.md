---
title: Future.Prelude
description: Bound to xprelude rather than to its last segment, which would shadow the prelude every module is given.
---

# Module `future.prelude`

```zirric
import future.prelude
```

|            |                                                                                                                                                                                                                                                                                            |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Module** | `future.prelude`                                                                                                                                                                                                                                                                           |
| **Source** | [`future/prelude/ZE-007-attribute-binding.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/future/prelude/ZE-007-attribute-binding.zirr), [`future/prelude/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/future/prelude/module-docs.zirr) |

Proposed additions to the prelude that have not landed yet.

> Prelude declarations still under proposal.

These are prelude declarations from proposals that have not been accepted. They follow the same rules as the rest of [`future`](../../future/index.md): they may change, and they may be removed.

[`Bound`](#bound) belongs to [ZE-007 Attribute Binding](https://zirric.knabel.dev/proposals/ZE-007-attribute-binding), which proposes calling an attribute's members on a value directly instead of going through the attribute.

## Dependencies

- [`future`](../../future/index.md)

---

## Contents

- **Attributes** — [`Bound`](#bound)
- **Constants** — [`ZE_007`](#ze_007)

---

## Attributes

### `Bound` {#bound}

<small>`future/prelude/ZE-007-attribute-binding.zirr:28`</small>

```zirric
attr Bound
```

A bound function of an attribute receives the targeted value as first argument.

```zirric
attr Error {
	@Bound()
	debug(err) -> String
}

@Error(fn(err) { return err.message })
data MessageError {
	message: String
}

MessageError("msg")[@Error].debug() // err-receiver passed automatically
```

---

## Constants

### `ZE_007` {#ze_007}

<small>`future/prelude/ZE-007-attribute-binding.zirr:10`</small>

```zirric
const ZE_007
```

ZE-007: Attribute Binding
https://zirric.knabel.dev/proposals/ZE-007-attribute-binding
