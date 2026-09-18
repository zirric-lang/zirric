# Cavefile manifests

Zirric packages declare dependencies and tasks in a `Cavefile`. A Cavefile is
regular Zirric source code that uses attributes from the
`cave` and `future.tasks` standard library modules.
For
implementation details, see `proposals/ZE-002-the-cavefile.md` and
`cave/manifest.zirr`.

## Dependencies

```zirric
import cave

@cave.Dependencies()
data Dependencies {
    @cave.Stdlib("prelude")
    prelude

    @cave.Local("../some-local-package")
    helpers

    @cave.Git("https://code.knabel.dev/zirric-lang/zirric")
    @cave.Version(">0.1.0")
    zirric
}
```

Only fields within the `@cave.Dependencies` data declaration participate in
dependency resolution.

## Tasks

```zirric
import future.tasks

@tasks.Name("generate")
@tasks.Help("Generates something")
@tasks.Exec("tasks/generate.zirr")
data GenerateTask {
    @tasks.Flag()
    @tasks.Name("dry")
    isDryRun: Bool
}
```

Tasks are registered by attributes such as `@tasks.Exec` or `@tasks.Call`. Flags
and positional arguments are expressed as fields with type hints and task attributes.
