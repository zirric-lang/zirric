---
title: ZE-013 - Mutability and Constants
description: Replace `let` with `const` and `var` to make mutability explicit at the declaration site.
---

# Mutability and Constants

::: callout tip Implemented
This proposal has been accepted and implemented. You can use this feature since Zirric [v0.1.0](/changelog/v0.1.0).
:::

## Introduction

This proposal replaces the `let` keyword introduced in [ZE-001](/proposals/ZE-001-base-language) with two distinct keywords — `const` for immutable bindings and `var` for mutable bindings — to make mutability explicit at the declaration site. `extern let` becomes `extern const` for the same reason.

[ZE-001](/proposals/ZE-001-base-language) did not specify mutability rules in sufficient detail, leaving parameter mutability, the semantics of reassignment, and the interaction between bindings and reference types as undefined behaviour. This proposal closes those open ends.

Function parameters are treated as constants and do not require a keyword.

## Motivation

[ZE-001](/proposals/ZE-001-base-language) introduces `let` as the declaration keyword for variables and includes an example showing reassignment (`x = 2`). The keyword `let` does not communicate whether a binding is constant or mutable — readers must scan all subsequent uses of the name to determine that. The keyword `const`, by contrast, directly describes the behavior: the binding holds a fixed value.

The same applies to extern values: `extern const void` communicates that `void` is a fixed, runtime-provided constant, while `const` makes that intent unambiguous without additional context.

Explicit mutability is a small syntactic cost that yields significant readability and analysis benefits: readers immediately know whether a name can change, and tooling can enforce immutability where declared.

Because [ZE-001](/proposals/ZE-001-base-language) is still in progress and reassignment of `let` bindings has not yet been implemented in Zirric, this is the right moment to make the change before the mutable form of `let` is ever shipped.

## Proposed Solution

Remove `let` and introduce two declarations in its place, splitting the single `let` into a constant form and a mutable form:

| Declaration         | Mutability                                                                            | Allowed position          |
| ------------------- | ------------------------------------------------------------------------------------- | ------------------------- |
| `const x = v`       | Binding is fixed — `x =` is forbidden; member and subscript mutation is still allowed | Local scope, module scope |
| `var x = v`         | Binding is rebindable — `x =`, `x.f =`, `x[i] =` are all allowed                      | Local scope, module scope |
| `extern const name` | Immutable, runtime-provided — replaces `extern let`                                   | Module scope only         |

There is no `extern var` because the runtime owns those bindings.

Function parameters are implicitly constant. No keyword is written. [ZE-001](/proposals/ZE-001-base-language) did not specify parameter mutability; this proposal establishes that parameters are always `const`.

```zirric
const x = 42       // x cannot be rebound
var counter = 0     // counter can be rebound

counter = counter + 1   // ok
x = 2                   // compile-time error: x is const

data Person { name, age }
const person = Person("John", 40)
person.age = 42         // ok — mutates the field; the binding person is not rebound
person = Person("Jane", 30) // compile-time error: person is const

fn increment(n) {     // n is implicitly const
    n = n + 1           // compile-time error: n is const
    return n
}
```

Extern constants look natural now:

```zirric
extern const void
```

## Detailed Design

### Syntax

```ebnf
decl_const = [attr_chain], "const", identifier, "=", expression ;
decl_var   = [attr_chain], "var",   identifier, "=", expression ;

decl_extern_const = "extern", "const", identifier ;

decl = [attr_chain], ( decl_const | decl_var | decl_fn | decl_union | decl_data | decl_attr | decl_extern_type | decl_extern_const | decl_extern_fn | decl_mod ) ;
```

The discard pattern `_ = expression` continues to work unchanged. `_` is special syntax and not a valid identifier; `_[0] = x` and `_.field = x` are not valid forms.

### Mutability rules

- A `const` binding prevents **rebinding** the name: `x = v` is a compile-time error after declaration. Member assignment (`x.field = v`) and subscript assignment (`x[i] = v`) are still permitted because they mutate the referenced object, not the binding itself.
- A `var` binding allows rebinding and all forms of mutation.
- Function parameters behave like `const` bindings.

