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
$ZIRRIC_STDLIB/
└── git/<package>/<version>/
    ├── Cavefile
    └── <submodule>/
$ZIRRIC_PACKAGES/
└── git/<package>/<version>/
    ├── Cavefile
    └── <submodule>/
<package>
├── Cavefile
├── <vendored-package>/
│   ├── Cavefile
│   └── <submodule>/
└── <submodule>/
```

Registries enumerate packages already cached locally, then consult remote sources when a version is missing. The current implementation focuses on Git and local paths.
