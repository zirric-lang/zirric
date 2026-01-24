---
title: Cave
description: Cavefile annotations and dependency metadata.
---

# Cave

The `cave` module defines annotations and types used by Cavefiles.

## Annotations

### Dependencies

```zirric
annotation Dependencies
```

Marks the current data structure as a dependencies manifest.

### Git

```zirric
annotation Git
```

Marks a field as a Git dependency with a URL and predicate. The field name
represents the import name of the dependency in this package.

Fields:

- `@String url`

### Local

```zirric
annotation Local
```

Marks a field as a local dependency with a path.

Fields:

- `@String path`

### Stdlib

```zirric
annotation Stdlib
```

Marks the current data structure as a standard library dependency.

Fields:

- `@String name`

### Version

```zirric
annotation Version
```

The version predicate for a Git dependency.

Fields:

- `@String predicate`

## Union

### Source

```zirric
union Source
```

The type of dependency: standard library, Git, or local.

Members:

- `Stdlib`
- `Git`
- `Local`
