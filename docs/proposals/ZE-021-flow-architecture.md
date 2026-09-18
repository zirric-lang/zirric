---
title: "ZE-021 - The flow Module: Model-Update-View Architecture"
status: Draft
---

# ZE-021 - The `flow` Module: Model-Update-View Architecture

::: callout draft Draft
This proposal is still a draft and is subject to change. Please do not cite or reference it as a finalized design. Features described here may not be implemented as described and cannot be used right now.
:::

## Introduction

This proposal introduces `flow`, a standard architecture for building interactive, testable UI components around a Model-Update-View loop, in the tradition of The Elm Architecture. It defines how components hold state, describe effects, compose into larger trees, render to a platform-neutral tree, and handle shared exclusive state such as keyboard focus — all without threading capability values through application code, and without committing to any one rendering target.

Zirric's UI story is text-based by design: any widget that cannot be meaningfully expressed as structured text does not need to be solved by the language. Concrete widget kits and platform backends are first-party companion packages outside the language distribution, not part of this proposal — `flow` only defines the architecture and the generic tree those packages render to and through.

## Motivation

Building UI without a shared architecture invites exactly the problems [ZE-018](./ZE-018-io-fmt-os.md) and [ZE-020](./ZE-020-capability-attributes-and-script.md) were written to avoid at the module level: ad hoc side effects scattered through view code, no consistent way to test what a component _does_ in response to input, and capability values (a clock, a shell) leaking into places that should only manipulate data. `flow` gives components a shape where state transitions are pure and inspectable, effects are data described and executed at a single boundary, and nested components compose without any of them needing to know how the environment is wired — or which platform they will ultimately render on.

## Proposed Solution

Zirric has no multiple-value returns, so `init` and `update` pack their result into a small `Step` value rather than returning a pair:

```zirric
mod flow

data Step { model: Any, cmd: @Cmd }

data Component {
    init: fn() -> Step
    update: fn(Any, Any) -> Step
    view: fn(Any, fn(Any) -> Void) -> @Renderable
    subs: [@Sub]
}
```

