---
title: "Zirric Styleguide"
description: "A guide to writing good and readable Zirric code."
---

# Styleguide

This guide captures the conventions used across the Zirric standard library. Favor clarity and consistency over cleverness.

Whitespace is not covered here, because it is not a matter of taste: run [`zirric fmt`](/tooling/code-formatter), which writes the canonical layout and has no style options. It leaves your line breaks alone, so how you group arguments and chain calls is still yours to decide.

## General principles

- Prefer descriptive names over abbreviations.
- Keep files small and focused.
- Match existing style in nearby modules.

## Modules

Zirric favors many small modules over a few large ones. Every source file declares the module it belongs to, by its fully qualified path — the package's base module path joined with the directory the file sits in. The last segment binds the module locally, so it is that segment a reader actually uses, and that segment worth choosing well.

Segment names use `snake_case` but should avoid underscores when possible. They should be short, expressive, and aligned with the declarations they contain.

Plural or singular follows what the module is about. A module of operations over one kind of value is plural; a module covering a domain or a capability is singular.

```zirric
// good
mod code.knabel.dev.zirric_lang.zirric.strings // operations on many strings
mod code.knabel.dev.zirric_lang.zirric.paths
mod code.knabel.dev.zirric_lang.zirric.results

mod code.knabel.dev.zirric_lang.zirric.io // a domain
mod code.knabel.dev.zirric_lang.zirric.math
mod code.knabel.dev.zirric_lang.zirric.reflect // reflect.typeOf

// bad
mod myapp.string_utils
mod myapp.json_parser
mod myapp.reflection // reflection.typeOf
```

Reach for `mod local = path` only when the last segment would collide with something the file already declares. A local name that differs from the module's own is one more thing a reader has to keep in mind.

```zirric
mod here = myapp.flow.node
```

### Submodules

A directory inside a module is a submodule, imported under a dotted path. Use one when a part of the module is large enough to stand alone but only makes sense next to its parent.

```zirric
import tests.runner
import reflect.packages

// rename when the qualified name gets long
import pkgs = reflect.packages
```

### Private declarations

A declaration whose name starts with `_` is not exported from its module. This is a language rule, not a hint, so reach for it instead of inventing an `internal` module.

```zirric
// good
extern fn _firstIndexOf(v: Like, needle: Like) -> Int

fn firstIndexOf(v: Like, needle: Like) -> Option {
	const index = _firstIndexOf(v, needle)
	// ...
}
```

Use it for the raw building block behind a friendlier public function, and for helpers that exist only to implement an attribute.
It is recommended to keep the module surface small.

## Data types

Data type names use `PascalCase`. Prefer nouns, pluralize only when the type is a collection, and avoid prefixing the module name.

```zirric
// good
data Person
data People
data Error // in module http

// bad
data person
data people
data HttpError // in module http
```

Capabilities are not data types — they are [attributes](#attributes). A type does not implement `Printable`; it carries `@Printable`.

## Unions

Union names use `PascalCase`. Choose singular nouns unless the union itself is a collection.

```zirric
// good
union Stateful {}
union JuristicPerson {}

// bad
union StateOrStore {}
union JuristicPersons {}
```

A union of the types a function will accept interchangeably is named `Like`, so it reads as a constraint at the call site.
This is especially useful in modules that are built around a type.

```zirric
// in module strings
union Like {
	Char
	String
}

fn isEmpty(v: Like) -> Bool
```

If the members inside the union are more relevant than the union name itself, define them at top level instead of nesting.

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

## Functions

Function names use `camelCase`. Names should read as verbs or verb phrases and form a sentence with the module name.

```zirric
// good
print
printLine
reflect.typeOf

// bad
print_line
reflect.reflectType
```

### The subject comes first

A function that operates on a value takes it as the first parameter. The module name then reads as the namespace and the first argument as the subject.

```zirric
// good
fs.readFile(fsys, "notes.txt")
arrays.map(items, transform)
random.int(source, 100)

// bad
fs.readFile("notes.txt", fsys)
```

Where a `data` type holds its behavior in fields, give each field a matching module function that takes the value first. Both spellings work, and the function form is the one that reads well in a chain.

```zirric
data Source {
	int: fn(Int) -> Int
	float: fn() -> Float
}

fn int(source: Source, bound: Int) -> Int {
	source.int(bound)
}
```

## Attributes

Attributes use `PascalCase`. Which shape to use depends on what the attribute is for:

| Shape     | Role                                  | Examples                             |
| --------- | ------------------------------------- | ------------------------------------ |
| Bare noun | Sets a value on a declaration         | `Name`, `Default`, `Version`, `Flag` |
| `Has…`    | The environment provides a capability | `HasFileSystem`, `HasSystemClock`    |
| `-able`   | The value itself can do something     | `Printable`, `Countable`, `Iterable` |
| `Any…`    | The union follows a known shape       | `AnyResult`, `AnyOption`             |

```zirric
// good
attr Name { text } // in module coding: sets the key a field travels under
attr HasFileSystem {
	fileSystem(self: @HasFileSystem) -> FileSystem
}

// bad
attr HasName { text } // not a capability
attr FileSystem { // reads as metadata, not as a requirement
	fileSystem(self) -> FileSystem
}
```

`Has…` is the shape a function requires of its environment, which is what keeps a dependency visible in the signature:

```zirric
fn report(env: @HasStandardWriter, value: @Printable) {
	const writer = HasStandardWriter(env).writer(env)
	fmt.fprintln(value, writer)
}
```

## Variables and constants

Variable names use `camelCase`. Names should be as long as their scope: short names for tight scopes, longer names for shared or public values. Reuse common names in well-known patterns.

Prefer constants if possible.

```zirric
// good
const name = "John"
const person = Person("John", 42)
const err = http.Error("Not found")

fn printPersonName(p) {
	println(p.name)
}

// bad
const n = "John"
const p = Person("John", 42)
const error = http.Error("Not found")

fn printName(n) {
	println(n.name)
}
```

## Errors

A failure the caller could reasonably expect is a [`Result`](/stdlib/results), never a `panic`. Reserve `panic` for a bug in the calling code — an index outside an array, an invariant that cannot hold — where continuing would only hide the mistake.

```zirric
// good
fn readConfig(fsys: FileSystem) -> Result {
	fs.readString(fsys, "config.yaml")
}

// bad
fn readConfig(fsys: FileSystem) -> String {
	const result = fs.readString(fsys, "config.yaml")
	switch result {
	case is Ok:
		return result.value
	case is Err:
		panic("no config") // the caller could have handled this
	}
}
```

Error values carry [`@Error`](/stdlib/prelude#error), which is how they render in diagnostics. Add context as it travels rather than replacing it:

```zirric
results.mapErr(result, fn(reason) { errors.wrap(reason, "loading config") })
```

## Imports and layout

- Group imports together near the top of the file.
- Keep one declaration per block to make attributes and docs obvious.
- Prefer blank lines to separate logical sections.

Document declarations with `//` comments directly above them. Avoid mid-sentence line breaks. The first line should be the most important one.
Comments should be written in markdown.

```zirric
// Returns the number of characters (runes) in v.
// Unlike len(), which counts bytes, this counts UTF-8 code points:
// len("café") is 5, count("café") is 4.
extern fn count(v: Like) -> Int
```
