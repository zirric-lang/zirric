---
title: "Zirric Styleguide"
description: "A guide to writing good and readable Zirric code."
---

# Styleguide

This guide captures the conventions used across the Zirric standard library and
proposals. Favor clarity and consistency over cleverness.

## General principles

- Prefer descriptive names over abbreviations.
- Keep files small and focused.
- Match existing style in nearby modules.

## Modules

Zirric favors many small modules over a few large ones. Keep each module focused
and consider hiding implementation details in an `internal` module.

Module names use `snake_case` but should avoid underscores when possible. Names
should be short, expressive, and usually plural. Keep module names aligned with
the declarations they contain.

```zirric
// good
mod http
mod json
mod strings
mod reflect // reflect.typeOf

// bad
mod http_client
mod json_parser
mod string_utils
mod reflection // reflection.typeOf
```

If a filename or its declarations could clash with another module, declare the
`mod` explicitly at the top of the file:

```zirric
mod my_module
```

## Data types

Data type names use `PascalCase`. Prefer nouns, pluralize only when the type is
a collection, and avoid prefixing the module name.

Witness types describe capabilities and may be suffixed with `able` or `ible`.

```zirric
// good
data Person
data People
data Printable // witness type
data Error // in module http

// bad
data person
data people
data Printer // witness type
data HttpError // in module http
```

## Functions

Function names use `camelCase`. Names should read as verbs or verb phrases and
form a sentence with the module name.

```zirric
// good
print
printLine
reflect.typeOf

// bad
print_line
reflect.reflectType
```

## Variables and Constants

Variable names use `camelCase`. Names should be as long as their scope: short
names for tight scopes, longer names for shared or public values. Reuse common
names in well-known patterns.

Prefer constants if possible.

```zirric
// good
const name = "John"
const person = Person("John", 42)
const err = http.Error("Not found")

fn printPersonName(p) {
    print(p.name)
}

// bad
const n = "John"
const p = Person("John", 42)
const error = http.Error("Not found")

fn printName(n) {
    print(n.name)
}
```

## Attributes

Annotations use `PascalCase` and should describe the property they declare.

```zirric
// good
attr Returns
attr HasKey
attr IsOptional

// bad
attr Return
attr Key
attr Optional
```

## Unions

Union names use `PascalCase`. Choose singular nouns unless the union itself is a
collection. For witness unions, describe the capability, optionally with a
`Witness` suffix.

```zirric
// good
union Stateful
union JuristicPerson
union FunctorWitness

// bad
union StateOrStore
union JuristicPersons
union Functor
```

If the members inside the union are more relevant than the union name itself, define
them at top level instead of nesting.

```zirric
// good
union Optional {
    data Some { value }
    data None
}

union Maybe {
    Optional
    Any
}

// bad
union Maybe {
    union Optional {
        data Some { value }
        data None
    }
    Any
}
```

## Imports and layout

- Group imports together near the top of the file.
- Keep one declaration per block to make attributes and docs obvious.
- Prefer blank lines to separate logical sections.