These rules govern **binding reassignment** only — they are enforced syntactically by the compiler by tracking which identifier was declared with which keyword. They say nothing about the values themselves.

**Value mutation through aliasing** is a separate, semantic concern. All values — including `data`, `Array`, and `Dict` — are reference types. Any binding, `const` or `var`, that refers to the same object will observe mutations made through any other binding:

```zirric
const arr = []
var arr2 = arr
arr2[0] = 99    // ok
// arr[0] is now 99 — arr and arr2 refer to the same object

data Person { name, age }
const p = Person("Alice", 30)
var p2 = p
p2.age = 99     // ok
// p.age is now 99 — p and p2 refer to the same object
```

### Subscript assignment

This proposal introduces subscript assignment for `Array` and `Dict`. Subscript assignment mutates the collection object directly; the binding itself is not rebound, so it is permitted on both `const` and `var` bindings:

```zirric
var items = [1, 2, 3]
items[0] = 99           // ok

const frozen = [1, 2, 3]
frozen[0] = 99          // ok — the binding is not rebound
frozen = [4, 5, 6]      // compile-time error: frozen is const
```

`Array` and `Dict` are `extern` types. Subscript assignment is a dedicated operation on those types, distinct from the field assignment defined for `data` types below.

### Member assignment

Field assignment on `data` values uses **in-place mutation** semantics: the field is updated directly on the existing object. No new instance is created. Like subscript assignment, the binding itself is not rebound, so field assignment is permitted on both `const` and `var` bindings:

```zirric
data Person {
    name
    age
}

var person = Person("John", 40)
person.age = 42     // ok
person = Person("Jane", 30) // ok — var binding can be rebound

const frozen = Person("Jane", 30)
frozen.age = 99     // ok — field mutation does not rebind frozen
frozen = Person("Bob", 20)  // compile-time error: frozen is const
```

Compound assignment operators `+=`, `-=`, `*=`, `/=`, `%=` desugar to a read followed by a write on the same lvalue:

```zirric
person.age += 2     // equivalent to: person.age = person.age + 2
items[0] -= 1       // equivalent to: items[0] = items[0] - 1
```

The lvalue is resolved once: subscript expressions and member accesses in the chain are not evaluated twice. Compound assignment on a `const` binding's direct name is still forbidden (`x += 1` where `x` is `const` is a compile-time error), but compound assignment through a member or subscript is allowed.

`extern` types do not support field assignment. `Array` and `Dict` support subscript assignment as described above.

#### Chains

All mutation forms propagate in place through the entire chain. The only restriction is **direct rebinding** of a `const` root binding.

Given:

```zirric
data Person {
    name
    age
}
data PersonList { friends }  // friends is an Array of Person

var personData = PersonList([Person("Alice", 30)])
const pd = PersonList([Person("Alice", 30)])
```

All three forms work on both `var` and `const` root bindings:

```zirric
// Field on data — mutates in place; permitted on const
personData.friends = []
pd.friends = []

// Subscript on Array — evaluates pd.friends[0] and replaces the element
personData.friends[0] = Person("Bob", 25)
pd.friends[0] = Person("Bob", 25)

// Chained field + subscript — evaluates personData.friends[0] and sets its age field directly
personData.friends[0].age = 99
pd.friends[0].age = 99

// Only direct rebinding of the root is forbidden for const
personData = PersonList([])  // ok — var
pd = PersonList([])          // compile-time error: pd is const
```

`personData.friends[0].age = 99` evaluates `personData.friends[0]` to obtain a reference to the `Person` object, then sets its `age` field directly on that object. No new instance is created.

#### EBNF

```ebnf
stmt_assign = lvalue, [ augmented_op ], "=", expression ;
lvalue = identifier
       | lvalue, "[", expression, "]"
       | lvalue, ".", identifier ;
augmented_op = "+" | "-" | "*" | "/" | "%" ;
```

### Module-scope declarations

Module-scope `const` and `var` declarations are evaluated **lazily** — a binding is initialised on first access. Circular dependencies between module-scope declarations do not produce a compile-time error; they produce a runtime error when the cycle is first encountered.

