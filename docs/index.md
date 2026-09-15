---
title: Zirric
description: A compact, expression-first language with attributes and a small runtime.
layout: "full"
toc: false
---

![Zirric](/assets/images/zirric.svg){ .size-small .align-center }

::: hero glow:true layout:split

# Explicit structure.

Zirric is a compact language that favors declarations, readable data modeling, and attributes that describe capabilities without interfaces.

::: button "Get Started" /guides/getting-started
::: button "Browse Proposals" /proposals

== side

```zirric
attr Countable {
    length(value: @Countable) -> Int
}

@Countable(fn(v) { return v.length })
data Bag {
    items
    length
}

fn summarize(bag: Bag) -> Result {
    const length = Countable(bag).length(bag)

    return if length > 0 {
        Ok(length)
    } else {
        Err("empty")
    }
}
```

:::

::: grids
::: grid
::: card "Declarations first" icon:rocket
Small set of primitives: `const`, `var`, `fn`, `data`, `union`, `attr`, `mod`, `import`.
:::
:::

::: grid
::: card "Expression-oriented" icon:code
`if` and `for` return values when you need them, so data flow stays explicit.
:::
:::

::: grid
::: card "Attributes over interfaces" icon:code
Capabilities are declared, composed, and exposed to tooling.
:::
:::

::: grid
::: card "Readable data types" icon:code
Records and tagged unions stay close to the domain and are easy to reason about.
:::
:::

:::
