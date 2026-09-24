---
title: Co
description: Routines that take turns, scopes that own them, and channels between them.
---

# Module `co`

```zirric
import co
```

|            |                                                                                                                                                                                                                                                                                                                                                                                                |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `co`                                                                                                                                                                                                                                                                                                                                                                                           |
| **Source** | [`co/co.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/co/co.zirr), [`co/helpers.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/co/helpers.zirr), [`co/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/co/module-docs.zirr), [`co/timers.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/co/timers.zirr) |

> Concurrency without parallelism, and without new syntax.

A blocking call blocks the whole program, which is fine until something has to wait on two things at once — a key and a timer, a slow job and a user who presses `q`. `co` adds routines for that, and nothing else: no `go` keyword, no send or receive operator, no `async`/`await`. Everything here is a function.

Two rules govern it. **One routine runs at a time, and it only gives way at a switch point** — [`send`](#send), [`receive`](#receive), [`select`](#select), [`wait`](#wait), [`sleep`](#sleep), iterating a channel, the end of a [`scope`](#scope-fn) body, and every stdlib call that waits on the outside world: [`io.read`](../io/index.md#read), [`io.write`](../io/index.md#write), `fmt.fprint*` and the functions of [`fs`](../fs/index.md), whatever reader, writer or filesystem they are given. Between two switch points, code runs as if it were alone, so a shared `var` or array needs no lock. And **every routine belongs to a scope, and a scope does not end before its routines** — [`scope`](#scope-fn) is the only way to get a [`Scope`](#scope-type), so nothing runs in the background that the code does not show.

A [`Channel`](#channel-type) is `@Iterable`, which is why a producer and a consumer need nothing new: `for line <- produce(s, ...)` receives until the producer closes. [`select`](#select) takes from the first ready channel in list order, so priority is visible at the call site and the same in every run.

[`Timer`](#timer) is a capability, passed in like [`fs.FileSystem`](../fs/index.md#filesystem) is. [`immediateTimer`](#immediatetimer) fires before a spawned body ever runs and [`neverTimer`](#nevertimer) never fires, so a timeout is tested in both directions without waiting for either.

[`cancel`](#cancel) stops a routine at the switch point it is parked on, and returns at once. What it cannot take back is a call already handed to the operating system: a write in flight reaches the file or the terminal, possibly after [`scope`](#scope-fn) has returned. A read is different, because bytes cannot be asked for twice — a host stream owns its reads, so a routine cancelled while waiting for input leaves the bytes behind for whoever reads next rather than discarding them.

## Dependencies

- [`time`](../time/index.md)

---

## Contents

- **Unions** — [`Selected`](#selected)
- **Data** — [`Cancelled`](#cancelled), [`Closed`](#closed), [`Received`](#received), [`TimedOut`](#timedout), [`Timer`](#timer)
- **Attributes** — [`HasTimer`](#hastimer)
- **Types** — [`Channel`](#channel-type), [`Routine`](#routine), [`Scope`](#scope-type)
- **Functions** — [`after`](#after), [`all`](#all), [`cancel`](#cancel), [`channel`](#channel-fn), [`close`](#close), [`first`](#first), [`immediateTimer`](#immediatetimer), [`map`](#map), [`merge`](#merge), [`neverTimer`](#nevertimer), [`produce`](#produce), [`receive`](#receive), [`scope`](#scope-fn), [`select`](#select), [`send`](#send), [`sleep`](#sleep), [`spawn`](#spawn), [`timeout`](#timeout), [`wait`](#wait)

---

## Unions

### `Selected` {#selected}

<small>`co/co.zirr:48`</small>

```zirric
union Selected {
	Received
	Closed
}
```

What [`select`](#select) found.

#### Cases

| Case       | Interpretation                                                                                        |
| ---------- | ----------------------------------------------------------------------------------------------------- |
| `Received` | A value was taken from channel.                                                                       |
| `Closed`   | channel is closed and drained, so it will never carry anything again. Drop it from the list, or stop. |

---

## Data

### `Cancelled` {#cancelled}

<small>`co/co.zirr:66`</small>

```zirric
data Cancelled
```

The routine was stopped by [`cancel`](#cancel).

---

### `Closed` {#closed}

<small>`co/co.zirr:58`</small>

```zirric
data Closed {
	channel: Channel
}
```

channel is closed and drained, so it will never carry anything again. Drop it from the list, or stop.

#### Fields

| Field     | Description             |
| --------- | ----------------------- |
| `channel` | The channel that ended. |

---

### `Received` {#received}

<small>`co/co.zirr:50`</small>

```zirric
data Received {
	channel: Channel
	value
}
```

A value was taken from channel.

#### Fields

| Field     | Description                      |
| --------- | -------------------------------- |
| `channel` | The channel the value came from. |
| `value`   | The value.                       |

---

### `TimedOut` {#timedout}

<small>`co/timers.zirr:44`</small>

```zirric
data TimedOut {
	limit: Duration
}
```

[`timeout`](#timeout) gave up before the body finished.

#### Fields

| Field   | Description                  |
| ------- | ---------------------------- |
| `limit` | The limit that was exceeded. |

---

### `Timer` {#timer}

<small>`co/timers.zirr:12`</small>

```zirric
data Timer {
	after: fn(Duration) -> Channel
}
```

The capability to wait for time to pass.
It is a plain value holding a function, so a test hands over one that fires at once or one that never fires, and waits for nothing.

#### Fields

| Field   | Description |
| ------- | ----------- |
| `after` |             |

---

## Attributes

### `HasTimer` {#hastimer}

<small>`co/timers.zirr:6`</small>

```zirric
attr HasTimer {
	timer(self: @HasTimer) -> Timer
}
```

Provides the capability to wait for time to pass, the way clock.HasMonotonicClock provides a clock.

#### Fields

| Field   | Description |
| ------- | ----------- |
| `timer` |             |

---

## Types

### `Channel` {#channel-type}

<small>`co/co.zirr:11`</small>

```zirric
extern type Channel
```

A queue between routines. It is @Iterable, so a `for` loop receives from it until it is closed and drained.

---

### `Routine` {#routine}

<small>`co/co.zirr:7`</small>

```zirric
extern type Routine
```

A running function, returned by [`spawn`](#spawn).

---

### `Scope` {#scope-type}

<small>`co/co.zirr:4`</small>

```zirric
extern type Scope
```

Owns routines. Only [`scope`](#scope-fn) creates one, so every [`spawn`](#spawn) traces back to a visible [`scope`](#scope-fn) call.

---

## Functions

### `after` {#after}

<small>`co/timers.zirr:17`</small>

```zirric
fn after(timer: Timer, d: Duration) -> Channel
```

A channel that receives one value after d has passed, and then closes.

---

### `all` {#all}

<small>`co/helpers.zirr:4`</small>

```zirric
fn all(bodies: [fn() -> Result]) -> [Result]
```

Runs every body at once and returns every result, in the order the bodies were given.

---

### `cancel` {#cancel}

<small>`co/co.zirr:27`</small>

```zirric
extern fn cancel(s: Scope) -> Void
```

Stops every routine spawned into s, each at the switch point it is parked on.
Neither the scope's body nor the calling routine is stopped, and buffered values are lost: cancelling closes nothing.

---

### `channel` {#channel-fn}

<small>`co/co.zirr:30`</small>

```zirric
extern fn channel(capacity: Int) -> Channel
```

A channel holding up to capacity values. A capacity of 0 makes every send wait for a receiver.

---

### `close` {#close}

<small>`co/co.zirr:41`</small>

```zirric
extern fn close(ch: Channel) -> Void
```

Says no more values will be sent. The values already in the channel are still received.
Closing a closed channel is a bug in the caller, so it stops the program.

---

### `first` {#first}

<small>`co/helpers.zirr:29`</small>

```zirric
fn first(bodies: [fn() -> Result]) -> Result
```

The result of whichever body finishes first. The rest are stopped.

---

### `immediateTimer` {#immediatetimer}

<small>`co/timers.zirr:28`</small>

```zirric
fn immediateTimer() -> Timer
```

A timer whose every [`after`](#after) has already fired. This is how code under a timeout is tested without waiting for one.

---

### `map` {#map}

<small>`co/helpers.zirr:12`</small>

```zirric
fn map(items: @Iterable, limit: Int, transform: fn(Any) -> Result) -> [Result]
```

Applies transform to every item, at most limit at a time, and returns the results in order.

---

### `merge` {#merge}

<small>`co/helpers.zirr:77`</small>

```zirric
fn merge(s: Scope, channels: [Channel]) -> Channel
```

All values from all channels in one, which closes once the last of them has.

---

### `neverTimer` {#nevertimer}

<small>`co/timers.zirr:38`</small>

```zirric
fn neverTimer() -> Timer
```

A timer that never fires, so the same code is tested as though its deadline were never reached.

---

### `produce` {#produce}

<small>`co/helpers.zirr:66`</small>

```zirric
fn produce(s: Scope, body: fn(Channel) -> Any) -> Channel
```

A generator: body sends into the channel, which closes when body returns.
The channel is unbuffered, so the producer moves in lockstep with its consumer. A producer that should run ahead is written with [`channel`](#channel-fn) and [`spawn`](#spawn) directly.

---

### `receive` {#receive}

<small>`co/co.zirr:37`</small>

```zirric
extern fn receive(ch: Channel) -> Option
```

Blocks until a value arrives, and returns it as Some. Once the channel is closed and drained, None, forever.

---

### `scope` {#scope-fn}

<small>`co/co.zirr:15`</small>

```zirric
extern fn scope(body: fn(Scope) -> Any) -> Any
```

Runs body with a fresh scope, waits for every routine spawned into it, and returns body's value.
Nothing outlives the block: a forgotten [`wait`](#wait) never loses work, and no routine is left running behind it.

---

### `select` {#select}

<small>`co/co.zirr:45`</small>

```zirric
extern fn select(channels: [Channel]) -> Selected
```

Blocks until one of the channels can be received from, and takes from the first ready one in list order.
Order is priority: visible at the call site and the same in every run. The cost is that a channel that is always ready starves the ones after it.

---

### `send` {#send}

<small>`co/co.zirr:34`</small>

```zirric
extern fn send(ch: Channel, value: Any) -> Void
```

Blocks until there is room, then puts value in.
Sending on a closed channel is a bug in the caller, so it stops the program.

---

### `sleep` {#sleep}

<small>`co/timers.zirr:22`</small>

```zirric
fn sleep(timer: Timer, d: Duration)
```

Blocks the calling routine for d. The other routines keep running.

---

### `spawn` {#spawn}

<small>`co/co.zirr:19`</small>

```zirric
extern fn spawn(s: Scope, body: fn() -> Result) -> Routine
```

Starts body in a new routine owned by s and returns at once.
The new routine runs when the spawning one reaches a switch point, not before.

---

### `timeout` {#timeout}

<small>`co/helpers.zirr:48`</small>

```zirric
fn timeout(timer: Timer, limit: Duration, body: fn() -> Result) -> Result
```

body's result, or Err(TimedOut(limit)) if it is still running when limit passes.

---

### `wait` {#wait}

<small>`co/co.zirr:23`</small>

```zirric
extern fn wait(routine: Routine) -> Result
```

Blocks until the routine ends, and returns what it ended with: its own Result, or Err(Cancelled()).
May be called more than once and from any routine, and answers the same each time.
