---
title: Clock
description: Wall-clock and monotonic time, as values a test can replace.
---

# Module `clock`

```zirric
import clock
```

|            |                                                                                                                                                                                                                  |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `clock`                                                                                                                                                                                                          |
| **Source** | [`clock/clock.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/clock/clock.zirr), [`clock/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/clock/module-docs.zirr) |

> Two kinds of clock, deliberately kept apart.

There are two clocks, and confusing them is a bug. [`SystemClock`](#systemclock) reads civil time and can jump when the host is corrected — use it to timestamp and to do calendar work. [`MonotonicClock`](#monotonicclock) only moves forward and has a meaningless origin — use it to measure how long something took.

The attributes keep them apart at the type level: code that measures elapsed time requires [`HasMonotonicClock`](#hasmonotonicclock) and therefore cannot be handed a wall clock by mistake.

Both are plain data holding a function, so a test supplies its own. [`fixed`](#fixed) freezes time, [`stepping`](#stepping) advances it predictably, and [`steppingMonotonic`](#steppingmonotonic) with a zero step makes every reading identical — which is how code under a timeout is tested without waiting for one.

## Dependencies

- [`time`](../time/index.md)

---

## Contents

- **Data** — [`MonotonicClock`](#monotonicclock), [`SystemClock`](#systemclock)
- **Attributes** — [`HasMonotonicClock`](#hasmonotonicclock), [`HasSystemClock`](#hassystemclock)
- **Functions** — [`fixed`](#fixed), [`instant`](#instant), [`now`](#now), [`stepping`](#stepping), [`steppingMonotonic`](#steppingmonotonic)

---

## Data

### `MonotonicClock` {#monotonicclock}

<small>`clock/clock.zirr:22`</small>

```zirric
data MonotonicClock {
	now: fn() -> Instant
}
```

A clock that only ever moves forward, whose origin carries no meaning.

#### Fields

| Field | Description |
| ----- | ----------- |
| `now` |             |

---

### `SystemClock` {#systemclock}

<small>`clock/clock.zirr:17`</small>

```zirric
data SystemClock {
	now: fn() -> Timestamp
}
```

A clock reading civil time, which can jump when the host is corrected.

#### Fields

| Field | Description |
| ----- | ----------- |
| `now` |             |

---

## Attributes

### `HasMonotonicClock` {#hasmonotonicclock}

<small>`clock/clock.zirr:12`</small>

```zirric
attr HasMonotonicClock {
	clock(self: @HasMonotonicClock) -> MonotonicClock
}
```

Provides the monotonic clock, for measuring how long something took.
Kept apart from HasSystemClock because a wall clock can jump, so code measuring elapsed time must not be handed one.

#### Fields

| Field   | Description |
| ------- | ----------- |
| `clock` |             |

---

### `HasSystemClock` {#hassystemclock}

<small>`clock/clock.zirr:6`</small>

```zirric
attr HasSystemClock {
	clock(self: @HasSystemClock) -> SystemClock
}
```

Provides the wall clock, for timestamping and calendar work.

#### Fields

| Field   | Description |
| ------- | ----------- |
| `clock` |             |

---

## Functions

### `fixed` {#fixed}

<small>`clock/clock.zirr:37`</small>

```zirric
fn fixed(at: Timestamp) -> SystemClock
```

A wall clock frozen at one time, for tests that must see a fixed date.

---

### `instant` {#instant}

<small>`clock/clock.zirr:32`</small>

```zirric
fn instant(c: MonotonicClock) -> Instant
```

The current monotonic reading. Subtract two to learn how much time passed between them.

---

### `now` {#now}

<small>`clock/clock.zirr:27`</small>

```zirric
fn now(c: SystemClock) -> Timestamp
```

The current wall-clock time.

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

A monotonic clock starting at the origin and advancing by step on every reading.
A step of zero makes every reading identical, which is how code under a timeout is tested without waiting.
