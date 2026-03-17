# Declarations

Zirric is driven by declarations. These constructs define values, types, and modules and are annotated with metadata.

## `var`

`var` introduces a variable. Variables are scoped and may be reassigned.

```zirric
var count = 0
count = count + 1
```

`_ = expr` explicitly discards a value.

| Declaration Scope | Valid | Access Scope |
| ----------------- | ----- | ------------ |
| Top-level         | Yes   | Module       |
| Local             | Yes   | Lexical      |
| Expression        | No    | -            |

## `const`

`const` introduces a constant. Constsants are scoped and cannot be reassigned.
But the values they refer to can be mutated if they are mutable types.

```zirric
const count = 0
count = count + 1
```

`_ = expr` explicitly discards a value.

| Declaration Scope | Valid | Access Scope |
| ----------------- | ----- | ------------ |
| Top-level         | Yes   | Module       |
| Local             | Yes   | Lexical      |
| Expression        | No    | -            |

## `fn`

Functions are declared with `fn name(params) { ... }`.

```zirric
fn greet(name) {
    "Hello, " + name
}
```

| Declaration Scope | Valid       | Access Scope |
| ----------------- | ----------- | ------------ |
| Top-level         | Yes         | Module       |
| Local             | Yes         | Lexical      |
| Expression        | Lambda only | -            |

## `data`

`data` declares record-like types with named fields.

```zirric
data Person {
    name
    age
}
```

Instances are created by calling the type name as a function.

```zirric
let alice = Person("Alice", 30)
```

| Declaration Scope | Valid | Access Scope |
| ----------------- | ----- | ------------ |
| Top-level         | Yes   | Module       |
| Local             | No    | -            |
| Expression        | No    | -            |

## `union`

`union` defines tagged unions. Inside a union declaration other `data` and `union` types may be declared. These will be available at top level.

```zirric
// union with nested declarations
union Option {
    data Some { value }
    data None { }
}

// union that references other declarations
union StringOption {
    String
    None // resuloves to the `None` defined above
}
```

| Declaration Scope | Valid | Access Scope |
| ----------------- | ----- | ------------ |
| Top-level         | Yes   | Module       |
| Local             | No    | -            |
| Expression        | No    | -            |

## `extern const`

`extern const` declares runtime-provided constants.
A compiler extension must provide the implementation for these declarations.

```zirric
extern const void
```

| Declaration Scope | Valid | Access Scope |
| ----------------- | ----- | ------------ |
| Top-level         | Yes   | Module       |
| Local             | No    | -            |
| Expression        | No    | -            |

## `extern fn`

`extern fn` declares runtime-provided function.
A compiler extension must provide the implementation for these declarations.

```zirric
extern fn print(msg)
```

| Declaration Scope | Valid | Access Scope |
| ----------------- | ----- | ------------ |
| Top-level         | Yes   | Module       |
| Local             | No    | -            |
| Expression        | No    | -            |

## `extern type`

`extern` declares runtime-provided types. In contrast to `data` types, these are opaque and immutable through Zirric code.
A compiler extension must provide the implementation for these declarations.

```zirric
extern type String {
    length
}
```

| Declaration Scope | Valid | Access Scope |
| ----------------- | ----- | ------------ |
| Top-level         | Yes   | Module       |
| Local             | No    | -            |
| Expression        | No    | -            |

## `attr`

`attr` defines metadata schemas used by tooling and the runtime.
Attributes can only be created statically.

```zirric
attr Returns {
    @Type(AnyType) type
}
```

| Declaration Scope | Valid | Access Scope |
| ----------------- | ----- | ------------ |
| Top-level         | Yes   | Module       |
| Local             | No    | -            |
| Expression        | No    | -            |

### Attributes on declarations

Attributes attach metadata and behavior to declarations and fields.

```zirric
@Deprecated("use newFn")
fn oldFn() { ... }

@Returns(String)
fn greet(@Type(String) name) { ... }

@SomeAttr()
union StringOption {
    // no annotation allowed here
    String
    @ToBool({ _ -> true })
    data None {}
}

@SomeAttr()
data Person {
    @SomeFieldAttr()
    name
    age
}
```

## `mod`

A `mod` declaration creates a reference to the current module that can be used as value.
While not required, it is often declared for readability even if unused.

```zirric
mod http
```

| Declaration Scope | Valid | Access Scope |
| ----------------- | ----- | ------------ |
| First top-level   | Yes   | File         |
| Top-level         | No    | -            |
| Local             | No    | -            |
| Expression        | No    | -            |

## `import`

An `import` declaration allows the access to other modules. All module scoped declarations are accessible through the imported module except the ones beginning with `_`.
Imports are qualified. Specific members can be imported directly.

The last name of an import path is the default alias for the imported module. Aliases can be defined explicitly to avoid name clashes or for readability.

For standard library modules like `prelude` or `future` a short form of the import path can be used.
Therefore an import of `prelude` resolved to `code.knabel.dev.zirric_lang.zirric.prelude`.

```zirric
import prelude // fully qualified: code.knabel.dev.zirric_lang.zirric.prelude
import code.knabel.dev.zirric_lang.zirric.future.tasks {
    Call // member imports allow direct import
}
// import with alias
import xprelude = code.knabel.dev.zirric_lang.zirric.future.prelude

_ = String // Special case: prelude members are automatically imported
_ = prelude.String // Qualified access through the import
_ = xprelude.Error // Qualified access through the alias

@tasks.Name("do") // Qualified member
@Call({ -> }) // Directly imported member
data DoTask {}
```

| Declaration Scope | Valid | Access Scope |
| ----------------- | ----- | ------------ |
| Top-level         | Yes   | File         |
| Local             | No    | -            |
| Expression        | No    | -            |