- **`Model` and `Msg` are per-component conventions, not generic parameters.** Zirric has no generics ([ZE-017](./ZE-017-type-hints.md)), so `flow`'s own declarations (`Step`, `Component`, `Cmd.run`, `Sub.listen`, `lift`, `Renderable.render`) type these positions `Any`. Each component declares its own concrete `Model`/`Msg` types (see the full example below) and narrows them back out of `Any` in its own `update`/`view` signatures.
- **`update` is pure.** It never performs I/O; it returns a description of the effect it wants (`@Cmd`), which only the runtime executes.
- **Effects are typed, comparable values**, not opaque closures, via the `@Cmd` protocol attribute — this keeps them assertable in tests without running them.
- **Continuous values and discrete events are different mechanisms.** A `Binding` carries a two-way value (a text field's contents); `Msg` is reserved for named, matchable events.
- **Shared exclusive state (e.g. focus) lives once**, in the nearest parent that owns every candidate, and is derived at render time rather than duplicated per child.
- **The environment (`Env`) appears exactly once**, at the runtime's entry point. It is never passed through `update`, `view`, or component application code.
- **Rendering produces a generic tree, resolved lazily.** A component never targets a platform directly; it produces `flow.Node` values that any backend — including one written after the component shipped — can walk and interpret.

## Detailed Design

### Cmd: effects as data

```zirric
attr Cmd {
    run(self: @Cmd, env: Any, dispatch: fn(Any) -> Void) -> Void
}

@Cmd(fn(self, env: @HasClock, dispatch) {
    dispatch(GotTime(env.now()))
})
data TickCmd {}

@Cmd(fn(self, env: @HasShell, dispatch) {
    dispatch(self.onResult(env.run(self.cmd)))
})
data RunShellCmd { cmd: [String], onResult: fn(proc.ProcResult) -> Any }

@Cmd(fn(self, env, dispatch) { void })
data NoneCmd {}   // shared, canonical no-op — import from flow, never redeclare per component
```

`Cmd.run` takes a `dispatch` callback rather than returning a `Msg` directly, since effects may complete asynchronously; the runtime is responsible for calling `update` with whatever `Msg` a `dispatch` call produces, one at a time.

Capability requirements on `env` use the `@HasX` attributes from [ZE-020](./ZE-020-capability-attributes-and-script.md) — a `TickCmd` declares it needs `@HasClock` without knowing or caring what concrete `Env` type the host application uses.

### Node: a platform-neutral rendering tree, in its own submodule

`Renderable.render` cannot return a backend-specific type (a DOM node, a terminal cell buffer) without making `flow` depend on one particular target. Instead it returns a value from `flow.node`, a small, zero-dependency submodule holding the rendering tree and the `Renderable` protocol together. Everything a widget kit or backend needs to render lives here and nowhere else, so `ui`, `term`, `html`, and `md` never pull in `Cmd`, `Sub`, `Component`, or any other part of `flow` they don't use:

```zirric
mod flow.node

union Node {
    data Text { value: String }
    data Group { children: [Node] }
    data Widget { value: @Renderable, dispatch: fn(Any) -> Void }
}

attr Renderable {
    render(self: @Renderable, dispatch: fn(Any) -> Void) -> Node
}
```

`flow` itself imports `flow.node` to use `Renderable`/`Widget` in `Component`/`MappedView`. **Zirric has no `export` declaration yet**, so `flow` cannot re-export `Renderable` under its own name — attempting that would require `Renderable` to live in `flow` instead, which `flow.node.Widget`'s own type hint needs, creating a cycle (`flow.node` needing `flow`, `flow` needing `flow.node`). Until `export` lands, component and backend authors import `Renderable` from `flow.node` directly, alongside `flow` itself:

```zirric
import flow { Component, Step, Binding }
import flow.node { Renderable }
```

This is a one-line, purely additive cost — `flow.node.Widget` keeps its correct `@Renderable` type hint rather than being loosened to `Any`, which would need to be tightened back once attribute conformance is statically enforced. Once `export` lands, `flow` can re-export `Renderable` and this second import line becomes unnecessary with no change to `Widget`, any backend, or any component's `view` signature.

`Widget` is a deferral marker, not a payload — it means "this child is itself a `@Renderable` value; don't resolve it now, resolve it at the depth the backend actually reaches it." It carries its own `dispatch` alongside the deferred value, rather than relying on whatever ambient dispatch the tree-walker happens to be passing around when it eventually resolves it — this is what lets `MappedView` (below) actually remap messages: without a dispatch bound to the `Widget` itself, a wrapper that changes _how_ a message should be interpreted would have no way to apply that change, since the value underneath wouldn't be resolved until some later point that had already lost track of the wrapping.

This matters for containers too, because a naive implementation that recurses into its children immediately (`Group(children.map(fn(c) { return c.render(dispatch) }))`) throws away the identity of every nested widget before a backend gets a chance to check whether that specific widget has a richer, backend-specific conformance. A correct container preserves children unresolved, passing its own dispatch through unchanged since it isn't remapping anything itself:

```zirric
// A container widget composes by deferring, not recursing:
@Renderable(fn(self, dispatch) {
    return Group(self.children.map(fn(c) { return Widget(c, dispatch) }))
})
```

A backend that wants to give a specific value special treatment — independent of `flow` or of whatever widget kit produced it — declares its own attribute and checks for it before falling back to the generic walk. Note `term` here depends on `flow.node` only, not on all of `flow`:

```zirric
mod term

import flow.node { Node, Text, Group, Widget, Renderable }
import io { HasWriter }

attr TermNode {
    toTerm(self: @TermNode) -> String
}

// `present` is curried on `out` so it can be handed to `flow.run` as a
// plain 2-arg (view, dispatch) callback, matching what `run` expects —
// `term` itself is the only place that needs to know about `out`.
fn present(out: @HasWriter) -> fn(@Renderable, fn(Any) -> Void) -> Void {
    return fn(view, dispatch) { renderAny(view, dispatch, out) }
}

fn renderAny(value: @Renderable, dispatch: fn(Any) -> Void, out: @HasWriter) -> Void {
    switch value {
    case is @TermNode:
        out.write(bytesFromString(value.toTerm()))
    case _:
        renderNode(value.render(dispatch), out)
    }
}

fn renderNode(node: Node, out: @HasWriter) -> Void {
    switch node {
    case is Text:
        out.write(bytesFromString(node.value))
    case is Group:
        for child <- node.children { renderNode(child, out) }
    case is Widget:
        // use the Widget's own bound dispatch, not any dispatch from
        // an outer scope — this is what makes MappedView's wrap apply
        renderAny(node.value, node.dispatch, out)
    }
}
```

Every widget is required to produce _some_ universal representation via `@Renderable`, even if it's only a placeholder string — that requirement is what makes "doesn't have to be solved in Zirric" concrete: a widget that can't be meaningfully expressed as text still has to degrade to something, it just isn't obligated to be good on every backend.

Because attribute conformance cannot be added to a declaration from outside its own module, and import cycles are forbidden, per-backend richness (`@TermNode`, a future `@DomNode`, ...) can only be attached to a type by that type's own author. A widget kit's built-ins therefore rely entirely on the generic `@Renderable` path to reach backends written after the widget kit shipped; only a widget's own author can opt it into a specific backend it already knows about.

### Composing Cmd and views: map, not rewrite

Nesting is handled by wrapping the dispatch sink, not by rewriting a tree:

```zirric
import flow.node { Widget }

@Cmd(fn(self, env, dispatch) {
    self.inner.run(env, fn(msg) { dispatch(self.wrap(msg)) })
})
data MappedCmd { inner: @Cmd, wrap: fn(Any) -> Any }

@Renderable(fn(self, dispatch) {
    // bind the wrapped dispatch into the Widget now, so whichever backend
    // resolves `self.inner` later uses this remapped dispatch, not its own
    return Widget(self.inner, fn(msg) { dispatch(self.wrap(msg)) })
})
data MappedView { inner: @Renderable, wrap: fn(Any) -> Any }
```

This composes at any nesting depth with one allocation per level and works uniformly regardless of how many levels deep a `Cmd` or view sits.

### Msg: wrap per mount point

Each place a child is mounted gets its own `Msg` wrapper variant, so messages from sibling instances of the same component type remain distinguishable:

```zirric
union AppMsg {
    data NameFieldMsg { msg: TextField.Msg }
    data EmailFieldMsg { msg: TextField.Msg }
    data ItemMsg { index: Int, msg: TodoItem.Msg }
}
```

### lift: factored delegation

```zirric
fn lift(childUpdate: fn(Any, Any) -> Step,
         get: fn(Any) -> Any,
         set: fn(Any, Any) -> Any,
         wrap: fn(Any) -> Any) -> fn(Any, Any) -> Step {
    return fn(msg, model) {
        const childStep = childUpdate(msg, get(model))
        return Step(set(model, childStep.model), MappedCmd(childStep.cmd, wrap))
    }
}
```

### Binding: continuous values without wrapping

```zirric
data Binding { get: fn() -> Any, set: fn(Any) -> Void }
```

A parent constructs a `Binding` closed over its own model and hands it directly to a child, bypassing `Msg` entirely for plain value sync. See the full example below for `Binding` used alongside widgets from a companion package.

### Focus: centralized, derived state

Exclusive state such as "which field is focused" lives once, in the parent, as data — never duplicated per child:

```zirric
data FocusId { key: String }

data FormModel {
    name: TextField.Model
    email: TextField.Model
    focus: Option   // Option<FocusId>
}

fn fieldView(fieldModel: TextField.Model, id: FocusId, focus: Option,
             dispatch: fn(AppMsg) -> Void) -> @Renderable {
    const isFocused = focus is Some && focus.value == id
    return TextFieldView(fieldModel, isFocused, fn() { dispatch(FocusRequested(id)) })
}
```

Keystrokes have no widget of origin, so they are delivered through a persistent subscription and routed by the parent based on current focus, not by `Msg` shape. Capturing raw key events isn't a byte-stream read — `io` doesn't yet have a capability for it, so this proposal introduces one, following the `@HasX` naming convention from [ZE-020](./ZE-020-capability-attributes-and-script.md):

```zirric
mod io

attr HasKeyEvents {
    onKeyDown(self: @HasKeyEvents, handler: fn(String) -> Void) -> Void
}
```

```zirric
attr Sub {
    listen(self: @Sub, env: Any, dispatch: fn(Any) -> Void) -> Void
}

@Sub(fn(self, env: @HasKeyEvents, dispatch) {
    env.onKeyDown(fn(key) { dispatch(self.toMsg(key)) })
})
data KeyDownSub { toMsg: fn(String) -> Any }
```

The message shape a key press produces is the _component's_ decision, not the architecture's — `run` never constructs or assumes any particular `Msg` variant. A form that needs key routing declares its own message type and supplies its own `KeyDownSub`, mapped through it:

```zirric
union FormMsg {
    data KeyPressed { key: String }
    data FocusRequested { id: FocusId }
    data NameFieldMsg { msg: TextField.Msg }
    data EmailFieldMsg { msg: TextField.Msg }
}

fn updateForm(msg: FormMsg, model: FormModel) -> Step {
    switch msg {
    case is KeyPressed:
        return routeKey(msg.key, model)
    case is FocusRequested:
        return Step(FormModel(model.name, model.email, Some(msg.id)), NoneCmd())
    case is NameFieldMsg:
        return lift(TextField.update, fn(m) { return m.name },
                     fn(m, c) { return FormModel(c, m.email, m.focus) },
                     NameFieldMsg)(msg.msg, model)
    case is EmailFieldMsg:
        return lift(TextField.update, fn(m) { return m.email },
                     fn(m, c) { return FormModel(m.name, c, m.focus) },
                     EmailFieldMsg)(msg.msg, model)
    case _:
        return Step(model, NoneCmd())
    }
}
```

```zirric
fn routeKey(key: String, model: FormModel) -> Step {
    switch model.focus {
    case is Some:
        switch model.focus.value.key {
        case "name":
            return lift(TextField.update, fn(m) { return m.name },
                         fn(m, c) { return FormModel(c, m.email, m.focus) },
                         NameFieldMsg)(TextField.KeyPressed(key), model)
        case "email":
            return lift(TextField.update, fn(m) { return m.email },
                         fn(m, c) { return FormModel(m.name, c, m.focus) },
                         EmailFieldMsg)(TextField.KeyPressed(key), model)
        case _:
            return Step(model, NoneCmd())
        }
    case _:
        return Step(model, NoneCmd())
    }
}
```

`routeKey` writes out one `lift(...)` call per field by hand, which is fine for two fields and repetitive past a handful. Once a form has more than a few, this wants to become a small map from focus key to a pre-built `lift(...)` closure, assembled once at composition time — `routeKey` then becomes a single lookup instead of a per-field branch. I'd hold off building that generalization until the hand-written form here actually gets unwieldy in a real component, rather than abstracting it up front.

The component that owns this form supplies its own subscription as part of its `Component` bundle:

```zirric
const formComponent = Component(
    initForm,
    updateForm,
    viewForm,
    [KeyDownSub(fn(key) { return KeyPressed(key) })]
)
```

A single `KeyDownSub` is started once at the runtime's entry point; `flow` v1 does not diff or re-subscribe per render — every component that needs keys lists its subscriptions once, in its `Component` value, and `run` starts each of them unconditionally at startup. Structural interactions specific to a platform — screen navigation (push/pop/replace, tabs, modals), scrolling, and similar — are expressed as ordinary `Cmd`/`Msg` vocabulary too, shared across every backend; each backend interprets the same commands however fits its platform (a terminal panel stack vs. a native navigation controller vs. browser history), which is out of scope for this proposal and left to `ui` and the platform backend packages.

### Runtime loop

`flow.run` takes a `present` callback supplied by the composition root, so the architecture itself never picks a rendering target. `Component` carries its subscriptions explicitly (see `Component`'s definition above), so `run` never has to construct or assume any application-specific `Msg` shape:

```zirric
fn run(component: Component, env: Any, present: fn(@Renderable, fn(Any) -> Void) -> Void) {
    const initial = component.init()
    var model = initial.model
    const dispatch = fn(msg) {
        const step = component.update(msg, model)
        model = step.model
        step.cmd.run(env, dispatch)
        present(component.view(model, dispatch), dispatch)
    }
    present(component.view(model, dispatch), dispatch)   // initial render, exactly once
    for sub <- component.subs { sub.listen(env, dispatch) }
    initial.cmd.run(env, dispatch)   // may synchronously call dispatch, which renders again on its own
}
```

The initial render happens once, before `initial.cmd` runs — if that `Cmd` dispatches synchronously, `dispatch`'s own `present` call covers the resulting state change, so the view is never rendered twice for the same model. If `initial.cmd` is `NoneCmd` (or otherwise never dispatches), the one render before it still shows the starting state.

`env` is constructed once here and never appears again in application code — every `Cmd`/`Sub` receives it only through this one call site.

### snapshot: rendering without running the loop

Not every use of a component needs the interactive loop — a static reference page generated from a form's default state, for instance, needs no `Cmd`, no subscription, and no `Env` at all. `snapshot` skips `run` entirely and renders `init`'s starting model directly:

```zirric
fn snapshot(component: Component) -> @Renderable {
    return component.view(component.init().model, fn(msg) { void })
}
```

Effects are deliberately not run here — `snapshot` is for the pure case. A component whose meaningful output depends on an effect having completed first is out of scope for this proposal; nothing about `Component`/`Cmd`/`Node` prevents adding a resolve-then-render helper later, once a concrete need for one is in hand.

### Full example: a labeled name field, rendered to a terminal

Widgets come from the `ui` companion package; rendering comes from `term`. Neither is part of `flow` itself.

```zirric
mod nameField

import future.prelude { Option, Some, None }
import flow { Step, Binding, NoneCmd }
import flow.node { Renderable }
import ui { Label, TextInput, Column }

data Model { name: String, valid: Option }

union Msg {
    data NameChanged { text: String }
    data ValidationResult { ok: Bool }
}

fn init() -> Step {
    return Step(Model("", None), NoneCmd())
}

fn update(msg: Msg, model: Model) -> Step {
    switch msg {
    case is NameChanged:
        return Step(Model(msg.text, None), ValidateNameCmd(msg.text))
    case is ValidationResult:
        return Step(Model(model.name, Some(msg.ok)), NoneCmd())
    case _:
        return Step(model, NoneCmd())
    }
}

fn view(model: Model, dispatch: fn(Msg) -> Void) -> @Renderable {
    const name = Binding(
        fn() { return model.name },
        fn(text) { dispatch(NameChanged(text)) }
    )
    return Column([ Label("Name"), TextInput(name.get(), name.set) ])
}

@Cmd(fn(self, env: @HasShell, dispatch) {
    const result = env.run(["validate-name", self.text])
    dispatch(ValidationResult(result.exitCode == 0))
})
data ValidateNameCmd { text: String }
```

```zirric
mod main

import flow { run, Component }
import term { present }
import nameField { init, update, view }
import os { stdout }
import io { Writer, HasWriter }

@HasWriter(fn(self, buf) { return self.out.write(buf) })
data TermEnv { out: Writer }

fn main() {
    const env = TermEnv(stdout())
    run(Component(init, update, view, []), env, present(env))
}
```

## Changes to the Standard Library

| Declaration                 | Module      | Kind    | Description                                                                       |
| --------------------------- | ----------- | ------- | --------------------------------------------------------------------------------- |
| `Step`                      | `flow`      | `data`  | Packed `(Model, @Cmd)` result of `init`/`update`                                  |
| `Component`                 | `flow`      | `data`  | `init` / `update` / `view` / `subs` bundle                                        |
| `Cmd`                       | `flow`      | `attr`  | Protocol for describable, executable effects                                      |
| `Sub`                       | `flow`      | `attr`  | Protocol for continuous, unrequested event sources                                |
| `NoneCmd`                   | `flow`      | `data`  | Canonical no-op `Cmd`; conforms to `@Cmd`                                         |
| `Binding`                   | `flow`      | `data`  | Two-way value sync without `Msg`                                                  |
| `MappedCmd`                 | `flow`      | `data`  | `Cmd` wrapper for nested dispatch                                                 |
| `MappedView`                | `flow`      | `data`  | `Renderable` wrapper for nested dispatch                                          |
| `lift`                      | `flow`      | `fn`    | Factored child-update delegation                                                  |
| `run`                       | `flow`      | `fn`    | Runtime entry point; owns the `Env`, takes `present`                              |
| `snapshot`                  | `flow`      | `fn`    | Render a component's initial state without running the loop                       |
| `Renderable`                | `flow.node` | `attr`  | Protocol for rendering to a `Node`; zero-dependency submodule                     |
| `Node`                      | `flow.node` | `union` | Platform-neutral rendering tree                                                   |
| `Text` / `Group` / `Widget` | `flow.node` | `data`  | `Node` variants; `Widget` defers re-dispatch and carries its own bound `dispatch` |
| `HasKeyEvents`              | `io`        | `attr`  | Capability for registering a raw key-down handler                                 |

## Companion Packages (out of scope for this proposal)

Widget kits and rendering backends are first-party but independently versioned — not part of the language distribution, added as an ordinary Cavefile dependency:

| Package | Repository                         | Provides                                                                                                                  |
| ------- | ---------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| `ui`    | `code.knabel.dev/zirric-lang/ui`   | Platform-neutral widgets conforming to `flow.node.Renderable`                                                             |
| `term`  | `code.knabel.dev/zirric-lang/term` | Terminal backend: walks `flow.node.Node`, provides `@TermNode` for optional richer per-widget rendering, `present`        |
| `html`  | `code.knabel.dev/zirric-lang/html` | Static HTML backend: walks `flow.node.Node`, provides `@HtmlNode` for optional richer per-widget markup, `renderToString` |
| `md`    | `code.knabel.dev/zirric-lang/md`   | Markdown backend: walks `flow.node.Node`, provides `@MdNode` for optional richer per-widget output, `renderToString`      |

None of the four packages depend on each other; each depends on `flow.node` only — not on the rest of `flow` (`Cmd`, `Sub`, `Component`, `lift`, `run`, `snapshot`, `Binding`) — so no cross-package attribute conformance is required for the baseline generic rendering path to work on any of them, and a change to the interactive-loop machinery cannot ripple into a widget kit or a rendering backend. Additional backends (mobile) are left for future proposals.

`ui` extends its interactive-form vocabulary (`Label`, `TextInput`, `Column`, `Row`, ...) with content-semantic widgets, so the same components render sensibly through `term`, `html`, and `md` alike:

| Widget                                         | Purpose                   | Typical mapping                             |
| ---------------------------------------------- | ------------------------- | ------------------------------------------- |
| `Heading { level: Int, text: String }`         | Section title             | `<h1>`–`<h6>` / `#`–`######` / bold line    |
| `Paragraph { text: String }`                   | Block of prose            | `<p>` / blank-line-separated text           |
| `List { items: [@Renderable], ordered: Bool }` | Bulleted or numbered list | `<ul>`/`<ol>` / `-`/`1.`                    |
| `ListItem { content: @Renderable }`            | One list entry            | `<li>` / list line                          |
| `Link { text: String, href: String }`          | Hyperlink                 | `<a href>` / `[text](href)` / `text (href)` |
| `CodeBlock { code: String, language: Option }` | Preformatted code         | `<pre><code>` / fenced code block           |
| `Emphasis { text: String }`                    | Italic emphasis           | `<em>` / `*text*`                           |
| `Strong { text: String }`                      | Bold emphasis             | `<strong>` / `**text**`                     |

These conform only to `@Renderable`, the same as every other built-in widget — no backend-specific attribute is required for correct output on `term`, `html`, or `md`; a backend may still add a richer conformance (e.g. `html`'s own `@HtmlNode` on `CodeBlock` to add syntax-highlighting classes) without any change to `ui` itself.

## Dependencies on Other Proposals

| Proposal                                                                                | Dependency                                                                                          |
| --------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| [ZE-018 I/O, Fmt, OS](./ZE-018-io-fmt-os.md)                                            | `Reader`/`Writer`-style mockable capability pattern that `Cmd`/`Sub` extend                         |
| [ZE-020 Capability Attributes and script](./ZE-020-capability-attributes-and-script.md) | `@HasX` attributes are how individual `Cmd`/`Sub` implementations declare what they need from `env` |
| [ZE-009 Option Values](./ZE-009-option-values.md)                                       | `Option` backs focus state (`Option<FocusId>`)                                                      |
| [ZE-016 Closure Syntax](./ZE-016-closure-syntax.md)                                     | `lift`, `Binding`, and dispatch-wrapping all rely on first-class closures                           |
| [ZE-017 Type Hints](./ZE-017-type-hints.md)                                             | Multi-attribute hints on `Cmd.run`'s `env` parameter                                                |

## Best Practices

- **Always return `Step`, never a bare `Model`, from `init`/`update`.** Since Zirric has no multi-value returns, `Step` is the one packed result type for the whole architecture — don't invent per-component ad hoc pairings (e.g. an array `[model, cmd]`) that lose field names and nominal type-checking on `is`.
- **Never pass `Env` through `update` or `view`.** If a function needs a capability, it belongs on a `@Cmd`/`@Sub` implementation's `run` signature, not as a parameter threaded through application logic.
- **Keep `data` fields on `Cmd`/`Sub` types to domain data only** — never store a capability instance (a `Writer`, a `Clock`) on a `Cmd`. Doing so breaks equality-based test assertions and reintroduces manual threading one layer up.
- **Import `NoneCmd` from `flow`; never redeclare it per component.** A locally-declared `data NoneCmd {}` is a distinct nominal type from every other component's, and won't be interchangeable with theirs.
- **Containers defer, they never recurse eagerly.** A `Column`-like widget should wrap each child in `Widget(...)`, not call `.render(dispatch)` on it immediately — eager resolution destroys a backend's ability to give a nested widget special treatment below the root.
- **A widget's `@Renderable` implementation must always produce a valid universal fallback.** Backend-specific richness (`@TermNode`, ...) is additive and optional; the generic path must work on its own.
- **Reserve `Msg` for discrete, named events.** Use `Binding` for anything that's really a continuous value sync (text fields, sliders, checkboxes) to avoid a wrapper variant per form field.
- **Centralize exclusive state in the nearest common parent** and derive per-child flags (`isFocused`, `isSelected`) at render time — never duplicate them into child `Model`s, where they can desync.
- **Wrap child `Msg` per mount point, with an explicit key for collections** (`ItemMsg { index, msg }`), not just per component type, so sibling instances stay distinguishable.
- **Reach for `lift` before hand-writing delegation.** Manually get/call/put-back/wrap sequences are the single largest source of boilerplate in comparable architectures; use the combinator.
- **Treat `@HasX` hints on `Cmd`/`Sub.run` as documentation of least authority**, even though they are not yet statically enforced — a `TickCmd` should only ever call through `@HasClock`, by convention, until attribute enforcement lands.
- **Start with one persistent `Sub` per event source.** Don't build per-render subscription diffing until a real use case (e.g. a screen-scoped WebSocket) demonstrates it's needed.
- **Don't build a widget a terminal can't degrade to text.** If a widget cannot be meaningfully expressed as structured text, it belongs outside `ui`'s scope entirely, per Zirric's text-based UI principle.
- **Until `export` lands, import `Renderable` from `flow.node` explicitly.** Any function typed `-> @Renderable` — every component's `view`, every backend's rendering entry point — needs `import flow.node { Renderable }` alongside `import flow { ... }`. This is a known, temporary ergonomic cost, not a design flaw to work around with a looser type hint elsewhere.
- **Use `snapshot`, not `run`, for one-shot output.** Static reference docs, generated pages, and anything else that only needs a component's initial rendering should call `snapshot` directly rather than starting and somehow truncating the interactive loop — `run` has no well-defined stopping point once a subscription and asynchronous `Cmd`s are in play.
- **Prefer content-semantic widgets over hand-formatted text.** Use `Heading`/`Paragraph`/`List`/`Link`/`CodeBlock` so the same component produces sensible output on `term`, `html`, and `md` alike, instead of baking Markdown syntax or HTML tags directly into a `Label`'s text.
