---
title: Tasks
description: Task annotations for Cavefile-based automation.
---

# Tasks

The `tasks` module defines annotations and enums for describing runnable tasks
in Cavefiles.

## Annotations

### Alias

```zirric
annotation Alias
```

Provides alternative names for the task, flag, or argument.

Fields:

- `@Array alias`

### Arg

```zirric
annotation Arg
```

Marks this field as a positional commandline argument.

### Call

```zirric
annotation Call
```

Indicates that this task is implemented by a function.

Fields:

- `@Function function`

### Exec

```zirric
annotation Exec
```

Indicates that this task is implemented in an external file.

Fields:

- `@String file`

### Flag

```zirric
annotation Flag
```

Marks this field as a commandline flag.

### Help

```zirric
annotation Help
```

A short help text for the task, flag, or argument.

Fields:

- `@String help`

### Import

```zirric
annotation Import
```

Indicates that this task uses a module for its implementation.

Fields:

- `@Module module`

### Name

```zirric
annotation Name
```

Renames the task, flag, or argument.

Fields:

- `@String name`

### Short

```zirric
annotation Short
```

Provides a short name for the flag. Not applicable to arguments.

Fields:

- `@Char short`

## Enum

### Task

```zirric
enum Task
```

Marks a data declaration as a task. Exactly one of `@Exec`, `@Call`, or `@Import`
is required.

Cases:

- `Exec`
- `Call`
- `Import`
