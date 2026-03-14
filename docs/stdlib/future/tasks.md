---
title: Future.Tasks
description: Experimental task annotations for Cavefile-based automation.
---

# Future.Tasks

The `tasks` module defines annotations and unions for describing runnable tasks
in Cavefiles, moved under `future/tasks` as part of the ZE-002 proposal work.

## Values

### let ZE_002

```zirric
let ZE_002 = "https://zirric.knabel.dev/proposals/ze-002-the-cavefile/"
```

ZE-002: The Cavefile.

## Annotations

### annotation Exec

```zirric
@Proposal(ZE_002)
annotation Exec
```

Indicates that this task is implemented in an external file.

Fields:

- `@String file` — The file that contains the implementation of the task.

### annotation Call

```zirric
@Proposal(ZE_002)
annotation Call
```

Indicates that this task is implemented by a function.

Fields:

- `@Function function` — The function to run for the task.

### annotation Import

```zirric
@Proposal(ZE_002)
annotation Import
```

Indicates that this task uses a module for its implementation. Must be declared
on the field of a data declaration with `@Dependencies`.

Fields:

- `@Module module` — The module to use for the task.

### annotation Name

```zirric
@Proposal(ZE_002)
annotation Name
```

Renames the task, flag or argument.

Fields:

- `@String name` — The new name.

### annotation Alias

```zirric
@Proposal(ZE_002)
annotation Alias
```

Provides alternative names for the task, flag or argument.

Fields:

- `@Array alias` — Alternative names.

### annotation Short

```zirric
@Proposal(ZE_002)
annotation Short
```

Provides a short name for the flag. Not applicable to arguments.

Fields:

- `@Char short` — The short name of the flag.

### annotation Help

```zirric
@Proposal(ZE_002)
annotation Help
```

A short help text for the task, flag or argument.

Fields:

- `@String help` — A short help text.

### annotation Flag

```zirric
@Proposal(ZE_002)
annotation Flag
```

Marks this field as a commandline flag.

### annotation Arg

```zirric
@Proposal(ZE_002)
annotation Arg
```

Marks this field as a positional commandline argument.

## Union

### union Task

```zirric
@Proposal(ZE_002)
union Task
```

Marks a data declaration as a task. The task can be executed from the command
line. Exactly one of these two annotations are required: `@RunFile`, `@Run` or
`@Import`.

Members:

- `Exec`
- `Call`
- `Import`
