---
title: Tasks
description: The attributes that turn a data declaration into a CLI task.
---

# Module `tasks`

> Declaring a subcommand, with typed flags and arguments.

```zirric
import tasks
```

|            |                                                                                                         |
| ---------- | ------------------------------------------------------------------------------------------------------- |
| **Module** | `tasks`                                                                                                 |
| **Source** | [`tasks/manifest.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/tasks/manifest.zirr) |

A task is a `data` declaration in the `Cavefile` carrying either [`Exec`](#exec) or [`Call`](#call). `zirric task` lists them and `zirric task run` — or `zirric x` — runs them.

The difference between the two is what happens when it runs. [`Exec`](#exec) runs a script as its own program, which reads its own arguments through `os.args()`. [`Call`](#call) invokes a function directly with an instance of the declaration, built from the parsed flag and argument values — so the fields are typed and the CLI does the parsing.

The remaining attributes describe the command line: [`Name`](#name) and [`Alias`](#alias) name it, [`Short`](#short) and [`Help`](#help) document it, and [`Flag`](#flag) and [`Arg`](#arg) mark which fields are flags and which are positional.

## Contents

- **Attributes** — [`Exec`](#exec), [`Call`](#call), [`Name`](#name), [`Alias`](#alias), [`Short`](#short), [`Help`](#help), [`Flag`](#flag), [`Arg`](#arg)
- **Unions** — [`Task`](#task)
- **Constants** — [`ZE_002`](#ze_002)

---

## Attributes

### `Exec` {#exec}

<small>`tasks/manifest.zirr:22`</small>

```zirric
@Proposal(ZE_002)
attr Exec {
	// The file that contains the implementation of the task.
	file: String
}
```

Indicates that this task is implemented in an external file.

#### Members

| Member | Signature      | Description                                            |
| ------ | -------------- | ------------------------------------------------------ |
| `file` | `file: String` | The file that contains the implementation of the task. |

---

### `Call` {#call}

<small>`tasks/manifest.zirr:29`</small>

```zirric
@Proposal(ZE_002)
attr Call {
	// The function to run for the task.
	function: Function
}
```

Indicates that this task is implemented by a function.

#### Members

| Member     | Signature            | Description                       |
| ---------- | -------------------- | --------------------------------- |
| `function` | `function: Function` | The function to run for the task. |

---

### `Name` {#name}

<small>`tasks/manifest.zirr:36`</small>

```zirric
@Proposal(ZE_002)
attr Name {
	// The new name.
	name: String
}
```

Renames the task, flag or argument.

#### Members

| Member | Signature      | Description   |
| ------ | -------------- | ------------- |
| `name` | `name: String` | The new name. |

---

### `Alias` {#alias}

<small>`tasks/manifest.zirr:43`</small>

```zirric
@Proposal(ZE_002)
attr Alias {
	// Alternative names.
	alias: Array
}
```

Provides alternative names for the task, flag or argument.

#### Members

| Member  | Signature      | Description        |
| ------- | -------------- | ------------------ |
| `alias` | `alias: Array` | Alternative names. |

---

### `Short` {#short}

<small>`tasks/manifest.zirr:51`</small>

```zirric
@Proposal(ZE_002)
attr Short {
	// The short name of the flag.
	short: Char
}
```

Provides a short name for the flag. Not applicable to arguments.

#### Members

| Member  | Signature     | Description                 |
| ------- | ------------- | --------------------------- |
| `short` | `short: Char` | The short name of the flag. |

---

### `Help` {#help}

<small>`tasks/manifest.zirr:58`</small>

```zirric
@Proposal(ZE_002)
attr Help {
	// A short help text.
	help: String
}
```

A short help text for the task, flag or argument.

#### Members

| Member | Signature      | Description        |
| ------ | -------------- | ------------------ |
| `help` | `help: String` | A short help text. |

---

### `Flag` {#flag}

<small>`tasks/manifest.zirr:65`</small>

```zirric
@Proposal(ZE_002)
attr Flag {}
```

Marks this field as a commandline flag.

---

### `Arg` {#arg}

<small>`tasks/manifest.zirr:69`</small>

```zirric
@Proposal(ZE_002)
attr Arg {}
```

Marks this field as a positional commandline argument.

---

## Unions

### `Task` {#task}

<small>`tasks/manifest.zirr:15`</small>

```zirric
@Proposal(ZE_002)
union Task {
	Exec
	Call
}
```

Marks a data declaration as a task. The task can be executed from the command line. Exactly one of @Exec or @Call is required.

#### Cases

| Case   | Interpretation                                        |
| ------ | ----------------------------------------------------- |
| `Exec` | Runs a script as its own program.                     |
| `Call` | Calls a function with the parsed flags and arguments. |

---

## Constants

### `ZE_002` {#ze_002}

<small>`tasks/manifest.zirr:9`</small>

```zirric
const ZE_002 = "https://zirric.knabel.dev/proposals/ze-002-the-cavefile/"
```

ZE-002: Error Handling https://zirric.knabel.dev/proposals/ze-002-the-cavefile/

---

## See also

- [Cavefile manifests](/cavefile) — what a whole file looks like.
- [`cave`](../cave/index.md) — the dependency half of a Cavefile.
- [Zirric CLI](/tooling/zirric-cli) — running the tasks you declare.
- [ZE-002 The Cavefile](/proposals/ZE-002-the-cavefile) — the design.
