---
title: Cave
description: Cavefile schema, dependency attributes, and package metadata.
---

# Cave

The `cave` module defines attributes and types used by Cavefiles, as part of
the ZE-002 proposal work.

## Values

### const ZE_002

```zirric
const ZE_002 = "https://zirric.knabel.dev/proposals/ze-002-the-cavefile/"
```

ZE-002: The Cavefile.

## Attributes

### attr Dependencies

```zirric
@Proposal(ZE_002)
attr Dependencies {}
```

Marks the current data structure as a dependencies manifest.

### attr Package

```zirric
@Proposal(ZE_002)
attr Package {
  url: String
}
```

Declares the canonical URL of the current package. Placed on the Cavefile's
`mod` declaration, e.g. `@cave.Package("https://...") mod mymodule`. Used to
derive the package's name and source instead of falling back to the project
directory name.

Fields:

- `url: String` — The canonical URL of the package.

### attr Version

```zirric
@Proposal(ZE_002)
attr Version {
  predicate: String
}
```

The version predicate for the Git dependency.

Fields:

- `predicate: String` — The version predicate string.

### attr Stdlib

```zirric
@Proposal(ZE_002)
attr Stdlib {
  name: String
}
```

Marks the current data structure as a standard library dependency.

Fields:

- `name: String` — The name of the standard library dependency.

### attr Git

```zirric
@Proposal(ZE_002)
attr Git {
  url: String
}
```

Marks a field as a Git dependency with a URL and predicate. The field name
represents the import name of the dependency in this package.

Fields:

- `url: String` — The URL of the Git repository.

### attr Local

```zirric
@Proposal(ZE_002)
attr Local {
  path: String
}
```

Marks a field as a local dependency with a path.

Fields:

- `path: String` — The local path to the dependency.

## Union

### union Source

```zirric
@Proposal(ZE_002)
union Source {
  Stdlib
  Git
  Local
}
```

The type of dependency: standard library, Git, or local.

Members:

- `Stdlib`
- `Git`
- `Local`
