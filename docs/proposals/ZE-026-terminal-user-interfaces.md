---
title: "ZE-026 - Terminal User Interfaces"
description: "A terminal capability in os, and a tui module family built on the Elm architecture and co, where updates return values and effects are data."
---

# Terminal User Interfaces

::: callout warning Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design. Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

This proposal adds two things:

- **`terminal`** describes a terminal as a capability value, and its input as a union of events.
- **`tui`** runs interactive terminal programs in the style of the Elm architecture. A program has three parts:
  - a model;
  - an `update` function that returns the next model;
  - a `view` function that renders the model to a string.

`os.terminal()` hands out the real terminal. Nothing else touches it. The loop is written in plain Zirric on top of the routines and channels of [ZE-025](https://zirric.knabel.dev/proposals/ZE-025-routines-and-channels/).

The design follows [Bubble Tea](https://github.com/charmbracelet/bubbletea) where Bubble Tea is right, and departs from it where Zirric has a better tool. The largest departure is the governing rule: **effects are values.**

An update never does anything. It returns a `tui.Next` holding the new model and a list of `tui.Effect` values, and the loop performs them. This has two consequences:

- Every step of a program is comparable with `==`.
- Every program can be replayed in a test without a terminal, a timer or a routine.

No language changes are proposed.

## Motivation

Since ZE-025, a program can wait on standard input and a timer at the same time. Its spinner example shows how far that gets: it reads single bytes and prints a new line per frame. Three host facilities are still missing, and without them a real interactive program is impossible, not just inconvenient:

1. **No raw mode.** Input arrives only after enter, echoed by the terminal. A key press cannot be a command.
2. **No key decoding.** An arrow key arrives as the three bytes `ESC [ A`, split across reads at the host's whim. Every program would parse escape sequences on its own, and get it wrong in a different way.
3. **No size.** There is no way to ask how wide the terminal is, or to hear that it changed, so nothing can be laid out or truncated.

These three need the host, and so belong in `os`. Beyond them, every TUI would write the same `co.select` loop by hand:

- read keys in a routine;
- run jobs in routines;
- arm timers;
- redraw without flicker;
- restore the terminal on the way out.

Worse, a text input written for one program's loop cannot be dropped into another's. Components only compose over a loop they share.

With this proposal, a counter is a complete program:

```zirric
mod code.example.counter

import fmt
import os
import terminal { Key }
import tui

// Handles one message. Only keys matter here; everything else keeps the model.
fn update(count: Int, msg) -> tui.Next {
	switch msg {
	case is Key:
		return switch msg.name {
		case "up":
			tui.keep(count + 1)
		case "down":
			tui.keep(count - 1)
		case "q":
			tui.Next(count, [tui.Quit()])
		case "ctrl+c":
			tui.Next(count, [tui.Quit()])
		case _:
			tui.keep(count)
		}
	}
	return tui.keep(count)
}

fn view(count: Int) -> String {
	return "Count: " + fmt.sprint(count) + "\n↑/↓ to change, q to quit"
}

const app = tui.Program(tui.keep(0), update, view, [])
_ = tui.run(app, os.terminal(), os.timer())
```

And its test needs no terminal:

```zirric
@tests.Test()
fn testUpTwiceThenQuit() -> Result {
	const trace = tui.replay(counter.app, terminal.Size(40, 5), [
		terminal.key("up"),
		terminal.key("up"),
		terminal.key("q")
	])
	return assert.all([
		assert.equal(2, trace.model),
		assert.isTrue(trace.quit)
	])
}
```

## Proposed Solution

| Module              | Where               | Kind                   | Contents                                                                                                              |
| ------------------- | ------------------- | ---------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `terminal`          | stdlib              | vocabulary, capability | `Terminal` value, `Event` union (`Key`, `Paste`, `Mouse`, `Resize`), `Size`, `Mode`, `HasTerminal`, a scripted double |
| `os`                | stdlib              | host                   | new: `terminal() -> Result`                                                                                           |
| `tui`               | stdlib              | loop                   | `Program`, `Next`, `Effect` union, `Ticked`, `run`, `replay`, `Trace`                                                 |
| `tui.style`         | stdlib              | pure                   | colors and text attributes as values; `paint`, `plain`                                                                |
| `tui.layout`        | stdlib              | pure                   | measuring and arranging blocks of text                                                                                |
| `widgets.spinner`   | first-party package | component              | reference component that uses effects                                                                                 |
| `widgets.textinput` | first-party package | component              | reference component that does not                                                                                     |

This proposal replaces the draft `term`, `ui` and `colors` packages. Where each piece lives, and why, is argued under Alternatives Considered.

### A program is data

```zirric
data Program {
	start: Next
	update: fn(Any, Any) -> Next
	view: fn(Any) -> String
	modes: [terminal.Mode]
}
```

| Field    | Meaning                                                                    |
| -------- | -------------------------------------------------------------------------- |
| `start`  | The initial model, and the effects to perform before the first message.    |
| `update` | Receives the current model and one message, and returns what happens next. |
| `view`   | Renders a model as text.                                                   |
| `modes`  | Asks the terminal for full screen, mouse tracking or bracketed paste.      |

A model is any value. It is usually a `data` type, and often a `union`: a program that is loading, ready or failed is most honestly modelled as three members. `update` then begins with a `switch` over the model.

A message is any value, from one of three sources:

- the terminal sends `terminal.Event` members;
- timers send `tui.Ticked`;
- `Call` and `Stream` effects send whatever their tasks produce.

`update` tells them apart with `switch` and `case is`, as it would any other union.

**`update` and `view` never wait.** They are ordinary functions without switch points: all waiting happens in the loop and in the routines effects start. This is a convention, not a check. An update that reads a file still works, but it holds up the frame, and it cannot be replayed.

### Next is the whole answer

```zirric
data Next {
	model
	effects: [Effect]
}

// Next(model, []): the common case of an update with nothing to do.
fn keep(model) -> Next
```

Zirric has no tuples and no multiple returns, and needs neither: an update's answer is one value with two fields. Because `==` compares type and fields, `tui.Next(0, [tui.Quit()]) == tui.Next(0, [tui.Quit()])` holds, so a test can compare a whole step at once.

### Effects are values

```zirric
union Effect {
	data Quit
	data After { delay: time.Duration, tag }
	data Call { task: fn() -> Any }
	data Stream { body: fn(co.Channel) -> Any }
	data Print { text: String }
}

data Ticked { tag }
```

| Effect              | What the loop does                                                                                                           |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `Quit()`            | Draws the final frame, cancels every routine the program started, restores the terminal, and returns `Ok(model)` from `run`. |
| `After(delay, tag)` | Delivers `Ticked(tag)` once `delay` has passed on the program's `co.Timer`.                                                  |
| `Call(task)`        | Runs `task` in a routine and delivers its return value as a message.                                                         |
| `Stream(body)`      | Runs `body` in a routine, as `co.produce` does, and delivers every value it sends as a message.                              |
| `Print(text)`       | Writes `text` above an inline program, where it scrolls away with the terminal's history.                                    |

`After` fires once. Something that repeats, such as a spinner, schedules the next `After` when it handles the current `Ticked`. A component that stops caring simply stops asking. There are no subscriptions to register, or to forget to cancel.

`Call` is for work with one answer: reading a file, asking a server. The task should return a message of the program's own type, so `update` knows what arrived:

```zirric
data Loaded { result: Result }

const load = tui.Call(fn() { return Loaded(fs.readString(fsys, path)) })
```

`Stream` is for work with many answers, such as following a log or reporting progress. Its body receives a channel of its own, and every value sent into it becomes a message:

```zirric
data Progress { done: Int }

const copying = tui.Stream(fn(ch) {
	var done = 0
	for path <- paths {
		_ = backUp(fsys, path)
		done = done + 1
		co.send(ch, Progress(done))
	}
})
```

The capabilities a task needs (`fsys` here) are captured from where the program was built. The program never calls `os`; the entry point does, and a test passes `fs.memory()` instead.

### The terminal is a capability

```zirric
union Event {
	data Key { name: String, text: String }
	data Paste { text: String }
	data Mouse { button: String, x: Int, y: Int }
	data Resize { width: Int, height: Int }
}

data Size { width: Int, height: Int }

union Mode {
	data FullScreen
	data MouseTracking
	data BracketedPaste
}

data Terminal {
	enter: fn([Mode]) -> Result
	leave: fn() -> Void
	size: fn() -> Size
	next: fn() -> Event?
	write: fn(String) -> Void
}
```

`Terminal` follows `fs.FileSystem` and `co.Timer`:

- the primitives go in the fields;
- conveniences go in module functions;
- the real one comes only from `os`;
- a double ships in the library.

`os.terminal()` returns a `Result`, because standard input or output may not be a terminal at all (a pipe, a CI log). That is an expected failure, not a bug.

The real terminal reads input on its own host stream, like `os.stdin()`, and decodes it into events, including `Resize` when the window changes.

Following ZE-025, **the module decides where a routine gives way, not the value behind it.** `terminal.next(t)` and `terminal.write(t, text)` are switch points whatever terminal `t` is, just as `io.read` is for any reader. A scripted terminal in a test therefore interleaves at the same points as the real one.

A key is identified by one canonical `name`, which is what `switch` matches:

| Kind      | Names                                                                                                               |
| --------- | ------------------------------------------------------------------------------------------------------------------- |
| Printable | the character itself: `"a"`, `"A"`, `"7"`, `"?"`; the space bar is `"space"`                                        |
| Editing   | `"enter"`, `"tab"`, `"backspace"`, `"delete"`, `"esc"`, `"insert"`                                                  |
| Movement  | `"up"`, `"down"`, `"left"`, `"right"`, `"home"`, `"end"`, `"pgup"`, `"pgdown"`                                      |
| Function  | `"f1"` … `"f12"`                                                                                                    |
| Modified  | modifiers in the fixed order `ctrl+`, `alt+`, `shift+`: `"ctrl+c"`, `"alt+enter"`, `"shift+tab"`, `"ctrl+alt+left"` |

`text` is what the key would insert, and `""` for anything that inserts nothing. A text input appends `key.text` and never needs to know the name of every printable key. An uppercase letter is `"A"`, not `"shift+a"`, because that is what the user typed.

### Rendering is a string

`view` returns a `String`: lines separated by `"\n"`, possibly carrying the escape sequences `tui.style` produces. Strings already compose with `+`, `strings`, `fmt` and `for` expressions, and a test can compare them. `tui.layout` measures and arranges strings, and `tui.style.plain` strips styling so a test can compare the text alone.

```zirric
const title = [style.Bold(), style.Fg(style.cyan)]

fn header(text: String, width: Int) -> String {
	return layout.pad(style.paint(text, title), width, layout.Center())
}
```

A style is an array of `style.Look` values, not a `data` type with a field per attribute. Positional construction of eight booleans and two colors would be unreadable. An array names only what it sets, can be reused as a `const`, and combines with `+`.

### Components are modules

There is no component attribute and no component type. By convention, a component is a module with a `data` type for its state and these functions:

| Function         | Signature                                                       | Required        |
| ---------------- | --------------------------------------------------------------- | --------------- |
| `new`            | `(…) -> T`                                                      | yes             |
| `update`         | `(T, Any) -> T`, or `(T, Any) -> tui.Next` when it has effects  | yes             |
| `view`           | `(T) -> String`                                                 | yes             |
| starting effects | `(T) -> tui.Effect`, named for what they start (`spinner.tick`) | when it has any |

A parent stores the component's state in its own model, forwards messages to its `update` and places its `view`. Nothing is registered, and a reader can follow every message by reading the parent's `update`.

## Detailed Design

### The loop

`tui.run(program, term, timer)` owns the terminal for the duration of the program:

```zirric
fn run(program: Program, term: terminal.Terminal!, timer: co.Timer) -> Result
```

`term` is the `Result` that `os.terminal()` returns, the way `fs.withFile` takes the result of `fs.open`, so the entry point needs no `switch` of its own. An `Err` is returned unchanged.

1. Enter the terminal with `program.modes`. Raw mode and a hidden cursor are always on. An `Err` from `enter` is returned.
2. Open a `co.scope`, which owns every routine the program starts. Start the input routine with `terminal.events`.
3. Take `program.start` as the current step and perform its effects.
4. Deliver `terminal.Resize` with the current size as the first message. Every program learns its size the same way it learns it later.
5. Until a `Quit` has been performed, repeat:
   1. Call `update(model, message)`. Its result must be a `Next`; anything else is a bug in the program, and the loop panics with a message naming `update`.
   2. Draw `view(model)`.
   3. Perform the step's effects in order. `After`, `Call` and `Stream` each spawn a routine into the scope that sends its messages into the loop's inbox.
   4. Wait with `co.select([input, inbox])` for the next message.
6. Cancel the scope, draw the final frame, leave the terminal, and return `Ok(model)`.

**Input comes first.** `select` takes the first ready channel in list order, so a key waiting alongside a task result is handled first. The user is never kept waiting behind work they may be about to cancel. Within the inbox, messages arrive in the order their routines sent them.

**The loop needs nothing beyond `co`.** In outline, leaving out drawing and `Print`:

```zirric
fn _loop(s: co.Scope, program: Program, t: terminal.Terminal, timer: co.Timer) -> Result {
	const input = terminal.events(s, t)
	const inbox = co.channel(0)
	var model = program.start.model
	var quitting = _perform(s, inbox, timer, program.start.effects)
	const size = terminal.size(t)
	var message = terminal.Resize(size.width, size.height)
	for !quitting {
		const next = _step(program, model, message)
		model = next.model
		_draw(t, program.view(model))
		quitting = _perform(s, inbox, timer, next.effects)
		if !quitting {
			const selected = co.select([input, inbox])
			switch selected {
			case is co.Closed:
				quitting = true              // standard input ended
			case is co.Received:
				message = selected.value
			}
		}
	}
	co.cancel(s)
	return Ok(model)
}

fn _perform(s: co.Scope, inbox: co.Channel, timer: co.Timer, effects: [Effect]) -> Bool {
	var quitting = false
	for effect <- effects {
		switch effect {
		case is Quit:
			quitting = true
		case is After:
			_ = co.spawn(s, fn() {
				co.sleep(timer, effect.delay)
				co.send(inbox, Ticked(effect.tag))
				return Ok(void)
			})
		case is Call:
			_ = co.spawn(s, fn() {
				co.send(inbox, effect.task())
				return Ok(void)
			})
		case is Stream:
			_ = co.spawn(s, fn() {
				for message <- co.produce(s, effect.body) {
					co.send(inbox, message)
				}
				return Ok(void)
			})
		}
	}
	return quitting
}
```

Each closure captures its own `effect`, a property ZE-025 pins with regression tests.

Because ZE-025 runs one routine at a time, `update` and `view` never race with a task: the model is only ever touched by the loop's own routine. Tasks run whenever the loop waits. A task that reads a file or talks to a server gives way at every switch point, so the spinner keeps turning. A task that computes for a long time without one stalls the screen, as it would stall any other routine. Such a task should be split into steps, or become a `Stream` that reports as it goes.

### Quitting and cancelling

`Quit` cancels the program's scope. By ZE-025's rules:

- The input routine and pending `After`s stop at once.
- A byte already on its way stays in the host stream.
- A `Call` in the middle of a write finishes the write.

Quitting therefore never waits on a server, and never leaves a half-issued write behind.

Effects listed after a `Quit` in the same step are still performed, but routines they start are cancelled immediately. A program that needs a task to finish before it exits waits for its result message, and only then returns `Quit()`.

`run` leaves the terminal after `Quit`, when input ends, and on every `Err`. A `panic` cannot be caught in Zirric, so the host restores the terminal before it prints a panic and exits. This is a requirement on the implementation of `os.terminal()`, not something Zirric code does.

`ctrl+c` is delivered as `terminal.key("ctrl+c")`, like any other key, and does not quit on its own. Raw mode turns it into input, and a program decides what it means: cancel a dialog, clear a field. `run` does not guess.

### Replay

```zirric
data Trace {
	model
	frames: [String]
	effects: [Effect]
	quit: Bool
}

fn replay(program: Program, size: terminal.Size, messages: [Any]) -> Trace
```

`replay` runs the same steps as `run`, in the same order: `start`, then `Resize(size)`, then each message. It records three things:

- the frame drawn after every step;
- every effect produced;
- whether a `Quit` was among them.

Messages after a `Quit` are ignored.

**Replay never performs an effect.** It starts no routine and needs no timer, so a replay is a pure function of its arguments.

- A `Call` is recorded, not called. A test that wants its result calls `effect.task()` itself, with whatever doubles it built the program with, and passes the result on as a message.
- An `After` is recorded, not waited for. A test that wants a tick passes a `tui.Ticked` it made.
- A `Stream` is recorded, not run. A test that wants its messages collects them with the tools ZE-025 already gives it:

```zirric
const messages = co.scope(fn(s) {
	return for message <- co.produce(s, effect.body) { message }
})
```

`terminal.key(name)` builds the `Key` a terminal would send for a name:

- `terminal.key("a") == terminal.Key("a", "a")`
- `terminal.key("down") == terminal.Key("down", "")`

`terminal.typed(text)` builds one key per character, so a test reads like the keystrokes it stands for.

### Running for real in a test

`replay` covers a program's logic. To test `run` itself, or a program end to end, `terminal.scripted` provides a terminal of a fixed size that hands out the given events and records what is written to it:

```zirric
@tests.Test()
fn testCounterEndToEnd() -> Result {
	const script = terminal.scripted(terminal.Size(40, 5), [terminal.key("up"), terminal.key("q")])
	return assert.equal(Ok(1), tui.run(counter.app, Ok(script.terminal), co.neverTimer()))
}
```

The scripted terminal's events interleave with task results wherever the scheduler reaches them. A test of this kind should not depend on a key arriving before or after a `Call`'s answer. Ordering is exactly what `replay` puts in the test's hands.

### Drawing

The loop draws line by line, and a line equal to the one already on screen is not rewritten. Lines are cut, never wrapped, at the terminal's width as `tui.layout.width` measures it. The loop therefore always knows how many rows a frame occupies.

| Mode             | Where the frame goes                                       | On exit                                            |
| ---------------- | ---------------------------------------------------------- | -------------------------------------------------- |
| inline (default) | from the cursor down, growing and shrinking with the frame | the final frame stays; the shell prompt follows it |
| `FullScreen()`   | the alternate screen, cut at the terminal's height         | the previous screen is restored                    |

`Print` writes above an inline frame. It is dropped in full-screen mode, where there is no history to write into.

Every frame is written with a single `terminal.write`, so it reaches the screen whole. `view` runs once per message, and a burst of messages draws a burst of frames. Line diffing keeps each one cheap. Coalescing frames would need a receive that does not block, which ZE-025 leaves for later; see Open Questions.

### Width

`tui.layout.width` counts terminal cells:

- escape sequences count zero;
- most characters count one;
- wide characters (CJK, most emoji) count two.

This needs the host's Unicode tables, so it is an `extern` with a Zirric-facing function over it, like `strings.count`. Neither `len` (bytes) nor `strings.count` (code points) is the width of text on a screen.

### Composition with the language

| Feature                     | Interaction                                                                                                                       |
| --------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `data`, `union`             | Models, messages and effects are ordinary values. Unions make phases explicit.                                                    |
| `switch` / `is`             | The only dispatch mechanism. `case is Key:` narrows, so `msg.name` is checked.                                                    |
| `==`                        | Steps, effects (except `Call` and `Stream`) and keys compare by value; tests rely on it.                                          |
| `co`                        | The loop is a scope; effects are routines; input and results are channels. No new rules.                                          |
| attributes                  | None required. `HasTerminal` exists for code that wants the capability from an environment.                                       |
| `for` expressions           | Build lists of lines; `continue` filters; collect a `Stream` in tests.                                                            |
| `?.` / `!.`                 | Unaffected. Tasks can use `!.` inside a function returning a result.                                                              |
| static checks               | Hints on `Program` document intent. The checker does not look inside function-typed hints yet, so they cannot raise false alarms. |
| formatter, LSP, tree-sitter | No changes.                                                                                                                       |

`Call` and `Stream` hold functions, and functions compare by identity. Tests check them with `is tui.Call` or `is tui.Stream`, and by running them.

### Deliberately not

- **No parallelism.** Inherited from ZE-025, and relied on: the model has one owner.
- **No layout engine.** `tui.layout` joins and pads strings; it does not solve constraints.
- **No focus manager, key-binding registry or command palette.** They are components or patterns, and belong in packages.
- **No global quit key.** `ctrl+c` is a key.
- **No subscriptions.** Repetition is an `After` scheduled again. A long-lived source is a `Stream`.
- **No view tree.** `view` returns a string.
- **No components in the standard library.** See Alternatives Considered.

## Changes to the Standard Library

All additions. Nothing existing changes.

### `os`

```zirric
// Returns the terminal attached to standard input and output, or an Err when either is not a terminal.
extern fn terminal() -> Result
```

### `terminal`

Vocabulary and capability, as above, plus:

```zirric
// Provides the terminal, for code that takes its capabilities from an environment.
attr HasTerminal {
	terminal(self: @HasTerminal) -> Terminal
}

fn enter(t: Terminal, modes: [Mode]) -> Result
fn leave(t: Terminal) -> Void
fn size(t: Terminal) -> Size
// The next event, or None() once input has ended. A switch point.
fn next(t: Terminal) -> Event?
// Writes text as one piece. A switch point.
fn write(t: Terminal, text: String) -> Void
// Every event, in a channel that closes when input ends.
fn events(s: co.Scope, t: Terminal) -> co.Channel

// The Key a terminal sends for a canonical name.
fn key(name: String) -> Key
// One Key per character of text.
fn typed(text: String) -> [Key]

// A terminal of a fixed size that hands out the given events, then ends its input,
// and records everything written to it.
fn scripted(size: Size, events: [Event]) -> Script

data Script {
	terminal: Terminal
	output: fn() -> String
}
```

### `tui`

`Program`, `Next`, `Effect` with `Quit`, `After`, `Call`, `Stream` and `Print`, `Ticked`, `Trace`, and:

```zirric
fn keep(model) -> Next
fn run(program: Program, term: terminal.Terminal!, timer: co.Timer) -> Result
fn replay(program: Program, size: terminal.Size, messages: [Any]) -> Trace
```

### `tui.style`

```zirric
union Color {
	data Ansi { index: Int }                        // 0–255
	data Rgb { red: Int, green: Int, blue: Int }    // 0–255 each
}

const black = Ansi(0)
const red = Ansi(1)
const green = Ansi(2)
const yellow = Ansi(3)
const blue = Ansi(4)
const magenta = Ansi(5)
const cyan = Ansi(6)
const white = Ansi(7)
const gray = Ansi(8)

union Look {
	data Fg { color: Color }
	data Bg { color: Color }
	data Bold
	data Faint
	data Italic
	data Underline
	data Reverse
}

// text with every look applied, reset at each line end so styles never leak.
fn paint(text: String, looks: [Look]) -> String
// text with every escape sequence removed.
fn plain(text: String) -> String
```

`run` decides from the terminal whether color is emitted at all (for example, `NO_COLOR`). `paint` does not, so a view is the same string in a test and on a screen. `replay` keeps styles; `plain` removes them.

### `tui.layout`

```zirric
union Align {
	data Start
	data Center
	data End
}

fn width(text: String) -> Int                    // cells of the widest line
fn height(text: String) -> Int                   // number of lines
fn pad(text: String, width: Int, align: Align) -> String
fn truncate(text: String, width: Int) -> String  // cut with "…"
fn row(blocks: [String], gap: Int) -> String     // side by side, top-aligned
fn column(blocks: [String]) -> String            // one below the other
fn box(text: String) -> String                   // a rounded border around text
```

## The Components Package

Components ship in a first-party package outside the standard library, working name `widgets`. This proposal specifies only its first two components. Their job is to show that the convention holds, both with effects and without. Further components (lists, viewports, tables, progress bars) are the package's business, and need no proposal.

### `widgets.spinner`

```zirric
data Spinner {
	frames: [String]
	index: Int
	interval: time.Duration
	tag
}

fn new(tag) -> Spinner
fn tick(s: Spinner) -> tui.Effect {
	return tui.After(s.interval, s.tag)
}
fn update(s: Spinner, msg) -> tui.Next {
	if msg is tui.Ticked && msg.tag == s.tag {
		const moved = Spinner(s.frames, (s.index + 1) % len(s.frames), s.interval, s.tag)
		return tui.Next(moved, [tick(moved)])
	}
	return tui.keep(s)
}
fn view(s: Spinner) -> String {
	return s.frames[s.index]
}
```

The `tag` is how two spinners in one program tell their ticks apart. It is any value the parent chooses, compared with `==`.

### `widgets.textinput`

```zirric
data TextInput {
	value: String
	cursor: Int
	placeholder: String
	focused: Bool
}

fn new(placeholder: String) -> TextInput
fn focus(input: TextInput) -> TextInput
fn blur(input: TextInput) -> TextInput
fn update(input: TextInput, msg) -> TextInput   // no effects, so no Next
fn view(input: TextInput) -> String
```

## A Complete Program

A todo picker. It reads a file while a spinner turns, lets the user move through the lines, and prints the chosen one.

```zirric
mod code.example.todos

import fmt
import fs
import os
import strings
import terminal { Key }
import tui
import tui.layout
import tui.style
import widgets.spinner

// The message the load task sends back.
data Loaded {
	result: Result
}

// Where the program is. Each phase holds only what it needs.
union Phase {
	data Loading { busy: spinner.Spinner }
	data Ready { items: [String], cursor: Int }
	data Failed { reason }
}

// Builds the program. The filesystem comes in, so tests can pass fs.memory().
fn program(fsys: fs.FileSystem, path: String) -> tui.Program {
	const busy = spinner.new("loading")
	const load = tui.Call(fn() { return Loaded(fs.readString(fsys, path)) })
	return tui.Program(tui.Next(Loading(busy), [load, spinner.tick(busy)]), update, view, [])
}

fn update(model: Phase, msg) -> tui.Next {
	if msg == terminal.key("ctrl+c") {
		return tui.Next(model, [tui.Quit()])
	}
	switch model {
	case is Loading:
		return whileLoading(model, msg)
	case is Ready:
		return whileReady(model, msg)
	}
	switch msg {
	case is Key:
		return tui.Next(model, [tui.Quit()])
	}
	return tui.keep(model)
}

fn whileLoading(model: Loading, msg) -> tui.Next {
	switch msg {
	case is Loaded:
		return tui.keep(fromFile(msg.result))
	case is tui.Ticked:
		const next = spinner.update(model.busy, msg)
		return tui.Next(Loading(next.model), next.effects)
	}
	return tui.keep(model)
}

fn fromFile(result: Result) -> Phase {
	switch result {
	case is Err:
		return Failed(result.reason)
	}
	const items = for line <- strings.split(result.value, "\n") {
		if line == "" { continue } else { line }
	}
	return Ready(items, 0)
}

fn whileReady(model: Ready, msg) -> tui.Next {
	switch msg {
	case is Key:
		return switch msg.name {
		case "up":
			tui.keep(move(model, -1))
		case "down":
			tui.keep(move(model, 1))
		case "enter":
			tui.Next(model, [tui.Quit()])
		case "q":
			tui.Next(model, [tui.Quit()])
		case _:
			tui.keep(model)
		}
	}
	return tui.keep(model)
}

fn move(ready: Ready, by: Int) -> Ready {
	const wanted = ready.cursor + by
	const last = len(ready.items) - 1
	const cursor = if wanted < 0 { 0 } else if wanted > last { last } else { wanted }
	return Ready(ready.items, cursor)
}

fn view(model: Phase) -> String {
	return switch model {
	case is Loading:
		spinner.view(model.busy) + " Reading todos…"
	case is Ready:
		viewReady(model)
	case _:
		style.paint("Could not read todos: " + fmt.sprint(model.reason), [style.Fg(style.red)])
	}
}

fn viewReady(ready: Ready) -> String {
	var lines = []
	var index = 0
	for item <- ready.items {
		const line = if index == ready.cursor {
			style.paint("> " + item, [style.Bold(), style.Fg(style.cyan)])
		} else {
			"  " + item
		}
		lines = append(lines, line)
		index = index + 1
	}
	return layout.column(lines)
}

// Entry point: the only place that reaches for os.
const finished = tui.run(program(os.fs(), "todo.txt"), os.terminal(), os.timer())
switch finished {
case is Err:
	fmt.fprintln(finished.reason, os.stderr())
	os.exit(1)
case is Ok:
	const last = finished.value
	if last is Ready {
		fmt.fprintln(last.items[last.cursor], os.stdout())
	}
}
```

And its tests:

```zirric
mod code.example.todos._t

import fs
import terminal
import tests
import tests.assert
import tui
import tui.style
import code.example.todos

const size = terminal.Size(40, 10)

@tests.Test()
fn testStartsLoadingAndTicking() -> Result {
	const trace = tui.replay(todos.program(fs.memory(), "todo.txt"), size, [])
	return assert.all([
		assert.isTrue(trace.model is todos.Loading),
		assert.isTrue(trace.effects[0] is tui.Call),
		assert.isTrue(trace.effects[1] is tui.After)
	])
}

@tests.Test()
fn testPicksSecondItem() -> Result {
	const fsys = fs.memory()
	_ = fs.writeString(fsys, "todo.txt", "milk\neggs\n")
	const program = todos.program(fsys, "todo.txt")
	// replay performs no effects: run the load here and hand its message in
	const load = tui.replay(program, size, []).effects[0]
	const trace = tui.replay(program, size, [load.task(), terminal.key("down"), terminal.key("enter")])
	return assert.all([
		assert.equal(todos.Ready(["milk", "eggs"], 1), trace.model),
		assert.isTrue(trace.quit),
		assert.equal("  milk\n> eggs", style.plain(trace.frames[len(trace.frames) - 1]))
	])
}

@tests.Test()
fn testMissingFileFails() -> Result {
	const program = todos.program(fs.memory(), "todo.txt")
	const load = tui.replay(program, size, []).effects[0]
	const trace = tui.replay(program, size, [load.task()])
	return assert.isTrue(trace.model is todos.Failed)
}
```

## Compatibility

Purely additive. No syntax, keyword, operator or checker rule changes, and no work for the formatter, language server or tree-sitter grammar. The draft `term`, `ui` and `colors` packages are superseded; nothing in the standard library depends on them.

## Dependencies on Other Proposals

| Proposal                                                                     | Dependency                                                                                                              |
| ---------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| [ZE-008](https://zirric.knabel.dev/proposals/ZE-008-error-handling/)         | `os.terminal()` and `run` report expected failures as `Result`s.                                                        |
| [ZE-010](https://zirric.knabel.dev/proposals/ZE-010-iterable/)               | Tests collect a `Stream` with a `for` expression over a channel.                                                        |
| [ZE-017](https://zirric.knabel.dev/proposals/ZE-017-type-hints/)             | `update` dispatches with `case is`; `Program`'s fields carry function type hints.                                       |
| [ZE-018](https://zirric.knabel.dev/proposals/ZE-018-io-fmt-os/)              | `os` is the only module that reaches the host; `terminal()` joins it.                                                   |
| [ZE-022](https://zirric.knabel.dev/proposals/ZE-022-static-checks/)          | Hints on `Program` must only ever report certainties.                                                                   |
| [ZE-024](https://zirric.knabel.dev/proposals/ZE-024-qualified-module-names/) | Examples use qualified module paths.                                                                                    |
| [ZE-025](https://zirric.knabel.dev/proposals/ZE-025-routines-and-channels/)  | The loop is a `co.scope`; effects are routines; `terminal.next` and `write` are switch points; timers are a `co.Timer`. |

## Alternatives Considered

### Do nothing: every program writes its own loop

Since ZE-025, a program can write its own `co.select` loop, as ZE-025's spinner example does. That is the strongest form of doing nothing, and it fails in two places:

- Raw mode, key decoding and size need the host, and by the design of ZE-018 only `os` reaches the host. At minimum, `os.terminal()` and the `terminal` module must exist.
- Hand-written loops do not compose. A text input written against one program's inbox and message types cannot be dropped into another program. The loop is the contract components are written against, just as `@io.Writer` is the contract for streams.

### Where each piece lives

The draft `term`, `ui` and `colors` packages split the domain differently. This proposal replaces them, with each piece placed by one question: what breaks if it changes?

- **`terminal` belongs in the standard library.** It is the only way to reach the host, like `fs` and `co.Timer`, and its real implementation lives in `os`.
- **`tui` belongs in the standard library.** It is small, depends only on `terminal`, `co` and `time`, and is the shared contract that makes packages composable. Two competing loops would split the ecosystem, and a loop that changes breaks every component at once. The standard library is where such a contract earns stability.
- **`tui.style` and `tui.layout` belong in the standard library, next to `tui`.** They are pure and small, and they are coupled to the loop: `layout.width` must skip exactly the escape sequences `style.paint` emits, and the loop cuts lines by that width. As a separate package (`colors`), that agreement would be an accident rather than a guarantee.
- **Components do not belong in the standard library.** They grow, churn and carry taste. The standard library is meant to be basic, and "which list widget" is not a basic question. A first-party package can move quickly, and third-party components compete on equal terms, because they all target the same `tui` contract.

### A program as an attribute on the model type

`@tui.App(update, view)` on `data Model` is closest to Bubble Tea's `Model` interface. It is rejected because a program is one value, not an open set of types, and nothing needs to discover it by reflection. By the standard library's own order, a `data` value is cheaper than an attribute. An attribute would also force the model to be a `data` type, when a `union` is often the better model.

### Commands as functions

In Bubble Tea, `tea.Cmd` is a function (`fn() -> Any`) that the runtime runs later. Making that the only kind of effect is rejected: a function cannot be compared, printed or inspected, so a test could not see that an update asked to quit. `Call` and `Stream` keep that flexibility for the two cases that need it, and ZE-025 is what now runs them.

### Letting update spawn routines itself

`update` could take a `co.Scope` and start its own routines. This is rejected because updates would stop being replayable, the model would gain a second writer, and every test would need a scope and a timer. Effects keep that power in the loop.

### Returning the model and effects separately

Effects could come from a second function `effects(model)`, or be stored in the model. This is rejected because it splits one decision across two places, and effects stored in the model must be cleared by hand after they run.

### Returning either a model or a `Next`

Letting `update` return a bare model would spare the common case its `tui.keep`. This is rejected because `update` would then return two shapes, the loop would have to guess which one it got, and a model that happens to be a `Next` would be misread. One return shape, no ambiguity.

### Lowercase helpers for effects and looks

Helpers such as `tui.quit()` and `style.bold()` beside `tui.Quit()` and `style.Bold()` are rejected. A field-less member is already constructed with a call, so the helpers would only be a second spelling of the same thing. `tui.keep` stays because it is not a spelling: it supplies the empty effect list.

### An immediate-mode or widget-tree API

ncurses and tview work with a draw call per cell or a tree of widgets. This is rejected because it hides the state in the widgets, and a test must drive a fake screen to see anything.

### A view tree or cell buffer

A tree or buffer, like Elm's `Html` or Ratatui's `Buffer`, is stronger for layout, since the renderer knows the structure. It is rejected for now: a string is what every other module already produces, and a tree brings a layout engine with it. A later proposal can add one beside strings.

### A `Style` data type with one field per attribute

This is Lip Gloss's `Style`. It is rejected because, without named construction ([ZE-003](https://zirric.knabel.dev/proposals/ZE-003-named-data-construction/), rejected), a ten-field positional constructor is unreadable. An array of `Look` values names only what it sets.

## Open Questions

1. **Coalescing frames.** The loop draws after every message. Draining whatever else is ready before drawing needs a `select` that does not block, which ZE-025 lists as a possible later addition. Is that addition worth making for this use, or is line diffing enough?
2. **Rebuilding models.** Updating one field of a large `data` model means passing every other field to its constructor again. Unions of small phases keep this in check in the examples. Is that enough, or does this deserve its own proposal?
3. **`Stream` and `Call`.** `Call` is a `Stream` that sends once. Keeping both makes the common case testable by calling a function. Is one effect with one test pattern better than two?
4. **Priority.** Input comes before task results. A held-down key could delay results, but never the reverse. Is that the right way round?
5. **Handing over the terminal.** Running `$EDITOR` or a pager from a TUI means leaving raw mode and giving standard input to a child process. ZE-025 notes this needs the host stream to be quiesced. Should `tui` gain an `Exec` effect once a process module exists?
6. **The components package.** Is `widgets` the right name, and should it live under `zirric-lang` alongside the standard library's repository?

## Acknowledgements

The architecture is [The Elm Architecture](https://guide.elm-lang.org/architecture/), by way of Charm's [Bubble Tea](https://github.com/charmbracelet/bubbletea), whose split into model, update, view and commands this proposal keeps. Elm's commands are also data interpreted by a runtime, but they are opaque; here they are an ordinary union.

Other prior art:

- `tui.style` and `tui.layout` borrow their scope from [Lip Gloss](https://github.com/charmbracelet/lipgloss).
- The component convention comes from [Bubbles](https://github.com/charmbracelet/bubbles).
- `replay` comes from Bubble Tea's `teatest`.
- [Ratatui](https://ratatui.rs) and [Textual](https://textual.textualize.io) informed the view-tree alternative.
