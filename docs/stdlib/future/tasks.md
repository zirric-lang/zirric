---
title: Future.Tasks
description: Experimental task attributes for Cavefile-based automation.
---

# Future.Tasks

The `tasks` module defines attributes and unions for describing runnable tasks
in Cavefiles, moved under `future/tasks` as part of the ZE-002 proposal work.

## Values

### let ZE_002

```zirric
let ZE_002 = "https://zirric.knabel.dev/proposals/ze-002-the-cavefile/"
```

ZE-002: The Cavefile.

## Attributes

### attr Exec

```zirric
@Proposal(ZE_002)
attr Exec
```

Indicates that this task is implemented in an external file.

Fields:

- `@String file` — The file that contains the implementation of the task.

### attr Call

```zirric
@Proposal(ZE_002)
attr Call
```

Indicates that this task is implemented by a function.

Fields:

- `@Function function` — The function to run for the task.

### attr Import

```zirric
@Proposal(ZE_002)
attr Import
```

Indicates that this task uses a module for its implementation. Must be declared
on the field of a data declaration with `@Dependencies`.

Fields:

- `@Module module` — The module to use for the task.

### attr Name

```zirric
@Proposal(ZE_002)
attr Name
```

Renames the task, flag or argument.

Fields:

- `@String name` — The new name.

### attr Alias

```zirric
@Proposal(ZE_002)
attr Alias
```

Provides alternative names for the task, flag or argument.

Fields:

- `@Array alias` — Alternative names.

### attr Short

```zirric
@Proposal(ZE_002)
attr Short
```

Provides a short name for the flag. Not applicable to arguments.

Fields:

- `@Char short` — The short name of the flag.

### attr Help

```zirric
@Proposal(ZE_002)
attr Help
```

A short help text for the task, flag or argument.

Fields:

- `@String help` — A short help text.

### attr Flag

```zirric
@Proposal(ZE_002)
attr Flag
```

Marks this field as a commandline flag.

### attr Arg

```zirric
@Proposal(ZE_002)
attr Arg
```

Marks this field as a positional commandline argument.

## Union

### union Task

```zirric
@Proposal(ZE_002)
union Task
```

Marks a data declaration as a task. The task can be executed from the command
line. Exactly one of these two attributes are required: `@RunFile`, `@Run` or
`@Import`.

Members:

- `Exec`
- `Call`
- `Import`
