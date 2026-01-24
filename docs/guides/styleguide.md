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
module http
module json
module strings
module reflect // reflect.typeOf

// bad
module http_client
module json_parser
module string_utils
module reflection // reflection.typeOf
```

If a filename or its declarations could clash with another module, declare the
`module` explicitly at the top of the file:

```zirric
module my_module
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

## Variables

Variable names use `camelCase`. Names should be as long as their scope: short
names for tight scopes, longer names for shared or public values. Reuse common
names in well-known patterns.

```zirric
// good
let name = "John"
let person = Person("John", 42)
let err = http.Error("Not found")

func printPersonName(p) {
    print(p.name)
}

// bad
let n = "John"
let p = Person("John", 42)
let error = http.Error("Not found")

func printName(n) {
    print(n.name)
}
```

## Annotations

Annotations use `PascalCase` and should describe the property they declare.

```zirric
// good
annotation Returns
annotation HasKey
annotation IsOptional

// bad
annotation Return
annotation Key
annotation Optional
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
- Keep one declaration per block to make annotations and docs obvious.
- Prefer blank lines to separate logical sections.
