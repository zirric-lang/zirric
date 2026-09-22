---
title: Errors
description: Combining, annotating and unwrapping error values.
---

# Module `errors`

```zirric
import errors
```

|            |                                                                                                                                                                                                                        |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `errors`                                                                                                                                                                                                               |
| **Source** | [`errors/errors.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/errors/errors.zirr), [`errors/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/errors/module-docs.zirr) |

> What to do with the reason inside an `Err`.

An error in Zirric is any value carrying the [`prelude.Error`](../prelude/index.md#error) attribute, which knows how to render itself for diagnostics. `errors` works with those values rather than with [`prelude.Result`](../prelude/index.md#result) itself.

[`join`](#join) and [`wrap`](#wrap) both return new `@Error` values whose [`debug`](#debug) output is composed from the errors they were given, so the context survives into whatever finally prints it.

## Contents

- **Functions** — [`debug`](#debug), [`join`](#join), [`unwrap`](#unwrap), [`wrap`](#wrap)

---

## Functions

### `debug` {#debug}

<small>`errors/errors.zirr:4`</small>

```zirric
fn debug(err: @Error) -> String
```

Returns a debug string for err, via its @Error attribute.

---

### `join` {#join}

<small>`errors/errors.zirr:9`</small>

```zirric
fn join(errs: [@Error]) -> Option
```

Returns an @Error combining every error in errs, or None if errs is empty.

---

### `unwrap` {#unwrap}

<small>`errors/errors.zirr:26`</small>

```zirric
fn unwrap(err: @Error) -> @Error
```

Returns the error wrap was called with, or err itself if it isn't wrapped.

---

### `wrap` {#wrap}

<small>`errors/errors.zirr:21`</small>

```zirric
fn wrap(err: @Error, message: String) -> @Error
```

Returns an @Error that adds context to err, debugging as "<message>: <err>".
