# Declarations

Zirric programs are built from declarations. Every named entity — values, functions, types, modules — is introduced by a declaration. This page covers how each declaration form works, its scoping rules, and how attributes and type hints attach to declarations.

For the grammar of each form, see [Syntax § Declarations](/specification/syntax#declarations).

## `const`

`const` introduces an immutable binding. Once assigned, it cannot be reassigned. The value itself may be mutable (e.g. an array), but the binding is fixed.

```zirric
const answer = 42
const name = "Zirric"
```

**Scoping.** `const` may appear at top level (module scope) or inside a block (lexical scope). Top-level constants are visible to the entire module and to importers. Local constants are visible from the point of declaration to the end of the enclosing block.

**Closure capture.** When a closure captures a `const`, it captures the value directly. The captured value is immutable and does not require an indirection cell.

| Context    | Valid | Visibility |
| ---------- | ----- | ---------- |
| Top-level  | Yes   | Module     |
| Local      | Yes   | Lexical    |
| Expression | No    | —          |

## `var`

`var` introduces a mutable binding that can be reassigned with `=`.

```zirric
var count = 0
count = count + 1
```

**Scoping.** Same rules as `const`: top-level or local. The `_ = expr` form discards the result, which is useful for calling functions for their side effects.

**Closure capture.** When a closure captures a `var`, the compiler wraps it in an upvalue cell. Both the enclosing scope and the closure share a reference to the same cell, so mutations from either side are visible to the other. See [Expressions § Closures](/specification/expressions#closures) for details.

| Context    | Valid | Visibility |
| ---------- | ----- | ---------- |
| Top-level  | Yes   | Module     |
| Local      | Yes   | Lexical    |
| Expression | No    | —          |

## `fn`

`fn` declares a named function. The body is a block of statements. The last expression in the block is the implicit return value if no explicit `return` is used.

```zirric
fn greet(name) {
	"Hello, " + name
}
```

**Parameters** are positional. Each parameter may carry a [type hint](#parameters-and-type-hints). A **return type hint** may follow the parameter list.

```zirric
fn add(a: Int, b: Int) -> Int {
	a + b
}
```

**Scoping.** Functions can be declared at top level or inside a block. Local functions capture variables from enclosing scopes (see [Expressions § Closures](/specification/expressions#closures)).

Functions are first-class values. A declared function can be passed as an argument, stored in a variable, or returned from another function.

| Context    | Valid        | Visibility |
| ---------- | ------------ | ---------- |
| Top-level  | Yes          | Module     |
| Local      | Yes          | Lexical    |
| Expression | Closure only | —          |

## `data`

`data` declares a record-like type with named fields. Instances are created by calling the type name as a constructor function.

```zirric
data Person {
	name
	age
}

const alice = Person("Alice", 30)
```

A `data` declaration without a body declares a zero-field type:

```zirric
data None
```

**Fields.** Fields are positional — the constructor expects arguments in declaration order. Each field may have [type hints](#parameters-and-type-hints) and [attributes](#attributes-on-declarations). Function-style fields declare a parameter list:

```zirric
data Greetable {
	greeting(ofValue)
}
```

**Field access.** Fields are accessed with `.`:

```zirric
alice.name // "Alice"
alice.age // 30
```

**Scoping.** `data` can only appear at the top level or nested inside a `union`. It cannot be declared inside a function body.

| Context    | Valid | Visibility |
| ---------- | ----- | ---------- |
| Top-level  | Yes   | Module     |
| In `union` | Yes   | Module     |
| Local      | No    | —          |

## `union`

`union` declares a nominal tagged union — a type whose values may be any of a fixed set of member types. Members can be existing types referenced by name, or inline `data` declarations.

```zirric
union Option {
	data Some { value }
	data None
}

union StringOption {
	String
	None
}
```

**Nested declarations.** `data` types declared inside a `union` body are hoisted to module scope. They are accessible both as standalone types and as members of the union.

**Membership.** A value belongs to a union if its concrete type is one of the declared members. This is checked at runtime. See [Type System § Union Types](/specification/typesystem#union-types) for membership semantics and discrimination.

**No attributes on members.** Union member references (like `String` or `None` above) cannot carry attributes. Attributes go on the `data` declarations themselves.

| Context   | Valid | Visibility |
| --------- | ----- | ---------- |
| Top-level | Yes   | Module     |
| Local     | No    | —          |

## `attr`

`attr` declares an attribute type. Attributes are metadata schemas that can be attached to declarations and fields. They follow the same field syntax as `data`.

```zirric
attr Deprecated {
	reason: String
}

attr Doc {
	description: String
}
```

An `attr` without a body declares a zero-field attribute type:

```zirric
attr Marker
```

**Scoping.** Attribute declarations can only appear at the top level.

**Attr vs. data.** Both `attr` and `data` declare types with named fields, but they differ in construction: attribute instances can only be created through `@Attr(args)` application syntax, never by calling them as functions at runtime. This distinction is enforced by the compiler. See [Type System § Attribute Types](/specification/typesystem#attribute-types) for their role in the type system.

| Context   | Valid | Visibility |
| --------- | ----- | ---------- |
| Top-level | Yes   | Module     |
| Local     | No    | —          |

## Attributes on Declarations

Attributes attach metadata to declarations and fields using the `@Name(args)` syntax. Parentheses are always required, even for zero-field attributes.

```zirric
@Deprecated("use newGreet instead")
fn oldGreet(name) {
	"Hi, " + name
}
```

### Where attributes can appear

Attributes can precede:

- **Top-level declarations**: `fn`, `data`, `union`, `attr`, `extern type`, `extern fn`, `extern const`, `const`, `var`
- **Fields** of `data`, `attr`, or `extern type`
- **Nested `data`** inside a `union`

Attributes **cannot** appear on:

- Union member references (e.g. `String` in a `union` body)
- Local declarations inside function bodies
- Expressions

### Validation

Only declared `attr` types can be applied as attributes. Applying a non-attribute type — such as `@String()` or `@Int()` — is a compile error. Type information on fields and parameters uses [type hints](#parameters-and-type-hints) instead.

### Multiple attributes

Multiple attributes may be stacked on a single declaration or field:

```zirric
@Doc("The user's full name")
@Deprecated("use displayName")
data LegacyUser {
	@Doc("First and last name")
	name: String
}
```

### Qualified attributes

Attributes from other modules are accessed through their qualified name:

```zirric
import json

@json.HasKey("user_name")
data User { name: String }
```

## `extern` Declarations

`extern` declarations introduce entities that are provided by the runtime or compiler extensions, not defined in Zirric source.

### `extern type`

Declares an opaque type whose implementation is in the runtime. Fields may be declared for access but the type cannot be constructed from Zirric code.

```zirric
extern type String {
	length: Int
}
```

### `extern fn`

Declares a function provided by the runtime.

```zirric
extern fn print(msg)
```

### `extern const`

Declares a constant provided by the runtime.

```zirric
extern const void: Void
```

All `extern` declarations can only appear at top level.

| Context   | Valid | Visibility |
| --------- | ----- | ---------- |
| Top-level | Yes   | Module     |
| Local     | No    | —          |

## `mod`

`mod` declares a reference to the current module as a value. It must be the first declaration in the file (after any shebang line). While not required, it is conventional to declare it for readability.

```zirric
mod http
```

The declared name becomes an identifier bound to the module object. This is useful for qualified attribute references when the module defines attributes.

| Context         | Valid | Visibility |
| --------------- | ----- | ---------- |
| First in file   | Yes   | File       |
| Other positions | No    | —          |

## `import`

`import` makes another module's declarations available in the current file.

```zirric
import prelude
import tasks { Call }
import xprelude = future.prelude
```

**Module path.** The import path is a dot-separated sequence of identifiers.

**Alias.** The last segment of the path is the default alias. An explicit alias can be provided with `alias = path`.

**Member imports.** Specific members can be imported directly into scope with `{ Name, OtherName }`. Without member imports, declarations are accessed through the module alias (e.g. `tasks.Call`).

**Prelude.** The `prelude` module is automatically available — its members can be used without an explicit import.

**Visibility.** Declarations whose names start with `_` are considered private and are not exported from a module.

| Context   | Valid | Visibility |
| --------- | ----- | ---------- |
| Top-level | Yes   | File       |
| Local     | No    | —          |

## Parameters and Type Hints

Parameters appear in `fn`, closures, `extern fn`, and function-style data fields. Each parameter is an identifier optionally followed by a type hint.

```zirric
fn greet(name: String) -> String {
	"Hello, " + name
}
```

### Type hint positions

Type hints can appear in several positions on declarations:

| Position            | Syntax           | Example                   |
| ------------------- | ---------------- | ------------------------- |
| Parameter           | `name: Type`     | `fn f(x: Int) { ... }`    |
| Field               | `name: Type`     | `data P { name: String }` |
| Return type         | `-> Type`        | `fn f() -> Bool { ... }`  |
| Variable / constant | `name: Type = …` | `const x: Int = 42`       |
| Extern const        | `name: Type`     | `extern const void: Void` |

### What type hints express

Type hints are documentation and tooling aids. They communicate the intended type to readers, editors, and the language server. The compiler records them for runtime `is` checks and `switch` matching, but Zirric does not perform static type checking — values may still flow dynamically.

For the full type hint grammar and composite forms (`[T]`, `[K: V]`, `fn(P) -> R`, `@Attr`), see [Syntax § Type Expressions](/specification/syntax#type-expressions) and [Type System § Type Hints](/specification/typesystem#type-hints).
