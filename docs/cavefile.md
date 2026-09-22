# Cavefile manifests

Zirric packages declare themselves, their dependencies and their tasks in a `Cavefile`. A Cavefile is regular Zirric source code that uses attributes from the `cave` and `tasks` standard library modules. For implementation details, see `proposals/ZE-002-the-cavefile.md` and `cave/manifest.zirr`.

## The package

The Cavefile's own `mod` declares the package's base module path. Every module of the package is named under it, so a package based on `code.knabel.dev.zirric_lang.ui` holds its `./flow/node` sources to `mod code.knabel.dev.zirric_lang.ui.flow.node`. See [Declarations § `mod`](/specification/declarations#mod).

The `@cave.Package()` data declaration is the manifest itself: its attributes describe the package, its fields declare the dependencies.

```zirric
mod code.knabel.dev.zirric_lang.ui

import cave

@cave.Package()
@cave.Git("https://code.knabel.dev/zirric-lang/ui")
@cave.Version("1.2.3")
@cave.LanguageVersion("^0.1.0")
@cave.Description("Widgets and layout for Zirric programs.")
@cave.Documentation("https://zirric.knabel.dev/ui")
data UI {
	@cave.Stdlib("prelude")
	prelude

	@cave.Local("../some-local-package")
	helpers

	@cave.Git("https://code.knabel.dev/zirric-lang/zirric")
	@cave.Version(">0.1.0")
	@cave.Description("The Zirric standard library")
	@cave.Documentation("https://zirric.knabel.dev")
	zirric
}
```

| Attribute                     | On the package                             | On a dependency                                 |
| ----------------------------- | ------------------------------------------ | ----------------------------------------------- |
| `@cave.Git(url)`              | the repository the package is published at | fetch the dependency from this repository       |
| `@cave.Stdlib(name)`          | —                                          | take the dependency from the standard library   |
| `@cave.Local(path)`           | —                                          | take the dependency from a path on disk         |
| `@cave.Version(pred)`         | the package's own version                  | which versions of the dependency are acceptable |
| `@cave.LanguageVersion(pred)` | which Zirric versions can build it         | which Zirric versions the dependency needs      |
| `@cave.Description(desc)`     | one line about the package                 | one line about the dependency                   |
| `@cave.Documentation(url)`    | where its documentation lives              | where the dependency's documentation lives      |

Only fields within the `@cave.Package` data declaration participate in dependency resolution.

`zirric cave new` writes this file for you, reading the Git remote to name the package and taking the rest from flags — see [the CLI](/tooling/zirric-cli#project-commands).

A `@cave.LanguageVersion` the running Zirric does not satisfy refuses the project outright: `zirric cave install`, `run`, `fmt`, `task` and the language server all stop with it. Only `zirric cave describe` still opens such a project, because reporting what the manifest says is how the refusal is explained. A development build that carries no version of its own is exempt, since it cannot be told apart from one that is too old.

`zirric cave describe` prints all of it — the package under `package:`, the dependencies under `dependencies:`:

```yaml
package:
  name: code.knabel.dev.zirric_lang.ui
  source: https://code.knabel.dev/zirric-lang/ui
  version: 1.2.3
  languageVersion: "^0.1.0"
  description: Widgets and layout for Zirric programs.
  documentation: https://zirric.knabel.dev/ui
```

## Tasks

```zirric
import tasks

@tasks.Name("generate")
@tasks.Help("Generates something")
@tasks.Exec("tasks/generate.zirr")
data GenerateTask {
	@tasks.Flag()
	@tasks.Name("dry")
	isDryRun: Bool
}
```

Tasks are registered by attributes such as `@tasks.Exec` or `@tasks.Call`. Flags and positional arguments are expressed as fields with type hints and task attributes.
