# Type System

Zirric is dynamically typed but strongly typed. Values carry their type at runtime and can flow through any variable, field, or parameter. There are no implicit conversions — a `String` is never silently treated as an `Int`. Type information is communicated through [type hints](#type-hints) and checked at runtime through [`is` expressions](/specification/expressions#is-expression) and [`switch` matching](/specification/expressions#switch).

This page covers the type categories, their construction and identity rules, type hints, runtime type checking, and protocol attributes.

## Type Categories

Zirric has four categories of user-declared types and a set of built-in extern types.

| Category        | Declared with | Constructible from Zirric | Runtime identity |
| --------------- | ------------- | ------------------------- | ---------------- |
| Data types      | `data`        | Yes (constructor call)    | Nominal          |
| Union types     | `union`       | No (members are)          | Membership-based |
| Attribute types | `attr`        | Only via `@Attr()` syntax | Nominal          |
| Extern types    | `extern type` | No (runtime-provided)     | Nominal          |

For declaration syntax, see [Declarations](/specification/declarations).

## Data Types

Data types are the most common types in Zirric. They store structured data as named fields.

```zirric
data Person {
	name: String
	age: Int
}
```

**Construction.** Call the type name as a function with positional arguments matching the field order:

```zirric
const alice = Person("Alice", 30)
```

**Identity.** Each `data` declaration creates a distinct type. Two `data` types with identical fields are not the same type. Type identity is nominal — it is determined by the declaration, not the structure.

**Field access.** Fields are accessed with `.`:

```zirric
alice.name // "Alice"
```

**Equality.** Two `data` instances are equal (`==`) if they have the same type and all fields are equal.

## Union Types

Union types declare that a value may be one of a fixed set of member types. Members can be existing types or inline `data` declarations.

```zirric
union Option {
	data Some { value }
	data None
}
```

**Membership.** A value belongs to a union if its concrete type is one of the declared members. Membership is checked at runtime. A value does not need to be "wrapped" in the union — if its type is a member, it _is_ a member.

```zirric
const x = Some(42)
x is Option // true — Some is a member of Option
x is Some // true — x is directly a Some
```

**Nested declarations.** `data` types declared inside a `union` body are hoisted to module scope. They exist as standalone types and as union members simultaneously.

**Discrimination.** Union values are typically discriminated using `switch` with `is` cases:

```zirric
fn unwrap(opt: Option) {
	switch opt {
	case is Some:
		opt.value
	case is None:
		void
	case _:
		void
	}
}
```

See [Expressions § Switch](/specification/expressions#switch) for the full matching semantics.

**Unions of unions.** A union can reference another union as a member. A value belongs to the outer union if it belongs to any of its members, transitively.

## Extern Types

Extern types are built-in types provided by the runtime. They cannot be constructed from Zirric code — instances are created by literals, declarations, or runtime operations.

| Type     | Created by                  | Fields        |
| -------- | --------------------------- | ------------- |
| `Int`    | Integer literals            | —             |
| `Float`  | Float literals              | —             |
| `Bool`   | `true`, `false`             | `toggle()`    |
| `String` | String literals             | `length: Int` |
| `Char`   | Runtime operations          | —             |
| `Array`  | Array literals              | `length: Int` |
| `Dict`   | Dict literals               | `length: Int` |
| `Func`   | `fn` declarations, closures | `arity: Int`  |
| `Void`   | `void` constant             | —             |
| `Module` | `mod` declarations          | (members)     |
| `Any`    | —                           | —             |

`Any` is a special type that all values belong to. It acts as a universal supertype.

**Field access on extern types.** Some extern types expose fields (like `String.length`). These are accessed with `.` just like `data` fields.

## Attribute Types

Attribute types are declared with `attr` and follow the same field syntax as `data`. However, they differ in a critical way: attribute instances can only be created through the `@Attr(args)` application syntax on declarations or fields. They cannot be called as constructor functions at runtime.

```zirric
attr Table {
	name: String
}

@Table("people")
data Person { name }
```

**Role in the type system.** Attribute types participate in type checking through `is @Attr` expressions and `switch case is @Attr` patterns. When you write `value is @Table`, the runtime checks whether the value's _type declaration_ carries the `@Table` attribute — not whether the value itself is a `Table` instance.

For how to declare and apply attributes, see [Declarations § attr](/specification/declarations#attr) and [Declarations § Attributes on Declarations](/specification/declarations#attributes-on-declarations).

### Protocol attributes

A common pattern in Zirric is to use attributes to describe type capabilities, similar to interfaces or protocols in other languages. The prelude defines attributes like `@Countable` and `@Iterable` that the runtime uses to drive behavior.

```zirric
attr Countable {
	length(value: @Countable) -> Int
}

attr Iterable {
	iterate(value: @Iterable, yield: Func) -> Void
}
```

Types opt into a protocol by applying the attribute with a function implementation:

```zirric
@Countable(fn(v) { return v.length })
@Iterable(_arrayIterate)
extern type Array {
	length: Int
}
```

**How `for` uses `@Iterable`.** When a `for` loop iterates over a value, the runtime looks up the `@Iterable` attribute on the value's type and calls the `iterate` function. The function receives the value and a `yield` callback. It calls `yield` with each element; if `yield` returns `false`, iteration stops (implementing `break`).

**How `len` uses `@Countable`.** The built-in `len` operation looks up the `@Countable` attribute on the value's type and calls the `length` function.

### Matching on attributes

`is @Attr` checks whether a value's type carries the attribute, not whether the value is an attribute instance:

```zirric
const items = [1, 2, 3]
items is @Iterable // true — Array has @Iterable
items is @Countable // true — Array has @Countable
```

This also works in `switch`:

```zirric
switch value {
case is @Iterable:
// value's type has @Iterable
case _:
	// fallback
}
```

### Accessing attributes at runtime

Attribute values attached to a type can be accessed at runtime through the `reflect` module:

```zirric
import future.reflect

const personType = reflect.typeOf(person)
const fields = reflect.fieldsOf(personType)
const attr = reflect.attribute(fields[0], json.HasKey)
```

## Type Hints

Type hints annotate declarations and parameters with type information. They appear after `:` on fields, parameters, and variables, and after `->` on function return types.

```zirric
fn greet(name: String) -> String {
	"Hello, " + name
}

data Pair {
	first: Int
	second: Int
}

const x: Int = 42
```

### What type hints express

Type hints communicate intended types to readers, editors, and the language server. The compiler records them for `is` checks and `switch` matching. Zirric does not perform static type checking — values flow dynamically regardless of hints.

### Composite type hints

Beyond simple type references, type hints support composite forms:

| Form            | Syntax          | Meaning                                          |
| --------------- | --------------- | ------------------------------------------------ |
| Named type      | `Type`          | A reference to a declared type                   |
| Qualified type  | `mod.Type`      | A type from another module                       |
| Array type      | `[Element]`     | An array of elements                             |
| Dict type       | `[Key: Value]`  | A dictionary with key and value types            |
| Function type   | `fn(P) -> R`    | A function with parameter and return types       |
| Attribute type  | `@Attr`         | A value whose type carries `@Attr`               |
| Multi-attribute | `@Attr1 @Attr2` | A value whose type carries all listed attributes |

```zirric
fn transform(items: [Int], f: fn(Int) -> String) -> [String] {
	return for item <- items {
		f(item)
	}
}

fn process(value: @Countable @Iterable) {
	// value's type must have both attributes
}
```

### Optional and Result Shorthands

`T?` and `T!` are shorthands for the two prelude unions that wrap a value:

```zirric
fn findUser(id: Int) -> User? { ... } // returns an Option: Some(user) or None()
fn readFile(path: String) -> String! { ... } // returns a Result: Ok(text) or Err(reason)
```

`T?` asks for an [`Option`](/stdlib/prelude) and `T!` for a [`Result`](/stdlib/prelude). The element — `User`, `String` — says what a present or successful value holds, and is treated exactly as an array's element is:

- **At runtime the container is all that is checked.** `findUser(1) is User?` asks only whether the value is an `Option`, the same way `xs is [String]` asks only whether it is an `Array`. Zirric has no generic types, so there is nothing about the element for a runtime check to look at.
- **Between two written hints the element is compared.** Passing a `String?` where an `Int?` is declared is reported, as passing `[String]` where `[Int]` is declared would be. An element unknown on either side still fits, so a plain `Option` satisfies a `String?` and vice versa.

```zirric
const name: String? = Some("Ada")
const age: Int? = name // reported: age is declared Int?, got String?
const any: Option = name // fine — a bare Option says nothing about what it holds
```

The suffixes stack (`T?!` is a `Result`, the outermost suffix winning) and apply to any type expression, `[Int]?` and `fn() -> Int?` included. They must sit directly on the type, with no space between the two; see [Syntax § Type Expressions](/specification/syntax#type-expressions).

### Type hints on declarations

See [Declarations § Parameters and Type Hints](/specification/declarations#parameters-and-type-hints) for all positions where type hints can appear.

## Type Checking with `is`

The `is` operator and `switch case is` patterns perform runtime type checks. The matching rules depend on the type hint form:

| Pattern        | Matches when                                               |
| -------------- | ---------------------------------------------------------- |
| `is Type`      | The value's concrete type is `Type`                        |
| `is UnionType` | The value's concrete type is a member of the union         |
| `is @Attr`     | The value's type declaration carries the `@Attr` attribute |

`is` always produces a `Bool`. In a `switch`, the matching case's body is executed.

```zirric
42 is Int // true
42 is Number // true (Number is union { Int, Float })
42 is String // false
42 is @Numeric // true (Int has @Numeric)
```

## Common Prelude Types

The prelude defines a small set of types and unions available without explicit import.

**`Number`** — a union of `Int` and `Float`:

```zirric
union Number {
	Int
	Float
}
```

**`Void`** — represents absence of a meaningful value. The constant `void` is the single instance. Functions without an explicit return value return `void`.

**`Any`** — the universal type. Every value is an `Any`. Useful as a type hint when no constraint is needed.

The prelude also defines common attributes (`@Deprecated`, `@Default`, `@Numeric`) and protocol attributes (`@Countable`, `@Iterable`). See [Standard Library § Prelude](/stdlib/prelude) for the full list.
