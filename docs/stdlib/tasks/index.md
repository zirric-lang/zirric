---
title: Tasks
description: The attributes that turn a data declaration into a CLI task.
---

# Module `tasks`

```zirric
import tasks
```

|            |                                                                                                                                                                                                                        |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `tasks`                                                                                                                                                                                                                |
| **Source** | [`tasks/manifest.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/tasks/manifest.zirr), [`tasks/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/tasks/module-docs.zirr) |

> Declaring a subcommand, with typed flags and arguments.

A task is a `data` declaration in the `Cavefile` carrying either [`Exec`](#exec) or [`Call`](#call). `zirric task` lists them and `zirric task run` — or `zirric x` — runs them.

The difference between the two is what happens when it runs. [`Exec`](#exec) runs a script as its own program, which reads its own arguments through `os.args()`. [`Call`](#call) invokes a function directly with an instance of the declaration, built from the parsed flag and argument values — so the fields are typed and the CLI does the parsing.

The remaining attributes describe the command line: [`Name`](#name) and [`Alias`](#alias) name it, [`Short`](#short) and [`Help`](#help) document it, and [`Flag`](#flag) and [`Arg`](#arg) mark which fields are flags and which are positional.

## Dependencies

- [`future`](../future/index.md)

---

## Contents

- **Unions** — [`Task`](#task)
- **Attributes** — [`Alias`](#alias), [`Arg`](#arg), [`Call`](#call), [`Exec`](#exec), [`Flag`](#flag), [`Help`](#help), [`Name`](#name), [`Short`](#short)
- **Constants** — [`ZE_002`](#ze_002)

---

## Unions

### `Task` {#task}

<small>`tasks/manifest.zirr:15`</small>

```zirric
union Task {
	Exec
	Call
}
```

Marks a data declaration as a task.
The task can be executed from the command line.
Exactly one of @Exec or @Call is required.

#### Cases

| Case   | Interpretation                                               |
| ------ | ------------------------------------------------------------ |
| `Exec` | Indicates that this task is implemented in an external file. |
| `Call` | Indicates that this task is implemented by a function.       |

---

## Attributes

### `Alias` {#alias}

<small>`tasks/manifest.zirr:43`</small>

```zirric
attr Alias {
	alias: Array
}
```

Provides alternative names for the task, flag or argument.

#### Fields

| Field   | Description        |
| ------- | ------------------ |
| `alias` | Alternative names. |

---

### `Arg` {#arg}

<small>`tasks/manifest.zirr:69`</small>

```zirric
attr Arg
```

Marks this field as a positional commandline argument.

---

### `Call` {#call}

<small>`tasks/manifest.zirr:29`</small>

```zirric
attr Call {
	function: Function
}
```

Indicates that this task is implemented by a function.

#### Fields

| Field      | Description                       |
| ---------- | --------------------------------- |
| `function` | The function to run for the task. |

---

### `Exec` {#exec}

<small>`tasks/manifest.zirr:22`</small>

```zirric
attr Exec {
	file: String
}
```

Indicates that this task is implemented in an external file.

#### Fields

| Field  | Description                                            |
| ------ | ------------------------------------------------------ |
| `file` | The file that contains the implementation of the task. |

---

### `Flag` {#flag}

<small>`tasks/manifest.zirr:65`</small>

```zirric
attr Flag
```

Marks this field as a commandline flag.

---

### `Help` {#help}

<small>`tasks/manifest.zirr:58`</small>

```zirric
attr Help {
	help: String
}
```

A short help text for the task, flag or argument.

#### Fields

| Field  | Description        |
| ------ | ------------------ |
| `help` | A short help text. |

---

### `Name` {#name}

<small>`tasks/manifest.zirr:36`</small>

```zirric
attr Name {
	name: String
}
```

Renames the task, flag or argument.

#### Fields

| Field  | Description   |
| ------ | ------------- |
| `name` | The new name. |

---

### `Short` {#short}

<small>`tasks/manifest.zirr:51`</small>

```zirric
attr Short {
	short: Char
}
```

Provides a short name for the flag.
Not applicable to arguments.

#### Fields

| Field   | Description                 |
| ------- | --------------------------- |
| `short` | The short name of the flag. |

---

## Constants

### `ZE_002` {#ze_002}

<small>`tasks/manifest.zirr:9`</small>

```zirric
const ZE_002
```

ZE-002: Error Handling
https://zirric.knabel.dev/proposals/ze-002-the-cavefile/
