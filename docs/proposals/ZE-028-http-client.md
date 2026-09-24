---
title: "ZE-028 - HTTP Client"
description: "HTTP requests as a standard library module, with the client as a value you pass in rather than reach for."
---

# HTTP Client

::: callout draft Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design. Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

A CLI talks to the network more often than to anything else but the file system: it checks for updates, calls an API, downloads a release, posts to a webhook. Zirric programs have no way to make an HTTP request today, other than running `curl` as a child process.

This proposal adds a standard library module, `http`, with requests, responses and a `Client` value. The client is a capability like `fs.FileSystem` or `co.Timer`: the real one comes from `os.http()`, and tests pass `http.stub(handler)` instead. Requests are switch points and cancellable, so they compose with the routines of [ZE-025](https://zirric.knabel.dev/proposals/ZE-025-routines-and-channels/). There is no new syntax, no new keyword and no new operator.

Three rules govern the design:

- **The network is a value.** No function in `http` reaches the network on its own. A program that talks HTTP says so in its signatures, and the only call that touches the real network is `os.http()`.
- **A response is not an error.** A `404` is information. `Err` means there was no response at all; `http.expectSuccess` turns an unwanted status into an `Err` in one call.
- **Buffered by default, streamed on request.** `send` returns the whole body as `Binary`. Large or endless bodies go through `http.open` and `http.withStream`, shaped like `fs.open` and `fs.withFile`.

## Motivation

Common HTTP tasks in the target domain:

- an update check or version lookup against a registry;
- a CLI for a web API: issues, deployments, chat messages, an LLM;
- downloading a release archive to disk, with progress;
- several requests at once, with a limit and a timeout.

Today, all of these shell out to `curl` through the process module. That works on a developer machine and falls apart everywhere else: `curl` may be missing (Windows, minimal containers), errors arrive as exit codes and text on standard error, headers have to be parsed from its output, and a test either hits the real network or replaces a whole executable on the `PATH`.

With `http`, an update check is a function over a `Client`:

```zirric
mod code.example.updates

import bytes
import coding
import http
import json

// The part of the registry's answer this program reads.
data Release {
	version: String
}

// Returns the latest released version of the package called name.
fn latestVersion(client: http.Client, name: String) -> String! {
	const body = http.expectSuccess(http.get(client, "https://registry.example.com/" + name))!.body
	const tree = json.parse(bytes.toString(body))
	if tree is Err {
		return tree
	}
	return Ok(coding.decodeWith(json, Release, tree.value)!.version)
}
```

Its test never opens a socket:

```zirric
mod code.example.updates._t

import bytes
import http
import tests
import tests.assert
import code.example.updates

@tests.Test()
fn testLatestVersion() -> Result {
	const client = http.stub(fn(request) {
		if request.url != "https://registry.example.com/zirric" {
			return http.respond(404, bytes.fromString(""))
		}
		return http.respond(200, bytes.fromString("{\"version\": \"0.3.0\"}"))
	})
	return assert.equal(Ok("0.3.0"), updates.latestVersion(client, "zirric"))
}

@tests.Test()
fn testOfflineIsErr() -> Result {
	return assert.isErr(updates.latestVersion(http.offline(), "zirric"))
}
```

Only the entry point reaches for the real network:

```zirric
const version = updates.latestVersion(os.http(), "zirric")
```

Concurrency needs nothing new. [ZE-025](https://zirric.knabel.dev/proposals/ZE-025-routines-and-channels/) already has a bounded map and a timeout:

```zirric
fn fetchAll(client: http.Client, timer: co.Timer, urls: [String]) -> [Result] {
	return co.map(urls, 4, fn(url) {
		co.timeout(timer, time.seconds(10), fn() { http.get(client, url) })
	})
}
```

## Proposed Solution

A new module, `http`:

| Declaration                                                                      | Meaning                                                                          |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `data Request { method, url, headers, body }`                                    | What to send. `method: String`, `url: String`, `headers: Dict`, `body: Binary`.  |
| `data Response { status, headers, body }`                                        | A complete response. `status: Int`, `headers: Dict`, `body: Binary`.             |
| `data Stream { status, headers, readFrom, closeWith }`                           | A response whose body is still arriving. Carries `@io.Reader`. Must be closed.   |
| `data Client { open: fn(Request) -> Result }`                                    | The capability to make requests. `open` returns `Ok(Stream)`.                    |
| `attr HasClient`                                                                 | Provides a `Client`, like `fs.HasFileSystem` provides a file system.             |
| `open(client: Client, request: Request) -> Result`                               | Sends `request`, returns `Ok(Stream)` once the status and headers have arrived.  |
| `close(stream: Stream) -> Result`                                                | Releases the connection. The unread rest of the body is dropped.                 |
| `withStream(opened: Result, body: fn(Stream) -> Result) -> Result`               | Runs `body` on an opened stream and closes it, whether or not `body` succeeded.  |
| `send(client: Client, request: Request) -> Result`                               | `open`, read the whole body, `close`. Returns `Ok(Response)`.                    |
| `get(client: Client, url: String) -> Result`                                     | `send` of a `GET` without headers or body.                                       |
| `post(client: Client, url: String, contentType: String, body: Binary) -> Result` | `send` of a `POST` with a `content-type`.                                        |
| `request(method: String, url: String) -> Request`                                | A request without headers or body.                                               |
| `withHeader(request: Request, name: String, value: String) -> Request`           | A copy of `request` with one more header value.                                  |
| `withBody(request: Request, contentType: String, body: Binary) -> Request`       | A copy of `request` with a body and its `content-type`.                          |
| `header(headers: Dict, name: String) -> Option`                                  | The first value of a header, looked up without regard to case.                   |
| `expectSuccess(result: Result) -> Result`                                        | Passes an `Err` and a `2xx` response through; any other status becomes an `Err`. |
| `query(url: String, params: Dict) -> String`                                     | `url` with `params` appended as an escaped query string.                         |
| `form(fields: Dict) -> Binary`                                                   | `fields` as an `application/x-www-form-urlencoded` body.                         |
| `respond(status: Int, body: Binary) -> Response`                                 | A response without headers. For stubs.                                           |
| `stub(handler: fn(Request) -> Response) -> Client`                               | A client that answers every request by calling `handler`. For tests.             |
| `offline() -> Client`                                                            | A client whose every request fails with `Unreachable`. For tests.                |
| `data InvalidRequest { reason: String }`                                         | `@Error`: the request could not be sent as given (bad URL, bad method).          |
| `data Unreachable { url: String, reason: String }`                               | `@Error`: no response arrived (DNS, connection, TLS, too many redirects).        |
| `data UnexpectedStatus { status: Int, body: Binary }`                            | `@Error`: returned by `expectSuccess` for a status outside `200`–`299`.          |

One addition outside `http`: `os.http() -> http.Client`, the real client.

### The client is a value

`Client` has exactly one primitive, `open`. Everything else in the module is a plain Zirric function over it, so a client with extra behavior is a function returning a `Client`:

```zirric
// A client that sends a bearer token with every request.
fn authorized(client: http.Client, token: String) -> http.Client {
	return http.Client(fn(request) {
		return http.open(client, http.withHeader(request, "authorization", "Bearer " + token))
	})
}
```

Logging, base URLs, retries and recording for tests are written the same way. There is no middleware concept, no registry and no global configuration: what a client does is visible where it is built.

### Streaming

`send` is right for API calls and small files. A download or a streamed answer uses `open` and `withStream`, exactly like a file:

```zirric
// Downloads url into path on fsys, chunk by chunk.
fn download(client: http.Client, fsys: fs.FileSystem, url: String, path: String) -> Result {
	return http.withStream(http.open(client, http.request("GET", url)), fn(stream) {
		if stream.status != 200 {
			return Err(errors.wrap(http.UnexpectedStatus(stream.status, bytes.fromString("")), url))
		}
		return fs.withFile(fs.create(fsys, path), fn(file) {
			for {
				const chunk = io.read(stream, 32768)
				if len(chunk) == 0 {
					return Ok(void)
				}
				_ = io.write(file, chunk)
			}
		})
	})
}
```

A `Stream` carries `@io.Reader`, so everything that reads from a reader reads from a response.

### Testing

`stub` turns a function into a server. The handler sees the full `Request`, so a test can check what was sent as easily as what comes back:

```zirric
@tests.Test()
fn testSendsToken() -> Result {
	var seen = ""
	const client = http.stub(fn(request) {
		seen = http.header(request.headers, "Authorization") ?? ""
		return http.respond(204, bytes.fromString(""))
	})
	_ = http.get(authorized(client, "secret"), "https://api.example.com/me")
	return assert.equal("Bearer secret", seen)
}
```

`offline` covers the other side: every code path that must survive a missing network is tested without pulling a cable.

## Detailed Design

### Requests

| Rule                                                                                        | Consequence                                                               |
| ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `url` must be absolute, with the scheme `http` or `https`.                                  | Otherwise `open` returns `Err(InvalidRequest(…))` without sending.        |
| `method` is sent as written. It must be a non-empty token; `request` does not uppercase it. | `"get"` is a different method from `"GET"`, as in HTTP.                   |
| `headers` maps lower-case names to arrays of values.                                        | Repeated headers keep all their values, in order.                         |
| `withHeader` lower-cases `name` and appends `value`.                                        | Case never decides whether two headers are the same.                      |
| `body` is always `Binary`. An empty body is sent as no body.                                | Text is converted with `bytes.fromString`; JSON with `json.format` first. |
| `content-length` and `host` are computed by the client and cannot be set.                   | A request cannot lie about its own framing.                               |

`Request` has no optional fields, since construction is positional ([ZE-003](https://zirric.knabel.dev/proposals/ZE-003-named-data-construction/) was rejected). `request`, `withHeader` and `withBody` fill that gap with plain functions. A JSON body reads:

```zirric
fn createIssue(client: http.Client, title: String) -> Result {
	const payload = json.format(["title": title])
	if payload is Err {
		return payload
	}
	const request = http.request("POST", "https://api.example.com/issues")
	return http.expectSuccess(http.send(client, http.withBody(request, "application/json", bytes.fromString(payload.value))))
}
```

### Responses

- `Response.headers` and `Stream.headers` use the same shape as requests: lower-case names, arrays of values.
- `header(headers, name)` lower-cases `name` and returns `Some` of the first value, or `None()`.
- `send` returns `Ok(Response)` for every status the server sends, `1xx` excluded (the client handles those itself).
- `expectSuccess(Ok(r))` is `Ok(r)` for `200 <= r.status < 300`, otherwise `Err(UnexpectedStatus(r.status, r.body))`. The body is kept because APIs explain their errors there. `expectSuccess(Err(e))` is `Err(e)`.
- `UnexpectedStatus` does not name the URL: a `Response` does not know its request, and hiding it there would be magic. Callers add context with `errors.wrap`, as for every other error.
- `send` reads the body into memory without a limit. Unknown or large bodies are what `open` is for.

### Streams

| Rule                                                                            | Consequence                                                             |
| ------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `open` returns once status and headers have arrived.                            | A program can decide on the status before reading any of the body.      |
| `io.read(stream, n)` returns up to `n` bytes, and an empty `Binary` at the end. | The same contract as every other `@io.Reader`.                          |
| A dropped connection mid-body ends the stream early.                            | `close` then returns `Err(Unreachable(…))`, so `withStream` reports it. |
| `close` on a closed stream does nothing.                                        | `withStream` and an explicit `close` never conflict.                    |
| A stream that is never closed holds its connection until the program ends.      | `withStream` is the recommended form, as `fs.withFile` is for files.    |

### Scheduling and cancelling

Under [ZE-025](https://zirric.knabel.dev/proposals/ZE-025-routines-and-channels/), `http` belongs to the outside world:

- `open`, `send`, `get`, `post` and reading a `Stream` through `io.read` are switch points that **always** give way, whether the client is `os.http()` or a `stub`. The module decides, not the value behind it, so a test with a stub sees the same interleaving points as production.
- A request in flight runs on its own goroutine, like a host stream. The routine that made it waits at an ordinary, cancellable switch point.

Cancelling follows ZE-025's split between what can be kept and what is already out:

| In flight                      | On cancel                                                                                                                                      |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Waiting for status and headers | Stops at once. The connection is aborted. The request may already have reached the server; whether it acted on it is unknown, as with a write. |
| Reading the body of a `Stream` | Stops at once. The connection is dropped.                                                                                                      |
| A `stub` handler               | Runs to its end, like any Zirric code between switch points. Its response is discarded.                                                        |

**A stopped routine never leaks a connection.** The host ties every open stream to the routine that opened it and releases it when that routine ends, however it ends. `withStream` is still the right form; this rule only makes cancellation safe.

There is no timeout in `Request` or `Client`. `co.timeout` is the one way to give up on something, for HTTP as for everything else.

### The real client

`os.http()` returns one client per process. Connections are reused between requests. Its behavior is fixed, not configurable:

| Concern      | Behavior                                                                                                    |
| ------------ | ----------------------------------------------------------------------------------------------------------- |
| TLS          | Verified against the system's root certificates. There is no switch to turn verification off.               |
| Redirects    | Followed up to ten times for `GET` and `HEAD`. Other methods return the `3xx` response as it is.            |
| Proxies      | Taken from `HTTP_PROXY`, `HTTPS_PROXY` and `NO_PROXY`, as most command line tools do.                       |
| Compression  | `gzip` is requested and decoded. The body is the decoded one; `content-encoding` is removed from `headers`. |
| `user-agent` | `zirric/<version>` unless the request sets one.                                                             |
| Cookies      | Not stored. A `set-cookie` header is returned like any other.                                               |

Anything a program needs beyond this, such as a base URL, a token or a retry, is a function over `Client` (see above), not a setting.

### Static checks, tooling

Nothing new is needed. `http` is ordinary Zirric: `data`, one attribute, functions and a few externs behind `os.http()`. A `Stream` passed where `@io.Reader` is expected passes [ZE-022](https://zirric.knabel.dev/proposals/ZE-022-static-checks/). There is no new grammar, so the formatter, language server and tree-sitter grammar need no work.

### Deliberately not

- **A server.** Zirric's domain is CLI and TUI programs, which call services far more often than they are one. A server would need its own proposal, built on the same `Request` and `Response`.
- **Module-level shortcuts without a client.** No `http.get(url)`. The network is always passed in.
- **Timeouts and retries as options.** `co.timeout` and a `for` loop with `co.sleep` already express them.
- **Non-`2xx` as `Err` by default.** `expectSuccess` is one call away.
- **Automatic JSON.** `json.parse`, `json.format` and `coding` already do it, and stay swappable.
- **Middleware, interceptors, hooks.** A function returning a `Client` is the whole mechanism.
- **Disabling TLS verification.** A test uses `stub`; a development server gets a trusted certificate.
- **Streaming request bodies, multipart helpers, WebSockets, HTTP/3 control, a cookie jar.** All can be added later without breaking anything.
- **A URL type.** URLs are strings; `query` covers the common escaping need. A `urls` module, like `paths`, is an open question.

## Changes to the Standard Library

- New module `http`, as listed above.
- `os.http() -> http.Client`.
- `os` depends on `http`, as it does on `fs`, `clock` and `random`.

## Compatibility

Nothing breaks. `http` is new, and `os` only gains a function.

## Dependencies on Other Proposals

| Proposal                                                                      | Dependency                                                                                               |
| ----------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| [ZE-008](https://zirric.knabel.dev/proposals/ZE-008-error-handling/)          | Every request returns a `Result`; `InvalidRequest`, `Unreachable` and `UnexpectedStatus` carry `@Error`. |
| [ZE-018](https://zirric.knabel.dev/proposals/ZE-018-io-fmt-os/)               | `Stream` carries `@io.Reader`; `os` hands out the real client.                                           |
| [ZE-019](https://zirric.knabel.dev/proposals/ZE-019-result-and-option-sugar/) | `!.body`, `??` and `!!` work on what `send` and `header` return.                                         |
| [ZE-021](https://zirric.knabel.dev/proposals/ZE-021-encoding-and-decoding/)   | JSON bodies go through `json` and `coding`, not through `http`.                                          |
| [ZE-025](https://zirric.knabel.dev/proposals/ZE-025-routines-and-channels/)   | Requests are switch points and cancellable; timeouts and concurrency come from `co`.                     |

## Alternatives Considered

### Do nothing: shell out to `curl`

The process module already lets a program run `curl`, and nothing new would be needed in the standard library. For a personal script on a developer machine, that is enough.

It loses because every property the standard library is built for is gone. `curl` is not everywhere; its errors are exit codes and text; headers and status must be parsed back out of its output; and a test can only replace it by putting a fake executable on the `PATH`. The one thing a CLI talks to most would be the one thing it cannot test.

### Module-level functions, no client

`http.get(url)`, as in most scripting languages, is shorter in a ten-line script. It hides the network inside every function that calls it, so nothing in a signature says a function needs it, and a test can only replace it with a global switch. That is the habit the standard library is meant to make visible at the import. `http.get(os.http(), url)` costs one argument.

### An `@http.Client` attribute instead of a `data` value

An attribute would let any type act as a client. But the set of behaviors is one function, and a `data` value with a function field already accepts any behavior: `stub`, `offline` and `authorized` are all plain `Client`s. An attribute adds a concept and buys nothing, which is also why `fs.FileSystem` and `random.Source` are values.

### Non-`2xx` responses as `Err`

Saves `expectSuccess` in the common case. But a `404` is often the answer a program wants ("does this release exist?"), a `304` is not a failure, and APIs return error details in bodies a program needs to read. Folding them into `Err` would force callers to dig the response back out of the error. Keeping the two apart is also what [ZE-008](https://zirric.knabel.dev/proposals/ZE-008-error-handling/) asks for: `Err` for what went wrong, values for what came back.

### Always streamed bodies

One response type instead of two. But every API call would have to open, read and close, and a forgotten `close` would leak a connection in the simplest script. `send` covers the common case; `Stream` stays one step away.

### Options on the client: timeouts, redirects, TLS, base URL

A configurable client, as in Go or Python, answers every need with a field. It also means every client needs defaults, every field needs documentation and tests, and a reader must look at where a client was built to know what a call does. Timeouts already live in `co`, a base URL or token is a three-line wrapper, and the rest are fixed on purpose. A setting can be added later; a setting cannot be taken back.

### Other module names

| Name       | Why not                                                                           |
| ---------- | --------------------------------------------------------------------------------- |
| `net.http` | There is no `net` to put next to it, and nesting would be the only reason for it. |
| `web`      | Suggests a server and HTML as much as a client.                                   |
| `fetch`    | Names one function, not the module.                                               |

## Open Questions

- **Redirects.** Should the final URL after redirects be part of `Response`? It matters for downloads behind short links, but the field would be meaningless for `stub` responses.
- **Body limits.** Should `send` take, or apply, a maximum body size, so an unexpected endless body cannot exhaust memory?
- **A `urls` module.** `query` covers building URLs. Parsing and joining them, like `paths` for file paths, may deserve its own module.
- **Recording.** A client that records real exchanges and replays them in tests would make stubs for large APIs cheap. Should the standard library ship it, or leave it to a package?
- **Streaming uploads.** A request body from an `@io.Reader` would allow uploading large files without loading them. Is it needed before a server exists?

## Acknowledgements

Prior art: Go's `net/http` and its `RoundTripper` for the single-function client; Python's `requests` and `httpx` for the buffered default; Rust's `reqwest` for keeping status checks explicit (`error_for_status`); Elixir's `Req` and Tesla for clients built by wrapping clients.