Module-scope `var` bindings may be read and reassigned by any function declared in the same module.

```zirric
var requestCount = 0

fn trackRequest() {
    requestCount = requestCount + 1
}
```

### Closures and upvalues

`var` bindings that are captured by a closure (function literal or `func` defined in the same scope) must be allocated as **upvalues** — heap-allocated cells shared between the enclosing scope and all capturing closures. This ensures that mutations performed inside a closure are visible to the outer scope and vice versa.

Multiple closures capturing the same `var` binding share the same upvalue cell, so mutations made by one closure are immediately visible to all others:

```zirric
var x = 0
const inc = fn() { x = x + 1 }
const get = fn() { return x }
inc()
inc()
get()   // 2 — both closures operate on the same cell
```

`const` bindings captured by a closure capture the current reference at closure creation time; no upvalue cell is needed.

```zirric
var total = 0
const add = fn(n) {
    total = total + n   // captures `total` via upvalue
}
add(1)
add(2)
// total == 3
```

### Attributes

Both `const` and `var` declarations may carry attribute chains, just as `let` did:

```zirric
@Int
const maxRetries = 5

@String
var greeting = "Hello"
```

### Breaking changes

`let` is removed from the language. Because [ZE-001](/proposals/ZE-001-base-language) is still in progress and `let` reassignment has not yet been implemented, no existing code relies on mutable `let` bindings. The migration is therefore straightforward for current users: every `let` declaration becomes either `const` or `var`, and every `extern let` becomes `extern const`.

| Old syntax        | New syntax                   |
| ----------------- | ---------------------------- |
| `let x = v`       | `const x = v` or `var x = v` |
| `extern let name` | `extern const name`          |

The transformation is mechanical:

1. `extern let` → `extern const`
2. `let name = v` where `name` is never reassigned → `const name = v`
3. `let name = v` where `name` is intended to be reassigned → `var name = v`

Since reassignment is not yet in use, in practice all current `let` declarations map directly to `const`.

## Changes to the Standard Library

- All `extern let` declarations in the standard library (e.g., `extern let void` in `prelude/shim.zirr`) become `extern const`.
- All `let` bindings in `.zirr` source files are migrated to `const` or `var` as appropriate.
- The `decl_let` EBNF rule in the language reference is replaced by `decl_const`, `decl_var`, and `decl_extern_const`, and the top-level `decl` production is updated accordingly.
- `Array` and `Dict` gain subscript assignment support.

## Alternatives Considered

### Keep `let` as immutable, add `var` for mutable

This is the Swift/Kotlin model. It would be less disruptive (no `let` removal), but [ZE-001](/proposals/ZE-001-base-language) already documents that `let` values may be reassigned, so keeping `let` as immutable would still be a breaking change without the benefit of a more descriptive keyword. Removing `let` entirely is cleaner.

### Keep `let`, add `mut let` or `let mut`

Adding a modifier avoids a new keyword but produces verbose syntax (`let mut x = 0`) and does not address `extern let`. The two-keyword approach (`const`/`var`) is more conventional and more readable.

### Copy-on-set semantics for `data` types

An alternative to uniform in-place mutation is to give `data` types **copy-on-set** semantics: `person.age = 42` would instead be equivalent to `person = Person(person.name, 42)`, creating a new instance and reassigning the binding. `Array` and `Dict` would keep in-place mutation semantics.

This matches the model of languages like Swift (structs) and Elm (records). However, because all types in Zirric are reference types, copy-on-set for `data` would be a deliberate departure from the general object model and would create a confusing inconsistency: `personData.friends = []` would update `personData` (copy-on-set), but `personData.friends[0] = x` would not (in-place mutation), even though the two lines look nearly identical. Uniform in-place mutation avoids this inconsistency.

## Acknowledgements

- The `const`/`var` dichotomy is well-established in Go and JavaScript (ES6).
- The upvalue mechanism for mutable captured variables follows the design used in Lua and the Lua-derived Crafting Interpreters implementation by Robert Nystrom.
