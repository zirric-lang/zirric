---
title: Future
description: Implementations of proposals that have not landed yet.
---

# Module `future`

> Proposals, implemented early and subject to change.

```zirric
import future
```

|            |                                                                                                       |
| ---------- | ----------------------------------------------------------------------------------------------------- |
| **Module** | `future`                                                                                              |
| **Source** | [`future/future.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/future/future.zirr) |

`future` holds implementations of [proposals](/proposals) that are not part of the language yet. Use them with care: they may change or disappear without notice, and the declarations of a rejected proposal are removed outright.

Once a proposal is accepted, its implementation moves to the module it belongs in. Not every proposal is represented here, and some cannot be — a few need syntax Zirric does not have yet.

[`Proposal`](#proposal) links a declaration back to the proposal that describes it.

## Contents

- **Attributes** — [`Proposal`](#proposal)

---

## Attributes

### `Proposal` {#proposal}

<small>`future/future.zirr:10`</small>

```zirric
attr Proposal {
	link: String
}
```

Links a declaration to the proposal that describes it, so tooling can point at the design behind an unstable API.

#### Members

| Member | Signature      | Description                                      |
| ------ | -------------- | ------------------------------------------------ |
| `link` | `link: String` | URL of the proposal describing this declaration. |

---

## See also

- [`future.prelude`](./prelude/index.md) — proposed prelude additions.
- [`future.reflect`](./reflect/index.md) — the earlier reflection sketch.
- [Proposals](/proposals) — what each of these is tracking.
