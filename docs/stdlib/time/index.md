---
title: Time
description: Durations, instants and timestamps.
---

# Module `time`

> Three types that refuse to be mixed up.

```zirric
import time
```

|            |                                                                                                                                                                                                                                                                                                               |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `time`                                                                                                                                                                                                                                                                                                        |
| **Source** | [`time/time.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/time/time.zirr), [`time/durations.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/time/durations.zirr), [`time/instants.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/time/instants.zirr) |

`time` defines the three values a clock deals in and the arithmetic over them. [`Duration`](#duration) is a span, [`Instant`](#instant) a monotonic reading, and [`Timestamp`](#timestamp) a point on the wall clock.

They are separate types on purpose. Subtracting two `Instant`s gives a `Duration`; adding a `Duration` to either moves it. Subtracting two `Timestamp`s is [`between`](#between). What you cannot do is measure elapsed time with wall-clock readings, because a wall clock jumps and the types will not let you pretend otherwise.

`Duration` is a primitive rather than a wrapper, so spans add, subtract and scale with the usual operators and print as `1.5s` rather than as a bare number.

## Contents

- **Types** — [`Duration`](#duration), [`Instant`](#instant), [`Timestamp`](#timestamp)
- **Functions** — [`nanoseconds`](#nanoseconds), [`microseconds`](#microseconds), [`milliseconds`](#milliseconds), [`seconds`](#seconds), [`minutes`](#minutes), [`hours`](#hours), [`zero`](#zero), [`asNanoseconds`](#asnanoseconds), [`asMilliseconds`](#asmilliseconds), [`asSeconds`](#asseconds), [`asMinutes`](#asminutes), [`asHours`](#ashours), [`absolute`](#absolute), [`formatDuration`](#formatduration), [`parseDuration`](#parseduration), [`origin`](#origin), [`since`](#since), [`between`](#between), [`fromEpoch`](#fromepoch), [`toEpoch`](#toepoch), [`format`](#format), [`parse`](#parse), [`year`](#year), [`month`](#month), [`day`](#day), [`hour`](#hour), [`minute`](#minute), [`second`](#second)

---

## Types

### `Duration` {#duration}

<small>`time/time.zirr:5`</small>

```zirric
extern type Duration {}
```

A span of time, in nanoseconds. A primitive rather than a wrapper, so that spans add, subtract and scale with the usual operators, and print as 1.5s rather than as a bare number.

---

### `Instant` {#instant}

<small>`time/time.zirr:9`</small>

```zirric
extern type Instant {}
```

A reading from a monotonic clock. Its origin is arbitrary, so it is only meaningful next to another Instant. Subtracting two gives a Duration; adding a Duration moves it. Nothing else is allowed, which is what keeps elapsed time honest.

---

### `Timestamp` {#timestamp}

<small>`time/time.zirr:13`</small>

```zirric
extern type Timestamp {}
```

A point on the wall clock, as nanoseconds since the Unix epoch. Distinct from Instant because wall clocks jump, so a Timestamp must never be used to measure how long something took.

---

## Functions

### `nanoseconds` {#nanoseconds}

<small>`time/durations.zirr:6`</small>

```zirric
extern fn nanoseconds(count: Int) -> Duration
```

A span of the given number of nanoseconds.

---

### `microseconds` {#microseconds}

<small>`time/durations.zirr:9`</small>

```zirric
extern fn microseconds(count: Int) -> Duration
```

A span of the given number of microseconds.

---

### `milliseconds` {#milliseconds}

<small>`time/durations.zirr:12`</small>

```zirric
extern fn milliseconds(count: Int) -> Duration
```

A span of the given number of milliseconds.

---

### `seconds` {#seconds}

<small>`time/durations.zirr:15`</small>

```zirric
extern fn seconds(count: Int) -> Duration
```

A span of the given number of seconds.

---

### `minutes` {#minutes}

<small>`time/durations.zirr:18`</small>

```zirric
extern fn minutes(count: Int) -> Duration
```

A span of the given number of minutes.

---

### `hours` {#hours}

<small>`time/durations.zirr:21`</small>

```zirric
extern fn hours(count: Int) -> Duration
```

A span of the given number of hours.

---

### `zero` {#zero}

<small>`time/durations.zirr:24`</small>

```zirric
extern fn zero() -> Duration
```

A span of no time at all.

---

### `asNanoseconds` {#asnanoseconds}

<small>`time/durations.zirr:27`</small>

```zirric
extern fn asNanoseconds(d: Duration) -> Int
```

The whole number of nanoseconds in d.

---

### `asMilliseconds` {#asmilliseconds}

<small>`time/durations.zirr:30`</small>

```zirric
extern fn asMilliseconds(d: Duration) -> Int
```

The whole number of milliseconds in d, discarding any remainder.

---

### `asSeconds` {#asseconds}

<small>`time/durations.zirr:33`</small>

```zirric
extern fn asSeconds(d: Duration) -> Float
```

The number of seconds in d, including any fraction.

---

### `asMinutes` {#asminutes}

<small>`time/durations.zirr:36`</small>

```zirric
extern fn asMinutes(d: Duration) -> Float
```

The number of minutes in d, including any fraction.

---

### `asHours` {#ashours}

<small>`time/durations.zirr:39`</small>

```zirric
extern fn asHours(d: Duration) -> Float
```

The number of hours in d, including any fraction.

---

### `absolute` {#absolute}

<small>`time/durations.zirr:42`</small>

```zirric
extern fn absolute(d: Duration) -> Duration
```

d without its sign.

---

### `formatDuration` {#formatduration}

<small>`time/durations.zirr:46`</small>

```zirric
extern fn formatDuration(d: Duration) -> String
```

A human reading of d, such as "1.5s" or "250ms". The result always reads back through parseDuration as the same span.

---

### `parseDuration` {#parseduration}

<small>`time/durations.zirr:50`</small>

```zirric
extern fn parseDuration(text: String) -> Result
```

Reads a span such as "1.5s", "250ms", "2h45m" or "-1.5h", failing with Err when the text is not one. Accepts the units ns, us, µs, ms, s, m and h.

---

### `origin` {#origin}

<small>`time/instants.zirr:4`</small>

```zirric
extern fn origin() -> Instant
```

The zero point of the monotonic scale. Its position carries no meaning; it exists so that instants can be constructed, which is mostly useful for test clocks.

---

### `since` {#since}

<small>`time/instants.zirr:7`</small>

```zirric
extern fn since(earlier: Instant, later: Instant) -> Duration
```

The span from earlier to later, which is negative when they are the other way round.

---

### `between` {#between}

<small>`time/instants.zirr:10`</small>

```zirric
extern fn between(earlier: Timestamp, later: Timestamp) -> Duration
```

The span between two points on the wall clock.

---

### `fromEpoch` {#fromepoch}

<small>`time/instants.zirr:13`</small>

```zirric
extern fn fromEpoch(nanoseconds: Int) -> Timestamp
```

A Timestamp the given number of nanoseconds after the Unix epoch.

---

### `toEpoch` {#toepoch}

<small>`time/instants.zirr:16`</small>

```zirric
extern fn toEpoch(t: Timestamp) -> Int
```

The number of nanoseconds from the Unix epoch to t.

---

### `format` {#format}

<small>`time/instants.zirr:19`</small>

```zirric
extern fn format(t: Timestamp) -> String
```

t rendered in RFC 3339 form, in UTC, such as "2024-03-05T14:30:00Z".

---

### `parse` {#parse}

<small>`time/instants.zirr:22`</small>

```zirric
extern fn parse(text: String) -> Result
```

Reads an RFC 3339 timestamp, failing with Err when the text is not one.

---

### `year` {#year}

<small>`time/instants.zirr:25`</small>

```zirric
extern fn year(t: Timestamp) -> Int
```

The calendar parts of t, in UTC.

---

### `month` {#month}

<small>`time/instants.zirr:26`</small>

```zirric
extern fn month(t: Timestamp) -> Int
```

The calendar month of `t`, in UTC, from 1.

---

### `day` {#day}

<small>`time/instants.zirr:27`</small>

```zirric
extern fn day(t: Timestamp) -> Int
```

The day of the month of `t`, in UTC, from 1.

---

### `hour` {#hour}

<small>`time/instants.zirr:28`</small>

```zirric
extern fn hour(t: Timestamp) -> Int
```

The hour of `t`, in UTC, from 0.

---

### `minute` {#minute}

<small>`time/instants.zirr:29`</small>

```zirric
extern fn minute(t: Timestamp) -> Int
```

The minute of `t`, in UTC, from 0.

---

### `second` {#second}

<small>`time/instants.zirr:30`</small>

```zirric
extern fn second(t: Timestamp) -> Int
```

The second of `t`, in UTC, from 0.

---

## See also

- [`clock`](../clock/index.md) — where these values come from.
- [`math`](../math/index.md) — arithmetic on the numbers the `as…` functions return.
