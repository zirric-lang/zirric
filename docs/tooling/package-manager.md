---
title: Package Manager
description: Cavefile manifests, module layout, and registry resolution.
---

# Cave Package Manager

Zirric packages are described by a `Cavefile`, which is itself Zirric code. The
Cavefile declares dependencies using attributes and is parsed by tooling based
on its type information rather than executed as a program.

For the authoritative design, see the
[ZE-002 Cavefile proposal](/proposals/ZE-002-the-cavefile#cavefile).

## Cavefile at a glance

A Cavefile is a Zirric module that declares dependencies through attributes.
The package manager reads it for types and metadata, not for runtime behavior.

```zirric
import cave

@cave.Dependencies()
data Dependencies {
    @cave.Local("../some-local-package")
    localPkg
}
```

## Modules and directory layout

Modules are discovered from the filesystem. Each directory with `.zirr` files
becomes a module; each file contributes sources to that module. The module URI
is derived from the directory path, and nested folders create nested modules.

The resolver expects packages to carry a `Cavefile` at the package root. It
discovers modules beneath that root and maps them to logical module URIs.

Example layout:

```
my-package/
├── Cavefile
├── http/
│   ├── client.zirr
│   └── server.zirr
├── math/
│   ├── list.zirr
│   └── ops.zirr
└── ui/
    └── button.zirr
```

## Registry layout

The registry layer resolves packages from local caches and remote sources. The
expected folder structure in increasing priority is:

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

Registries enumerate packages already cached locally, then consult remote
sources when a version is missing. The current implementation focuses on Git
and local paths.
