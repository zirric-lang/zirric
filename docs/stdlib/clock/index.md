---
title: Clock
description: Wall-clock and monotonic time, as values a test can replace.
---

# Module `clock`

> Two kinds of clock, deliberately kept apart.

```zirric
import clock
```

|            |                                                                                                   |
| ---------- | ------------------------------------------------------------------------------------------------- |
| **Module** | `clock`                                                                                           |
| **Source** | [`clock/clock.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/clock/clock.zirr) |

There are two clocks, and confusing them is a bug. [`SystemClock`](#systemclock) reads civil time and can jump when the host is corrected — use it to timestamp and to do calendar work. [`MonotonicClock`](#monotonicclock) only moves forward and has a meaningless origin — use it to measure how long something took.

The attributes keep them apart at the type level: code that measures elapsed time requires [`HasMonotonicClock`](#hasmonotonicclock) and therefore cannot be handed a wall clock by mistake.

Both are plain data holding a function, so a test supplies its own. [`fixed`](#fixed) freezes time, [`stepping`](#stepping) advances it predictably, and [`steppingMonotonic`](#steppingmonotonic) with a zero step makes every reading identical — which is how code under a timeout is tested without waiting for one.

## Contents

- **Attributes** — [`HasSystemClock`](#hassystemclock), [`HasMonotonicClock`](#hasmonotonicclock)
- **Data** — [`SystemClock`](#systemclock), [`MonotonicClock`](#monotonicclock)
- **Functions** — [`now`](#now), [`instant`](#instant), [`fixed`](#fixed), [`stepping`](#stepping), [`steppingMonotonic`](#steppingmonotonic)

---

## Attributes

### `HasSystemClock` {#hassystemclock}

<small>`clock/clock.zirr:6`</small>

```zirric
attr HasSystemClock {
	clock(self: @HasSystemClock) -> SystemClock
}
```

Provides the wall clock, for timestamping and calendar work.

#### Members

| Member  | Signature                                     | Description                               |
| ------- | --------------------------------------------- | ----------------------------------------- |
| `clock` | `clock(self: @HasSystemClock) -> SystemClock` | The wall clock this environment provides. |

---

### `HasMonotonicClock` {#hasmonotonicclock}

<small>`clock/clock.zirr:12`</small>

```zirric
attr HasMonotonicClock {
	clock(self: @HasMonotonicClock) -> MonotonicClock
}
```

Provides the monotonic clock, for measuring how long something took. Kept apart from HasSystemClock because a wall clock can jump, so code measuring elapsed time must not be handed one.

#### Members

| Member  | Signature                                           | Description                                    |
| ------- | --------------------------------------------------- | ---------------------------------------------- |
| `clock` | `clock(self: @HasMonotonicClock) -> MonotonicClock` | The monotonic clock this environment provides. |

---

## Data

### `SystemClock` {#systemclock}

<small>`clock/clock.zirr:17`</small>

```zirric
data SystemClock {
	now: fn() -> Timestamp
}
```

A clock reading civil time, which can jump when the host is corrected.

#### Fields

| Field | Signature                | Description                                              |
| ----- | ------------------------ | -------------------------------------------------------- |
| `now` | `now: fn() -> Timestamp` | Reads the current wall-clock time. Prefer [`now`](#now). |

---

### `MonotonicClock` {#monotonicclock}

<small>`clock/clock.zirr:22`</small>

```zirric
data MonotonicClock {
	now: fn() -> Instant
}
```

A clock that only ever moves forward, whose origin carries no meaning.

#### Fields

| Field | Signature              | Description                                              |
| ----- | ---------------------- | -------------------------------------------------------- |
| `now` | `now: fn() -> Instant` | Takes a monotonic reading. Prefer [`instant`](#instant). |

---

## Functions

### `now` {#now}

<small>`clock/clock.zirr:27`</small>

```zirric
fn now(c: SystemClock) -> Timestamp
```

The current wall-clock time.

---

### `instant` {#instant}

<small>`clock/clock.zirr:32`</small>

```zirric
fn instant(c: MonotonicClock) -> Instant
```

The current monotonic reading. Subtract two to learn how much time passed between them.

---

### `fixed` {#fixed}

<small>`clock/clock.zirr:37`</small>

```zirric
fn fixed(at: Timestamp) -> SystemClock
```

A wall clock frozen at one time, for tests that must see a fixed date.

---

### `stepping` {#stepping}

<small>`clock/clock.zirr:42`</small>

```zirric
fn stepping(from: Timestamp, step: Duration) -> SystemClock
```

A wall clock starting at from and advancing by step on every reading, for tests that need time to pass predictably.

---

### `steppingMonotonic` {#steppingmonotonic}

<small>`clock/clock.zirr:53`</small>

```zirric
fn steppingMonotonic(step: Duration) -> MonotonicClock
```

A monotonic clock starting at the origin and advancing by step on every reading. A step of zero makes every reading identical, which is how code under a timeout is tested without waiting.

---

## See also

- [`time`](../time/index.md) — the `Timestamp`, `Instant` and `Duration` these produce.
- [`os`](../os/index.md#systemclock) — the host's real clocks.
