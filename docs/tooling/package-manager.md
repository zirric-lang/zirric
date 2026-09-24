---
title: Package Manager
description: Cavefile manifests, module layout, and registry resolution.
---

# Package Manager

Zirric packages are described by a `Cavefile`, which is itself Zirric code. The Cavefile declares dependencies using attributes and is parsed by tooling based on its type information rather than executed as a program. `zirric cave install` resolves what it declares; `zirric cave describe` prints what the tooling read.

For the authoritative design, see the [ZE-002 Cavefile proposal](/proposals/ZE-002-the-cavefile#the-package-manifest).

## Cavefile at a glance

A Cavefile is a Zirric module that describes the package and declares its dependencies through attributes. The package manager reads it for types and metadata, not for runtime behavior.

```zirric
mod my_package

import cave

@cave.Package()
@cave.Version("1.0.0")
data MyPackage {
	@cave.Local("../some-local-package")
	localPkg
}
```

## Modules and directory layout

Modules are discovered from the filesystem. Each directory with `.zirr` files becomes a module; each file contributes sources to that module. The module URI is derived from the directory path, and nested folders create nested modules.

The resolver expects packages to carry a `Cavefile` at the package root. It discovers modules beneath that root and maps them to logical module URIs, each named under the base path the Cavefile's own `mod` declares. Every source file states that name in full — see [Declarations § `mod`](/specification/declarations#mod).

Example layout, for a package based on `my_package`:

```
my-package/
├── Cavefile          // mod my_package
├── http/
│   ├── client.zirr   // mod my_package.http
│   └── server.zirr   // mod my_package.http
├── math/
│   ├── list.zirr     // mod my_package.math
│   └── ops.zirr      // mod my_package.math
└── ui/
    └── button.zirr   // mod my_package.ui
```

## Registry layout

The registry layer resolves packages from local caches and remote sources. The expected folder structure in increasing priority is:

```
$ZIRRIC_REGISTRY/
├── git/<package>/<version>/
│   ├── Cavefile
│   └── <submodule>/
└── stdlib/<package>/<version>/
    └── <submodule>/
<package>
├── Cavefile
├── <vendored-package>/
│   ├── Cavefile
│   └── <submodule>/
└── <submodule>/
```

Registries enumerate packages already cached locally, then consult remote sources when a version is missing. The current implementation focuses on Git and local paths.

`stdlib/` is the exception: it is written, never read. The standard library is embedded in the `zirric` binary and always resolved from there, so each run mirrors those sources next to the `git/` clones purely so they can be opened and read on disk. The version in the path is the version of the binary that wrote them, which keeps several installed toolchains from overwriting each other.

The mirrored files are read-only, so an editor opening one refuses to save over it. Editing a copy would change nothing anyway: the next run restores it from the binary.
