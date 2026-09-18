---
title: Tasks
description: Task attributes for Cavefile-based automation.
---

# Tasks

The `tasks` module defines attributes and unions for describing runnable tasks
in Cavefiles, as part of the ZE-002 proposal work.

## Values

### const ZE_002

```zirric
const ZE_002 = "https://zirric.knabel.dev/proposals/ze-002-the-cavefile/"
```

ZE-002: The Cavefile.

## Attributes

### attr Exec

```zirric
@Proposal(ZE_002)
attr Exec {
  file: String
}
```

Indicates that this task is implemented in an external file.

Fields:

- `file: String` — The file that contains the implementation of the task.

### attr Call

```zirric
@Proposal(ZE_002)
attr Call {
  function: Function
}
```

Indicates that this task is implemented by a function.

Fields:

- `function: Function` — The function to run for the task.

### attr Name

```zirric
@Proposal(ZE_002)
attr Name {
  name: String
}
```

Renames the task, flag or argument.

Fields:

- `name: String` — The new name.

### attr Alias

```zirric
@Proposal(ZE_002)
attr Alias {
  alias: Array
}
```

Provides alternative names for the task, flag or argument.

Fields:

- `alias: Array` — Alternative names.

### attr Short

```zirric
@Proposal(ZE_002)
attr Short {
  short: Char
}
```

Provides a short name for the flag. Not applicable to arguments.

Fields:

- `short: Char` — The short name of the flag.

### attr Help

```zirric
@Proposal(ZE_002)
attr Help {
  help: String
}
```

A short help text for the task, flag or argument.

Fields:

- `help: String` — A short help text.

### attr Flag

```zirric
@Proposal(ZE_002)
attr Flag {}
```

Marks this field as a commandline flag.

### attr Arg

```zirric
@Proposal(ZE_002)
attr Arg {}
```

Marks this field as a positional commandline argument.

For a task using `@Call`, a `@Flag`/`@Arg` field's type must be `Bool`, `String`, or `Int` — the CLI uses it to register a real, typed flag or argument. This restriction doesn't apply to `@Exec` tasks, which parse their own arguments from `os.args()` instead.

## Union

### union Task

```zirric
@Proposal(ZE_002)
union Task
```

Marks a data declaration as a task. The task can be executed from the command
line. Exactly one of `@Exec` or `@Call` is required.

Members:

- `Exec`
- `Call`
