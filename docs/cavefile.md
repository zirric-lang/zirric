# Cavefile manifests

Zirric packages declare dependencies and tasks in a `Cavefile`. A Cavefile is
regular Zirric source code that uses annotations from the `cave` and
`cave.tasks` standard library modules. For implementation details, see
`proposals/ZE-002-the-cavefile.md` and `stdlib/cave/manifest.zirr`.

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
    future
}
```

Only fields within the `@cave.Dependencies` data declaration participate in
dependency resolution.

## Tasks

```zirric
import cave.tasks

@tasks.Name("generate")
@tasks.Help("Generates something")
@tasks.Exec("tasks/generate.zirr")
data GenerateTask {
    @Bool
    @tasks.Flag()
    @tasks.Name("dry")
    isDryRun
}
```

Tasks are registered by annotations such as `@tasks.Exec` or `@tasks.Call`. Flags
and positional arguments are expressed as annotated fields.
