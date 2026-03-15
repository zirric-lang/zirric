---
title: Future.Cave
description: Experimental Cavefile schema, dependency attributes, and package metadata.
---

# Future.Cave

The `cave` module defines attributes and types used by Cavefiles, moved under
`future/cave` as part of the ZE-002 proposal work.

## Values

### let ZE_002

```zirric
let ZE_002 = "https://zirric.knabel.dev/proposals/ze-002-the-cavefile/"
```

ZE-002: The Cavefile.

## Attributes

### attr Dependencies

```zirric
@Proposal(ZE_002)
attr Dependencies
```

Marks the current data structure as a dependencies manifest.

### attr Version

```zirric
@Proposal(ZE_002)
attr Version
```

The version predicate for the Git dependency.

Fields:

- `@String predicate` — The version predicate string.

### attr Stdlib

```zirric
@Proposal(ZE_002)
attr Stdlib
```

Marks the current data structure as a standard library dependency.

Fields:

- `@String name` — The name of the standard library dependency.

### attr Git

```zirric
@Proposal(ZE_002)
attr Git
```

Marks a field as a Git dependency with a URL and predicate. The field name
represents the import name of the dependency in this package.

Fields:

- `@String url` — The URL of the Git repository.

### attr Local

```zirric
@Proposal(ZE_002)
attr Local
```

Marks a field as a local dependency with a path.

Fields:

- `@String path` — The local path to the dependency.

## Union

### union Source

```zirric
@Proposal(ZE_002)
union Source
```

The type of dependency: standard library, Git, or local.

Members:

- `Stdlib`
- `Git`
- `Local`
