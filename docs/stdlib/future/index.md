---
title: Future
description: Implementations of proposals that have not landed yet.
---

# Module `future`

```zirric
import future
```

|            |                                                                                                                                                                                                                        |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `future`                                                                                                                                                                                                               |
| **Source** | [`future/future.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/future/future.zirr), [`future/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/future/module-docs.zirr) |

> Proposals, implemented early and subject to change.

`future` holds implementations of [proposals](https://zirric.knabel.dev/proposals) that are not part of the language yet. Use them with care: they may change or disappear without notice, and the declarations of a rejected proposal are removed outright.

Once a proposal is accepted, its implementation moves to the module it belongs in. Not every proposal is represented here, and some cannot be — a few need syntax Zirric does not have yet.

[`Proposal`](#proposal) links a declaration back to the proposal that describes it.

## Contents

- **Attributes** — [`Proposal`](#proposal)

---

## Attributes

### `Proposal` {#proposal}

<small>`future/future.zirr:5`</small>

```zirric
attr Proposal {
	link: String
}
```

Links a declaration to the proposal that describes it.
Written on anything a proposal introduced, here or elsewhere, so a reader can find the design the declaration came from.

#### Fields

| Field  | Description                                                                           |
| ------ | ------------------------------------------------------------------------------------- |
| `link` | The proposal's page, e.g. "https://zirric.knabel.dev/proposals/ZE-002-the-cavefile/". |
